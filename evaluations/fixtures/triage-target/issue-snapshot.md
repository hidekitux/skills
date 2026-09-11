# Issue snapshot

The export was captured for a planning review. Every item is still open in the
tracker unless the entry says otherwise.

## #41 Add an audit trail

Goal: record an audit event for every account change.

Scope:

- Add an event writer under `src/audit/`.
- Redact account identifiers before persistence.
- Document retention behavior.

Acceptance criteria:

- Account identifiers never appear in stored events.
- Retention is configurable.

## #42 Record account audit events

Goal: record an audit event for every account change.

Scope:

- Add an event writer under `src/audit/`.
- Preserve the actor role in each event.
- Add a retention configuration example.

Acceptance criteria:

- Account identifiers are redacted before persistence.
- Actor roles remain queryable.
- Retention is configurable.

## #43 Rename the import command

Goal: update documentation and scripts from `cmd/legacy-import` to
`cmd/import-data`.

Acceptance criteria:

- No current documentation invokes `cmd/legacy-import`.
- The supported command is `cmd/import-data`.

## #44 Add CSV import support

Goal: support CSV records in the import service.

Scope:

- Parse CSV rows in `src/import/csv.go`.
- Wire the parser into `cmd/import-data`.
- Add parser and command tests.

Acceptance criteria:

- CSV rows parse into import records.
- The command accepts a CSV input path.
- Parser and command tests pass.

## #45 Split deployment configuration

Goal: split deployment settings into independently deployable modules.

Dependencies: blocked by #46 because the module boundary must be agreed first.

Priority: High.

## #46 Define the deployment module boundary

Goal: document the module boundary needed by #45.

Dependencies: none stated.

Priority: Medium.

## #47 Refresh the release banner

Goal: change the release banner text in the web shell.

Dependencies: none stated.

Priority: Low.
