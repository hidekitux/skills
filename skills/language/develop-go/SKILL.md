---
name: develop-go
description: Apply a Go project's own conventions, build commands, and test commands while writing or changing Go code, and verify the change with the project's formatter, vet, and tests. Use it whenever a task writes, changes, or tests Go code in a module with a go.mod, such as adding a function or package, fixing a failing go test, or updating go.mod dependencies, even when the request does not mention conventions or verification. Do not use it for a project without Go code.
license: Apache-2.0
---

# Develop Go

## Todo List

1. **in progress:** Confirm the Go module, its Go version, and the project's own commands.
2. Make the requested Go change following the project's conventions.
3. Verify the change with the project's formatter, vet, and tests, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives Go
knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the Go commands and conventions.

Stop and say that this skill does not apply when the repository has no
`go.mod` and no `.go` files.

## 1. Discover the project

- Find the module root from `go.mod` and read its `module` path, its `go`
  version, and any `toolchain` line. Read `go.work` when it exists; a workspace
  changes which modules a command builds.
- Find the project's own entry point before using raw `go` commands: a
  `mise.toml` task, a `Makefile` target, a `Taskfile.yml` task, or scripts in
  the repository's contributor documentation. Prefer that entry point, because
  it carries the project's flags, build tags, and environment; a raw
  `go test ./...` can pass locally while the project's own check fails.
- Note the checks the project already runs, such as `staticcheck`,
  `golangci-lint`, or `govulncheck`, from its task files or CI workflows. Do not
  add a linter the project does not use.

## 2. Change the code

- Follow the package layout already in use. Put code that other modules must
  not import under `internal/`.
- Keep package names short, lower case, and without underscores. Name
  exported identifiers for how callers read them, such as `bytes.Buffer`
  rather than `bytes.BytesBuffer`.
- Return errors instead of panicking. Wrap an error with context using
  `fmt.Errorf("read config: %w", err)`, and compare with `errors.Is` or
  `errors.As`.
- Pass `context.Context` as the first parameter of a function that does I/O or
  can be cancelled. Do not store it in a struct.
- Define an interface in the package that uses it, and keep it small.
- Close what you open with `defer` right after a successful open, and check the
  error of a `Close` that writes.
- Start a goroutine only with a clear owner and a way to stop it; protect
  shared state with a mutex or a channel.
- Write table-driven tests with `t.Run` subtests. Use `t.Helper()` in test
  helpers and `t.TempDir()` for files. Keep tests independent of their order.
- Change `go.mod` only through `go get` or `go mod tidy`, and keep `go.sum` in
  the same change. Do not raise the `go` or `toolchain` version unless the task
  asks for it.

## 3. Verify

Run the project's entry point for each check when it has one. Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Format | `gofmt -l .` | Prints nothing |
| Vet | `go vet ./...` | Exits 0 |
| Build | `go build ./...` | Exits 0 |
| Test | `go test ./...` | Exits 0 |
| Race | `go test -race ./...` when the change touches goroutines or shared state | Exits 0 |
| Modules | `go mod tidy -diff` when the change touches imports | Prints nothing |

`go mod tidy -diff` needs Go 1.23 or later. With an older Go, run
`go mod tidy` and report the change it makes to `go.mod` and `go.sum`.

Fix a failure the change caused before handoff. Report a failure the change did
not cause as an existing failure, with the command and its output, instead of
fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing tool. When a workflow skill owns
the task, return these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Name the file,
command, or output behind every claim about the project.
