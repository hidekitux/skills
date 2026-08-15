#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${root}/.mise/bin:${PATH}"
depth="${FSL_DEPTH:-8}"
found=0

while IFS= read -r -d '' spec; do
  found=1
  echo "Mutating ${spec} at depth ${depth}"
  fslc mutate "${spec}" --depth "${depth}"
done < <(find specs -type f -name '*.fsl' -print0)

if [[ "${found}" -eq 0 ]]; then
  echo "No FSL specs found in specs/."
fi
