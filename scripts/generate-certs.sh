#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  ./scripts/generate-certs.sh [--output-dir DIR] [--server-name NAME] [--server-ip IP] [--days N] [--force]

Options:
  --output-dir DIR      Output directory for generated certificates. Default: ./certs
  --server-name NAME    Server certificate DNS SAN / CN. Default: localhost
  --server-ip IP        Server certificate IP SAN. Default: 127.0.0.1
  --days N              Certificate validity in days. Default: 3650
  --force               Overwrite existing certificate files
  -h, --help            Show this help message

Generated files:
  ca.key, ca.crt
  server.key, server.crt
  client.key, client.crt
EOF
}

OUTPUT_DIR="./certs"
SERVER_NAME="localhost"
SERVER_IP="127.0.0.1"
DAYS="3650"
FORCE="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --server-name)
      SERVER_NAME="$2"
      shift 2
      ;;
    --server-ip)
      SERVER_IP="$2"
      shift 2
      ;;
    --days)
      DAYS="$2"
      shift 2
      ;;
    --force)
      FORCE="true"
      shift
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

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required but not found in PATH" >&2
  exit 1
fi

case "$DAYS" in
  ''|*[!0-9]*)
    echo "--days must be a positive integer" >&2
    exit 1
    ;;
esac

mkdir -p "$OUTPUT_DIR"

for file in ca.key ca.crt server.key server.crt client.key client.crt; do
  if [[ -e "$OUTPUT_DIR/$file" && "$FORCE" != "true" ]]; then
    echo "Refusing to overwrite existing file: $OUTPUT_DIR/$file" >&2
    echo "Re-run with --force if you want to replace the generated certificates." >&2
    exit 1
  fi
done

umask 077

SERVER_EXT="$(mktemp)"
CLIENT_EXT="$(mktemp)"
cleanup() {
  rm -f "$SERVER_EXT" "$CLIENT_EXT"
}
trap cleanup EXIT

cat >"$SERVER_EXT" <<EOF
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth
subjectAltName=DNS:${SERVER_NAME},IP:${SERVER_IP}
EOF

cat >"$CLIENT_EXT" <<'EOF'
basicConstraints=CA:FALSE
keyUsage=digitalSignature,keyEncipherment
extendedKeyUsage=clientAuth
EOF

echo "Generating CA certificate..."
openssl genrsa -out "$OUTPUT_DIR/ca.key" 4096 >/dev/null 2>&1
openssl req -x509 -new -nodes \
  -key "$OUTPUT_DIR/ca.key" \
  -sha256 \
  -days "$DAYS" \
  -out "$OUTPUT_DIR/ca.crt" \
  -subj "/CN=nextunnel-ca" >/dev/null 2>&1

echo "Generating server certificate..."
openssl genrsa -out "$OUTPUT_DIR/server.key" 4096 >/dev/null 2>&1
openssl req -new \
  -key "$OUTPUT_DIR/server.key" \
  -out "$OUTPUT_DIR/server.csr" \
  -subj "/CN=${SERVER_NAME}" >/dev/null 2>&1
openssl x509 -req \
  -in "$OUTPUT_DIR/server.csr" \
  -CA "$OUTPUT_DIR/ca.crt" \
  -CAkey "$OUTPUT_DIR/ca.key" \
  -CAcreateserial \
  -out "$OUTPUT_DIR/server.crt" \
  -days "$DAYS" \
  -sha256 \
  -extfile "$SERVER_EXT" >/dev/null 2>&1

echo "Generating client certificate..."
openssl genrsa -out "$OUTPUT_DIR/client.key" 4096 >/dev/null 2>&1
openssl req -new \
  -key "$OUTPUT_DIR/client.key" \
  -out "$OUTPUT_DIR/client.csr" \
  -subj "/CN=nextunnel-client" >/dev/null 2>&1
openssl x509 -req \
  -in "$OUTPUT_DIR/client.csr" \
  -CA "$OUTPUT_DIR/ca.crt" \
  -CAkey "$OUTPUT_DIR/ca.key" \
  -CAcreateserial \
  -out "$OUTPUT_DIR/client.crt" \
  -days "$DAYS" \
  -sha256 \
  -extfile "$CLIENT_EXT" >/dev/null 2>&1

rm -f \
  "$OUTPUT_DIR/server.csr" \
  "$OUTPUT_DIR/client.csr" \
  "$OUTPUT_DIR/ca.srl"

echo
echo "Certificates generated in: $OUTPUT_DIR"
echo "  CA:      $OUTPUT_DIR/ca.crt"
echo "  Server:  $OUTPUT_DIR/server.crt / $OUTPUT_DIR/server.key"
echo "  Client:  $OUTPUT_DIR/client.crt / $OUTPUT_DIR/client.key"
echo
echo "Use server_name=${SERVER_NAME} in the client TLS config."
