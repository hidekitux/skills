---
name: develop-kotlin
description: Apply a Kotlin project's own conventions, Gradle tasks, and test commands while writing or changing Kotlin code, and verify the change through the project's Gradle wrapper. Use it whenever a task writes, changes, or tests Kotlin code or Gradle Kotlin DSL build files, such as in a JVM library, a Kotlin Multiplatform module, an Android module, or a Minecraft mod, even when the request does not mention Gradle or verification. Do not use it for a project without Kotlin code.
license: Apache-2.0
---

# Develop Kotlin

## Todo List

1. **in progress:** Confirm the Gradle build, the Kotlin and JVM versions, and the project's own tasks.
2. Make the requested Kotlin change following the project's conventions.
3. Verify the change with the project's Gradle tasks, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives Kotlin
and Gradle knowledge to the task that is already running. It does not choose
what to change, and it creates no Issue, Pull Request, branch, or release. When
the task belongs to a workflow skill such as `implement-issue`, `write-tests`,
or `debug-code`, that skill keeps ownership of the plan, the commits, and the
handoff; this skill supplies the Kotlin commands and conventions. A framework
skill, such as one for Android, Compose Multiplatform, or a Minecraft mod
loader, can build on this skill for the language and Gradle rules.

Stop and say that this skill does not apply when the repository has no `.kt`
files. A Gradle Kotlin DSL build file (`.kts`) alone marks a Gradle build, not
Kotlin code, so a Java project with `build.gradle.kts` is out of scope.

## 1. Discover the project

- Find the build: `settings.gradle.kts` or `settings.gradle`, the modules it
  includes, and the Gradle wrapper (`gradlew`). Use `./gradlew`, never a Gradle
  installed on the machine, so the build runs the version in
  `gradle/wrapper/gradle-wrapper.properties`. When the project has no wrapper,
  say so and report the Gradle checks as not run instead of using another
  Gradle version, because a different Gradle can pass or fail for reasons the
  project does not have.
- Read the Kotlin version and plugin versions from the version catalog
  (`gradle/libs.versions.toml`) when it exists, otherwise from the `plugins`
  block. Read the JVM target from `kotlin { jvmToolchain(...) }` or
  `compilerOptions`.
- Find the checks the project runs, such as `ktlint`, `detekt`, `spotless`, or
  `binary-compatibility-validator`, from the build files and CI workflows. Do
  not add a tool the project does not use.
- For a Kotlin Multiplatform module, list its targets and source sets
  (`commonMain`, `commonTest`, `jvmMain`, `iosMain`, and so on) before you
  place code.

## 2. Change the code

- Follow the package and module layout already in use. In a multiplatform
  module, put code in the narrowest source set that every caller can reach,
  and use `expect` and `actual` only when a platform API is required.
- Prefer `val` over `var`, and immutable collections in public signatures.
- Keep nullability in the type. Do not use `!!`; handle `null` with `?.`,
  `?:`, `requireNotNull`, or an early return with a reason.
- Use a `data class` for a value, a `sealed` interface or class for a closed
  set of states whose members carry different data, and an `enum class` for a
  fixed set of constants, which may share properties.
- Launch coroutines in a scope that someone owns and cancels, such as a
  `viewModelScope` or a scope the caller passes in. Do not use `GlobalScope`.
  Switch dispatchers with `withContext` at the I/O boundary.
- Throw an exception only for a programming error or an unrecoverable state;
  return a result type when the caller is expected to handle the failure.
- Keep the project's API mode. When `explicitApi()` is on, declare visibility
  and return types for public declarations.
- Add a dependency through the version catalog when the project has one, and
  do not upgrade Gradle, the Kotlin plugin, or the Android Gradle plugin unless
  the task asks for it.
- Write tests with the framework the module already uses, such as `kotlin.test`,
  JUnit 5, or Kotest. In a multiplatform module, put shared tests in
  `commonTest`.

## 3. Verify

Run the project's own task when it defines one. Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Build and test one module | `./gradlew :<module>:check` | Exits 0 |
| Build and test everything | `./gradlew check` when the change crosses modules | Exits 0 |
| Format or lint | The project's task, such as `./gradlew ktlintCheck`, `./gradlew detekt`, or `./gradlew spotlessCheck` | Exits 0 |
| Public API | `./gradlew apiCheck` when the project uses the binary-compatibility validator | Exits 0 |

When the machine lacks the JDK or SDK the build needs, report that the check
could not run, with the error, instead of claiming it passed. Fix a failure
the change caused before handoff. Report a failure the change did not cause as
an existing failure instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing JDK. When a workflow skill owns
the task, return these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
