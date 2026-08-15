#!/usr/bin/env bash
set -euo pipefail

fsl_version="4.2.0"
download_base="https://github.com/ymm-oss/fsl/releases/download/v${fsl_version}"
cache_root="${RUNNER_TEMP:-${TMPDIR:-/tmp}}/skills-fslc"
bin_dir="${FSLC_BIN_DIR:-${cache_root}/bin}"
bin_path="${bin_dir}/fslc"

case "$(uname -s):$(uname -m)" in
  Linux:x86_64)
    asset="fslc-linux-x64"
    expected_sha256="194bfdc65586eec280a0d608bf0312af00a5a8628df4d8bdf0ce712349e247ab"
    ;;
  Darwin:arm64)
    asset="fslc-macos-arm64"
    expected_sha256="ca07ff2edc3faba9724a5152395281dce3ba673f3550cbec5d32d20e97e6ddf4"
    ;;
  *)
    echo "Unsupported platform for fslc ${fsl_version}: $(uname -s) $(uname -m)" >&2
    exit 2
    ;;
esac

if [[ -x "${bin_path}" ]] && [[ "$(shasum -a 256 "${bin_path}" | awk '{print $1}')" == "${expected_sha256}" ]]; then
  echo "fslc ${fsl_version} is already installed at ${bin_path}"
  exit 0
fi

mkdir -p "${bin_dir}"
temp_path="$(mktemp "${TMPDIR:-/tmp}/fslc.XXXXXX")"
trap 'rm -f "${temp_path}"' EXIT

curl --fail --location --proto '=https' --tlsv1.2 --silent --show-error \
  "${download_base}/${asset}" --output "${temp_path}"

actual_sha256="$(shasum -a 256 "${temp_path}" | awk '{print $1}')"
if [[ "${actual_sha256}" != "${expected_sha256}" ]]; then
  echo "fslc ${fsl_version} checksum verification failed" >&2
  echo "expected: ${expected_sha256}" >&2
  echo "actual:   ${actual_sha256}" >&2
  exit 1
fi

install -m 0755 "${temp_path}" "${bin_path}"
echo "Installed fslc ${fsl_version} at ${bin_path}"
