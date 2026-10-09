---
name: develop-minecraft-forge
description: Apply a Minecraft Forge or NeoForge mod's own registration, event, resource, and client-code conventions while writing or changing the mod, and verify the change through the project's Gradle wrapper with its build, data generation, and game test runs. Use it whenever a task adds or changes blocks, items, entities, events, lang entries, models, loot tables, or other code or resources in a mod built with ForgeGradle, NeoGradle, or ModDevGradle, even when the request does not mention Forge, NeoForge, or verification. Do not use it for a Fabric or Quilt mod; use develop-minecraft-fabric there.
license: Apache-2.0
---

# Develop Minecraft Forge

## Todo List

1. **in progress:** Confirm the loader, its Gradle plugin, the Minecraft, loader, mappings, and Java versions, the mod id, and the project's own tasks and runs.
2. Make the requested mod change following the project's registration, event, and resource conventions.
3. Verify the change through the Gradle wrapper, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `game` category. It gives Minecraft
Forge and NeoForge knowledge to the task that is already running. It does not
choose what to change, and it creates no Issue, Pull Request, branch, or
release. When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the mod loader commands and
conventions.

Use `develop-java` for the Java language and Gradle rules, and
`develop-kotlin` as well when the mod has Kotlin code. This skill adds only
the rules that the mod loader brings.

Stop and say that this skill does not apply when the build applies no Forge or
NeoForge Gradle plugin and the project has no `mods.toml` or
`neoforge.mods.toml`. A Fabric mod, marked by `fabric.mod.json` and the
Fabric Loom plugin, is out of scope; use `develop-minecraft-fabric` there. A
Quilt mod, marked by `quilt.mod.json` and the Quilt Loom plugin, is out of
scope too.

## 1. Discover the project

- Find the loader from the Gradle plugin in `build.gradle`,
  `build.gradle.kts`, or `settings.gradle(.kts)`: ForgeGradle
  (`net.minecraftforge.gradle`) for Forge, NeoGradle
  (`net.neoforged.gradle.userdev`) or ModDevGradle (`net.neoforged.moddev`)
  for NeoForge.
- Read `gradle.properties` for `minecraft_version`, the loader version
  (`forge_version` or `neo_version`), the mappings channel and version (such
  as `mapping_channel` and `mapping_version`, or the Parchment properties),
  and `mod_id`. When the build sets a value directly, read it there instead.
- Read the metadata file: `src/main/resources/META-INF/mods.toml` for Forge
  and for NeoForge before 20.5, `src/main/resources/META-INF/neoforge.mods.toml`
  for NeoForge 20.5 and later. Its `modId` must equal `mod_id`.
- Find the main mod class annotated with `@Mod`, the `DeferredRegister`
  fields it registers on the mod event bus, and the event subscribers
  (`@EventBusSubscriber` classes and listeners added to an `IEventBus`).
- Find data generation: a `data` run in the build and a `GatherDataEvent`
  subscriber, with output usually in `src/generated/resources/`. Find the
  resources under `src/main/resources/assets/<modid>/` and
  `src/main/resources/data/<modid>/`, and any game tests.
- Read the Java level from the toolchain. Minecraft 1.17 needs Java 16,
  1.18 to 1.20.4 need Java 17, and 1.20.5 to 1.21.x need Java 21. The
  year-based versions from 26.1 need Java 25; check the loader's release notes
  for a version newer than this list.
- Read the Gradle task name for each run from the build: a run named `data`
  is the task `runData`, and a run named `gameTestServer` is
  `runGameTestServer`. Newer ModDevGradle projects split data generation into
  `clientData` and `serverData` runs.

## 2. Change the code

- Register every block, item, entity, and other registry object through a
  `DeferredRegister` that is registered on the mod event bus, and hold the
  returned `DeferredHolder`, `DeferredItem`, `DeferredBlock`, or
  `RegistryObject`. Never create a registry object in a static initializer
  outside a register call, because an object built outside the
  `DeferredRegister` is never registered and fails when the game uses it.
- From Minecraft 1.21.2, an item's or block's properties must carry its
  registry id, or the game fails at startup with an "id not set" error. Use
  `registerItem` or `registerSimpleItem` (and the block equivalents), which
  set it, or call `setId` on the properties when you use plain `register`.
- Keep client-only code, such as rendering, screens, and key bindings, out of
  common code. Put it in a class that only a `Dist.CLIENT` subscriber or a
  `FMLEnvironment.dist` check reaches, because a dedicated server has no
  client classes and fails to load a class that refers to them.
- Use the mod id namespace for every resource location, registry name,
  translation key, and resource path.
- Put lang entries in `assets/<modid>/lang/en_us.json`, such as
  `item.<modid>.<name>` for an item and `block.<modid>.<name>` for a block.
- Add item and block models, block states, recipes, tags, and loot tables
  through the data generation providers when the project uses data
  generation. Otherwise add them as JSON files under `assets/<modid>/` and
  `data/<modid>/`, following the folder names the Minecraft version uses.
- Do not change `minecraft_version`, the loader version, the mappings, the
  Java version, the Gradle plugin version, the mod id, the `mods.toml` or
  `neoforge.mods.toml` metadata, or the configured runs unless the task asks
  for it, because every change can break every Minecraft and loader class
  reference, resource path, or saved world that uses the mod.

## 3. Verify

Run the project's own task when it defines one. Otherwise run, through the
wrapper:

| Check | Command | Pass condition |
| --- | --- | --- |
| Build | `./gradlew build` | Exits 0 |
| Data generation | `./gradlew runData`, or the project's data run, when the project uses data generation | Exits 0 and the generated files hold the change |
| Game tests | `./gradlew runGameTestServer` when the project has game tests | Exits 0 and every required test passes |

Run `runClient` or `runServer` only when the task asks for it, because each
one launches the game. After data generation, read the changed files under the
output folder and confirm the new entries.

When the project has no Gradle wrapper or the machine lacks the JDK the
Minecraft version needs, report the checks as not run, with the reason,
instead of using a Gradle installed on the machine or claiming a pass. Fix a
failure the change caused before handoff. Report a failure the change did not
cause as an existing failure instead of fixing unrelated code.

## Handoff

Report each verification command, its exit status, and its result. Name every
check you skipped and why, such as a missing wrapper or JDK, or a client run
the task did not ask for. When a workflow skill owns the task, return these
results to it; that skill owns the next phase.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Choose the plain word
over an inflated or Latinate one: write `use` rather than `utilize` and `is`
rather than `serves as`. Name the file, command, or output behind every claim
about the project.
