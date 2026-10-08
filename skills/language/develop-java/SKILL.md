---
name: develop-java
description: Apply a Java project's own conventions, Gradle or Maven tasks, and test commands while writing or changing Java code, and verify the change through the project's Gradle wrapper or Maven wrapper, or through the build command its CI runs when it has no wrapper. Use it whenever a task writes, changes, or tests Java code, such as in a library, a service, an Android module, or a Minecraft mod, even when the request does not mention Gradle, Maven, or verification. Do not use it for a project without `.java` files, such as a Kotlin-only Gradle project; use develop-kotlin there.
license: Apache-2.0
---

# Develop Java

## Todo List

1. **in progress:** Confirm the build tool and its wrapper, the Java toolchain or release level, and the project's own tasks.
2. Make the requested Java change following the project's conventions.
3. Verify the change with the project's Gradle or Maven tasks, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives Java,
Gradle, and Maven knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the Java commands and
conventions. A framework skill, such as one for Android or a Minecraft mod
loader, can build on this skill for the language and build rules.

Stop and say that this skill does not apply when the repository has no
`.java` files. A Kotlin-only Gradle project is out of scope; use
`develop-kotlin` there.

## 1. Discover the project

- Find the build tool. Gradle uses `settings.gradle.kts` or `settings.gradle`
  and the wrapper `gradlew`; Maven uses `pom.xml` and the wrapper `mvnw`. List
  the modules that the settings file or the parent `pom.xml` includes.
- Use `./gradlew` or `./mvnw` when the project has a wrapper, rather than a
  Gradle or Maven installed on the machine, so the build runs the version that the project pins in
  `gradle/wrapper/gradle-wrapper.properties` or
  `.mvn/wrapper/maven-wrapper.properties`. When the project has no wrapper,
  use the command its CI workflow or contributor documentation runs, such as
  `mvn verify`, and name that source in the handoff. When neither exists, say
  so and report the build checks as not run instead of guessing a version,
  because a different version can pass or fail for reasons the project does
  not have.
- Read the Java level from `java { toolchain { languageVersion } }` or
  `options.release` in a Gradle build, and from `maven.compiler.release` or the
  compiler plugin's `<release>` in a Maven build. Do not raise it unless the
  task asks for it.
- Find the checks the project runs, such as Spotless, Checkstyle, Error Prone,
  PMD, or SpotBugs, from the build files and CI workflows. Do not add a tool
  the project does not use.
- Find the project's own entry point, such as a `mise.toml` task or a
  `Makefile` target, and prefer it over a raw wrapper command, because it
  carries the project's flags and environment.

## 2. Change the code

- Follow the package layout already in use. When `module-info.java` exists,
  keep its `exports` and `requires` in step with the change, and do not export
  a package only for a test.
- Use features of the Java level in use, such as records, switch expressions,
  and text blocks, only when that release allows them.
- Return `Optional` for a value that can be absent. Do not use `Optional` for a
  field or a parameter, because it adds a wrapper without removing the null
  check at the boundary.
- Close every resource with try-with-resources.
- Prefer immutable values: make fields `final`, and return unmodifiable
  collections such as `List.copyOf` from public methods.
- Throw a specific exception, such as `IllegalArgumentException` for a bad
  argument, with a message that names the value. Never swallow an exception
  in an empty `catch`; handle it, or rethrow it with the cause attached.
- Avoid raw types. Give every generic type its type arguments.
- Add a dependency through the version catalog or the `dependencyManagement`
  section when the project has one, and do not upgrade Gradle, Maven, or a
  plugin unless the task asks for it.
- Write tests with the framework and assertion library the module already
  uses, such as JUnit 5 and AssertJ, in the matching test package.

## 3. Verify

Run the project's own task when it defines one. Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Build and test one Gradle module | `./gradlew :<module>:check` | Exits 0 |
| Build and test a Gradle build | `./gradlew check` when the change crosses modules | Exits 0 |
| Build and test a Maven build | `./mvnw verify`, or `./mvnw -pl <module> -am verify` for one module | Exits 0 |
| Tests in one module | `./gradlew :<module>:test` or `./mvnw -pl <module> -am test` while iterating | Exits 0 |
| Format | The project's task, such as `./gradlew spotlessCheck` or `./mvnw spotless:check` | Exits 0 |

When the machine lacks the wrapper or the JDK the build needs, report that the
check could not run, with the error, instead of claiming it passed. Fix a
failure the change caused before handoff. Report a failure the change did not
cause as an existing failure instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing wrapper or JDK. When a workflow
skill owns the task, return these results to it; that skill owns the next
phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
