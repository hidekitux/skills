#!/usr/bin/env bash
# Validate that every published skill can be installed for each supported host.

set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
validation_tmp=$(mktemp -d "${TMPDIR:-/tmp}/skills-host-validation.XXXXXX")
trap 'rm -rf "$validation_tmp"' EXIT

skill_names=()
while IFS= read -r skill_file; do
  skill_names+=("$(basename "$(dirname "$skill_file")")")
done < <(find "$repo_root/skills" -type f -name SKILL.md -print | LC_ALL=C sort)

if ((${#skill_names[@]} == 0)); then
  echo "No publishable skills found." >&2
  exit 1
fi

for host in codex claude-code; do
  case "$host" in
    codex) install_root=".agents/skills" ;;
    claude-code) install_root=".claude/skills" ;;
  esac

  host_root="$validation_tmp/$host"
  mkdir -p "$host_root"
  git -C "$host_root" init -q
  (
    cd "$host_root"
    gh skill install "$repo_root" --from-local --all --agent "$host" --scope project
  )

  for skill_name in "${skill_names[@]}"; do
    skill_file="$host_root/$install_root/$skill_name/SKILL.md"
    if [[ ! -f "$skill_file" ]]; then
      echo "$host did not install $skill_name at $install_root." >&2
      exit 1
    fi
  done

  echo "$host installation validated: ${#skill_names[@]} skill(s) in $install_root."
done
