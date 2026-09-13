#!/usr/bin/env bash
set -euo pipefail

setup_root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
setup_state_file="${setup_root}/.agents/setup-state"
source "${setup_root}/scripts/setup/environment-state.sh"
setup_environment_export

setup_lock_acquire() {
  local lock_root=${SETUP_LOCK_ROOT:-${setup_root}/.agents/locks}
  local lock_dir="${lock_root}/setup"
  local started_at

  [[ "${SETUP_LOCK_HELD:-0}" == "1" ]] && return 0
  mkdir -p "${lock_root}"
  started_at=$(date +%s)
  while ! mkdir "${lock_dir}" 2>/dev/null; do
    if (( $(date +%s) - started_at >= 120 )); then
      echo "setup lock timeout; confirm that no setup process remains, then retry" >&2
      return 1
    fi
    sleep 0.1
  done
  printf '%s\n' "$$" > "${lock_dir}/owner"
  export SETUP_LOCK_HELD=1
  setup_lock_dir=${lock_dir}
  setup_lock_owned=1
  trap setup_lock_release EXIT
}

setup_lock_release() {
  [[ "${setup_lock_owned:-0}" == "1" ]] || return 0
  rm -f "${setup_lock_dir}/owner"
  rmdir "${setup_lock_dir}" 2>/dev/null || true
  setup_lock_owned=0
}

state_value() {
  local key="$1"

  [[ -f "${setup_state_file}" ]] || return 0
  awk -F= -v key="${key}" '$1 == key { print substr($0, index($0, "=") + 1); exit }' "${setup_state_file}"
}

hash_paths() {
  (
    cd "${setup_root}"
    {
      for path in "$@"; do
        if [[ -d "${path}" ]]; then
          find "${path}" -type f -print
        elif [[ -f "${path}" ]]; then
          printf '%s\n' "${path}"
        fi
      done
    } | sort | while IFS= read -r path; do
      printf '%s\t' "${path}"
      git hash-object -- "${path}"
    done | git hash-object --stdin
  )
}

bootstrap_inputs() {
  hash_paths "mise.toml" ".githooks" "scripts/setup"
}

validator_inputs() {
  hash_paths "go.mod" "go.sum" "cmd/validate-commit-message" "internal/commitlint"
}

ensure_link() {
  local target="$1"
  local destination="$2"

  mkdir -p "$(dirname "${destination}")"
  if [[ -L "${destination}" ]] && [[ "$(readlink "${destination}")" == "${target}" ]]; then
    return 0
  fi
  if [[ -e "${destination}" || -L "${destination}" ]]; then
    rm -f "${destination}"
  fi
  ln -s "${target}" "${destination}"
}

write_state() {
  local status="$1"
  local revision="$2"
  local bootstrap_fingerprint="$3"
  local validator_fingerprint="$4"
  local state_dir
  local temporary

  state_dir=$(dirname "${setup_state_file}")
  mkdir -p "${state_dir}"
  temporary=$(mktemp "${setup_state_file}.tmp.XXXXXX")
  cat > "${temporary}" <<EOF
format=1
status=${status}
revision=${revision}
bootstrap_inputs=${bootstrap_fingerprint}
validator_inputs=${validator_fingerprint}
EOF
  mv "${temporary}" "${setup_state_file}"
}
