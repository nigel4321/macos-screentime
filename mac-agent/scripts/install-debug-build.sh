#!/usr/bin/env bash
# Install a CI-built MacAgent.app debug bundle into /Applications.
#
# Strips the com.apple.quarantine xattr that browser-downloaded zips
# inherit, otherwise Gatekeeper rejects the unsigned debug build with
# "MacAgent is damaged and can't be opened" — proper codesign + notarize
# is roadmap §3.9.
#
# Usage:
#   install-debug-build.sh [path-to-zip]
#
# When called with no argument, picks the most recent
# ~/Downloads/MacAgent-debug-*.zip.
set -euo pipefail

if [[ "${1:-}" != "" ]]; then
    ZIP_PATH="$1"
else
    # shellcheck disable=SC2012  # ls -t for mtime ordering is intentional
    ZIP_PATH=$(ls -t "$HOME"/Downloads/MacAgent-debug-*.zip 2>/dev/null | head -1)
    if [[ -z "$ZIP_PATH" ]]; then
        echo "error: no MacAgent-debug-*.zip in ~/Downloads and no argument given" >&2
        echo "usage: $0 [path-to-zip]" >&2
        exit 1
    fi
    echo "Using latest download: $ZIP_PATH"
fi

if [[ ! -f "$ZIP_PATH" ]]; then
    echo "error: $ZIP_PATH not found" >&2
    exit 1
fi

WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

echo "Unzipping into $WORK_DIR"
ditto -x -k "$ZIP_PATH" "$WORK_DIR"

APP_PATH="$WORK_DIR/MacAgent.app"
if [[ ! -d "$APP_PATH" ]]; then
    echo "error: $ZIP_PATH did not contain MacAgent.app at the top level" >&2
    exit 1
fi

echo "Stripping com.apple.quarantine recursively"
xattr -dr com.apple.quarantine "$APP_PATH"

DEST="/Applications/MacAgent.app"
if [[ -d "$DEST" ]]; then
    echo "Removing existing $DEST"
    rm -rf "$DEST"
fi

echo "Installing to $DEST"
ditto "$APP_PATH" "$DEST"

echo
echo "Done. Launch with: open '$DEST'"
