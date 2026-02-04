#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="${DOCS_SOURCE_DIR:-$ROOT_DIR/docs}"
ARCHIVE_PATH="${DOCS_ARCHIVE_PATH:-$ROOT_DIR/.generated/datumctl-docs.tar.gz}"

if [ ! -d "${DOCS_DIR}" ]; then
  echo "Docs directory not found: ${DOCS_DIR}" >&2
  exit 1
fi

mkdir -p "$(dirname "${ARCHIVE_PATH}")"
rm -f "${ARCHIVE_PATH}"
tar -C "${DOCS_DIR}" -czf "${ARCHIVE_PATH}" .
