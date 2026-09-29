package project

// Issue is one triage input record from `gh issue list`.
type Issue struct {
	Number int64        `json:"number"`
	Title  string       `json:"title"`
	State  string       `json:"state"`
	URL    string       `json:"url"`
	Labels []issueLabel `json:"labels"`
}

type issueLabel struct {
	Name string `json:"name"`
}
