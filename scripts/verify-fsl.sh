#!/usr/bin/env bash
set -euo pipefail

cache_root="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/skills-fslc"
export PATH="${FSLC_BIN_DIR:-${cache_root}/bin}:${PATH}"
depth="${FSL_DEPTH:-8}"
found=0

while IFS= read -r -d '' spec; do
  found=1
  echo "Checking ${spec}"
  fslc check "${spec}"
  echo "Verifying ${spec} at depth ${depth}"
  fslc verify "${spec}" --depth "${depth}"
done < <(find specs -type f -name '*.fsl' -print0)

if [[ "${found}" -eq 0 ]]; then
  echo "No FSL specs found in specs/."
fi
