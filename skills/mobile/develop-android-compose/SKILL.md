---
name: develop-android-compose
description: Apply an Android project's own conventions, Gradle tasks, and test commands while writing or changing an Android app or library built with the Android Gradle plugin, with UI in Jetpack Compose or in Views, and verify the change through the project's Gradle wrapper. Use it whenever a task changes Android screens, composables, ViewModels, resources, the manifest, or Android tests, even when the request does not mention Compose, Gradle, or verification. Use it together with develop-kotlin for Kotlin and Gradle rules and develop-java for Java sources. Do not use it for a project with no Android Gradle plugin (`com.android.application` or `com.android.library`) in any build file, such as a plain Kotlin or Java JVM project; use develop-kotlin or develop-java there.
license: Apache-2.0
---

# Develop Android Compose

## Todo List

1. **in progress:** Confirm the Android modules and their plugins, the SDK levels, the AGP, Kotlin, and Compose versions, the variants, and the project's own tasks.
2. Make the requested Android change following the project's architecture and Compose conventions.
3. Verify the change with the project's Gradle tasks for the module and variant, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives Android,
Jetpack Compose, and Android Gradle plugin knowledge to the task that is
already running. It does not choose what to change, and it creates no Issue,
Pull Request, branch, or release. When the task belongs to a workflow skill
such as `implement-issue`, `write-tests`, or `debug-code`, that skill keeps
ownership of the plan, the commits, and the handoff; this skill supplies the
Android commands and conventions.

Use `develop-kotlin` for the Kotlin language and Gradle rules, and
`develop-java` for Java sources. This skill adds only the Android and Compose
rules on top of them.

Stop and say that this skill does not apply when no build file applies the
Android Gradle plugin (`com.android.application` or `com.android.library`). A
plugin applied through a version catalog alias counts when the catalog entry
has one of those ids. A plain Kotlin or Java JVM project is out of scope; use
`develop-kotlin` or `develop-java` there.

## 1. Discover the project

- List the modules in `settings.gradle.kts` or `settings.gradle`, and the
  plugin each module applies: `com.android.application` for an app,
  `com.android.library` for a library. Name the module in every Gradle task,
  such as `:app`, because a task without a module path runs in every module.
- Read compileSdk, minSdk, and targetSdk from each module's `android { }`
  block, and the Android Gradle plugin (AGP), Kotlin, and Compose BOM versions
  from `gradle/libs.versions.toml` when it exists.
- Confirm how Compose is set up: `buildFeatures { compose = true }`, the
  Compose compiler plugin `org.jetbrains.kotlin.plugin.compose`, and the
  `platform(libs.androidx.compose.bom)` dependency. A module without them uses
  Views; follow that module's layouts and do not add Compose to it unless the
  task asks.
- List the build types and product flavors. A variant name joins them, such as
  `freeDebug`, and the Gradle task names use it, such as
  `assembleFreeDebug`.
- Read `src/main/AndroidManifest.xml` for the activities, the permissions, and
  the application class.
- Find the architecture already in use: ViewModels, dependency injection with
  Hilt or Koin, and Navigation (Navigation Compose or fragments). Follow it
  instead of adding another library.
- Find the lint configuration (`lint { }` in the build file, `lint.xml`, or
  `lint-baseline.xml`) and the test setup: JUnit local tests in `src/test`,
  Robolectric, instrumented tests in `src/androidTest`, and Compose UI tests
  with `createComposeRule`.
- Find the Gradle wrapper (`gradlew`) and the Android SDK, which Gradle finds
  through `ANDROID_HOME` or `sdk.dir` in `local.properties`. Note a missing
  wrapper or SDK now, because every check in section 3 needs both.
- Find the project's own entry point, such as a `mise.toml` task or a
  `Makefile` target, and prefer it over a raw wrapper command, because it
  carries the project's flags and environment.

## 2. Change the code

- Hoist state. Keep a composable stateless: it takes the state as parameters
  and reports events through lambda parameters such as `onClick`. Keep the
  state in the caller or in a ViewModel.
- Give a composable that emits UI a `modifier: Modifier = Modifier` parameter,
  first among the optional parameters, and apply it to the outermost element,
  so the caller controls size and placement.
- Keep a value across recompositions with `remember`, and across
  configuration changes and process death with `rememberSaveable`.
- Run side effects only in an effect handler, such as `LaunchedEffect`,
  `DisposableEffect`, or `SideEffect`, never directly in the composable body,
  because the body can run many times.
- Collect a `Flow` or `StateFlow` in the UI with
  `collectAsStateWithLifecycle()`, so collection stops while the screen is not
  visible.
- Add a `@Preview` for a new composable when the module already uses previews.
- Put user-visible text in `res/values/strings.xml` and read it with
  `stringResource(R.string.<name>)` in Compose or `getString` in Views, so
  translators can find it. Do not hard-code text in UI code.
- Launch coroutines in `viewModelScope` in a ViewModel and in `lifecycleScope`
  in an activity or fragment, so they end with their owner.
- Do not change compileSdk, targetSdk, or minSdk, and do not upgrade AGP,
  unless the task asks for it, because each one changes platform behavior or
  the build for every module.
- Do not regenerate `lint-baseline.xml` or relax the `lint { }` or `lint.xml`
  configuration to make a lint check pass; fix the reported issue, or report
  it when it predates the change.
- Add a permission to the manifest only when the task needs it, because each
  permission is visible to users and to store review.
- Add a dependency through the version catalog, and take Compose library
  versions from the BOM instead of pinning each one.
- Write tests where the project keeps them: local tests in `src/test` for
  logic, and Compose UI or instrumented tests in `src/androidTest` for UI.

## 3. Verify

Run the project's own task when it defines one. Otherwise run, for the module
and variant you changed:

| Check | Command | Pass condition |
| --- | --- | --- |
| Build | `./gradlew :app:assembleDebug`, or `./gradlew :<module>:assemble<Variant>` | Exits 0 |
| Local tests | `./gradlew :app:testDebugUnitTest`, or `./gradlew :<module>:test<Variant>UnitTest` | Exits 0 |
| Lint | `./gradlew :app:lintDebug`, or `./gradlew :<module>:lint<Variant>` | Exits 0 |
| Instrumented tests | `./gradlew :app:connectedDebugAndroidTest` only when a device or emulator is connected | Exits 0 |

When the project has no Gradle wrapper, or the Android SDK is missing because
neither `ANDROID_HOME` nor `sdk.dir` in `local.properties` is set, report the
checks as not run, with the reason, instead of claiming they passed. Report
the instrumented tests as not run when no device or emulator is connected.
Fix a failure the change caused before handoff. Report a failure the change
did not cause as an existing failure instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing wrapper, a missing Android SDK,
or no connected device. When a workflow skill owns the task, return these
results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
