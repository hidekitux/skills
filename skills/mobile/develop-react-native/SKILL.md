---
name: develop-react-native
description: Apply a React Native project's own conventions, Expo or bare workflow, and test commands while writing or changing a React Native app, and verify the change with the project's typecheck, lint, Jest tests, and, when the toolchains are available, native builds. Use it whenever a task changes React Native screens, components, navigation, app config, native modules, or React Native tests, even when the request does not mention Expo, Metro, or verification. Use it together with develop-typescript for the language, package manager, scripts, and type and lint checks, and develop-react for component and hook rules. Do not use it for a project with no `react-native` dependency in any package.json, such as a React web app; use develop-react there.
license: Apache-2.0
---

# Develop React Native

## Todo List

1. **in progress:** Confirm the Expo or bare workflow, the React Native and Expo SDK versions, the New Architecture setting, the navigation library, the Metro config, the test setup, and the native modules.
2. Make the requested React Native change following the project's components, navigation, and styling conventions.
3. Verify the change with the project's typecheck, lint, and test scripts, and native builds when available, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives React
Native and Expo knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the React Native commands and
conventions.

Use `develop-typescript` for the language, the package manager, the
package.json scripts, and the type and lint checks, and `develop-react` for
component and hook rules. Use `develop-swift` and `develop-android-compose`
only for native module code in `ios/` and `android/`. This skill adds only the
React Native and Expo rules on top of them.

Stop and say that this skill does not apply when no package.json lists
`react-native` as a dependency. A React web app is out of scope; use
`develop-react` there.

## 1. Discover the project

- Decide between Expo and the bare workflow. An `expo` dependency and an
  `app.json` or `app.config.js` or `app.config.ts` file mark an Expo project.
  An Expo project without `ios/` and `android/` folders is a managed project
  that generates them with prebuild (Continuous Native Generation, CNG). A
  project with those folders and no `expo` dependency uses the bare React
  Native CLI workflow.
- Read the `react-native` and `expo` versions from package.json, and the
  New Architecture setting: `newArchEnabled` in the Expo app config,
  `newArchEnabled` in `android/gradle.properties`, and `RCT_NEW_ARCH_ENABLED`
  in `ios/Podfile` or its environment.
- Find the navigation library: React Navigation (`@react-navigation/*`) or
  Expo Router (`expo-router` and an `app/` route directory).
- Read `metro.config.js` when it exists, for custom resolvers, asset
  extensions, or monorepo watch folders.
- Find the test setup: Jest with the `jest-expo` or `react-native` preset in
  `jest.config.*` or the `jest` field of package.json, React Native Testing
  Library (`@testing-library/react-native`), and Detox or Maestro for
  end-to-end tests.
- List the native modules: local modules in `modules/`, `ios/`, or
  `android/`, config plugins in the app config, and CocoaPods in
  `ios/Podfile`.

## 2. Change the code

- Build UI from core components: `View` for layout, `Text` for every string,
  because React Native renders text only inside `<Text>`, `Pressable` for
  touch targets, and `FlatList` with `keyExtractor` for a long or unbounded
  list. Do not use web elements such as `<div>` or `<span>`.
- Style with `StyleSheet.create`, or with the styling library the project
  already uses.
- Branch on `Platform.OS`, or split a file into `.ios.tsx` and `.android.tsx`,
  only when the behavior must differ by platform.
- Give an interactive or meaningful element `accessibilityRole` and
  `accessibilityLabel`, so screen readers can announce it.
- In an Expo project, add a native dependency with `npx expo install <name>`,
  so its version matches the Expo SDK. In a bare project, use the package
  manager from `develop-typescript`, then run `pod install` in `ios/`.
- Do not edit `ios/` or `android/` in a managed Expo project, because prebuild
  overwrites them; change the app config or a config plugin instead.
- Do not change the React Native or Expo SDK version, the bundle identifier
  or application id, signing, the New Architecture setting, the Metro config,
  the Jest preset, or `ios/Podfile` unless the task asks for it, because each
  one changes the build or the tests for both platforms or the identity in
  the stores.
- Add screens through the navigation library already in use, React
  Navigation or Expo Router, and do not add a second one.
- Write tests with React Native Testing Library and the project's Jest preset,
  next to the existing tests and with their file naming.

## 3. Verify

Run the project's script for each check when it has one, through the package
manager that `develop-typescript` detects:

| Check | Command | Pass condition |
| --- | --- | --- |
| Typecheck | `npm run typecheck`, or `npx --no-install tsc --noEmit` | Exits 0 |
| Lint | `npm run lint` when the project defines it | Exits 0 |
| Test | `npm test`, or `npx --no-install jest` | Exits 0 |
| Expo health | `npx expo-doctor` in an Expo project, when the network and `node_modules` are available | Exits 0 and reports no failed check |
| Native build | `npx expo run:ios` or `npx expo run:android` in Expo; `xcodebuild` in `ios/` or `./gradlew assembleDebug` in `android/` in a bare project | Exits 0 |
| End-to-end | The project's Detox or Maestro command, such as `npx detox test` or `maestro test` | Exits 0 |

Run the native build and the end-to-end tests only when Xcode, the Android
SDK, and a simulator, emulator, or device are available. Otherwise report them
as not run, with the reason. When `node_modules` is missing, report the
JavaScript checks as not run with the error instead of claiming they passed.
Fix a failure the change caused before handoff. Report a failure the change
did not cause as an existing failure instead of fixing unrelated code.

## Handoff

Report whether the project uses Expo or the bare workflow and how you
detected it, each verification command, its exit status, and its result. Name
every check you skipped and why, such as a missing `node_modules`, no
simulator, or no Android SDK. When a workflow skill owns the task, return
these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
