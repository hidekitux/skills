---
name: develop-minecraft-fabric
description: Apply a Minecraft Fabric mod's own conventions for registration, entrypoints, Mixins, assets, and data generation, and verify the change through the project's Gradle wrapper with Fabric Loom tasks. Use it whenever a task adds or changes content, client code, Mixins, lang entries, or game tests in a mod built with Fabric Loom and described by `fabric.mod.json`, even when the request does not mention Fabric, Loom, or verification. Do not use it for a Forge or NeoForge mod; use develop-minecraft-forge there.
license: Apache-2.0
---

# Develop Minecraft Fabric

## Todo List

1. **in progress:** Confirm the Loom build, `gradle.properties`, `fabric.mod.json`, the entrypoints, the Mixin configs, and the Java version.
2. Make the requested mod change following the project's registration, side, and asset conventions.
3. Verify the change through the Gradle wrapper, and record each command and result.
4. Complete the list only when every verification command has a recorded result; hand off the commands, results, and any skipped check.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Scope

This skill is a technology skill in the `game` category. It gives Fabric and
Fabric Loom knowledge to the task that is already running. It does not choose
what to change, and it creates no Issue, Pull Request, branch, or release.
When the task belongs to a workflow skill such as `implement-issue`,
`write-tests`, or `debug-code`, that skill keeps ownership of the plan, the
commits, and the handoff; this skill supplies the Fabric commands and
conventions.

Use `develop-java` for the Java language and Gradle rules, and
`develop-kotlin` for a mod written with `fabric-language-kotlin`. This skill
does not repeat those rules.

Stop and say that this skill does not apply when the project has neither a
Fabric Loom plugin (`fabric-loom` or `net.fabricmc.fabric-loom`) nor a
`fabric.mod.json`. A Forge or NeoForge mod, marked
by `mods.toml`, `neoforge.mods.toml`, or a ForgeGradle or NeoGradle plugin, is
out of scope; use `develop-minecraft-forge` there, and do not add Fabric files
to it.

## 1. Discover the project

- Find the Loom plugin, `id 'fabric-loom'`, `id("fabric-loom")`, or
  `net.fabricmc.fabric-loom` for the year-based versions, in
  `build.gradle` or `build.gradle.kts`, and the plugin repositories in the
  settings file.
- Read `gradle.properties`: `minecraft_version`, `yarn_mappings` or the
  official Mojang mappings (`loom.officialMojangMappings()` in the build),
  `loader_version`, and `fabric_version`. The mappings decide the names: Yarn
  uses `Identifier` and `Registries`, and Mojang mappings use
  `BuiltInRegistries` with `ResourceLocation` before 1.21.11 and `Identifier`
  from 1.21.11. Use the names the existing code uses. When the
  build has no mappings dependency, do not add one.
- Read `src/main/resources/fabric.mod.json`: the mod `id`, the `entrypoints`
  (`main`, `client`, `fabric-datagen`, `fabric-gametest`), the `mixins` list,
  and `depends`.
- Open each Mixin config JSON that `fabric.mod.json` lists, and note its
  `package` and its `mixins`, `client`, and `server` arrays.
- Check for `splitEnvironmentSourceSets()` in the `loom` block. When it is on,
  client code lives in `src/client`, and `src/main` cannot reference client
  classes.
- Find the `ModInitializer` and `ClientModInitializer` classes that the
  entrypoints name, and any registry helper the project already has, such as
  a `ModItems.register` method.
- Read the Java version from `options.release` or the toolchain in the build.
  Minecraft 1.20.5 to 1.21.x require Java 21; 1.18 through 1.20.4 require
  Java 17. The year-based versions from 26.1 require Java 25 and ship without
  obfuscation, so Yarn mappings end at 1.21.11 and the build has no mappings
  dependency; check the Fabric release notes for a version newer than this
  list.

## 2. Change the code

- Register content from the `ModInitializer` through the project's own
  registry helper, or through `Registry.register` with an identifier in the
  mod id namespace, such as `Identifier.of(MOD_ID, "sapphire")`. Since
  Minecraft 1.21.2, item and block settings need their registry key, or the
  game fails at startup. Follow the existing helper, which sets it; without
  one, build the key and pass it to the settings before registering under the
  same key: with Yarn names, `RegistryKey.of(RegistryKeys.ITEM, id)` and
  `new Item.Settings().registryKey(key)`; with Mojang names,
  `ResourceKey.create(Registries.ITEM, id)` and
  `new Item.Properties().setId(key)`.
- Keep client-only code, such as renderers, screens, key bindings, and
  client networking receivers, in the `client` entrypoint or the client
  source set. Code that the server loads must not reference client classes.
- Prefer a Fabric API event or callback over a Mixin. Write a Mixin only when
  no API hook exists, keep it small, target one method precisely, and declare
  it in the Mixin config array for its side: `client`, `server`, or `mixins`
  for both.
- Put display names in `src/main/resources/assets/<modid>/lang/en_us.json`
  with the game's key format, such as `item.<modid>.sapphire`. Put models and
  textures under the same `assets/<modid>/` tree.
- When the project has a `fabric-datagen` entrypoint, add recipes, loot
  tables, tags, models, and lang entries through its data providers instead
  of writing the generated JSON by hand.
- Do not change `minecraft_version`, the mappings, `loader_version`,
  `fabric_version`, the Loom version, the Java version, the mod `id`, the
  `depends` block, the existing Mixin configs, or
  `splitEnvironmentSourceSets()` unless the task asks for it (adding a Mixin
  config that a needed Mixin requires is part of that Mixin, not a change to
  this list), because each change can break the mod's class references,
  resource paths, or load order.

## 3. Verify

Run the project's own task when it defines one. Otherwise run through the
wrapper:

| Check | Command | Pass condition |
| --- | --- | --- |
| Compile, remap, and test | `./gradlew build` | Exits 0 |
| Data generation | `./gradlew runDatagen` when the project has a `fabric-datagen` entrypoint | Exits 0, and the generated files match the change |
| Game tests | `./gradlew runGametest` when the project has game tests | Exits 0 |

Run `./gradlew runClient` or `./gradlew runServer` only when the task asks for
it, because each starts the game.

When the project has no Gradle wrapper or the machine lacks the JDK that the
Minecraft version needs, report the checks as not run, with the reason,
instead of using another Gradle or claiming they passed. Fix a failure the
change caused before handoff. Report a failure the change did not cause as an
existing failure instead of fixing unrelated code.

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
