#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT_DIR}/proto"
OUT_DIR="${ROOT_DIR}/app/generated"

REQUEST_PROTO="${PROTO_DIR}/analysis/v1/request.proto"
RESULT_PROTO="${PROTO_DIR}/analysis/v1/result.proto"

mkdir -p "${OUT_DIR}/analysis/v1"

if ! python -c "import grpc_tools.protoc" >/dev/null 2>&1; then
  echo "grpcio-tools is not installed in the current Python environment."
  echo "Install dependencies first: pip install -r ${ROOT_DIR}/requirements.txt"
  exit 1
fi

python -m grpc_tools.protoc \
  -I "${PROTO_DIR}" \
  --python_out="${OUT_DIR}" \
  "${REQUEST_PROTO}" \
  "${RESULT_PROTO}"

echo "Protobuf generated in ${OUT_DIR}/analysis/v1"
