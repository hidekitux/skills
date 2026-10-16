---
name: develop-ios
description: Apply an iOS or iPadOS app's own conventions, Xcode schemes, and test commands while writing or changing an app built with Xcode, with UI in SwiftUI or UIKit, and verify the change with `xcodebuild` on an iOS Simulator. Use it whenever a task changes an iOS app's views, view models, localized strings, Info.plist keys, assets, or unit or UI tests in an .xcodeproj, an .xcworkspace, an XcodeGen project.yml, or a Tuist Project.swift, even when the request does not mention SwiftUI, Xcode, or verification. Use it together with develop-swift for the Swift language, concurrency, package, and test-framework rules. Do not use it for a repository with no .xcodeproj, .xcworkspace, XcodeGen project.yml, or Tuist Project.swift that defines an iOS target, such as a Swift package with no app target; use develop-swift there.
license: Apache-2.0
---

# Develop iOS

## Todo List

1. **in progress:** Confirm the Xcode project or generator, the schemes and targets, the deployment target, the Info.plist keys, entitlements, localization, dependencies, and the project's own commands.
2. Make the requested iOS change following the project's SwiftUI or UIKit conventions.
3. Verify the change with `xcodebuild` build and test on an iOS Simulator and the configured lint or format checks, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives iOS,
iPadOS, SwiftUI, UIKit, and Xcode knowledge to the task that is already
running. It does not choose what to change, and it creates no Issue, Pull
Request, branch, or release. When the task belongs to a workflow skill such
as `implement-issue`, `write-tests`, or `debug-code`, that skill keeps
ownership of the plan, the commits, and the handoff; this skill supplies the
iOS commands and conventions.

Use `develop-swift` for the Swift language, concurrency, Swift package, and
test-framework rules. This skill adds only the iOS app, UI, and Xcode rules
on top of them.

Stop and say that this skill does not apply when the repository has no
`.xcodeproj`, `.xcworkspace`, XcodeGen `project.yml`, or Tuist `Project.swift`
that defines an iOS target. A Swift package with no app target is out of
scope; use `develop-swift` there.

## 1. Discover the project

- Find the build. Look for a project generator first: XcodeGen reads
  `project.yml` and Tuist reads `Project.swift`. With a generator, edit the
  generator source and regenerate, never the generated `.xcodeproj`, because
  the next generation overwrites it. Without one, find the `.xcodeproj`, and
  the `.xcworkspace` when it combines projects or CocoaPods.
- List the schemes and targets with `xcodebuild -list` and `-project` or
  `-workspace`. Note the app target, the unit test target, and the UI test
  target, and the scheme that runs each test target.
- Read the deployment target (`IPHONEOS_DEPLOYMENT_TARGET`, or
  `deploymentTarget` in `project.yml`), because it decides which APIs the code
  may call without an availability check.
- Read the Info.plist keys, from the `Info.plist` file or the
  `INFOPLIST_KEY_*` build settings, and the entitlements file
  (`*.entitlements`).
- Find the asset catalog (`Assets.xcassets`) and the localization files: a
  String Catalog (`Localizable.xcstrings`) or `.strings` files.
- Find the dependencies: Swift packages in the project or in `project.yml`,
  and CocoaPods in a `Podfile`. With CocoaPods, build the `.xcworkspace`, not
  the `.xcodeproj`.
- Find the checks the project configures, `.swiftlint.yml` for SwiftLint and
  `.swift-format` for swift-format, and the project's own entry point, such as
  a `Makefile` target, a `mise.toml` task, or a fastlane lane. Prefer that
  entry point, because it carries the project's scheme, destination, and
  flags.

## 2. Change the code

- In SwiftUI, keep each view small; move a part that grows into its own view.
- Own state in one place. Use `@State` for state local to a view. On iOS 17
  and later, use an `@Observable` model, with `@Bindable` where a view needs
  bindings to it; on an earlier deployment target, use `ObservableObject`
  with `@StateObject` and `@ObservedObject`. Follow the pattern the project
  already uses.
- Start async work with the `.task` modifier, so it is cancelled when the view
  goes away. Do no work in `body`, because SwiftUI can call it many times.
- Add a `#Preview` for a new view when the project already uses previews.
- In UIKit, follow the project's existing controllers, layout code or
  storyboards, and navigation.
- Put user-facing text through localization. A SwiftUI `Text("key")` literal
  is localizable already; use `String(localized:)` in non-UI code. When the
  project has a String Catalog, add each new key to it in the same change,
  because Xcode adds extracted keys only when it builds. Do not build
  user-facing text from concatenated strings.
- Support accessibility: give an image or icon-only control an accessibility
  label, use text styles such as `.body` so Dynamic Type scales the text, and
  use system colors and fonts so dark mode and contrast settings work.
- Add an Info.plist usage-description key, such as
  `NSCameraUsageDescription`, or a capability only when the task needs it,
  because users and App Review see each one.
- Do not change the deployment target, the bundle identifier, signing, the
  development team, provisioning profiles, entitlements, Info.plist keys the
  task does not need, or other build settings unless the task asks for it,
  because each one can break the build, signing, or the release for every
  target.
- Prefer editing source files over the project file. In an Xcode project
  without a generator, a file added inside a synchronized folder group
  (Xcode 16 and later) joins the target automatically; otherwise add the file
  to the target in the project file, or the build does not compile it. With a
  generator, a file under a listed source path joins the target on
  regeneration.
- Write tests where the project keeps them: unit tests in the unit test
  target and UI tests with `XCUIApplication` in the UI test target.

## 3. Verify

Run the project's own entry point for each check when it has one. With a
generator, generate the project first, such as `xcodegen generate` or
`tuist generate`. Pick the device from `xcrun simctl list devices available`.
Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Schemes | `xcodebuild -list` with `-project <name>.xcodeproj` or `-workspace <name>.xcworkspace` | Lists the scheme to build and test |
| Build and test | `xcodebuild -project <name>.xcodeproj` or `-workspace <name>.xcworkspace`, then `-scheme <scheme> -destination 'platform=iOS Simulator,name=<device>' build test` | Exits 0 |
| Focused test | The same command with `-only-testing:<target>/<test>` | Exits 0 |
| Lint | `swiftlint lint` when the project has `.swiftlint.yml` | Exits 0 and reports no new violation |
| Format | `swift format lint --recursive .` when the project has `.swift-format` | Prints no finding |

When the machine lacks Xcode, the project generator, or an iOS Simulator
runtime, such as on Linux, report each check as not run, with the error,
instead of claiming it passed. Fix a failure the change caused before
handoff. Report a failure the change did not cause as an existing failure,
with the command and its output, instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing Xcode, generator, or simulator
runtime. When a workflow skill owns the task, return these results to it;
that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
