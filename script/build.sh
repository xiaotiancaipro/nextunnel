#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
cd "${ROOT_DIR}"

APP_NAME="nextunnel-server"
MAIN_GO="${ROOT_DIR}/main.go"
OUT_DIR="${ROOT_DIR}/bin"

VERSION="$(
  sed -n -E 's/^[[:space:]]*var[[:space:]]+version[[:space:]]*=[[:space:]]*"([^"]*)".*/\1/p' "${MAIN_GO}" | head -n1
)"
if [[ -z "${VERSION}" ]]; then
  echo "error: cannot read version from ${MAIN_GO} (expect: var version = \"...\")" >&2
  exit 1
fi

LDFLAGS="-s -w"
export CGO_ENABLED=0

PLATFORMS=(
  "darwin:amd64"
  "darwin:arm64"
  "linux:amd64"
  "linux:arm64"
  "windows:amd64"
  "windows:arm64"
)

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

echo "Building ${APP_NAME} ${VERSION} -> ${OUT_DIR}"

for entry in "${PLATFORMS[@]}"; do

  GOOS="${entry%%:*}"
  GOARCH="${entry##*:}"
  export GOOS GOARCH

  suffix=""
  [[ "${GOOS}" == "windows" ]] && suffix=".exe"

  artifact="${APP_NAME}-${VERSION}-${GOOS}-${GOARCH}${suffix}"
  out_path="${OUT_DIR}/${artifact}"

  echo "  ${GOOS}/${GOARCH} -> ${artifact}"

  go build -trimpath -ldflags "${LDFLAGS}" -o "${out_path}" .

done

echo "Done"
