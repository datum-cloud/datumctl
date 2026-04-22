#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${MAN_PAGES_OUTPUT_DIR:-$ROOT_DIR/.generated/man/man1}"

rm -rf "${OUTPUT_DIR}"
mkdir -p "${OUTPUT_DIR}"

pushd "${ROOT_DIR}" >/dev/null
go run main.go docs generate-man-pages --output-dir "${OUTPUT_DIR}"
popd >/dev/null
