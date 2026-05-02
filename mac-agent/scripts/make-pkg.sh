#!/bin/bash
#
# Build a tamper-resistant install .pkg for MacAgent (ROADMAP §1.14).
#
# Composes:
#   - /Applications/MacAgent.app                       (the agent binary)
#   - /Library/LaunchAgents/com.macagent.MacAgent.plist (system LaunchAgent)
#   - postinstall script (chowns plist, launchctl bootstraps the active user)
#
# into a single distribution-style .pkg suitable for notarization.
#
# Usage:
#   make-pkg.sh --app /path/to/MacAgent.app --version 0.1.0 [--out dist/]
#
# Without --sign, produces an unsigned .pkg. macOS Gatekeeper will refuse
# to install an unsigned .pkg by double-click, but `installer -pkg` from a
# terminal still works for local testing.
#
# Signing (ROADMAP §3.9 dependency) is opt-in via env var:
#   MACAGENT_INSTALLER_SIGN_IDENTITY="Developer ID Installer: Foo (TEAMID)"
# productbuild then signs the outer distribution package; the binary inside
# MacAgent.app must be Developer-ID-Application-signed separately (mac.yml).
#
# Notarization is left to a separate step (see ROADMAP §3.9):
#   xcrun notarytool submit dist/MacAgent-<v>.pkg --keychain-profile AC --wait
#   xcrun stapler staple   dist/MacAgent-<v>.pkg

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INSTALLER_DIR="${REPO_ROOT}/mac-agent/installer"
PLIST_SRC="${INSTALLER_DIR}/com.macagent.MacAgent.plist"
POSTINSTALL_SRC="${INSTALLER_DIR}/postinstall"

APP_PATH=""
VERSION=""
OUT_DIR="${REPO_ROOT}/mac-agent/dist"

usage() {
    sed -n '/^# Usage/,/^$/p' "$0" | sed 's/^# \{0,1\}//'
    exit 64
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --app)     APP_PATH="$2"; shift 2 ;;
        --version) VERSION="$2";  shift 2 ;;
        --out)     OUT_DIR="$2";  shift 2 ;;
        -h|--help) usage ;;
        *) echo "unknown arg: $1" >&2; usage ;;
    esac
done

if [[ -z "${APP_PATH}" ]]; then
    echo "make-pkg.sh: --app PATH is required" >&2
    usage
fi
if [[ ! -d "${APP_PATH}" ]]; then
    echo "make-pkg.sh: app bundle not found at ${APP_PATH}" >&2
    exit 1
fi
if [[ ! -f "${PLIST_SRC}" ]] || [[ ! -f "${POSTINSTALL_SRC}" ]]; then
    echo "make-pkg.sh: installer assets missing under ${INSTALLER_DIR}" >&2
    exit 1
fi

# Derive version from the .app's Info.plist if not supplied. Keeps the .pkg
# version in lockstep with whatever xcodebuild produced.
if [[ -z "${VERSION}" ]]; then
    VERSION=$(/usr/libexec/PlistBuddy -c "Print :CFBundleShortVersionString" \
        "${APP_PATH}/Contents/Info.plist" 2>/dev/null || true)
fi
if [[ -z "${VERSION}" ]]; then
    echo "make-pkg.sh: could not derive --version from Info.plist; pass --version" >&2
    exit 1
fi

mkdir -p "${OUT_DIR}"
WORK_DIR=$(mktemp -d -t macagent-pkg-XXXXXXXX)
trap 'rm -rf "${WORK_DIR}"' EXIT

# Lay out the .pkg payload root mirroring the target install paths. pkgbuild
# walks this with --root and reproduces the same paths under "/".
PAYLOAD_ROOT="${WORK_DIR}/root"
mkdir -p "${PAYLOAD_ROOT}/Applications"
mkdir -p "${PAYLOAD_ROOT}/Library/LaunchAgents"

# ditto preserves bundle structure (symlinks under Frameworks/, code-signing
# resources). cp -R is unsafe for signed app bundles.
APP_BASENAME="$(basename "${APP_PATH}")"
PAYLOAD_APP="${PAYLOAD_ROOT}/Applications/${APP_BASENAME}"
ditto "${APP_PATH}" "${PAYLOAD_APP}"
cp "${PLIST_SRC}" "${PAYLOAD_ROOT}/Library/LaunchAgents/"

# Strip com.apple.quarantine from the payload copy. The flag is a download
# marker carried in by the source .app (e.g. a CI artifact zip extracted by
# Safari) and survives the ditto above. Left in place, it causes Gatekeeper
# to reject the installed app on first launch with a misleading "damaged"
# message. Once §3.9's notarized release pipeline lands, stapler will have
# already cleared this on the binary — the strip becomes a no-op then.
xattr -dr com.apple.quarantine "${PAYLOAD_APP}" 2>/dev/null || true

# Scripts dir holds postinstall (and preinstall, if we ever need one).
# pkgbuild expects executable files here; we copy with mode preserved.
SCRIPTS_DIR="${WORK_DIR}/scripts"
mkdir -p "${SCRIPTS_DIR}"
install -m 0755 "${POSTINSTALL_SRC}" "${SCRIPTS_DIR}/postinstall"

COMPONENT_PKG="${WORK_DIR}/MacAgent-component.pkg"
OUTPUT_PKG="${OUT_DIR}/MacAgent-${VERSION}.pkg"

echo "make-pkg.sh: building component pkg (version ${VERSION})"
pkgbuild \
    --identifier "com.macagent.MacAgent.pkg" \
    --version    "${VERSION}" \
    --root       "${PAYLOAD_ROOT}" \
    --scripts    "${SCRIPTS_DIR}" \
    --install-location / \
    "${COMPONENT_PKG}"

# productbuild wraps the component into a distribution-style pkg. Notarization
# only accepts distribution packages — a bare component pkg from pkgbuild will
# be rejected by `notarytool submit`.
SIGN_ARGS=()
if [[ -n "${MACAGENT_INSTALLER_SIGN_IDENTITY:-}" ]]; then
    echo "make-pkg.sh: signing with '${MACAGENT_INSTALLER_SIGN_IDENTITY}'"
    SIGN_ARGS=(--sign "${MACAGENT_INSTALLER_SIGN_IDENTITY}")
else
    echo "make-pkg.sh: MACAGENT_INSTALLER_SIGN_IDENTITY unset — producing UNSIGNED pkg"
    echo "make-pkg.sh: this build will not pass Gatekeeper or notarization (ROADMAP §3.9)"
fi

productbuild \
    --package "${COMPONENT_PKG}" \
    --version "${VERSION}" \
    --identifier "com.macagent.MacAgent.distribution" \
    ${SIGN_ARGS[@]+"${SIGN_ARGS[@]}"} \
    "${OUTPUT_PKG}"

echo "make-pkg.sh: wrote ${OUTPUT_PKG}"
