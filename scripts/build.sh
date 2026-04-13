#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/build.sh [--targets TARGETS] [--output-dir DIR] [--name NAME] [--main PACKAGE] [--cgo-enabled 0|1]

Options:
  --targets TARGETS     Comma-separated GOOS/GOARCH pairs.
                        Default: darwin/amd64,darwin/arm64,linux/amd64,linux/arm64,windows/amd64,windows/arm64
  --output-dir DIR      Output directory for built binaries. Default: ./bin
  --name NAME           Output binary base name. Default: nextunnel
  --main PACKAGE        Go package to build. Default: .
  --cgo-enabled 0|1     CGO_ENABLED value. Default: 0
  -h, --help            Show this help message

Examples:
  ./scripts/build.sh
  ./scripts/build.sh --targets linux/amd64,linux/arm64
  ./scripts/build.sh --output-dir ./release --name ntunnel
EOF
}

OUTPUT_DIR="./bin"
BINARY_NAME="nextunnel"
MAIN_PACKAGE="."
CGO_VALUE="0"
TARGETS="darwin/amd64,darwin/arm64,linux/amd64,linux/arm64,windows/amd64,windows/arm64"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --targets)
      TARGETS="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --name)
      BINARY_NAME="$2"
      shift 2
      ;;
    --main)
      MAIN_PACKAGE="$2"
      shift 2
      ;;
    --cgo-enabled)
      CGO_VALUE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if ! command -v go >/dev/null 2>&1; then
  echo "go is required but not found in PATH" >&2
  exit 1
fi

if [[ "$CGO_VALUE" != "0" && "$CGO_VALUE" != "1" ]]; then
  echo "--cgo-enabled must be 0 or 1" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IFS=',' read -r -a TARGET_LIST <<<"$TARGETS"
if [[ ${#TARGET_LIST[@]} -eq 0 ]]; then
  echo "No build targets provided" >&2
  exit 1
fi

case "$OUTPUT_DIR" in
  /*)
    OUTPUT_PATH="$OUTPUT_DIR"
    ;;
  *)
    OUTPUT_PATH="${ROOT_DIR}/${OUTPUT_DIR#./}"
    ;;
esac

mkdir -p "$OUTPUT_PATH"

pushd "$ROOT_DIR" >/dev/null

BUILT_FILES=()

for target in "${TARGET_LIST[@]}"; do
  if [[ ! "$target" =~ ^[A-Za-z0-9_]+/[A-Za-z0-9_]+$ ]]; then
    echo "Invalid target: $target" >&2
    echo "Expected format: GOOS/GOARCH" >&2
    exit 1
  fi

  goos="${target%%/*}"
  goarch="${target##*/}"
  ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  output_file="${OUTPUT_PATH}/${BINARY_NAME}_${goos}_${goarch}${ext}"
  echo "Building ${goos}/${goarch} -> ${output_file}"

  env \
    CGO_ENABLED="$CGO_VALUE" \
    GOOS="$goos" \
    GOARCH="$goarch" \
    go build -trimpath -o "$output_file" "$MAIN_PACKAGE"

  BUILT_FILES+=("$output_file")
done

popd >/dev/null

echo
echo "Build completed successfully. Generated files:"
for file in "${BUILT_FILES[@]}"; do
  echo "  $file"
done
