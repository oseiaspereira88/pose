#!/usr/bin/env bash
# Exercise the documented migration paths against fixture repositories
# (spec pose-sdd-migration-acquisition, R4).
#
# The guides in docs-site/docs/migrate/ make specific claims about what
# transfers and what does not. Those claims are the part users act on, and a
# guide that quietly drifts from the importer is worse than no guide — it
# produces confident wrong expectations. This script asserts the claims rather
# than merely running the commands.

set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT="$PWD"
FIXTURES="$ROOT/tests/import/fixtures"

BIN="$(mktemp -d)/pose"
go -C pose-mcp build -o "$BIN" ./cmd/pose

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }
ok()   { printf '\033[32m  ok\033[0m %s\n' "$1"; }

# A throwaway POSE instance so imports write somewhere real without touching
# this repository's own .pose/ tree.
new_instance() {
  local dir; dir="$(mktemp -d)"
  git -C "$dir" init -q
  "$BIN" install "$dir" >/dev/null 2>&1 || fail "pose install failed in $dir"
  printf '%s' "$dir"
}

echo "==> spec-kit: dry run writes nothing and reports provenance"
out="$("$BIN" import spec-kit "$FIXTURES/spec-kit/.specify/specs" --dry-run 2>&1)" \
  || fail "spec-kit dry run exited non-zero"
# The guide shows this exact shape, so assert the shape rather than a vague
# "it ran". requirements=2 is the FR-* extraction the guide tells readers to
# sanity-check before writing anything.
grep -q "requirements=2" <<<"$out" || fail "dry run did not extract both FR-* requirements"
grep -q "written=0" <<<"$out" || fail "dry run reported writes"
grep -q "dry_run=true" <<<"$out" || fail "dry run did not identify itself as one"
# The guide promises unmapped sections surface as curation notes rather than
# vanishing. "Out Of Scope" is in the fixture precisely to prove that.
grep -qi "out of scope" <<<"$out" || fail "unmapped section was dropped instead of reported"
ok "dry run reports requirement count, no writes and unmapped sections"

echo "==> spec-kit: import produces a draft spec with imported plan and tasks"
inst="$(new_instance)"
( cd "$inst" && "$BIN" import spec-kit "$FIXTURES/spec-kit/.specify/specs" >/dev/null ) \
  || fail "spec-kit import failed"
spec="$inst/.pose/specs/customer-export/spec.md"
[ -f "$spec" ] || fail "imported spec not written to $spec"
grep -q "^status: draft" "$spec" || fail "imported spec is not draft — the guide says status never carries over"
grep -q "Imported Implementation Plan" "$spec" || fail "plan.md did not reach the Technical Plan"
grep -q "Implement the CSV writer" "$spec" || fail "tasks.md did not reach Tasks"
grep -q "^depends_on:$" "$spec" || fail "depends_on should be empty — the guide says dependencies do not carry over"
grep -q "Import Provenance" "$spec" || fail "written spec lacks the Import Provenance section the guide describes"
grep -q "R1:" "$spec" || fail "FR-* requirements did not become R-IDs in the written spec"
grep -q "R2:" "$spec" || fail "second FR-* requirement missing from the written spec"
ok "imported spec matches what the guide promises"

echo "==> spec-kit: re-import refuses rather than clobbering curated work"
if ( cd "$inst" && "$BIN" import spec-kit "$FIXTURES/spec-kit/.specify/specs" >/dev/null 2>&1 ); then
  fail "re-import overwrote an existing spec; the guide promises it stops"
fi
ok "re-import refused"

echo "==> openspec: change folder import picks up proposal, design and tasks"
inst2="$(new_instance)"
( cd "$inst2" && "$BIN" import openspec "$FIXTURES/openspec/openspec/changes/add-2fa" >/dev/null ) \
  || fail "openspec import failed"
spec2="$(find "$inst2/.pose/specs" -name spec.md | head -1)"
[ -n "$spec2" ] || fail "no spec written by the openspec import"
grep -q "Imported Design" "$spec2" || fail "design.md did not reach the Technical Plan"
grep -q "Implement TOTP enrollment" "$spec2" || fail "tasks.md did not reach Tasks"
grep -q "R1:" "$spec2" || fail "### Requirement: sections did not become R-IDs"
grep -q "R2:" "$spec2" || fail "second requirement missing"
ok "openspec change import matches the guide"

echo "==> openspec: a spec with no '### Requirement:' section is refused"
bare="$(mktemp -d)/specs/thing"
mkdir -p "$bare"
printf '# Thing Specification\n\n## Purpose\n\nProse only, no requirements.\n' > "$bare/spec.md"
if "$BIN" import openspec "$bare/spec.md" --dry-run >/dev/null 2>&1; then
  fail "openspec accepted a spec with no requirements; the guide promises it fails outright"
fi
ok "requirement-less openspec source refused"

printf '\n\033[32mAll documented migration claims hold.\033[0m\n'
