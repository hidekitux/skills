---
name: develop-react
description: Apply a React project's own framework, conventions, and package.json scripts while writing or changing React components and hooks, and verify the change with the project's typecheck, lint, tests, and build. Use it whenever a task writes, changes, or tests React code in an application or component library built with Vite, Next.js (App Router or Pages Router), Remix or React Router, or legacy Create React App, such as adding a component state, fixing a hook, or adding a Testing Library test, even when the request does not mention React or verification. Do not use it when no package.json lists a `react` dependency, or for a React Native app.
license: Apache-2.0
---

# Develop React

## Todo List

1. **in progress:** Confirm the React version, the framework or bundler, the styling, state, routing, and test setup, and the `develop-typescript` project facts.
2. Make the requested React change following the project's conventions.
3. Verify the change with the project's typecheck, lint, tests, and build scripts, and record each command and result.
4. Complete the list only when every verification command has a recorded result or a stated reason it was not run; hand off the commands, results, and skipped checks.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `web` category. It gives React
knowledge to the task that is already running. It does not choose what to
change, and it creates no Issue, Pull Request, branch, or release. When the
task belongs to a workflow skill such as `implement-issue`, `write-tests`, or
`debug-code`, that skill keeps ownership of the plan, the commits, and the
handoff; this skill supplies the React commands and conventions.

This skill builds on `develop-typescript`. Follow `develop-typescript` for the
language rules, the package manager, the `package.json` scripts, the module
system, and the typecheck, lint, and format checks; this skill does not repeat
them. It covers React with whatever framework or bundler the project already
uses and does not choose one for the project.

Stop and say that this skill does not apply when no package.json in the
repository lists `react` in `dependencies`, `devDependencies`, or
`peerDependencies`. When a package.json lists `react-native`, stop and say that
a React Native app is out of scope for this skill.

## 1. Discover the project

- Read the `react` and `react-dom` versions from package.json. Note features
  that depend on the version, such as `use`, Actions, and `ref` as a prop in
  React 19.
- Find the framework or bundler from package.json and its config file: Vite
  (`vite.config.*`), Next.js (`next.config.*`), Remix or React Router
  (`@remix-run/*`, `react-router` with `react-router.config.*`), or legacy
  Create React App (`react-scripts`).
- For Next.js, find the router mode: an `app/` directory is the App Router and
  a `pages/` directory is the Pages Router; both can exist. In the App Router,
  a file that starts with `"use client"` is a Client Component, and every other
  component is a Server Component.
- Find the styling approach: CSS Modules (`*.module.css`), Tailwind
  (`tailwind.config.*` or `@import "tailwindcss"`), styled-components, or
  another library already in use.
- Find the state and data libraries, such as TanStack Query (React Query),
  Redux Toolkit, or Zustand, and the routing library.
- Find the test setup: Vitest or Jest with Testing Library
  (`@testing-library/react`), the test environment (`jsdom` or `happy-dom`),
  and end-to-end tests with Playwright or Cypress.
- Check whether the ESLint config enables `eslint-plugin-react-hooks`.
- Find the i18n setup when there is one, such as `react-i18next`, `next-intl`,
  or `react-intl`, and its message files.

## 2. Change the code

- Write function components and hooks. Do not add a class component.
- Follow the Rules of Hooks: call hooks only at the top level of a component
  or custom hook, never in a condition, loop, or nested function. Name a custom
  hook `use<Name>`.
- Derive a value during render from props and state instead of copying it into
  state and syncing it with an effect. Use `useMemo` only when the calculation
  is measurably slow.
- Use `useEffect` only to synchronize with an external system, such as a
  subscription, a timer, or a browser API. List every reactive value it reads
  in the dependency list, return a cleanup function when it starts something,
  and do not silence the `react-hooks/exhaustive-deps` rule.
- Give each list item a stable key from the data, such as an ID. Never use the
  array index as the key for a list that can be reordered, filtered, or edited.
- Keep state in the lowest component that needs it, and lift it only to the
  nearest common parent of the components that share it.
- Use semantic HTML and accessible names: a `<button>` for an action rather
  than a clickable `<div>`, a `<label>` for each form control, and `alt` text
  for each meaningful image.
- When the project has an i18n setup, add user-facing text through it, in each
  locale file the project keeps, instead of hard-coding the string.
- In the Next.js App Router, keep a component as a Server Component unless it
  needs state, effects, event handlers, or browser APIs. Put `"use client"` on
  the smallest component that needs it.
- Follow the project's styling approach, file layout, and component naming.
- Test behavior the user can see with Testing Library: query by role, label,
  or text (`getByRole`, `getByLabelText`, `getByText`), and drive input with
  `@testing-library/user-event` when the project has it. Do not assert on
  component state, props, or CSS class names.
- Do not change the React version, the framework or its version, or the
  Next.js router mode unless the task asks for it.

## 3. Verify

Run each check through the project's script and the package manager that
`develop-typescript` detected:

| Check | Command | Pass condition |
| --- | --- | --- |
| Typecheck | `npm run typecheck` | Exits 0 |
| Lint | `npm run lint` | Exits 0 |
| Unit and component tests | `npm test` | Exits 0 |
| Build | `npm run build`, which runs `next build`, `vite build`, or the framework's build | Exits 0 |
| End-to-end tests | The project's script, such as `npm run test:e2e` | Exits 0 |

Run end-to-end tests only when the project has them and a browser is
available; otherwise report them as not run with the reason. When Node.js, the
package manager, or `node_modules` is unavailable, report each affected check
as not run with the error instead of claiming it passed. Fix a failure the
change caused before handoff. Report a failure the change did not cause as an
existing failure, with the command and its output.

## Handoff

Report the React version, the framework or bundler and how you detected it,
each verification command, its exit status, and its result. Name every check
you skipped and why, such as a missing browser for end-to-end tests. When a
workflow skill owns the task, return these results to it; that skill owns the
next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
