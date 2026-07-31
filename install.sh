#!/bin/sh
# minepulse installer — fetches the right prebuilt binary from GitHub Releases,
# verifies its checksum, and installs it. No Go toolchain or package manager needed.
#
#   curl -fsSL https://raw.githubusercontent.com/docked-titan-foundation/minepulse/main/install.sh | sh
#
# Env overrides:
#   VERSION   version to install (e.g. v1.2.0); default: latest release
#   BINDIR    install directory; default: /usr/local/bin (falls back to ~/.local/bin)
#   OS ARCH   override target detection (linux|darwin, amd64|arm64)
set -eu

REPO="docked-titan-foundation/minepulse"
BINARY="minepulse"

info() { printf '%s\n' "minepulse: $*" >&2; }
err() {
	printf '%s\n' "minepulse: error: $*" >&2
	exit 1
}

have() { command -v "$1" >/dev/null 2>&1; }

# ── HTTP helper (curl or wget) ────────────────────────────────────────────────
if have curl; then
	dl() { curl -fsSL "$1" -o "$2"; }
	fetch() { curl -fsSL "$1"; }
elif have wget; then
	dl() { wget -qO "$2" "$1"; }
	fetch() { wget -qO- "$1"; }
else
	err "need curl or wget"
fi

# ── Detect OS / arch ──────────────────────────────────────────────────────────
OS="${OS:-$(uname -s | tr '[:upper:]' '[:lower:]')}"
case "$OS" in
linux | darwin) ;;
*) err "unsupported OS '$OS' (this installer covers linux and darwin; on Windows use Scoop — see the roadmap, or download from Releases)" ;;
esac

ARCH="${ARCH:-$(uname -m)}"
case "$ARCH" in
x86_64 | amd64) ARCH="amd64" ;;
aarch64 | arm64) ARCH="arm64" ;;
*) err "unsupported arch '$ARCH' (supported: amd64, arm64)" ;;
esac

# ── Resolve version ───────────────────────────────────────────────────────────
VERSION="${VERSION:-}"
if [ -z "$VERSION" ]; then
	info "resolving latest release…"
	VERSION="$(fetch "https://api.github.com/repos/${REPO}/releases/latest" |
		grep '"tag_name":' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
	[ -n "$VERSION" ] || err "could not resolve the latest release (rate-limited? set VERSION=vX.Y.Z)"
fi
NUM="${VERSION#v}" # goreleaser asset names carry no leading v

ASSET="${BINARY}_${NUM}_${OS}_${ARCH}.tar.gz"
CHECKSUMS="${BINARY}_${NUM}_checksums.txt"
BASE="https://github.com/${REPO}/releases/download/${VERSION}"

# ── Download + verify ─────────────────────────────────────────────────────────
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT INT TERM

info "downloading ${ASSET} (${VERSION})…"
dl "${BASE}/${ASSET}" "${TMP}/${ASSET}" || err "download failed: ${BASE}/${ASSET}"

if dl "${BASE}/${CHECKSUMS}" "${TMP}/${CHECKSUMS}" 2>/dev/null; then
	expected="$(grep " ${ASSET}\$" "${TMP}/${CHECKSUMS}" | awk '{print $1}')"
	if [ -n "$expected" ]; then
		if have sha256sum; then
			actual="$(sha256sum "${TMP}/${ASSET}" | awk '{print $1}')"
		elif have shasum; then
			actual="$(shasum -a 256 "${TMP}/${ASSET}" | awk '{print $1}')"
		else
			actual=""
			info "no sha256 tool found; skipping checksum verification"
		fi
		if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
			err "checksum mismatch for ${ASSET} (expected ${expected}, got ${actual})"
		fi
		[ -n "$actual" ] && info "checksum verified"
	fi
else
	info "checksums not available; skipping verification"
fi

tar -xzf "${TMP}/${ASSET}" -C "${TMP}" || err "failed to extract ${ASSET}"
[ -f "${TMP}/${BINARY}" ] || err "archive did not contain ${BINARY}"
chmod +x "${TMP}/${BINARY}"

# ── Install ───────────────────────────────────────────────────────────────────
BINDIR="${BINDIR:-/usr/local/bin}"
install_to() { # dir
	mkdir -p "$1" 2>/dev/null || return 1
	if [ -w "$1" ]; then
		mv "${TMP}/${BINARY}" "$1/${BINARY}"
	elif have sudo; then
		info "installing to $1 (needs sudo)…"
		sudo mv "${TMP}/${BINARY}" "$1/${BINARY}"
	else
		return 1
	fi
}

if install_to "$BINDIR"; then
	DEST="$BINDIR"
else
	DEST="${HOME}/.local/bin"
	info "cannot write ${BINDIR}; installing to ${DEST} instead"
	install_to "$DEST" || err "could not install to ${DEST}"
fi

info "installed ${BINARY} ${VERSION} to ${DEST}/${BINARY}"
case ":${PATH}:" in
*":${DEST}:"*) ;;
*) info "note: ${DEST} is not on your PATH — add it, e.g. export PATH=\"${DEST}:\$PATH\"" ;;
esac
info "run: ${BINARY} --help"
