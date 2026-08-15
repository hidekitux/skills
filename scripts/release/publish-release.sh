#!/usr/bin/env bash
# Publish only after the repository's complete local release gates succeed.
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: mise run release:publish -- vX.Y.Z" >&2
  exit 2
fi

tag="$1"
mise run validate

skill_creator_root="${SKILL_CREATOR_ROOT:-${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator}"
if [ -f "$skill_creator_root/scripts/quick_validate.py" ]; then
  mise run validate-skill-creator
else
  echo "skill-creator validator unavailable; skipping Codex-specific evidence." >&2
fi

mise run verify-release -- "$tag"
exec gh skill publish --tag "$tag"
