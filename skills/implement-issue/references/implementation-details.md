# Implementation details

Read this reference when an Issue has no plan or when a task is ready to commit.

## No-plan exemption

Proceed without a plan only when both conditions are recorded in the handoff:

1. The Issue states an established cause and repository or Issue evidence confirms it.
2. The change has exactly one defensible implementation approach.

If either condition fails, route to `plan-issue`. Re-evaluate when implementation reveals a decision with more than one defensible answer.

## Commit

- Commit each completed task on the Issue branch at task granularity and stage only that task's files.
- Use a single-sentence header `type: summary #<number>` from the repository commitlint enum. Do not create validation-only commits; fix failures in the task commit.
- Record the commit hash with the task's files, commands, and results. Do not push; `create-pr` owns publication.
