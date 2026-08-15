#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)

git -C "${root}" config core.hooksPath .githooks
exec bash "${root}/scripts/register-local-skills.sh"
