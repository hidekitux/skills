---
name: develop-kotlin-multiplatform
description: Apply a Kotlin Multiplatform module's targets, source set hierarchy, expect and actual declarations, and iOS framework export while writing or changing shared code, and verify the change per target through the project's Gradle wrapper. Use it whenever a task writes, changes, or tests code in a module that applies the `org.jetbrains.kotlin.multiplatform` plugin and shares code across Android, iOS, JVM, JS or Wasm, or native targets, such as adding a shared function for an Android and iOS app or fixing a failing commonTest, even when the request does not mention source sets, targets, or Swift. Do not use it for a project whose build files apply no Kotlin Multiplatform plugin; use `develop-kotlin` for a JVM-only Kotlin project.
license: Apache-2.0
---

# Develop Kotlin Multiplatform

## Todo List

1. **in progress:** Confirm the multiplatform module, its targets, its source set hierarchy, its `expect` and `actual` declarations, its iOS framework export, and the project's own Gradle tasks.
2. Make the requested change in the narrowest source set that every caller can reach, following the project's conventions.
3. Verify the change with the project's Gradle tasks for each target the machine can build, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and every target or check that did not run.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives Kotlin
Multiplatform knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the multiplatform commands and
conventions.

Use `develop-kotlin` alongside this skill for the Kotlin language rules, the
Gradle wrapper rule, and the version catalog rule; this skill does not repeat
them. Use `develop-swift` for Swift code in the iOS host app that calls the
shared module.

Stop and say that this skill does not apply when no build file applies the
Kotlin Multiplatform plugin (`org.jetbrains.kotlin.multiplatform`,
`kotlin("multiplatform")`, or a version catalog alias that resolves to it).
A JVM-only Kotlin project is out of scope; use `develop-kotlin` there.

## 1. Discover the project

- Find the multiplatform modules: the `include` lines in
  `settings.gradle.kts`, and each module's `build.gradle.kts` that applies the
  plugin. Read the Kotlin version from `gradle/libs.versions.toml` or the
  `plugins` block.
- List the targets declared in `kotlin { }`, such as `androidTarget()` or an
  `androidLibrary { }` block from the `com.android.kotlin.multiplatform.library`
  plugin, `iosArm64()`, `iosSimulatorArm64()`, `iosX64()`, `jvm()`, `js()`,
  `wasmJs()`, `macosArm64()`, and `linuxX64()`. The targets decide which
  platform APIs a source set can use and which test tasks exist.
- Map the source set hierarchy. Without manual `dependsOn` calls, the default
  hierarchy template creates the intermediate source sets from the targets,
  such as `commonMain` to `nativeMain` to `appleMain` to `iosMain`, each with
  a matching `*Test` source set. A build that calls `dependsOn` itself
  replaces the template, so read those calls instead. Check the directories
  under `<module>/src/` against the hierarchy.
- List the existing `expect` and `actual` declarations
  (`grep -rnE 'expect |actual ' <module>/src`), and note which source set
  holds each `actual`. Every target must reach one `actual` for each
  `expect`.
- Find the iOS export: the `binaries.framework { }` block with its `baseName`
  and `isStatic`, an `export(...)` of other modules, the
  `kotlin("native.cocoapods")` plugin with its `cocoapods { }` block, an
  `XCFramework` for a Swift Package Manager binary target, or an Xcode build
  phase that runs `embedAndSignAppleFrameworkForXcode`. The export decides
  which declarations Swift sees and under which module name.
- Find the shared libraries already in `commonMain` dependencies, such as
  `kotlinx-coroutines`, `kotlinx-serialization`, `kotlinx-datetime`, Ktor, or
  SQLDelight, and Swift interop tools such as SKIE or KMP-NativeCoroutines.
  Reuse them before writing platform code.
- Find the project's own tasks, such as a `mise.toml` task, a `Makefile`
  target, or the commands in its CI workflows, and prefer them, because they
  carry the module paths and targets the project actually checks.

## 2. Change the code

- Put code in the narrowest source set that every caller can reach. Code that
  only Android and the JVM need belongs in their source sets, not in
  `commonMain`; code that Android and iOS both call belongs in `commonMain`.
- Prefer common Kotlin and multiplatform libraries over `expect` and
  `actual`. Use `expect` and `actual` only for a platform API, keep each
  `expect` declaration small, and put each `actual` in the widest source set
  that has the API, such as `appleMain` for Foundation. When the platform
  code needs state or several functions, declare an interface in
  `commonMain` with one implementation per platform instead, because an
  interface can be faked in `commonTest`.
- Do not use JVM-only APIs such as `java.*`, `System.currentTimeMillis()`, or
  `String.format` in `commonMain`. They compile for the JVM target and fail
  for native and web targets, so a JVM-only build hides the error.
- Write shared tests in `commonTest` with `kotlin.test`, and put a test of
  platform code in that platform's test source set, such as `iosTest`.
- Keep the public API Swift-friendly. Swift sees the module through an
  Objective-C header, so a `suspend` function becomes a completion handler or
  `async` call, a `Flow` loses its element type, a `sealed` hierarchy is not
  exhaustive in a `switch`, and default arguments disappear. Only exceptions
  listed in `@Throws` reach Swift as errors; any other exception crashes the
  app. Use SKIE or KMP-NativeCoroutines only when the project already uses
  them. Keep platform-only helpers `internal` so they stay out of the header.
- Do not add or remove a target, and do not change the framework `baseName`,
  `isStatic`, or the CocoaPods or Swift Package Manager setup, unless the task
  asks for it, because each change breaks the host app's build.

## 3. Verify

Run the project's own task when it defines one. Otherwise run, with
`<module>` such as `shared`:

| Check | Command | Pass condition |
| --- | --- | --- |
| Common code | `./gradlew :<module>:compileKotlinMetadata` | Exits 0 |
| All tests | `./gradlew :<module>:allTests`, or `./gradlew :<module>:check` to add the project's lint tasks | Exits 0, and the output lists a test task for every declared target |
| One target | `./gradlew :<module>:jvmTest`, `:<module>:iosSimulatorArm64Test`, `:<module>:jsTest`, or the Android unit test task from `./gradlew :<module>:tasks --all` | Exits 0 |
| iOS framework | `./gradlew :<module>:linkDebugFrameworkIosSimulatorArm64` when the change touches the public API | Exits 0 |

Apple target tests and framework links need macOS with Xcode. On another host,
Gradle skips the Apple targets with a warning, so `allTests` can exit 0
without running them; read the output, and report each skipped target, such
as `iosSimulatorArm64` on Linux, as not run. When the project has no Gradle
wrapper or the machine lacks the JDK, Xcode, or the Android SDK, report the
check as not run with the error instead of claiming it passed. Fix a failure
the change caused before handoff. Report a failure the change did not cause
as an existing failure instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result, and name
the targets each command covered. Name every target and check that did not
run and why, such as iOS targets on a Linux host. When a workflow skill owns
the task, return these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
