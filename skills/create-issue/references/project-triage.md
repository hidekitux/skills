# Project triage

- Read the repository-declared Project configuration at `.github/issue-project.toml` before creating the Issue. It names the Project and the Status, Priority, and Scope fields with their options and the default Priority. A missing or invalid configuration is a blocker; report it instead of guessing.
- After the Issue exists, add it to the declared Project exactly once. Confirm it has no existing item with `gh project item-list`, then add it by URL with `gh project item-add`; when an item already exists, reuse it and never create a duplicate.
- Set Status to `Backlog`. Derive Scope from the Issue type: Feature→`Feature`, Bug→`Bug`, Documentation→`Docs`, Maintenance→`Maintenance`, Improvement→`Improvement`, Security→`Security`, Release→`Release`. Set the user-selected Priority, or the declared default when the user has no preference.
- Resolve Project number, field IDs, and option IDs from the declared names at runtime with `gh project list` and `gh project field-list`; never hard-code this repository's Project identity or IDs.
- Apply each value with `gh project item-edit` using the declared field name and option name (one call per field). Fail safely when Project access is unavailable or the configuration is ambiguous: do not mutate, and report the exact diagnostic.
- Report the Issue URL plus the resulting Project item and its Status, Priority, and Scope values in the handoff.
