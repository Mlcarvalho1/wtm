#!/usr/bin/env bash
# wtm installation wizard — macOS and Linux.
#
# Usage:
#   ./install.sh            interactive
#   ./install.sh -y         accept defaults, don't prompt
#   curl -fsSL https://raw.githubusercontent.com/Mlcarvalho1/wtm/main/install.sh | bash
#
# Checks for git, tmux and Go, offers to install whichever are missing via
# the platform's package manager, builds/installs the wtm binary, makes
# sure it ends up on $PATH, and writes a default config file.

set -euo pipefail

MODULE="github.com/Mlcarvalho1/wtm"
MIN_GO_MAJOR=1
MIN_GO_MINOR=21
ASSUME_YES=0

# ---------------------------------------------------------------- helpers --

bold() { printf '\033[1m%s\033[0m' "$1"; }
info() { printf '  %s\n' "$1"; }
step() { printf '\n%s %s\n' "$(bold '==>')" "$1"; }
warn() { printf '  %s %s\n' "$(bold '!')" "$1" >&2; }
die() { printf '\n%s %s\n' "$(bold 'error:')" "$1" >&2; exit 1; }

have() { command -v "$1" >/dev/null 2>&1; }

confirm() {
	# confirm "question" — defaults to yes. Always yes under -y / non-tty.
	local prompt="$1"
	if [ "$ASSUME_YES" = 1 ] || [ ! -t 0 ]; then
		return 0
	fi
	local reply
	read -r -p "  $prompt [Y/n] " reply || true
	case "$reply" in
		[nN]*) return 1 ;;
		*) return 0 ;;
	esac
}

while [ $# -gt 0 ]; do
	case "$1" in
		-y|--yes) ASSUME_YES=1 ;;
		-h|--help)
			printf 'Usage: %s [-y|--yes]\n' "$0"
			exit 0
			;;
		*) die "unknown argument: $1" ;;
	esac
	shift
done

# --------------------------------------------------------------- platform --

OS="$(uname -s)"
case "$OS" in
	Darwin) PLATFORM=macos ;;
	Linux) PLATFORM=linux ;;
	*) die "wtm's installer supports macOS and Linux only (detected: $OS)." ;;
esac

PKG_MANAGER=""
if [ "$PLATFORM" = macos ]; then
	have brew && PKG_MANAGER=brew
else
	if have apt-get; then PKG_MANAGER=apt
	elif have dnf; then PKG_MANAGER=dnf
	elif have pacman; then PKG_MANAGER=pacman
	elif have apk; then PKG_MANAGER=apk
	elif have zypper; then PKG_MANAGER=zypper
	fi
fi

pkg_install() {
	# pkg_install <name> — install a package with whatever manager we found.
	local name="$1"
	case "$PKG_MANAGER" in
		brew) brew install "$name" ;;
		apt) sudo apt-get update && sudo apt-get install -y "$name" ;;
		dnf) sudo dnf install -y "$name" ;;
		pacman) sudo pacman -Sy --noconfirm "$name" ;;
		apk) sudo apk add "$name" ;;
		zypper) sudo zypper install -y "$name" ;;
		*) return 1 ;;
	esac
}

step "wtm installation wizard ($PLATFORM)"

# -------------------------------------------------------------------- git --

step "Checking for git"
if have git; then
	info "found $(git --version)"
else
	warn "git not found."
	if [ -n "$PKG_MANAGER" ] && confirm "Install git with $PKG_MANAGER?"; then
		pkg_install git || die "failed to install git — install it manually and re-run."
	else
		die "git is required. Install it manually (e.g. https://git-scm.com/downloads) and re-run."
	fi
fi

# ------------------------------------------------------------------- tmux --

step "Checking for tmux"
if have tmux; then
	info "found $(tmux -V)"
else
	warn "tmux not found."
	if [ -n "$PKG_MANAGER" ] && confirm "Install tmux with $PKG_MANAGER?"; then
		pkg_install tmux || die "failed to install tmux — install it manually and re-run."
	else
		die "tmux is required. Install it manually (e.g. https://github.com/tmux/tmux/wiki/Installing) and re-run."
	fi
fi

# --------------------------------------------------------------------- go --

go_version_ok() {
	# go_version_ok <version-string like "go1.23.4"> — compare against minimum.
	local v="${1#go}"
	local major="${v%%.*}"
	local rest="${v#*.}"
	local minor="${rest%%.*}"
	if [ "$major" -gt "$MIN_GO_MAJOR" ]; then return 0; fi
	if [ "$major" -eq "$MIN_GO_MAJOR" ] && [ "$minor" -ge "$MIN_GO_MINOR" ]; then return 0; fi
	return 1
}

step "Checking for Go"
if have go; then
	GO_VER="$(go env GOVERSION 2>/dev/null || go version | awk '{print $3}')"
	if go_version_ok "$GO_VER"; then
		info "found $GO_VER"
	else
		warn "found $GO_VER, but wtm needs go${MIN_GO_MAJOR}.${MIN_GO_MINOR}+."
		if [ -n "$PKG_MANAGER" ] && confirm "Try upgrading Go with $PKG_MANAGER?"; then
			pkg_install go || die "failed to upgrade Go — install a newer toolchain manually from https://go.dev/dl/ and re-run."
		else
			die "install a newer Go toolchain from https://go.dev/dl/ and re-run."
		fi
	fi
else
	warn "Go not found."
	if [ -n "$PKG_MANAGER" ] && confirm "Install Go with $PKG_MANAGER?"; then
		pkg_install go || die "failed to install Go — install it manually from https://go.dev/dl/ and re-run."
	else
		die "Go is required to build wtm. Install it from https://go.dev/dl/ and re-run."
	fi
fi

have go || die "go still not on PATH after install — open a new shell and re-run this script."

# --------------------------------------------------------------- install --

step "Building and installing wtm"

SCRIPT_SOURCE="${BASH_SOURCE[0]:-}"
SCRIPT_DIR=""
if [ -n "$SCRIPT_SOURCE" ] && [ -f "$SCRIPT_SOURCE" ]; then
	SCRIPT_DIR="$(cd "$(dirname "$SCRIPT_SOURCE")" && pwd)"
fi
if [ -n "$SCRIPT_DIR" ] && [ -f "$SCRIPT_DIR/go.mod" ] && grep -q "^module $MODULE\$" "$SCRIPT_DIR/go.mod" 2>/dev/null; then
	info "installing from local checkout at $SCRIPT_DIR"
	(cd "$SCRIPT_DIR" && go install ./cmd/wtm) || die "go install failed."
else
	info "installing $MODULE@latest"
	go install "$MODULE/cmd/wtm@latest" || die "go install failed."
fi

GOBIN="$(go env GOBIN)"
if [ -z "$GOBIN" ]; then
	GOBIN="$(go env GOPATH)/bin"
fi
info "installed to $GOBIN/wtm"

# ------------------------------------------------------------------- path --

step "Checking PATH"
case ":$PATH:" in
	*":$GOBIN:"*)
		info "$GOBIN is already on PATH"
		;;
	*)
		warn "$GOBIN is not on PATH."
		SHELL_RC=""
		case "$(basename "${SHELL:-}")" in
			zsh) SHELL_RC="$HOME/.zshrc" ;;
			bash) SHELL_RC="$HOME/.bashrc" ;;
			fish) SHELL_RC="$HOME/.config/fish/config.fish" ;;
		esac
		if [ -n "$SHELL_RC" ] && confirm "Add $GOBIN to PATH in $SHELL_RC?"; then
			if [ "$(basename "${SHELL:-}")" = fish ]; then
				printf '\nset -gx PATH %s $PATH\n' "$GOBIN" >>"$SHELL_RC"
			else
				printf '\nexport PATH="%s:$PATH"\n' "$GOBIN" >>"$SHELL_RC"
			fi
			info "updated $SHELL_RC — restart your shell (or 'source $SHELL_RC') to pick it up."
		else
			info "add this to your shell profile manually:"
			info "    export PATH=\"$GOBIN:\$PATH\""
		fi
		;;
esac

# ----------------------------------------------------------------- config --

step "Setting up config"

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/wtm"
CONFIG_FILE="$CONFIG_DIR/config.yaml"
DEFAULT_WORKTREE_ROOT="$HOME/wt"

if [ -f "$CONFIG_FILE" ]; then
	info "config already exists at $CONFIG_FILE — leaving it untouched."
else
	WORKTREE_ROOT="$DEFAULT_WORKTREE_ROOT"
	if [ "$ASSUME_YES" != 1 ] && [ -t 0 ]; then
		read -r -p "  Worktree root directory [$DEFAULT_WORKTREE_ROOT]: " reply || true
		[ -n "$reply" ] && WORKTREE_ROOT="$reply"
	fi
	mkdir -p "$CONFIG_DIR"
	mkdir -p "$WORKTREE_ROOT"
	printf 'worktree_root: %s\nrepos: []\n' "$WORKTREE_ROOT" >"$CONFIG_FILE"
	info "wrote $CONFIG_FILE (worktree_root: $WORKTREE_ROOT)"
fi

# ------------------------------------------------------------------- done --

step "All set"
info "run 'wtm' from inside any git repo to register it and open the fleet view."
info "see USAGE.md for keybindings and a full walkthrough."
