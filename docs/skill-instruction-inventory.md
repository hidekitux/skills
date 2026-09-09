# Skill instruction inventory

Issue #197 keeps one inventory for every published skill. The inventory records
which `SKILL.md` sections load on every invocation, which sections load only
under a named condition, which rules a repository check enforces, and which
duplications the compaction evidence permits removing.

`docs/skill-instruction-inventory.yml` is the machine-readable record. The
`measure-instructions` command checks that it covers every `SKILL.md`, counts
the files with the pinned `cl100k_base` encoding from
`github.com/pkoukk/tiktoken-go` `v0.1.6`, and reports the measured counts.

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
