#!/usr/bin/env bash
# ==============================================================================
# release-candidate.sh — One-command release gate
# ==============================================================================
# Baseline:
#   - make precheck
#   - go vet ./...
#   - upgrade-path migration test (v5 -> v6)
#
# Optional full provider matrix:
#   SAME_RC_FULL_MATRIX=1 .scripts/release-candidate.sh
# ==============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo ""
echo "Release candidate gate"
echo "  Repo: $REPO_ROOT"

cd "$REPO_ROOT"

echo ""
echo "1) Baseline precheck"
make precheck

echo ""
echo "2) Vet"
GOCACHE=/tmp/go-build go vet ./...

echo ""
echo "3) Upgrade-path migration test"
GOCACHE=/tmp/go-build go test ./internal/store -run TestOpenPath_MigratesLegacyV5ToV6 -count=1

if [ "${SAME_RC_FULL_MATRIX:-0}" = "1" ]; then
    echo ""
    echo "4) Full provider matrix"
    make provider-smoke-full
else
    echo ""
    echo "4) Full provider matrix"
    echo "  SKIP  optional (set SAME_RC_FULL_MATRIX=1 to enable)"
fi

echo ""
echo "Release candidate gate complete."
