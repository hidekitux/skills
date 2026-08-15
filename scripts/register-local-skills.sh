#!/usr/bin/env bash
set -euo pipefail

root=$(git rev-parse --show-toplevel)
source_root="${root}/skills"

found=0
status=0

for manifest in "${source_root}"/*/SKILL.md; do
  [ -f "${manifest}" ] || continue

  found=1
  skill_name=$(basename "$(dirname "${manifest}")")
  target="../../skills/${skill_name}"

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
done

if [ "${found}" -eq 0 ]; then
  echo "No top-level skills found under ${source_root}." >&2
  exit 1
fi

exit "${status}"
