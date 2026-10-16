---
name: develop-flutter
description: Apply a Flutter project's own conventions, state management, localization, and test commands while writing or changing a Flutter app or package, and verify the change with the project's formatter, analyzer, and tests. Use it whenever a task changes Flutter widgets, screens, navigation, state, localized text, assets, or widget, golden, or integration tests, even when the request does not mention Flutter, localization, or verification. Use it together with develop-dart for the Dart language, analyzer, and pub rules. Do not use it for a project whose pubspec.yaml has no `flutter` SDK dependency, such as a pure Dart package; use develop-dart there.
license: Apache-2.0
---

# Develop Flutter

## Todo List

1. **in progress:** Confirm the Flutter package, its SDK constraint and version pin, its platform folders, its state management, navigation, localization, and test setup, and the project's own commands.
2. Make the requested Flutter change following the project's widget, state, and localization conventions.
3. Verify the change with the project's formatter, analyzer, and tests, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `mobile` category. It gives Flutter
knowledge to the task that is already running. It does not choose what to
change, and it creates no Issue, Pull Request, branch, or release. When the
task belongs to a workflow skill such as `implement-issue`, `write-tests`, or
`debug-code`, that skill keeps ownership of the plan, the commits, and the
handoff; this skill supplies the Flutter commands and conventions.

Use `develop-dart` for the Dart language, analyzer, and pub rules. This skill
adds only the Flutter rules on top of them. Use `develop-swift` or
`develop-android-compose` only for platform channel code or other native code
in the `ios/` or `android/` folder.

Stop and say that this skill does not apply when `pubspec.yaml` has no
`flutter` SDK dependency (`sdk: flutter` under `dependencies`). A pure Dart
package is out of scope; use `develop-dart` there.

## 1. Discover the project

- Read the SDK constraint under `environment:` in `pubspec.yaml`, and the
  Flutter version pin from `.fvmrc`, the `.fvm/` folder, or `mise.toml`. When
  the project pins Flutter with FVM, run every command as `fvm flutter`;
  when `mise.toml` pins it, run the pinned version, because another Flutter
  version can format, analyze, or resolve packages differently.
- List the platform folders that exist: `ios/`, `android/`, `web/`, `macos/`,
  `windows/`, and `linux/`. Build only for a platform the project has.
- Find the state management already in use from the dependencies and the
  code: Provider, Riverpod, Bloc, or plain `setState`.
- Find the navigation already in use: `Navigator` with routes, or
  `go_router`.
- Check for localization: `flutter_localizations` in the dependencies, an
  `l10n.yaml` file, and ARB files, usually in `lib/l10n/`. Read the template
  ARB file that `l10n.yaml` names, and whether `pubspec.yaml` sets
  `generate: true` under `flutter:`.
- Find the test setup: widget tests with `flutter_test` under `test/`, golden
  tests that call `matchesGoldenFile`, and an `integration_test/` folder.
- Read `analysis_options.yaml`, usually `package:flutter_lints/flutter.yaml`,
  and the rules it changes.
- Find the project's own entry point, such as a `Makefile` target, a melos
  script, or a `mise.toml` task, and prefer it over a raw `flutter` command,
  because it carries the project's flags.

## 2. Change the code

- Build screens from small widgets instead of one large `build` method. Give
  each widget a `const` constructor when its fields allow it, and create it
  with `const` where you can, so Flutter can skip rebuilding it.
- Keep `build` pure: no side effects, no async work, and no state changes in
  it, because Flutter can call it many times. Start work in `initState`, an
  event handler, or the project's state management.
- Dispose every controller you create, such as a `TextEditingController`,
  `AnimationController`, or `ScrollController`, in `dispose`.
- Read colors, text styles, and sizes from `Theme.of(context)` and
  `MediaQuery` instead of hard-coding them.
- When the project uses localization, put every user-facing string through the
  generated localizations, such as `AppLocalizations.of(context)!.noNotes`.
  Add the key to the template ARB file, then run `flutter gen-l10n`, or rely on
  `generate: true` to generate the code on the next `flutter` command. Do not
  hard-code user-facing text in widget code.
- Give a custom control or an image without text a label through `Semantics`
  or the widget's `semanticLabel`, so screen readers can describe it.
- Use the state management library and the navigation the project already
  has. Do not add another one.
- Add a package with `flutter pub add <package>`, or
  `flutter pub add dev:<package>` for a test or tool package.
- Do not change the Flutter SDK constraint or version pin, the platform
  folders' bundle IDs, application IDs, signing, development team, or
  provisioning profiles, the minimum OS versions, `l10n.yaml`, or
  `analysis_options.yaml` unless the task asks for it, because each one
  changes the build or the release for every user.
- Write widget tests with `testWidgets`, `WidgetTester.pumpWidget`, and
  `find`. Wrap a widget that reads localizations in a `MaterialApp` with the
  project's localization delegates.

## 3. Verify

Run the project's entry point for each check when it has one. Otherwise run,
with `fvm flutter` in place of `flutter` when the project pins Flutter with
FVM:

| Check | Command | Pass condition |
| --- | --- | --- |
| Format | `dart format --output=none --set-exit-if-changed .` | Exits 0 |
| Analyze | `flutter analyze` | Exits 0 |
| Test | `flutter test` | Exits 0 |
| Golden tests | The project's update command, such as `flutter test --update-goldens`, only when the task changes visuals | Exits 0, and the changed golden files show only the intended change |
| Build | `flutter build <platform>` only when that platform's toolchain is installed | Exits 0 |
| Integration tests | `flutter test integration_test` only when a device or simulator is available | Exits 0 |

When the machine has no Flutter SDK, report each check as not run, with the
error, instead of claiming it passed. Report the build and integration tests
as not run when the toolchain, device, or simulator is missing. Fix a failure
the change caused before handoff. Report a failure the change did not cause as
an existing failure, with the command and its output, instead of fixing
unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing Flutter SDK or no available
device. When a workflow skill owns the task, return these results to it; that
skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
