# Skill instruction inventory

Issue #197 keeps one inventory for every published skill. The inventory records
which `SKILL.md` sections load on every invocation, which sections load only
under a named condition, which rules a repository check enforces, and which
duplications the compaction evidence permits removing.

`docs/skill-instruction-inventory.yml` is the machine-readable record. The
`measure-instructions` command checks that it covers every `SKILL.md`, counts
the files with the pinned `cl100k_base` encoding from
`github.com/pkoukk/tiktoken-go` `v0.1.6`, and reports the current counts. It
does not compare them with the recorded counts.

Use a `conditional-reference` entry only when `load_condition` names the
trigger and `reference` points to a file below that skill's directory. Keep
Todo, safety, output, authority, and handoff rules in `SKILL.md` even when a
host currently enforces part of them. A section marked
`removable-duplication` needs paired full and compact evidence before the
compact form replaces it.

The `before_tokens` value is the count from the Issue branch base and
`before_commit` records that base. The `after_tokens` value is the count from
the compact source and `after_commit` records the compact source commit. Equal
counts record an audited skill whose entry point remains unchanged because the
evidence did not justify further reduction.

These token counts are the provenance of that compaction, not current sizes. A
later change to a `SKILL.md` updates only its section list when a heading
changes; it leaves `before_tokens`, `after_tokens`, and the commits as they
were. `check-instruction-inventory` in `mise run check:repository` checks the
coverage and the sections, not the counts.
