# Skill graph

`workflow/skill-graph.yml` is the authoritative source for reusable skill
capabilities, artifacts, prerequisites, authority boundaries, outcomes, and
handoffs. The graph describes contracts only. It does not execute skills,
select models, or enforce host permissions.

## Read the graph

Use `cmd/read-skill-graph` when a consumer needs the full graph or one named
skill. The command returns versioned JSON. Use `cmd/validate-skill-graph` to
check the graph, the catalog, discovered skill paths, and this document.

## Skill inventory

The inventory below is a human-facing view of the graph. The repository check
requires every graph node to occur exactly once between the markers.

<!-- skills:graph-inventory:start -->
- `analyze-project`
- `audit-workflow-enforcement`
- `bootstrap-project`
- `create-issue`
- `create-pr`
- `debug-code`
- `deliver-change`
- `fix-pr`
- `implement-issue`
- `improve-project`
- `merge-pr`
- `plan-issue`
- `refactor-code`
- `resolve-defect`
- `review-pr`
- `write-tests`
<!-- skills:graph-inventory:end -->

## Validation boundary

The graph validator checks structural completeness, reference integrity,
authority values, documentation drift, and bounded cycles. It does not prove
that a skill follows its `SKILL.md` prose or that a host enforces the declared
authority. Behavioral evaluation and cross-skill FSL replay own those claims.
