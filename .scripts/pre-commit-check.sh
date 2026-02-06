#!/usr/bin/env bash
# ==============================================================================
# pre-commit-check.sh — Vault Data Leak Prevention
# ==============================================================================
# Blocks commits that contain personal identifiers, client-sensitive patterns,
# or real vault data. SAME references vault structure generically in its code
# (e.g. _PRIVATE/, 01_Projects/) — that's expected product behavior.
#
# Hard blocks: personal identity, client names, local paths, real API keys.
# These should NEVER appear anywhere in this repo.
# ==============================================================================

set -euo pipefail

RED=""
GREEN=""
YELLOW=""
RESET=""
if [ -t 1 ] 2>/dev/null; then
    RED="\033[0;31m"
    GREEN="\033[0;32m"
    YELLOW="\033[1;33m"
    RESET="\033[0m"
fi

# --- Hard block: personal/client data that must NEVER appear ---
HARD_PATTERNS=(
    # Personal identifiers
    "REDACTED"
    "REDACTED"
    "REDACTED"
    "REDACTED"
    "REDACTED"
    "REDACTED"

    # Local paths (real machine paths, not generic references)
    "/Users/REDACTED"
    "C:\\\\Users\\\\Sean"
    "REDACTED"
    "REDACTED"
    "REDACTED"

    # Client-sensitive (load from blocklist if present)
    "REDACTED_CLIENT"
    "REDACTED_CLIENT"
    "REDACTED_CLIENT"

    # Real API keys (actual key prefixes, not documentation references)
    "sk-ant-api03"
    "sk-proj-[A-Za-z0-9]{20}"
    "AIzaSy[A-Za-z0-9]{30}"
)

# Build grep pattern
PATTERN=""
for p in "${HARD_PATTERNS[@]}"; do
    if [ -z "$PATTERN" ]; then
        PATTERN="$p"
    else
        PATTERN="$PATTERN|$p"
    fi
done

# --- Gather staged files ---
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)

if [ -z "$STAGED_FILES" ]; then
    exit 0
fi

# --- Scan ---
FOUND=0
MATCHES=""

while IFS= read -r file; do
    [ ! -f "$file" ] && continue

    # Skip binary files
    lc_file=$(echo "$file" | tr '[:upper:]' '[:lower:]')
    case "$lc_file" in
        *.png|*.jpg|*.jpeg|*.gif|*.webp|*.exe|*.dll|*.so|*.dylib|*.wasm) continue ;;
    esac

    # Skip this hook itself
    [ "$file" = ".scripts/pre-commit-check.sh" ] && continue
    [ "$file" = ".git/hooks/pre-commit" ] && continue

    SCAN_OUTPUT=$(git show ":$file" 2>/dev/null | grep -inE "$PATTERN" 2>/dev/null || true)

    if [ -n "$SCAN_OUTPUT" ]; then
        FOUND=$((FOUND + 1))
        MATCHES="${MATCHES}\n${YELLOW}--- ${file} ---${RESET}\n${SCAN_OUTPUT}\n"
    fi
done <<< "$STAGED_FILES"

# --- Report ---
if [ "$FOUND" -gt 0 ]; then
    echo ""
    echo -e "${RED}BLOCKED: Personal/client data detected in ${FOUND} file(s)${RESET}"
    echo ""
    echo -e "$MATCHES"
    echo ""
    echo "This repo must not contain personal identifiers, client names, or real paths."
    echo ""
    echo "To fix:"
    echo "  1. Remove the flagged content"
    echo "  2. Use synthetic/generic data instead"
    echo ""
    echo "To bypass (emergency only):"
    echo "  git commit --no-verify"
    echo ""
    exit 1
fi

exit 0
