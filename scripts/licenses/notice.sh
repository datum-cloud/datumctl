#!/usr/bin/env bash
# Regenerates NOTICE from the third-party dependencies of every platform the
# release ships (see .goreleaser.yaml). go-licenses only sees the packages
# built for one GOOS, and some dependencies are platform-specific (the Windows
# credential store, D-Bus on Linux), so each platform is reported separately
# and the sections are merged, one per package, sorted by name.
set -euo pipefail

GO_LICENSES_VERSION=v1.6.0
PLATFORMS=(linux darwin windows)

cd "$(dirname "$0")/../.."

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Build go-licenses for the host; GOOS below applies to the report only.
GOBIN="$tmp/bin" go install "github.com/google/go-licenses@${GO_LICENSES_VERSION}"

mkdir "$tmp/sections"
for goos in "${PLATFORMS[@]}"; do
  GOOS="$goos" CGO_ENABLED=0 "$tmp/bin/go-licenses" report ./... \
    --ignore go.datum.net/datumctl \
    --template scripts/licenses/notice.tmpl 2>"$tmp/$goos.log" |
    awk -v dir="$tmp/sections" '
      /^```$/ { fence = !fence }
      !fence && /^## / {
        name = substr($0, 4)
        gsub("/", "%", name)
        out = dir "/" name
        printf "" > out
      }
      out != "" { print >> out }
    '
done

# The fence toggle above assumes no license text contains a standalone ```
# line. Verify each section held together before trusting the merge: one
# name, matching the header, and fences that closed in pairs. A desynced
# section fails loudly here instead of corrupting NOTICE silently.
verify_section() {
  awk '
    NR == 1 {
      if ($0 !~ /^## /) { err = "missing \"## name\" header" }
      else { header_name = substr($0, 4) }
    }
    NR == 2 && $0 != "" { if (err == "") err = "missing blank line after header" }
    NR == 3 {
      if ($0 !~ /^\* Name: /) { if (err == "") err = "missing \"* Name:\" line" }
      else {
        name_line = substr($0, 9)
        if (name_line != header_name) {
          if (err == "") err = "name mismatch: header \"" header_name "\" vs \"" name_line "\""
        }
      }
    }
    /^\* Name: / { name_count++ }
    /^```$/ { fence_count++ }
    END {
      if (err == "") {
        if (name_count != 1) err = "expected exactly one \"* Name:\" line, found " name_count
        else if (fence_count % 2 != 0) err = "odd number of ``` fence lines (" fence_count ")"
      }
      if (err != "") { print err; exit 1 }
    }
  ' "$1"
}

for section in "$tmp"/sections/*; do
  if ! err=$(verify_section "$section"); then
    echo "error: malformed license section for $(basename "$section"): $err" >&2
    exit 1
  fi
done

# Each section ends with a blank line from the template.
find "$tmp/sections" -type f | sort -f | xargs cat > NOTICE

# Surface packages go-licenses could not find a license for; they appear in
# NOTICE as "License: Unknown".
grep -h '^E' "$tmp"/*.log | sed 's/.*Failed to find license for \([^:]*\):.*/warning: no license found for \1/' | sort -u >&2 || true
