#!/usr/bin/env bash
# scripts/gen-winres.sh — generate Windows PE resource metadata (.syso) via go-winres.
#
# Embeds company/product name, version, and manifest into the .exe so Windows
# Explorer and antivirus heuristics see legitimate PE metadata. This is one of
# the most effective cheap mitigations against AV false positives on Go binaries.
#
# Usage:
#   ./scripts/gen-winres.sh                 # use winres/winres.json as-is
#   ./scripts/gen-winres.sh 0.3.0           # patch version into winres.json first
#
# The .syso files are written to cmd/pharos/ and auto-linked by the Go toolchain
# for windows/* targets. Non-Windows builds ignore them.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WINRES_JSON="$ROOT/winres/winres.json"
OUT_DIR="$ROOT/cmd/pharos"

# Ensure go-winres is available (check GOPATH/bin too, since CI may not have it on PATH)
GOPATH_BIN="$(go env GOPATH)/bin"
if ! command -v go-winres >/dev/null 2>&1 && [ -x "$GOPATH_BIN/go-winres" ]; then
  export PATH="$GOPATH_BIN:$PATH"
fi
if ! command -v go-winres >/dev/null 2>&1; then
  echo "  Installing go-winres..."
  go install github.com/tc-hib/go-winres@latest
  export PATH="$GOPATH_BIN:$PATH"
fi

# Optionally patch the version into winres.json
if [[ $# -ge 1 ]]; then
  VERSION="$1"
  # Pad to 4-part quad (e.g. 0.3.0 -> 0.3.0.0, 1.2.3 -> 1.2.3.0)
  QUAD="$(echo "$VERSION" | awk -F. '{printf "%d.%d.%d.0", $1+0, $2+0, $3+0}')"
  cat > "$WINRES_JSON" <<EOF
{
  "RT_VERSION": {
    "#1": {
      "0000": {
        "fixed": {
          "file_version": "$QUAD",
          "product_version": "$QUAD"
        },
        "info": {
          "0409": {
            "CompanyName": "udit-001",
            "FileDescription": "Pharos - CLI for AI-guided learning workspaces",
            "FileVersion": "$VERSION",
            "InternalName": "pharos",
            "LegalCopyright": "MIT License",
            "OriginalFilename": "pharos.exe",
            "ProductName": "Pharos",
            "ProductVersion": "$VERSION"
          }
        }
      }
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0409": {
        "identity": {
          "name": "Pharos",
          "version": "$QUAD"
        },
        "description": "CLI + read-only web dashboard for AI-guided learning workspaces",
        "execution-level": "as invoker",
        "dpi-awareness": "PerMonitorV2",
        "use-common-controls-v6": true
      }
    }
  }
}
EOF
  echo "  Patched winres/winres.json -> v$VERSION"
fi

mkdir -p "$OUT_DIR"
echo "  Generating .syso files into $OUT_DIR..."
go-winres make --arch amd64,arm64 --in "$WINRES_JSON" --out "$OUT_DIR/rsrc"

echo "  ✓ Windows PE resources generated"
