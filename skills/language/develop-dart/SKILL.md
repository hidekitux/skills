---
name: develop-dart
description: Apply a Dart project's own conventions, analyzer rules, and test commands while writing or changing Dart code, and verify the change with the project's formatter, analyzer, and tests. Use it whenever a task writes, changes, or tests Dart code in a package or application with a pubspec.yaml, including a Flutter project, such as adding a function or library, fixing a failing test, or adding a dependency, even when the request does not mention lints or verification. Do not use it for a project with no pubspec.yaml and no `.dart` files.
license: Apache-2.0
---

# Develop Dart

## Todo List

1. **in progress:** Confirm the Dart package, its SDK constraint, its analyzer rules, and the project's own commands.
2. Make the requested Dart change following the project's conventions.
3. Verify the change with the project's formatter, analyzer, and tests, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `language` category. It gives Dart
knowledge to the task that is already running. It does not choose what to
change, and it creates no Issue, Pull Request, branch, or release. When the
task belongs to a workflow skill such as `implement-issue`, `write-tests`, or
`debug-code`, that skill keeps ownership of the plan, the commits, and the
handoff; this skill supplies the Dart commands and conventions. A framework
skill, such as one for Flutter, can build on this skill.

Stop and say that this skill does not apply when the repository has no
`pubspec.yaml` and no `.dart` files.

## 1. Discover the project

- Find each package root from `pubspec.yaml` and read its `name`, its SDK
  constraint under `environment: sdk:`, and its dependencies. Write only code
  that the lowest SDK in that constraint supports, because users of the
  package may run that SDK.
- Decide whether the project uses Flutter: a pubspec that lists the `flutter`
  SDK dependency (`sdk: flutter`) under `dependencies` is a Flutter project.
  Run `flutter` commands for it, such as `flutter analyze` and `flutter test`,
  because they load the Flutter SDK and its test bindings, which `dart` alone
  does not.
- Read `analysis_options.yaml`: the rule set it includes, such as
  `package:lints/recommended.yaml`, `package:flutter_lints/flutter.yaml`, or
  `package:very_good_analysis/analysis_options.yaml`, and every rule it
  enables, disables, or raises to an error. Write code that passes those rules
  instead of adding `// ignore:` comments.
- Find the project's own entry point before using raw `dart` commands: a
  `Makefile` target, a melos script in `melos.yaml` or under the `melos:` key
  of the root `pubspec.yaml`, or a `mise.toml` task. Prefer that entry point,
  because it carries the project's flags and runs every package in a
  workspace; a raw `dart test` in one package can pass while the project's own
  check fails.
- Note a pub workspace (`workspace:` in the root `pubspec.yaml`) or a melos
  monorepo, and run commands for the package you change.

## 2. Change the code

- Follow the library layout already in use. Put implementation files under
  `lib/src/` and export the public API from a library file in `lib/`.
- Name libraries, files, and directories in lower_snake_case, such as
  `string_utils.dart`. Name types in UpperCamelCase and other identifiers in
  lowerCamelCase.
- Keep sound null safety. Do not use the `!` operator unless the value cannot
  be null for a reason you can state; handle `null` with `?.`, `??`, or an
  early return.
- Prefer `final` for a variable that is not reassigned and `const` for a value
  known at compile time, including `const` constructors.
- Await every `Future`, or handle it explicitly. Wrap a `Future` you mean not
  to await in `unawaited()` from `dart:async`, so the analyzer and the reader
  see that the choice is deliberate.
- Add a dependency with `dart pub add <package>`, or
  `dart pub add dev:<package>` for a test or tool dependency, so
  `pubspec.yaml` and `pubspec.lock` stay consistent. In a Flutter project, use
  `flutter pub add`. Do not raise the SDK constraint unless the task asks for
  it.
- Write tests with `package:test`: `test()` for a case, `group()` for related
  cases, and `expect()` with a matcher. Put them under `test/` with names that
  end in `_test.dart`. In a Flutter project, use `flutter_test` for widget
  tests.

## 3. Verify

Run the project's entry point for each check when it has one. Otherwise run:

| Check | Command | Pass condition |
| --- | --- | --- |
| Format | `dart format --output=none --set-exit-if-changed .` | Exits 0 |
| Analyze | `dart analyze`, adding `--fatal-infos` only when the project's own commands or CI use it | Exits 0 |
| Test | `dart test` | Exits 0 |

In a Flutter project, run `flutter analyze` and `flutter test` instead of
`dart analyze` and `dart test`; `dart format` covers Flutter code too.

When the machine has no Dart or Flutter SDK, report each check as not run,
with the error, instead of claiming it passed. Fix a failure the change caused
before handoff. Report a failure the change did not cause as an existing
failure, with the command and its output, instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing Dart SDK. When a workflow skill
owns the task, return these results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
