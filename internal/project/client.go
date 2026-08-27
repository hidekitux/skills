package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes one gh CLI command and returns its combined output.
type Runner interface {
	Run(args ...string) (string, error)
}

// GH runs gh through the local binary.
type GH struct{}

// Run executes gh with the given arguments, keeping the process output in
// the error so scope and API failures stay actionable and classifiable.
func (GH) Run(args ...string) (string, error) {
	out, err := exec.Command("gh", args...).CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		if output != "" {
			return string(out), fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, output)
		}
		return string(out), fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// AccessError reports that Project access is unavailable so callers can fail
// safely instead of mutating live data.
type AccessError struct {
	Message string
}

func (e *AccessError) Error() string { return e.Message }

// Client performs Project reads and idempotent mutations for one owner.
type Client struct {
	Run   Runner
	Owner string
}

// NewClient returns a Client for the given owner and command runner.
func NewClient(run Runner, owner string) *Client {
	return &Client{Run: run, Owner: owner}
}

// run executes a gh command, classifying missing-scope failures as an
// AccessError so live operations fail safely with actionable diagnostics.
func (c *Client) run(args ...string) (string, error) {
	out, err := c.Run.Run(args...)
	if err != nil {
		if strings.Contains(err.Error(), "missing required scopes") {
			return "", &AccessError{Message: "GitHub Project access is unavailable: " +
				"authentication lacks the required scopes (read:project for reads, " +
				"project for mutations); refresh with: gh auth refresh -s read:project,project"}
		}
		return "", err
	}
	return out, nil
}

type projectListEntry struct {
	Number int64  `json:"number"`
	Title  string `json:"title"`
	ID     string `json:"id"`
}

type projectListDTO struct {
	Projects []projectListEntry `json:"projects"`
}

// ProjectTarget resolves the declared Project number and node id, preferring
// the configured number and otherwise requiring the title to match exactly
// one Project visible to the current authentication.
func (c *Client) ProjectTarget(cfg *Config) (int64, string, error) {
	if cfg.Project.Number != 0 {
		out, err := c.run("project", "view", fmt.Sprint(cfg.Project.Number), "--owner", c.Owner, "--format", "json")
		if err != nil {
			return 0, "", err
		}
		var view struct {
			Number int64  `json:"number"`
			ID     string `json:"id"`
		}
		if err := json.Unmarshal([]byte(out), &view); err != nil {
			return 0, "", fmt.Errorf("project view output cannot be parsed: %w", err)
		}
		if view.ID == "" {
			return 0, "", errors.New("project view returned no project id")
		}
		return view.Number, view.ID, nil
	}
	out, err := c.run("project", "list", "--owner", c.Owner, "--format", "json")
	if err != nil {
		return 0, "", err
	}
	var list projectListDTO
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return 0, "", fmt.Errorf("project list output cannot be parsed: %w", err)
	}
	matches := []projectListEntry{}
	for _, project := range list.Projects {
		if project.Title == cfg.Project.Title {
			matches = append(matches, project)
		}
	}
	switch len(matches) {
	case 0:
		return 0, "", fmt.Errorf("no Project titled %q found for owner %q", cfg.Project.Title, c.Owner)
	case 1:
		return matches[0].Number, matches[0].ID, nil
	default:
		return 0, "", fmt.Errorf("Project title %q for owner %q is ambiguous; found %d Projects",
			cfg.Project.Title, c.Owner, len(matches))
	}
}

// ProjectNumber resolves the declared Project number.
func (c *Client) ProjectNumber(cfg *Config) (int64, error) {
	number, _, err := c.ProjectTarget(cfg)
	return number, err
}

type fieldOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type fieldDTO struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Options []fieldOption `json:"options"`
}

type fieldListDTO struct {
	Fields []fieldDTO `json:"fields"`
}

// resolvedFields maps every field role to its live IDs and the declared
// options with their live IDs, failing before any mutation when a declared
// field or option is missing from the configured Project.
func (c *Client) resolvedFields(cfg *Config, number int64) (map[string]fieldDTO, error) {
	out, err := c.run("project", "field-list", fmt.Sprint(number), "--owner", c.Owner, "--format", "json")
	if err != nil {
		return nil, err
	}
	var list fieldListDTO
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("project field-list output cannot be parsed: %w", err)
	}
	byName := map[string]fieldDTO{}
	for _, field := range list.Fields {
		byName[field.Name] = field
	}
	resolved := map[string]fieldDTO{}
	for _, role := range RequiredFields {
		declared, _ := cfg.Field(role)
		live, ok := byName[declared.Name]
		if !ok {
			return nil, fmt.Errorf("Project field %q declared as fields.%s is missing from the configured Project",
				declared.Name, role)
		}
		liveOptionIDs := map[string]string{}
		for _, option := range live.Options {
			liveOptionIDs[option.Name] = option.ID
		}
		for _, option := range declared.Options {
			if _, ok := liveOptionIDs[option]; !ok {
				return nil, fmt.Errorf("option %q for Project field %q is missing from the configured Project",
					option, declared.Name)
			}
		}
		live.Options = nil
		for _, option := range declared.Options {
			live.Options = append(live.Options, fieldOption{ID: liveOptionIDs[option], Name: option})
		}
		resolved[role] = live
	}
	return resolved, nil
}

// optionID returns the live option ID for a resolved field role.
func optionID(fields map[string]fieldDTO, role, option string) (string, error) {
	field, ok := fields[role]
	if !ok {
		return "", fmt.Errorf("field role %q is unresolved", role)
	}
	for _, candidate := range field.Options {
		if candidate.Name == option {
			return candidate.ID, nil
		}
	}
	return "", fmt.Errorf("option %q is not configured for field role %q", option, role)
}

// itemLimit keeps one Project item-list read bounded while covering the
// supported migration size.
const itemLimit = "100"

// fieldValueDTO is one entry of the fieldValues array that older gh versions
// emit; newer gh versions flatten known single-select values into top-level
// item fields instead.
type fieldValueDTO struct {
	Field struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"field"`
	OptionID string `json:"optionId"`
	Value    any    `json:"value"`
}

// itemDTO is one Project item from "gh project item-list". Newer gh versions
// expose the Status, Priority, and Scope option names as top-level string
// fields; older versions nest them in fieldValues.
type itemDTO struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Priority string `json:"priority"`
	Scope    string `json:"scope"`
	Content  struct {
		URL string `json:"url"`
	} `json:"content"`
	FieldValues []fieldValueDTO `json:"fieldValues"`
}

type itemListDTO struct {
	Items []itemDTO `json:"items"`
}

// Items returns every Project item with its content URL and field values.
func (c *Client) Items(number int64) ([]itemDTO, error) {
	out, err := c.run("project", "item-list", fmt.Sprint(number), "--owner", c.Owner, "--limit", itemLimit, "--format", "json")
	if err != nil {
		return nil, err
	}
	var list itemListDTO
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, fmt.Errorf("project item-list output cannot be parsed: %w", err)
	}
	return list.Items, nil
}

// addItemUnchecked adds the Issue to the Project without a duplicate
// pre-check. Callers that already listed items must guarantee the Issue has
// no existing item before calling it.
func (c *Client) addItemUnchecked(number int64, issueURL string) (string, error) {
	out, err := c.run("project", "item-add", fmt.Sprint(number), "--owner", c.Owner, "--url", issueURL, "--format", "json")
	if err != nil {
		return "", err
	}
	var item itemDTO
	if err := json.Unmarshal([]byte(out), &item); err != nil {
		return "", fmt.Errorf("project item-add output cannot be parsed: %w", err)
	}
	if item.ID == "" {
		return "", errors.New("project item-add returned no item id")
	}
	return item.ID, nil
}

// ItemForIssue returns the sole Project item for an Issue URL or an error.
// A missing item returns ok=false; more than one item is an ambiguity error.
func (c *Client) ItemForIssue(number int64, issueURL string) (itemDTO, bool, error) {
	out, err := c.run("project", "item-list", fmt.Sprint(number), "--owner", c.Owner, "--limit", itemLimit, "--format", "json")
	if err != nil {
		return itemDTO{}, false, err
	}
	var list itemListDTO
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return itemDTO{}, false, fmt.Errorf("project item-list output cannot be parsed: %w", err)
	}
	items := []itemDTO{}
	for _, item := range list.Items {
		if item.Content.URL == issueURL {
			items = append(items, item)
		}
	}
	switch len(items) {
	case 0:
		return itemDTO{}, false, nil
	case 1:
		return items[0], true, nil
	default:
		return itemDTO{}, false, fmt.Errorf("Issue %s has %d Project items; exactly one is required", issueURL, len(items))
	}
}

// ItemIDForIssue returns the sole Project item id for an Issue URL, or ""
// when the Issue has no item. More than one item is an ambiguity error.
func (c *Client) ItemIDForIssue(number int64, issueURL string) (string, error) {
	item, ok, err := c.ItemForIssue(number, issueURL)
	if err != nil || !ok {
		return "", err
	}
	return item.ID, nil
}

// AddItem adds the Issue exactly once, reusing the sole existing item when it
// is already present.
func (c *Client) AddItem(number int64, issueURL string) (string, error) {
	existing, err := c.ItemIDForIssue(number, issueURL)
	if err != nil {
		return "", err
	}
	if existing != "" {
		return existing, nil
	}
	return c.addItemUnchecked(number, issueURL)
}

// SetSingleSelect sets one single-select field value on a Project item by
// GraphQL node IDs, the documented machine form of `gh project item-edit`.
func (c *Client) SetSingleSelect(projectID, itemID, fieldID, optionID string) error {
	_, err := c.run("project", "item-edit", "--id", itemID, "--field-id", fieldID,
		"--project-id", projectID, "--single-select-option-id", optionID)
	return err
}

// FieldValues is the Project field state for one Issue.
type FieldValues struct {
	Status   string
	Priority string
	Scope    string
}

// itemFieldNames resolves the current option name for every field role on an
// item, preferring the flattened status/priority/scope fields newer gh
// versions emit and falling back to the fieldValues array.
func itemFieldNames(item itemDTO, fields map[string]fieldDTO) (map[string]string, error) {
	flattened := map[string]string{
		"status":   item.Status,
		"priority": item.Priority,
		"scope":    item.Scope,
	}
	optionNames := map[string]string{}
	for _, role := range RequiredFields {
		field := fields[role]
		for _, option := range field.Options {
			optionNames[option.ID] = option.Name
		}
	}
	current := map[string]string{}
	for _, role := range RequiredFields {
		if value := flattened[role]; value != "" {
			current[role] = value
			continue
		}
		declaredName := fields[role].Name
		for _, fieldValue := range item.FieldValues {
			if fieldValue.Field.Name != declaredName {
				continue
			}
			if fieldValue.OptionID != "" {
				name, ok := optionNames[fieldValue.OptionID]
				if !ok {
					return nil, fmt.Errorf("Project item carries an unknown option id %q for field %q",
						fieldValue.OptionID, declaredName)
				}
				current[role] = name
				break
			}
			if text, ok := fieldValue.Value.(string); ok && text != "" {
				current[role] = text
				break
			}
		}
	}
	return current, nil
}
