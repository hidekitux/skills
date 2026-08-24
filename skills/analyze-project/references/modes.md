# Investigation modes

Modes run inside the shared Discover → Investigate → Validate → Prioritize → Report flow in `SKILL.md`. Load only the mode the objective calls for; multi-mode runs keep the shared evidence, prioritization, and report rules. A mode never changes the read-only guardrails: findings cite file or command evidence, recommendations name their next-owner skill, and change candidates go to `create-issue`.

## errors

- **Objective:** Analyze error messages, logs, stack traces, and failed runs; stop before any fix.
- **Method:** Classify the error (category, surface, first occurrence, frequency); collect the exact error text, logs, stack traces, and reproduction conditions (input, environment, command); form root-cause hypotheses, each citing the file or command that produced its evidence; recommend remediation and reproduction steps.
- **Evaluation criteria:** Hypotheses with evidence rank above unverified explanations; diagnosis ends at recommendations.
- **Guardrails:** Never fix code or claim a verified fix; reproduction through fix belongs to `debug-code`.
- **Next owner:** `debug-code`; governed change candidates go to `create-issue`.

## tests

- **Objective:** Analyze test runs, failures, coverage signals, and suite structure.
- **Method:** Classify failures (suite, environment, first failing assertion or symptom, order dependence); correlate failures with recent changes and CI runs; identify coverage or case-design gaps tied to concrete missed defects.
- **Evaluation criteria:** Failure claims cite actual run output; gap claims name a scenario the suite misses today.
- **Guardrails:** Never write, rewrite, or fix tests; test design belongs to `write-tests`.
- **Next owner:** `write-tests`; governed change candidates go to `create-issue`.

## dependencies

- **Objective:** Analyze dependency declarations, lockfiles, and available upgrade evidence.
- **Method:** Inventory declared and locked dependencies; check drift from declared versions, compatibility constraints, and license exposure; surface vulnerability warnings from a scanner or audit command run now.
- **Evaluation criteria:** Every risk cites a manifest or lockfile path and current command output; upstream advisories without current tool output are not findings.
- **Guardrails:** Never apply upgrades or modify lockfiles; never claim upstream advisories or a scanner prove a project is secure without current output.
- **Next owner:** `create-issue` (governed Change flow).

## docs

- **Objective:** Analyze documentation against current repository behavior.
- **Method:** Audit command references, setup and release instructions, configuration examples, architecture claims, and required outputs or artifacts; run commands when safe to verify them.
- **Evaluation criteria:** Findings cite the document path and the behavior evidence (command result or code path); commands that cannot be run are flagged as unverified, never assumed.
- **Guardrails:** Never edit documents; never claim a command works without running it when it can be run safely.
- **Next owner:** `create-issue`.

## performance

- **Objective:** Analyze runtime measurements, resource usage, and hot paths.
- **Method:** Collect repeatable measurements for the scoped workload; identify slow paths and resource pressure; correlate symptoms with changes or configuration; separate measured evidence from hypotheses and label hypotheses as such.
- **Evaluation criteria:** Improvement claims require repeatable measurement; hypotheses without measurement rank below measured findings.
- **Guardrails:** Never optimize code or change configuration; never claim an improvement without repeatable measurement; never benchmark a production system without prior approval or safety review.
- **Next owner:** `create-issue`.

## security

- **Objective:** Analyze attack surface, secret handling, and trust boundaries.
- **Method:** Map attack surface and trust boundaries from entry points, inputs, authentication, and network paths; check credential and secret handling (storage, rotation, exposure in logs or artifacts); review input validation, authentication, and network paths; recommend mitigation candidates.
- **Evaluation criteria:** Findings cite file or command evidence; severity reflects exploitability and reachability.
- **Guardrails:** Never apply security changes; never claim a project is secure because a scanner or pattern check passed; threat-model and vulnerability-report artifacts stay with dedicated security workflows.
- **Next owner:** `create-issue`.