---
name: develop-compose-multiplatform
description: Apply a Compose Multiplatform project's conventions, shared UI layout, resources, and Gradle tasks while writing or changing Compose UI shared across Android, iOS, desktop, and web, and verify the change through the project's Gradle wrapper. Use it whenever a task writes, changes, or tests UI in a module that applies the `org.jetbrains.compose` plugin, directly or through a version catalog alias, such as a `composeApp` module with `commonMain` screens, even when the request does not mention Compose Multiplatform, resources, or verification. Do not use it for a project with no `org.jetbrains.compose` plugin in any build file or version catalog alias, such as an Android-only Compose app.
license: Apache-2.0
---

# Develop Compose Multiplatform

## Todo List

1. **in progress:** Confirm the targets, the shared UI module, the platform entry points, the resources, and the project's own Gradle tasks.
2. Make the requested UI change in the shared source set following the project's conventions.
3. Verify the change with the project's Gradle tasks for each target, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and every target that did not build.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives Compose
Multiplatform knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the Compose Multiplatform
commands and conventions.

Use it alongside other technology skills instead of repeating their rules:
`develop-kotlin` for the language and Gradle rules,
`develop-kotlin-multiplatform` for targets, source sets, and `expect` and
`actual`, `develop-android-compose` for an Android-only module in the same
repository, and `develop-swift` for the iOS host app.

Stop and say that this skill does not apply when no build file applies the
`org.jetbrains.compose` plugin, either by id or through a version catalog alias
whose entry in `gradle/libs.versions.toml` has that id. An Android-only Compose
project is out of scope; use `develop-android-compose` there.

## 1. Discover the project

- Find the module that applies `org.jetbrains.compose` together with
  `org.jetbrains.kotlin.plugin.compose`, often `composeApp`. Read its plugin
  versions from the version catalog (`gradle/libs.versions.toml`) when it
  exists, otherwise from the `plugins` block.
- List the targets from the `kotlin { }` block, such as `androidTarget()`,
  `jvm("desktop")`, `iosArm64()`, `iosSimulatorArm64()`, and `wasmJs`. The
  target name sets the source set and task names: `jvm("desktop")` gives
  `desktopMain` and `desktopTest`, not `jvmMain` and `jvmTest`.
- Find the entry point of each platform: `MainActivity` in `androidMain`, the
  `MainViewController` function in `iosMain` that wraps the shared UI in
  `ComposeUIViewController`, `main.kt` with `application { Window { } }` in the
  desktop source set, and `main.kt` with `ComposeViewport` in `wasmJsMain`.
  Each one calls the shared root composable, often `App()`.
- Find the resources in `src/commonMain/composeResources/`, such as
  `values/strings.xml` and `drawable/`. Read the package of the generated `Res`
  class from an existing import or from
  `compose.resources { packageOfResClass }`; the default is derived from the
  project group and module name, such as
  `<group>.composeapp.generated.resources`.
- Find the navigation, lifecycle, and view model libraries already in use,
  such as `org.jetbrains.androidx.navigation` or
  `org.jetbrains.androidx.lifecycle`, and the Gradle wrapper (`gradlew`). When
  the project has no wrapper, say so and report the Gradle checks as not run,
  as `develop-kotlin` explains.

## 2. Change the code

- Put shared UI in `commonMain` and use only multiplatform APIs there. Do not
  import `android.*` or a JVM-only or iOS-only API in `commonMain`, because the
  other targets then fail to compile.
- Load user-facing strings and images through Compose Multiplatform
  resources: add the entry to `composeResources/values/strings.xml` or
  `drawable/`, then read it with `stringResource(Res.string.<name>)` or
  `painterResource(Res.drawable.<name>)`. Do not hard-code user-facing text,
  because a hard-coded string cannot be translated through
  `values-<locale>/strings.xml`.
- Keep platform-specific UI behind `expect` and `actual`, or pass it in from
  the platform entry point as a parameter or a composable lambda, so that the
  shared code stays free of platform imports.
- Follow the Compose rules that Android uses: hoist state to the caller, take
  `modifier: Modifier = Modifier` as the first optional parameter, and run side
  effects in `LaunchedEffect`, `DisposableEffect`, or another effect handler
  instead of in the composable body.
- Use the navigation and lifecycle libraries the project already has; do not
  add a second one.
- Do not change the Compose Multiplatform, Kotlin, or Android Gradle plugin
  version, the targets, the module layout, or `packageOfResClass` unless the
  task asks for it, because these versions must stay compatible with each
  other and every import of `Res` depends on its package.

## 3. Verify

Run the project's own task when it defines one. Otherwise run, through the
wrapper:

| Check | Command | Pass condition |
| --- | --- | --- |
| Android build | `./gradlew :<android module>:assembleDebug`, where the Android module applies `com.android.application`: `:composeApp` with AGP 8, or a separate module such as `:androidApp` with AGP 9 | Exits 0 |
| Tests | `./gradlew :composeApp:<jvm target>Test`, such as `jvmTest` or `desktopTest`, or `./gradlew :composeApp:allTests` | Exits 0 |
| iOS build | `./gradlew :composeApp:linkDebugFrameworkIosSimulatorArm64` on macOS, or the Xcode build of the iOS host app | Exits 0 |
| Desktop run | `./gradlew :composeApp:run` only when the task asks to run the app | The window opens |
| Web run | `./gradlew :composeApp:wasmJsBrowserDevelopmentRun` only when the task asks for it | The page loads |

A target can fail to build on this machine for a reason the change did not
cause: the iOS build needs macOS and Xcode, and the Android build needs the
Android SDK. Report each target that cannot build here as not run, with the
error, instead of claiming it passed. Fix a failure the change caused before
handoff.

## Handoff

Report each verification command, its exit status, and its result. Name every
target that did not build and why, such as a missing Android SDK or a machine
that is not macOS. When a workflow skill owns the task, return these results to
it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
