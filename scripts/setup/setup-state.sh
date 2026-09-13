#!/usr/bin/env bash
set -euo pipefail

setup_root=${SETUP_ROOT:-$(git rev-parse --show-toplevel)}
setup_state_file="${setup_root}/.agents/setup-state"

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
