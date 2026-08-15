---
name: audit-workflow-enforcement
description: Audit a repository for mandatory workflow rules that lack effective enforcement. Use when adding or reviewing hooks, CI checks, release gates, privacy checks, license evidence, or human-process controls that require a bounded subagent review.
license: Apache-2.0
---

# Audit Workflow Enforcement

## Todo List

1. **in progress:** Read repository instructions and inventory mandatory rules, their sources, and current enforcement evidence.
2. Run the repository's deterministic validation tasks and record every result.
3. Assign the remaining context-dependent checks to independent, read-only low-cost subagents.
4. Classify every rule as enforced, partial, or documented-only; propose an owner and automation boundary for each gap.
5. Complete the list only when the evidence matrix and unresolved risks are handed off; state any skipped check explicitly.

Keep exactly one item in progress. Mark an item complete only after its evidence exists. When native task tracking is unavailable, maintain this list as a Markdown checklist in the conversation.

## Workflow

1. Read `AGENTS.md`, contributor guidance, Skills, workflow files, hooks, tasks, and validators. Treat server-side GitHub controls as separate evidence from local hooks.
2. Run `mise run validate`. For release work, use `mise run release:publish -- vX.Y.Z`; do not replace its gate sequence with direct `gh skill publish`.
3. Do not ask subagents to rerun deterministic checks or edit files. Give each a distinct, read-only question and require file/line or command-output evidence.
4. Combine the results in a matrix: rule, source, enforcement, status, remaining gap, and recommendation. `Enforced` means the control blocks or rejects violations; a checklist or prose instruction is `documented-only`.

## Subagent Audit

Use subagents only for the following non-deterministic checks:

- Contextual disclosure: private-but-not-secret URLs, personal data, or generated noise that pattern checks cannot identify safely.
- Process evidence: whether a Skill's Todo List, intended-file review, and handoff instructions can be followed from the current artifact.
- Enforcement boundaries: whether a rule requires server administration, a human approval, or external-state evidence rather than a repository script.

Assign at most three independent, read-only tasks. Ask each to return findings with evidence and a severity; do not disclose expected findings or implementation plans. The parent agent resolves conflicts and never treats a subagent claim as a passing deterministic check.

Read the host note before selecting a subagent model: [Codex](references/hosts/codex.md) or [Claude Code](references/hosts/claude-code.md). If the requested low-cost model selector is unavailable, use the host's lowest-cost capable model and report the fallback.

## Boundaries

- Scripts can reject known tokens, local/private network URLs, user paths, unreviewed `mise` tools, missing test mappings, and a release sequence invoked through `release:publish`.
- Scripts cannot reliably determine whether a public-looking URL, prose, or file is sensitive in its business context, whether an agent actually maintained a Todo List, or whether a user intended every changed file.
- The `release:publish` task is the required automated path, but it cannot prevent a user with shell and GitHub authority from directly invoking `gh skill publish`. Report that residual bypass rather than claiming server-side enforcement.
