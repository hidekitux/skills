---
name: write-docs
description: Write or update a project's documentation, such as a README, a guide, a how-to, a contributor document, or a reference page, with every claim checked against the code and the project's own writing rules applied, then hand the change to implement-issue. Use it whenever a user asks to write, update, fix, or bring documentation up to date, including when a README or guide no longer matches the code, even if the request does not mention sources or verification. Do not use it to change code, configuration, or behavior.
license: Apache-2.0
---

# Write Docs

## Todo List

1. **in progress:** Resolve the document, its readers, its scope, and the project's writing rules.
2. Collect the source in the repository for every claim the document will make.
3. Write or update the document within the agreed scope.
4. Verify every claim, command, path, and link, then complete the list only when each claim is verified or listed as unverified; hand off the change to `implement-issue`.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Resolve and Guard

- This is a document-layer skill. It writes documentation files in the working
  tree and hands the change to `implement-issue`, which commits it on the Issue
  branch. It does not commit, create a branch, an Issue, or a Pull Request.
- Resolve the target file to write or change, who reads it, and what it must
  cover, from the request or its Issue. Do not widen the scope to other files.
- Change documentation only. When the request pairs a documentation change
  with a code, configuration, or behavior change, write the documentation for
  the code as it is today, leave the other change undone, record it, and hand
  it to `implement-issue`. Do not describe the requested behavior as if it
  existed.
- Stop and ask only when you cannot write the documentation without a decision
  that is not about wording, such as choosing between two conflicting
  behaviors in the code or stating a policy the project has not set.
- Find the project's writing rules before writing: a style guide, a glossary,
  `CONTRIBUTING.md`, or a linter configuration such as `.markdownlint.yaml` or
  `.vale.ini`. They take precedence; see "Writing quality" for the rules that
  apply when the project has none.

## Collect sources

- For each claim the document will make, find its source: the function
  signature, the command definition, the configuration key, the default value,
  or the test that shows the behavior. Read the source, not an older document.
- Prefer the project's own entry points in examples, such as a `mise.toml` task
  or a `Makefile` target, over raw tool commands, because readers run what the
  project supports.

## Write

- State what the reader can do and how, with the command or code they need.
- Name each thing in full on first mention and use that term throughout.
  Use the project's glossary terms.
- Keep examples short and runnable as written.
- Keep the existing structure and headings unless the request asks to change
  them.

## Verify

- Run each command the document shows when it is safe to run, or check its
  definition when it is not, and record the result.
- Check that every path and file name exists, and that every internal link and
  anchor resolves.
- Run the project's documentation checks when it has them, such as a Markdown
  linter or a link checker.
- List every claim you could not verify, with the reason.

## Handoff

Report the changed files, a table of claims with their sources, every claim
left unverified, and every request left undone because it was not
documentation. Hand the working-tree change and this record to
`implement-issue`.

## Writing quality

These rules bind the documentation files this skill writes into the project
and the handoff report. Follow the project's own writing rules. Where the
project states none, follow [persistent prose](references/persistent-prose.md).
