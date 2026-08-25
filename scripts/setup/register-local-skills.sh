#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
source_root="${root}/skills"
stamp="${root}/.agents/worktree-snapshot"

found=0
status=0

# Project Git hooks and shared commitlint live in the common directory shared
# by every worktree, so only the snapshot-dependent registrations below are
# keyed by the checked-out revision.
revision=$(git rev-parse HEAD)
if [[ -f "${stamp}" ]] && [[ "$(cat "${stamp}")" == "${revision}" ]]; then
  echo "Local skill registration is current for ${revision}"
  exit 0
fi

while IFS= read -r manifest; do
  [ -f "${manifest}" ] || continue

  found=1
  skill_dir=$(dirname "${manifest}")
  skill_name=$(basename "${skill_dir}")
  # Canonical repository-relative skill path under skills/, e.g. "refactor-code"
  # for a flat skill or "skills/refactor-code" for a namespaced skill.
  skill_rel=${skill_dir#"${source_root}/"}
  target="../../skills/${skill_rel}"

  for host_root in "${root}/.agents/skills" "${root}/.claude/skills"; do
    destination="${host_root}/${skill_name}"
    mkdir -p "${host_root}"

    if [ -L "${destination}" ]; then
      if [ "$(readlink "${destination}")" != "${target}" ]; then
        echo "Preserving unexpected symbolic link: ${destination}" >&2
        status=1
      fi
      continue
    fi

    if [ -e "${destination}" ]; then
      echo "Preserving existing non-symbolic-link entry: ${destination}" >&2
      status=1
      continue
    fi

    ln -s "${target}" "${destination}"
  done
done < <(find "${source_root}" -type f -name SKILL.md | sort)

if [ "${found}" -eq 0 ]; then
  echo "No skills found under ${source_root}." >&2
  exit 1
fi

if [ "${status}" -eq 0 ]; then
  mkdir -p "$(dirname "${stamp}")"
  printf '%s\n' "${revision}" > "${stamp}"
  echo "Registered ${revision} local skills"
fi

exit "${status}"
