---
name: develop-typescript
description: Apply a TypeScript or JavaScript project's own conventions, package manager, and package.json scripts while writing or changing code on Node.js, and verify the change with the project's typecheck, lint, format check, tests, and build. Use it whenever a task writes, changes, or tests `.ts`, `.tsx`, `.js`, `.mjs`, or `.cjs` code or edits package.json, such as adding an exported function, fixing a failing Vitest, Jest, or node:test test, or adding a dependency, even when the request does not mention the package manager or verification. Do not use it for a project with no `.ts`, `.tsx`, `.js`, `.mjs`, or `.cjs` source files and no package.json.
license: Apache-2.0
---

# Develop TypeScript

## Todo List

1. **in progress:** Confirm the package manager, the package.json scripts, the module system, and the TypeScript settings.
2. Make the requested TypeScript or JavaScript change following the project's conventions.
3. Verify the change with the project's typecheck, lint, format check, tests, and build, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives
TypeScript, JavaScript, and Node.js knowledge to the task that is already
running. It does not choose what to change, and it creates no Issue, Pull
Request, branch, or release. When the task belongs to a workflow skill such as
`implement-issue`, `write-tests`, or `debug-code`, that skill keeps ownership
of the plan, the commits, and the handoff; this skill supplies the TypeScript
commands and conventions. A framework skill, such as one for React or React
Native, can build on this skill.

Stop and say that this skill does not apply when the repository has no `.ts`,
`.tsx`, `.js`, `.mjs`, or `.cjs` source files and no package.json.

## 1. Discover the project

- Find the package manager from the lockfile: `package-lock.json` for npm,
  `pnpm-lock.yaml` for pnpm, `yarn.lock` for Yarn, and `bun.lockb` or
  `bun.lock` for Bun. Read the `packageManager` field in package.json too.
  When the lockfile and the field disagree, report the conflict instead of
  choosing silently. When there is no lockfile, say so and use the manager
  that `packageManager` names, or npm when the field is missing. Use one
  manager for every command, because a second manager writes a second
  lockfile that resolves different versions.
- Read the `scripts` in package.json before running any raw tool. Prefer
  `npm run <script>`, or the matching `pnpm run`, `yarn run`, or `bun run`
  command, because the script carries the project's flags, config paths, and
  file globs; a raw `tsc` or `eslint` call can pass while the project's own
  script fails. In a workspace, find the package that owns the change from the
  `workspaces` field or `pnpm-workspace.yaml`.
- Read the module system: the `"type"` field in package.json (`"module"` for
  ESM, missing or `"commonjs"` for CommonJS), the `.mjs` and `.cjs` extensions,
  and `module` and `moduleResolution` in tsconfig.json. Write imports in the
  form that system needs: under `NodeNext` the import names the emitted `.js`
  file, and a project that runs `.ts` files directly with Node.js type
  stripping (`allowImportingTsExtensions`, or `node --test` on `.ts` files)
  imports the `.ts` file. Type stripping rejects syntax that needs compiling,
  such as `enum`, `namespace`, and constructor parameter properties, so do not
  add it to such a project.
- Read tsconfig.json, including any file it `extends`: `strict` and related
  flags such as `noUncheckedIndexedAccess`, `target`, `lib`, and `noEmit`.
  Read the Node.js version from `engines`, `.nvmrc`, `.node-version`, or
  `mise.toml`.
- Find the test runner (Vitest, Jest, or `node:test`) and the lint and format
  tools (ESLint, Prettier, or Biome) from package.json, their config files,
  and CI workflows. Do not add a tool the project does not use.

## 2. Change the code

- Keep `strict` on and do not weaken a compiler flag to make an error go away.
- Do not use `any`. Take an unknown value as `unknown` and narrow it with
  `typeof`, `instanceof`, `in`, or a type guard before use.
- Do not use a non-null assertion (`value!`) unless a comment states why the
  value cannot be `null` or `undefined`; prefer a check or an early return.
- Declare parameter and return types on every exported function, class
  method, and constant, so the public API does not change when the
  implementation does.
- Use `async` and `await` and handle errors with `try`/`catch` or a returned
  result. Do not leave a floating promise: `await` it, return it, or mark a
  deliberate fire-and-forget call with `void` and handle its rejection.
- Follow the file layout, naming, and export style already in use, such as
  named exports or a barrel `index.ts`.
- Write tests with the runner the project already uses, in its existing test
  directory and file naming.
- Add a dependency only when the standard library or an existing dependency
  cannot do the job. Install it with the project's manager, such as
  `npm install <name>` or `pnpm add -D <name>`, so the lockfile changes in the
  same change. Do not edit the lockfile by hand, and do not change the Node.js,
  TypeScript, or package manager version unless the task asks for it.

## 3. Verify

Run the project's script for each check when it has one, through the detected
manager. Otherwise run the tool directly:

| Check | Command | Pass condition |
| --- | --- | --- |
| Typecheck | `npm run typecheck`, or `npx --no-install tsc --noEmit` (`tsc -b` for a tsconfig with `references`) | Exits 0 |
| Lint | `npm run lint`, or the project's installed linter such as `npx --no-install eslint .` or `npx --no-install biome lint .` | Exits 0 |
| Format | `npm run format:check`, or `npx --no-install prettier --check .` or `npx --no-install biome format .` | Exits 0 |
| Test | `npm test` | Exits 0 |
| Build | `npm run build` when the project defines it | Exits 0 |

Use `--no-install` with `npx`, because without a local install `npx tsc`
downloads an unrelated `tsc` package and `npx biome` an old `biome` package
instead of TypeScript and `@biomejs/biome`.

Replace `npm run` and `npx` with the detected manager's commands, such as
`pnpm run` and `pnpm exec`. When `node_modules` is missing, install with the
command that keeps the lockfile fixed: `npm ci`,
`pnpm install --frozen-lockfile`, `yarn install --immutable` (Yarn 2 or later)
or `yarn install --frozen-lockfile` (Yarn 1), or
`bun install --frozen-lockfile`. When there is no lockfile, do not run an
install that creates one unless the task asks for it. When Node.js, the
manager, or `node_modules` is unavailable, report each affected check as not
run with the error instead of claiming it passed.
Fix a failure the change caused before handoff. Report a failure the change
did not cause as an existing failure, with the command and its output, instead
of fixing unrelated code.

## Handoff

Report the detected package manager and how you detected it, each verification
command, its exit status, and its result. Name every check you skipped and why,
such as a missing `node_modules`. When a workflow skill owns the task, return
these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
