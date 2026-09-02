# Repository guidance

- Publishable skills belong in `skills/<skill-name>/` or `skills/<namespace>/<skill-name>/`.
- Give every skill a `SKILL.md` with `name` and `description` frontmatter. The `name` must match its directory name.
- License every published skill as `Apache-2.0`, the repository standard. Do not add a different license or a non-Apache dependency bundled with a published skill without explicit user approval. Development and CI tools are not bundled dependencies; keep them pinned in `mise.toml`, use them only for development or checks, and review their licenses when adding them.
- Keep `LICENSE` as the unmodified Apache-2.0 legal text and keep the repository copyright attribution in `NOTICE`.
- Set the `NOTICE` copyright owner from the repository owner's confirmed identity. Update the year only when new copyrightable material is added or materially updated; do not create annual-only copyright commits.
- Keep `SKILL.md` concise. Put optional scripts, detailed references, and output assets in `scripts/`, `references/`, and `assets/` below the skill root.
- Do not put credentials, tokens, private URLs, or user data in the repository.
- Use mise as the project command entry point. Run supported-platform repository checks with `mise run validate:all`; do not document or automate a direct replacement command when a mise task exists.
- Keep the repository's required tools and their pinned versions in `mise.toml`. Do not add a tool, a version, or a task unless this repository actually needs it.
- Before publishing, run `mise run validate:all` from the repository root. It
  includes installation validation for Codex and Claude Code. When
  `skill-creator` is available, also run `mise run validate:skill-creator` as
  additional Codex-specific authoring evidence; it is not required to use a
  skill in Claude Code.
- Before publishing, align every catalog version with the release tag, commit the release contents, and run `mise run verify:release -- vX.Y.Z`. Publish only the verified commit with `gh skill publish --tag vX.Y.Z`; never overwrite an existing release tag.

## Todo List contract

- Every publishable skill must begin each invocation by creating and maintaining a **Todo List**. For a one-step request, use one item; do not omit the list.
- Represent the requested outcome as small, observable items. Include discovery or scope confirmation, implementation, validation, and handoff when they apply.
- Keep exactly one item in progress. Mark an item complete only after its stated result or evidence exists; add, remove, or reorder items when the agreed scope changes.
- Use the host's native Todo List or task-tracking capability when it is available. Otherwise, present and maintain an equivalent Markdown checklist in the conversation. The skill must not depend on a proprietary task tool.
- Before handoff, complete or explicitly explain every remaining Todo List item. Keep the final report consistent with the list and the checks actually run.

## Creating and updating skills

- When `skill-creator` is available, invoke it before creating a new skill or
  making a substantial update to an existing one. In hosts where it is not
  available, use `docs/skill-brief-template.md` to collect the same design
  inputs, then run the repository validations.
- Treat this repository as the explicit destination: initialize a new publishable skill below `skills/`, not in a local agent directory.
- Before authoring, collect concrete trigger examples, the intended output, non-goals, required tools or data, and a verification method. Use `docs/skill-brief-template.md` when this information is not already present in the request.
- Design every new or substantially updated skill to follow the Todo List contract above. Put its task-specific initial items and completion evidence in `SKILL.md`; do not rely only on this repository guidance being present at installation time.
- Follow `skill-creator`'s initialization and validation workflow. Use its `init_skill.py` for a new skill, its `quick_validate.py` for the skill-level check, and `gh skill publish --dry-run` for repository-level validation.
- Generate `agents/openai.yaml` through `skill-creator` when creating a Codex-facing skill. Read its `openai_yaml.md` reference first and keep the generated UI metadata aligned with `SKILL.md`.
- Add `scripts/`, `references/`, and `assets/` only when they are reusable. Test every added script with a representative input.
- Forward-test complex or high-impact skills with realistic requests that do not reveal the expected answer.
- When bootstrapping a project, make mise the standard entry point. Define only the applicable `format:all`, `lint:all`, `test:all`, `check:all`, and `verify:fsl` tasks; `check` should compose the relevant validations.

## Host compatibility

- Keep the core behavior, output contract, and safety rules in the skill's `SKILL.md`; do not duplicate a complete skill per agent.
- Put a host-specific capability note at `skills/<skill-name>/references/hosts/<host>.md` only when the difference changes execution, safety, or the output. State the capability, the preferred path, the fallback, and how to verify the result.
- Use `agents/openai.yaml` only for Codex UI metadata. Put repository-level configuration examples and installation notes in `hosts/codex/` or `hosts/claude-code/`; these directories are not publishable skills.
- Do not put shared source in `.codex/`, `.claude/`, or `.agents/`. Those hidden directories are local installation state and are intentionally ignored.

- Use FSL for stateful workflow contracts such as review, validation, publishing, versioning, and deprecation. Do not claim that FSL verifies the prose instructions in a `SKILL.md`.
- Place a skill-owned FSL source file in `skills/<skill-name>/specs/`; expose it in the repository through a relative symbolic link at `specs/<skill-name>/`. Place only repository-owned or cross-skill FSL sources directly in `specs/`. Before authoring or changing one, obtain confirmation of a formalization memo for choices that affect behavior. Expose FSL validation through `mise run verify:fsl` after changes.

## Writing quality

- Name a thing in full on first mention and reuse that exact term for every later mention. Do not introduce a shortened or varied form without defining it.
- Choose the plain word. Do not use an inflated style word, a Latinate padding word, or `serves as` where `is` fits.
- Choose a word people actually use. A replacement nobody says aloud fails the rules in either language, and so does prose assembled from such words.
- In Japanese prose, replace a katakana loanword or an inline English word when a common Japanese word says the same thing, and keep terms already established in the domain. Keep every identifier, command, and path in its original form.
- Name one concept with one term in each language. Take the Japanese term for an English term this repository uses from `docs/term-glossary.md`, and decide a term the glossary omits by the test in `docs/writing-style.md`.
- Make every sentence add a fact the reader did not have. Delete a sentence that loses nothing.
- State a position and give its reason. Do not present two options and commit to neither.
- Cite the file, command, or output behind every claim about the repository.
- See `docs/writing-style.md` for the full rules. They sit in three parts: one for both languages, one for English, and one for Japanese. Read the shared part and the part for the language you are writing in.
