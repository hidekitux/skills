#!/usr/bin/env bash
set -euo pipefail

root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
source_root="${root}/skills"

found=0
status=0
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/register-local-skills.XXXXXX")
trap 'rm -rf "${tmp_dir}"' EXIT
desired_names="${tmp_dir}/desired-names"
touch "${desired_names}"

while IFS= read -r manifest; do
  [ -f "${manifest}" ] || continue

  found=1
  skill_dir=$(dirname "${manifest}")
  skill_name=$(basename "${skill_dir}")
  printf '%s\n' "${skill_name}" >> "${desired_names}"
done < <(find "${source_root}" -type f -name SKILL.md | sort)

if [ "${found}" -eq 0 ]; then
  echo "No skills found under ${source_root}." >&2
  exit 1
fi

sort -u "${desired_names}" -o "${desired_names}"

normalize_absolute_path() {
  local path="$1"
  local part
  local normalized=""
  local -a parts

  case "${path}" in
    /*) path="${path#/}" ;;
    *) return 1 ;;
  esac

  IFS=/ read -r -a parts <<< "${path}"
  for part in "${parts[@]}"; do
    case "${part}" in
      ""|.) continue ;;
      ..)
        if [ "${normalized}" = "/" ]; then
          continue
        fi
        if [[ "${normalized}" == */* ]]; then
          normalized="${normalized%/*}"
          [ -n "${normalized}" ] || normalized="/"
        else
          normalized="/"
        fi
        ;;
      *)
        if [ -z "${normalized}" ] || [ "${normalized}" = "/" ]; then
          normalized="/${part}"
        else
          normalized="${normalized}/${part}"
        fi
        ;;
    esac
  done

  printf '%s\n' "${normalized:-/}"
}

canonical_source_root=$(normalize_absolute_path "${source_root}")

is_repository_owned_link() {
  local destination="$1"
  local link_target
  local candidate
  local normalized

  link_target=$(readlink "${destination}") || return 1
  case "${link_target}" in
    /*) candidate="${link_target}" ;;
    *) candidate="$(dirname "${destination}")/${link_target}" ;;
  esac
  normalized=$(normalize_absolute_path "${candidate}") || return 1
  case "${normalized}" in
    "${canonical_source_root}"/*) return 0 ;;
    *) return 1 ;;
  esac
}

reconcile_host() {
  local host_root="$1"
  local manifest
  local skill_dir
  local skill_name
  local skill_rel
  local target
  local destination
  local existing_target
  local entry_name

  mkdir -p "${host_root}"

  while IFS= read -r manifest; do
    [ -f "${manifest}" ] || continue

    skill_dir=$(dirname "${manifest}")
    skill_name=$(basename "${skill_dir}")
  # Canonical repository-relative skill path under skills/, e.g. "process/plan-issue"
  # for a category skill or "vendor/refactor-code" for an arbitrary namespace.
    skill_rel=${skill_dir#"${source_root}/"}
    target="../../skills/${skill_rel}"
    destination="${host_root}/${skill_name}"

    if [ -L "${destination}" ]; then
      existing_target=$(readlink "${destination}")
      if [ "${existing_target}" = "${target}" ]; then
        continue
      fi
      if is_repository_owned_link "${destination}"; then
        rm "${destination}"
        ln -s "${target}" "${destination}"
        echo "Updated repository-owned symbolic link: ${destination}"
      else
        echo "Preserving non-owned symbolic link: ${destination}; remove or rename it before rerunning setup." >&2
        status=1
      fi
      continue
    fi

    if [ -e "${destination}" ]; then
      echo "Preserving existing non-symbolic-link entry: ${destination}; remove or rename it before rerunning setup." >&2
      status=1
      continue
    fi

    ln -s "${target}" "${destination}"
  done < <(find "${source_root}" -type f -name SKILL.md | sort)

  for destination in "${host_root}"/*; do
    [ -e "${destination}" ] || [ -L "${destination}" ] || continue
    entry_name=$(basename "${destination}")
    grep -Fqx "${entry_name}" "${desired_names}" && continue

    if [ -L "${destination}" ] && is_repository_owned_link "${destination}"; then
      rm "${destination}"
      echo "Removed stale repository-owned symbolic link: ${destination}"
    elif [ -L "${destination}" ]; then
      echo "Preserving non-owned symbolic link: ${destination}; remove or rename it before rerunning setup." >&2
      status=1
    else
      echo "Preserving existing non-symbolic-link entry: ${destination}; remove or rename it before rerunning setup." >&2
      status=1
    fi
  done
}

reconcile_host "${root}/.agents/skills"
reconcile_host "${root}/.claude/skills"

exit "${status}"
