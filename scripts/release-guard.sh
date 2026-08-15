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
# Prefer the justfile wrappers: `just check-deps`, `just tag-release vX.Y.Z`.

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

# default_branch resolves the branch releases are cut from. Not every repo in
# this family uses "main" — `wav` is on "master" — so this must be discovered
# rather than assumed.
default_branch() {
	local b
	b=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null || true)
	b=${b#origin/}
	if [ -z "$b" ]; then
		for candidate in main master; do
			if git rev-parse --verify --quiet "refs/remotes/origin/${candidate}" >/dev/null; then
				b=$candidate
				break
			fi
		done
	fi
	echo "${b:-main}"
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
		if [[ "$line" == *" -> "* ]]; then
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
		return 0
	fi

	# A handful of untagged commits is normal mid-development, so a small count
	# is only a warning. Past the threshold it becomes a reportable state, so the
	# scheduled workflow can raise the same nag it raises for stale deps —
	# otherwise "work stuck on main" is detected and then silently dropped, which
	# is exactly how algo-fft reached 97 unreleased commits.
	warn "$count commit(s) since $tag — consider cutting a release"
	git log --oneline "${tag}..HEAD" | head -5 | sed 's/^/      /'
	[ "$count" -gt 5 ] && echo "      … and $((count - 5)) more"

	if [ "$count" -ge "${UNRELEASED_THRESHOLD:-20}" ]; then
		red "  ✗ $count commits is past the ${UNRELEASED_THRESHOLD:-20}-commit threshold — release or explain why not"
		return 1
	fi

	return 0
}

# semver_field VERSION INDEX -> the numeric field (1=major, 2=minor, 3=patch)
semver_field() {
	echo "${1#v}" | cut -d. -f"$2" | cut -d- -f1
}

cmd_gate() {
	local version="$1"

	echo "Release gate for ${version}"
	echo

	# --- shape of the version string -------------------------------------
	# `return 1` explicitly, never a bare `return`: fail() ends in an assignment,
	# which succeeds, so a bare return here would report the gate as PASSED and
	# cmd_tag would go on to tag and push an invalid version.
	if ! [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		fail "'$version' is not a vX.Y.Z tag"
		return 1
	fi

	# --- refresh tags before anything is compared against them -------------
	# The base must be read AFTER fetching, or a release tagged elsewhere since
	# the last fetch is invisible and both the monotonic check and the API
	# comparison silently run against a stale predecessor.
	if ! git fetch --quiet origin --tags; then
		fail "could not fetch tags from origin — cannot verify ${version} against the real latest release"
		return 1
	fi

	local base
	base=$(latest_tag)

	# --- working tree and branch -----------------------------------------
	local dirty=0
	if [ -n "$(git status --porcelain)" ]; then
		fail "working tree is dirty"
		dirty=1
	else
		ok "working tree clean"
	fi

	local branch expected
	branch=$(git rev-parse --abbrev-ref HEAD)
	expected=$(default_branch)
	if [ "$branch" != "$expected" ]; then
		fail "on branch '$branch', expected '$expected'"
	else
		ok "on $branch"
	fi

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
		# Accept "## [0.8.0]", "## v0.8.0", "## [v0.8.0]" — but not "Unreleased",
		# and not a near-miss like "## 0.8.0-rc1" or "## 0.8.0.1". The heading must
		# END at the version, so the trailing group may only start with something
		# that cannot continue a version string. Dots are escaped because an
		# unescaped "." would let 0.8.0 match a heading for 0x8y0.
		local ver_re
		ver_re=$(printf '%s' "${version#v}" | sed 's/[.]/[.]/g')
		if grep -qiE "^##+ *\[?v?${ver_re}\]?([^0-9.-].*)?$" CHANGELOG.md; then
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
				# A MAJOR bump counts too — graduating v0.x -> v1.0.0 signals the
				# incompatibility at least as loudly, and comparing minors alone
				# would reject it (1.0.0 has minor 0, which is not > 8).
				if [ "$vmaj" -gt "$bmaj" ] || [ "$vmin" -gt "$bmin" ]; then
					required_ok=1
				fi
				if [ $required_ok -eq 0 ]; then
					fail "incompatible changes require a minor or major bump for v0.x (v0.$((bmin + 1)).0 or v$((bmaj + 1)).0.0 or later), got ${version}"
				else
					ok "version bump signals the incompatible changes"
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

	# Check the gate explicitly rather than leaning on `set -e`. The dispatcher
	# invokes cmd_tag inside a `||` list to capture its status, and that context
	# disables errexit for everything it calls — so an unchecked `cmd_gate` here
	# would let a failed gate fall straight through into `git tag` and `git push`.
	if ! cmd_gate "$version"; then
		return 1
	fi
	if [ "$FAILED" -ne 0 ]; then
		red "refusing to tag ${version}"
		return 1
	fi

	echo
	echo "Tagging ${version}…"
	git tag -a "$version" -m "${version}

$(if [ -f CHANGELOG.md ]; then
		# Same heading-boundary rule as the gate, for the same reason: a
		# "## 0.8.0-rc1" section preceding the real one would otherwise be
		# picked up and embedded as this tag's release notes.
		awk -v v="$(printf '%s' "${version#v}" | sed 's/[.]/[.]/g')" '
      $0 ~ "^##+ *\\[?v?" v "\\]?([^0-9.-].*)?$" {f=1; next}
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

# STATUS carries a subcommand's own return value; FAILED carries any fail() call.
# Both must reach the exit code — `exit "$FAILED"` alone would discard the
# threshold result from cmd_unreleased, which reports through its return value.
STATUS=0

case "${1:-}" in
deps) cmd_deps || STATUS=$? ;;
unreleased) cmd_unreleased || STATUS=$? ;;
gate)
	[ $# -ge 2 ] || {
		red "usage: $0 gate vX.Y.Z"
		exit 2
	}
	cmd_gate "$2" || STATUS=$?
	;;
tag)
	[ $# -ge 2 ] || {
		red "usage: $0 tag vX.Y.Z"
		exit 2
	}
	cmd_tag "$2" || STATUS=$?
	;;
*)
	echo "usage: $0 {deps|unreleased|gate vX.Y.Z|tag vX.Y.Z}" >&2
	exit 2
	;;
esac

if [ "$FAILED" -ne 0 ] && [ "$STATUS" -eq 0 ]; then
	STATUS=1
fi

exit "$STATUS"
