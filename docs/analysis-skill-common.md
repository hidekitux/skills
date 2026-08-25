# Analysis skill common contract

## Purpose

Every `analyze-*` skill shares one core design: it investigates a scoped area,
gathers evidence, prioritizes findings, and reports recommendations without
changing anything. `analyze-project`
([#76](https://github.com/hidekitux/skills/issues/76)) is the current owner
skill; error, tests, dependencies, docs, performance, and security analysis
fold in as investigation modes (the separate proposals
[#77](https://github.com/hidekitux/skills/issues/77)-
[#82](https://github.com/hidekitux/skills/issues/82) are superseded by
[#112](https://github.com/hidekitux/skills/issues/112)). New analysis skills
follow this contract and the layer rules in
[skill-layers.md](skill-layers.md).

## Discovery

- State the analysis objective and scope before investigating; record what is
  in and out of scope.
- Build an inventory of candidate areas from the repository layout, recent
  history, issue tracker, and CI runs.
- List the inputs the skill may read: repository paths, command output, logs,
  dependency manifests, and test or build results.

## Evidence

- Every finding cites file or command evidence; a finding without evidence is
  not reported.
- Record the commands run and their resulting output or artifacts.
- Distinguish observed facts from inferences, and label inference as such.

## Prioritization

- Rank findings by impact and confidence, and separate quick wins from
  structural issues.
- Group findings that share one root cause so recommendations stay minimal.

## Report format

- Produce a report with an executive summary, findings (each with severity,
  evidence, and location), prioritized recommendations, and a handoff section.
- Findings are recommendations only; the report never contains Issue-creation
  or Pull Request-creation instructions.

## Todo List contract

- Begin by creating a Todo List with discovery, investigation, validation, and
  handoff items; keep exactly one item in progress.
- Complete an item only when its evidence exists; explain unfinished items at
  handoff.

## Read-only guardrails

- Never edit files, run mutating commands, or create Issues or Pull Requests.
- Never document Issue-creation or Pull Request-creation rules; turning
  candidates into Issues belongs to `create-issue`.

## Handoff

- The report names the next-owner skill for each recommendation.
- Candidates for change are handed to `create-issue`; only `create-issue` may
  create Issues from them.
- Analysis stays read-only even when the report recommends changes.
