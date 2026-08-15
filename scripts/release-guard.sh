#!/usr/bin/env bash
#
# release-guard.sh — keeps this module releasable and keeps its siblings current.
#
# This exists because of a concrete failure. In August 2026 the algo-* family
# had drifted onto three different algo-fft versions at once: algo-pde pinned
# v0.6.15, algo-dsp pinned v0.7.3, algo-acoustics pinned v0.6.11, and algo-fft's
# own main sat 97 commits past its latest tag with a CHANGELOG section for a
# version nobody had ever tagged. Because algo-fft's generic PlanReal2D and
# PlanReal3D had changed signature between the v0.6 and v0.7 lines, no single
# upgrade anywhere could compile, and untangling it took a full day.
#
# Three distinct mistakes produced that, and this script blocks each one:
#
#   1. Work accumulated on main and was never tagged, so downstream could not
#      take fixes even when they existed.        -> `unreleased`
#   2. Consumers sat on old sibling versions indefinitely, and nothing said so.
#      -> `deps`
#   3. Exported API was removed without the version signalling it. Note that
#      semver EXEMPTS v0.x from this ("anything MAY change at any time"), so
#      `gorelease` alone will happily approve a patch bump across a breaking
#      change. The whole algo-* family is v0.x, so that exemption is exactly
#      the hole we fell through. This script refuses it anyway.  -> `gate`
#
# Usage:
#   scripts/release-guard.sh deps            # sibling deps at their latest tags?
#   scripts/release-guard.sh unreleased      # untagged work sitting on main?
#   scripts/release-guard.sh gate vX.Y.Z     # all release preconditions
#   scripts/release-guard.sh tag  vX.Y.Z     # gate, then create and push the tag
#
# Prefer the justfile wrappers: `just check-deps`, `just release vX.Y.Z`.

set -euo pipefail

export GOPRIVATE="${GOPRIVATE:-github.com/cwbudde}"

SIBLING_PREFIX="github.com/cwbudde/"

red() { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
yellow() { printf '\033[33m%s\033[0m\n' "$*"; }
fail() {
	red "  ✗ $*"
	FAILED=1
}
ok() { green "  ✓ $*"; }
warn() { yellow "  ! $*"; }

FAILED=0

# latest_tag is the version the release is measured against. RELEASE_GUARD_BASE
# overrides it — needed when re-cutting a release, and used by this script's own
# self-test to replay the KernelEightStep removal that motivated the API rule.
latest_tag() {
	if [ -n "${RELEASE_GUARD_BASE:-}" ]; then
		echo "$RELEASE_GUARD_BASE"
		return
	fi
	git tag --list 'v*' --sort=-v:refname | head -1
}

# sibling_modules lists every github.com/cwbudde/* module this one requires,
# direct or indirect. Indirect ones matter too: a stale indirect pin is how an
# incompatible algo-fft reached algo-acoustics through two different paths.
sibling_modules() {
	go list -m -f '{{if not .Main}}{{.Path}}{{end}}' all 2>/dev/null |
		grep "^${SIBLING_PREFIX}" || true
}

cmd_deps() {
	echo "Checking sibling dependencies are at their latest tags…"
	local mods
	mods=$(sibling_modules | tr '\n' ' ')

	if [ -z "${mods// /}" ]; then
		ok "no ${SIBLING_PREFIX}* dependencies"
		return
	fi

	# `go list -m -u` resolves each module's latest tag from the proxy/origin.
	# Anything with an .Update field is behind.
	local out
	# shellcheck disable=SC2086
	out=$(go list -m -u -f '{{.Path}} {{.Version}}{{if .Update}} -> {{.Update.Version}}{{end}}' $mods 2>/dev/null)

	while IFS= read -r line; do
		[ -z "$line" ] && continue
		if [[ $line == *" -> "* ]]; then
			fail "$line"
		else
			ok "$line"
		fi
	done <<<"$out"

	if [ "$FAILED" -ne 0 ]; then
		echo
		red "Sibling dependencies are behind their latest tags."
		echo "Bump them with:  go get <module>@<version> && go mod tidy"
		echo "If a bump is deliberately deferred, say so in PLAN.md and re-run."
	fi
}

cmd_unreleased() {
	echo "Checking for untagged work on this branch…"
	local tag count
	tag=$(latest_tag)

	if [ -z "$tag" ]; then
		warn "no version tags yet — nothing released from this module"
		return
	fi

	count=$(git rev-list "${tag}..HEAD" --count)
	if [ "$count" -eq 0 ]; then
		ok "HEAD is $tag"
	else
		# Deliberately a warning, not a failure. Untagged work is normal
		# mid-development; it only becomes a problem when it is forgotten.
		# The scheduled CI job is what turns a large number into a nag.
		warn "$count commit(s) since $tag — consider cutting a release"
		git log --oneline "${tag}..HEAD" | head -5 | sed 's/^/      /'
		[ "$count" -gt 5 ] && echo "      … and $((count - 5)) more"
	fi
}

# semver_field VERSION INDEX -> the numeric field (1=major, 2=minor, 3=patch)
semver_field() {
	echo "${1#v}" | cut -d. -f"$2" | cut -d- -f1
}

cmd_gate() {
	local version="$1"
	local base
	base=$(latest_tag)

	echo "Release gate for ${version}"
	echo

	# --- shape of the version string -------------------------------------
	if ! [[ $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		fail "'$version' is not a vX.Y.Z tag"
		return
	fi

	# --- working tree and branch -----------------------------------------
	local dirty=0
	if [ -n "$(git status --porcelain)" ]; then
		fail "working tree is dirty"
		dirty=1
	else
		ok "working tree clean"
	fi

	local branch
	branch=$(git rev-parse --abbrev-ref HEAD)
	if [ "$branch" != "main" ]; then
		fail "on branch '$branch', expected main"
	else
		ok "on main"
	fi

	git fetch --quiet origin --tags 2>/dev/null || true
	if git rev-parse --verify --quiet "origin/${branch}" >/dev/null; then
		if [ "$(git rev-parse HEAD)" != "$(git rev-parse "origin/${branch}")" ]; then
			fail "HEAD differs from origin/${branch} — push or pull first"
		else
			ok "in sync with origin/${branch}"
		fi
	fi

	# --- tag does not already exist --------------------------------------
	if git rev-parse --verify --quiet "refs/tags/${version}" >/dev/null ||
		git ls-remote --exit-code --tags origin "refs/tags/${version}" >/dev/null 2>&1; then
		fail "tag ${version} already exists"
	else
		ok "tag ${version} is free"
	fi

	# --- monotonic ---------------------------------------------------------
	if [ -n "$base" ]; then
		if [ "$(printf '%s\n%s\n' "$base" "$version" | sort -V | tail -1)" != "$version" ]; then
			fail "${version} does not sort after ${base}"
		else
			ok "${version} follows ${base}"
		fi
	fi

	# --- siblings current --------------------------------------------------
	echo
	cmd_deps
	echo

	# --- CHANGELOG ---------------------------------------------------------
	if [ -f CHANGELOG.md ]; then
		# Accept "## [0.8.0]", "## v0.8.0", "## [v0.8.0]" — but not "Unreleased".
		if grep -qiE "^##+ *\[?v?${version#v}\]?" CHANGELOG.md; then
			ok "CHANGELOG.md has a section for ${version}"
		else
			fail "CHANGELOG.md has no section for ${version}"
			echo "      A tag whose changelog section does not exist is how the"
			echo "      phantom algo-fft v0.7.5 happened. Promote [Unreleased]."
		fi
	fi

	# --- API compatibility -------------------------------------------------
	if [ -z "$base" ]; then
		warn "no previous tag — skipping API comparison"
	elif [ "$dirty" -eq 1 ]; then
		# gorelease refuses to run against a dirty repo. Skipping loudly matters:
		# silently treating "could not compare" as "nothing changed" would defeat
		# the entire point of this check.
		fail "cannot compare API against ${base} while the tree is dirty — commit first, then re-run"
	else
		echo "Comparing exported API against ${base} (this downloads ${base})…"
		local report status
		set +e
		report=$(go run golang.org/x/exp/cmd/gorelease@latest -base="$base" -version="$version" 2>&1)
		status=$?
		set -e

		# gorelease exits non-zero both when the version is wrong AND when it
		# could not run at all. Only the former has a summary section; treat the
		# latter as a hard tooling failure rather than an API verdict.
		if ! echo "$report" | grep -q '^# summary'; then
			fail "gorelease could not run — API compatibility is UNVERIFIED"
			echo "$report" | tail -5 | sed 's/^/      /'
			echo
			red "refusing to release ${version}"
			return 1
		fi

		if [ $status -ne 0 ]; then
			fail "gorelease rejected ${version}"
			echo "$report" | sed -n '/^# summary/,$p' | sed 's/^/      /'
		else
			ok "gorelease accepts ${version} as a semver-valid release"
		fi

		# The rule gorelease will NOT enforce for us. Semver says v0.x may break
		# anything at any time, so gorelease approves a patch bump across a
		# removed symbol. For this family that is precisely the hole that let
		# KernelEightStep and the PlanReal2D generics reach consumers unannounced,
		# so require the bump to be visible in the version regardless.
		if echo "$report" | grep -q '^## incompatible changes'; then
			local bmaj bmin vmaj vmin
			bmaj=$(semver_field "$base" 1)
			bmin=$(semver_field "$base" 2)
			vmaj=$(semver_field "$version" 1)
			vmin=$(semver_field "$version" 2)

			echo
			yellow "  Incompatible API changes since ${base}:"
			echo "$report" | awk '/^## incompatible changes/{f=1;next} /^#/{f=0} f' |
				grep -v '^$' | sed 's/^/      /'
			echo

			local required_ok=0
			if [ "$bmaj" -eq 0 ]; then
				# v0.x: we require a MINOR bump, stricter than semver demands.
				[ "$vmin" -gt "$bmin" ] && required_ok=1
				if [ $required_ok -eq 0 ]; then
					fail "incompatible changes require a minor bump for v0.x (v0.$((bmin + 1)).0 or later), got ${version}"
				else
					ok "minor bump signals the incompatible changes"
				fi
			else
				[ "$vmaj" -gt "$bmaj" ] && required_ok=1
				if [ $required_ok -eq 0 ]; then
					fail "incompatible changes require a major bump, got ${version}"
				else
					ok "major bump signals the incompatible changes"
				fi
			fi

			if [ $required_ok -eq 1 ] && [ -f CHANGELOG.md ]; then
				if ! grep -qiE 'breaking|incompatible|removed' CHANGELOG.md; then
					warn "CHANGELOG.md does not mention 'breaking'/'removed' anywhere — is the break documented?"
				fi
			fi
		else
			[ -n "$base" ] && ok "no incompatible API changes since ${base}"
		fi
	fi

	echo
	if [ "$FAILED" -ne 0 ]; then
		red "refusing to release ${version}"
		return 1
	fi
	green "${version} is clear to release"
}

cmd_tag() {
	local version="$1"
	cmd_gate "$version"

	echo
	echo "Tagging ${version}…"
	git tag -a "$version" -m "${version}

$(if [ -f CHANGELOG.md ]; then
		awk -v v="${version#v}" '
      $0 ~ "^##+ *\\[?v?" v "\\]?" {f=1; next}
      f && /^##+ /{exit}
      f {print}
    ' CHANGELOG.md | head -40
	fi)"
	git push origin "$version"
	green "pushed ${version}"

	echo
	echo "Renovate will open bump PRs against the consumers within the hour."
	echo "To do it by hand now, in each consuming repo:"
	echo "    go get $(go list -m)@${version} && go mod tidy"
}

case "${1:-}" in
deps) cmd_deps ;;
unreleased) cmd_unreleased ;;
gate)
	[ $# -ge 2 ] || {
		red "usage: $0 gate vX.Y.Z"
		exit 2
	}
	cmd_gate "$2"
	;;
tag)
	[ $# -ge 2 ] || {
		red "usage: $0 tag vX.Y.Z"
		exit 2
	}
	cmd_tag "$2"
	;;
*)
	echo "usage: $0 {deps|unreleased|gate vX.Y.Z|tag vX.Y.Z}" >&2
	exit 2
	;;
esac

exit "$FAILED"
