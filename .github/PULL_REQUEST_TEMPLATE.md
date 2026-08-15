## Issue

<!-- Keep this section first. Start change work with the branch's matching Closes #123 line, then add one Closes line per additional Issue; keep every reference here. Use Tracks #123 only for releases. -->

## Summary

<!-- What capability or repository behavior changes? -->

<!-- Title: [Type]: Summary in sentence case. It need not match any linked Issue title. -->

## Validation

<!-- List every command or evidence with its actual result. -->

## Skill checklist

- [ ] `name` matches the skill directory name.
- [ ] `description` explains both the capability and when to use it.
- [ ] The skill creates and maintains a Todo List, with a portable Markdown fallback.
- [ ] Todo List completion has observable evidence and matches the handoff.
- [ ] Scripts and `allowed-tools` were reviewed.
- [ ] No secrets, tokens, or private user data are included.
- [ ] `CATALOG.yml` was updated when a publishable skill changed.
- [ ] `gh skill publish --dry-run` passed.
- [ ] When an FSL specification changed, `mise run mutate-fsl` was run and survivors were reviewed.
