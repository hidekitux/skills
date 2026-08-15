#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)

git -C "${root}" config core.hooksPath .githooks
bash "${root}/scripts/setup/register-local-skills.sh"
echo "Git commit hooks enabled via core.hooksPath=.githooks"
