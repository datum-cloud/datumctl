#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${CLI_DOCS_OUTPUT_DIR:-$ROOT_DIR/.generated/cli-docs}"
ARCHIVE_PATH="${CLI_DOCS_ARCHIVE_PATH:-$ROOT_DIR/.generated/datumctl-cli-docs.tar.gz}"

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

pushd "${ROOT_DIR}" >/dev/null
go run main.go docs generate-cli-docs --output-dir "${OUTPUT_DIR}"
popd >/dev/null

mkdir -p "$(dirname "${ARCHIVE_PATH}")"
tar -C "${OUTPUT_DIR}" -czf "${ARCHIVE_PATH}" .
