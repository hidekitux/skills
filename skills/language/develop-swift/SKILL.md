---
name: develop-swift
description: Apply a Swift project's own conventions, build commands, and test commands while writing or changing Swift code in a Swift package or an Xcode project, and verify the change with `swift build`, `swift test`, or `xcodebuild`. Use it whenever a task writes, changes, or tests Swift code, such as adding a function or type to a package with a Package.swift, fixing a failing XCTest or Swift Testing test, or resolving a Swift 6 concurrency diagnostic in an .xcodeproj or .xcworkspace, even when the request does not mention conventions or verification. Do not use it for a project without `.swift` files.
license: Apache-2.0
---

# Develop Swift

## Todo List

1. **in progress:** Confirm the Swift package or Xcode project, its tools version, platforms, and language mode, and the project's own commands.
2. Make the requested Swift change following the project's conventions.
3. Verify the change with the project's build, tests, and configured lint or format checks, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives Swift
knowledge to the task that is already running. It does not choose what to
change, and it creates no Issue, Pull Request, branch, or release. When the
task belongs to a workflow skill such as `implement-issue`, `write-tests`, or
`debug-code`, that skill keeps ownership of the plan, the commits, and the
handoff; this skill supplies the Swift commands and conventions. A framework
skill, such as one for iOS, can build on this skill for the language, package,
and build rules.

Stop and say that this skill does not apply when the repository has no
`.swift` files.

## 1. Discover the project

- Find the build. A Swift package has a `Package.swift`; an Xcode project has
  an `.xcodeproj`, and an `.xcworkspace` when it combines projects or
  packages. A repository can have both, such as an app project that depends
  on a local package.
- Read the `// swift-tools-version:` line on the first line of `Package.swift`,
  the `platforms` list, and the language mode from `swiftLanguageModes` (tools
  version 6.0 and later) or `swiftLanguageVersions` (earlier), plus any
  per-target `swiftSettings` such as `.swiftLanguageMode(.v5)`. A package with
  tools version 6.0 and no language mode setting builds in the Swift 6 mode,
  which makes data-race checks errors. Read the default actor isolation too:
  `.defaultIsolation(MainActor.self)` in `Package.swift`, or the
  `SWIFT_DEFAULT_ACTOR_ISOLATION` build setting, makes unannotated code run on
  the main actor and changes what needs `Sendable` or `nonisolated`. In an
  Xcode project, read the `SWIFT_VERSION`, `SWIFT_STRICT_CONCURRENCY`, and
  `SWIFT_DEFAULT_ACTOR_ISOLATION` build settings. Do not change the tools
  version, the platforms, the language mode, or the default actor isolation
  unless the task asks for it, because each change can break every caller and
  every target.
- Find the project's own entry point before using raw `swift` or `xcodebuild`
  commands: a `Makefile` target, a `mise.toml` task, a fastlane lane in
  `fastlane/Fastfile`, or scripts named in the contributor documentation.
  Prefer that entry point, because it carries the project's scheme,
  destination, and flags; a raw `swift test` can pass while the project's own
  check fails.
- Find the checks the project configures: `.swiftlint.yml` for SwiftLint and
  `.swift-format` for swift-format, and the commands its CI workflows run. Do
  not add a linter or formatter the project does not use.
- Find the test framework from the existing tests: `import XCTest` with
  `XCTestCase` subclasses, or `import Testing` with `@Test` and `#expect`.

## 2. Change the code

- Follow the target and folder layout already in use. Keep access control
  minimal: leave declarations `internal` by default, and mark them `public`
  only when they are part of the module's API.
- Use a value type (`struct` or `enum`) by default. Use a `class` only when
  the value needs identity or shared mutable state that callers must observe.
- Prefer `let` over `var`.
- Handle an optional with `guard let` or `if let`. Do not use a force unwrap
  (`!`) or `try!` unless a comment states why it cannot fail.
- Throw an error type the caller can match, such as an `enum` that conforms to
  `Error`, or typed `throws(SomeError)` when the project already uses it. Do
  not swallow an error with an empty `catch` or with `try?` when the caller
  needs to know about the failure.
- Use `async` and `await` and actors for concurrent work. Keep `Sendable`
  conformance and `@MainActor` isolation correct instead of silencing a
  diagnostic; do not add `@unchecked Sendable` or `nonisolated(unsafe)`
  without a comment that states why the access is safe.
- Write tests with the framework the target already uses, XCTest or Swift
  Testing. Do not mix a second framework into a test target that uses one.
- Follow the existing SwiftLint and swift-format configuration, and do not
  disable a rule to make a check pass.
- Add a package dependency in `Package.swift` and run `swift package resolve`
  so `Package.resolved` changes in the same change. Do not run
  `swift package update` unless the task asks for newer versions.

## 3. Verify

Run the project's entry point for each check when it has one. Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Package build | `swift build` | Exits 0 |
| Package test | `swift test` | Exits 0 |
| Xcode schemes | `xcodebuild -list` with `-project <name>.xcodeproj` or `-workspace <name>.xcworkspace` | Lists the scheme to test |
| Xcode test | `xcodebuild -project <name>.xcodeproj` or `-workspace <name>.xcworkspace`, then `-scheme <scheme> -destination '<destination>' test`, such as `-destination 'platform=macOS'` or `-destination 'platform=iOS Simulator,name=<device>'` from `xcrun simctl list devices available` | Exits 0 |
| Lint | `swiftlint lint` when the project has `.swiftlint.yml` | Exits 0 and reports no new violation |
| Format | `swift format lint --recursive .` when the project has `.swift-format` | Prints no finding |

When the machine lacks Xcode, the Swift toolchain, or the simulator runtime
the destination needs, report the check as not run, with the error, instead
of claiming it passed. Fix a failure the change caused before handoff. Report
a failure the change did not cause as an existing failure, with the command
and its output, instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing Xcode or simulator runtime. When
a workflow skill owns the task, return these results to it; that skill owns
the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
