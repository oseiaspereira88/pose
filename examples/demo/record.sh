#!/usr/bin/env bash
# The launch proof scenario (spec pose-launch-proof-demo).
#
# Builds a disposable repository where every conventional check passes — tests,
# vet, build — and the delivery still cannot close, then closes it through the
# gate: trace, review, close. The reviewer is this script, standing in for a
# person or a second agent, and says so in its conclusion.
#
# That sequence is the entire argument for POSE in about fifteen seconds. An
# all-green recording would show a tool that agrees with the agent, which is
# what every reader already assumes exists.
#
# Usage:
#   bash examples/demo/record.sh --verify   # assert the scenario still behaves
#   bash examples/demo/record.sh            # run it slowly, for screen capture
#
# The recording itself is made by pointing a terminal recorder (asciinema,
# vhs, a screen capture) at the non-verify run. Keeping the scenario as a
# script rather than a video file means it can be re-recorded when output
# changes instead of quietly becoming a stale artifact.

set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT="$PWD"

VERIFY=0
for arg in "$@"; do
  case "$arg" in
    --verify) VERIFY=1 ;;
    *) echo "usage: bash examples/demo/record.sh [--verify]" >&2; exit 2 ;;
  esac
done

export POSE_LOCALE=en
BUILD_WORK="$(mktemp -d)"
trap 'rm -rf "$BUILD_WORK"' EXIT
WORK="$BUILD_WORK/demo"
mkdir "$WORK"
BIN="$BUILD_WORK/pose"
go -C pose-mcp build -o "$BIN" ./cmd/pose >/dev/null

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }

# Pacing exists only for the recording. --verify runs flat out.
beat() { [ "$VERIFY" -eq 1 ] || sleep "${1:-1}"; }
say()  { [ "$VERIFY" -eq 1 ] || printf '\n\033[1;36m%s\033[0m\n' "$1"; }
run()  { [ "$VERIFY" -eq 1 ] || printf '\033[1m$ %s\033[0m\n' "$*"; }

cd "$WORK"
git init -q
git config user.email demo@example.com
git config user.name  Demo
mkdir -p exporter
cat > exporter/go.mod <<'EOF'
module example.com/exporter

go 1.26
EOF
git add -A && git commit -qm "initial"
commit() { git add -A && git commit -qm "$1" -m "POSE-Spec: ${2:-customer-export}"; }

"$BIN" install "$WORK" >/dev/null 2>&1 || fail "pose install failed"
commit "Adopt POSE" pose-onboarding
"$BIN" new-spec customer-export >/dev/null 2>&1 || fail "new-spec failed"
SPEC="$(find .pose/specs -name '*customer-export*' -type f | head -1)"
[ -n "$SPEC" ] || fail "spec not scaffolded"

python3 - "$SPEC" <<'PY'
import sys, io, re
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
s = s.replace("### Goal\n<!-- What this feature delivers, in one sentence. -->",
              "### Goal\nExport customer records as CSV for audit.")
s = s.replace("### Business value\n<!-- Why it is worth doing now. -->",
              "### Business value\nAuditors currently request exports by hand.")
s = re.sub(r'- R1: \n', '- R1: The exporter shall write customer records as CSV.\n', s, count=1)
s = s.replace("- modified: path/to/file", "- created: exporter/exporter.go\n- created: exporter/exporter_test.go")
io.open(p, 'w', encoding='utf-8').write(s)
PY
DIGEST="$("$BIN" start spec:customer-export 2>&1 | sed -n 's/^start.digest=//p')"
"$BIN" start spec:customer-export --apply --digest "$DIGEST" >/dev/null 2>&1 || fail "start failed"
commit "Start customer-export"

cat > exporter/exporter.go <<'EOF'
package exporter

import "strings"

// CSV joins customer records into one CSV line.
func CSV(records []string) string { return strings.Join(records, ",") }
EOF
cat > exporter/exporter_test.go <<'EOF'
package exporter

import "testing"

func TestCSVJoinsRecords(t *testing.T) {
	if got := CSV([]string{"a", "b"}); got != "a,b" {
		t.Fatalf("got %q", got)
	}
}
EOF
commit "Export customers as CSV"

say "The agent implemented the feature and reported it finished."
beat 2

say "Every conventional check agrees."
run "go test ./..."
( cd exporter && go test ./... ) || fail "the demo's own tests must pass"
beat 1
run "go vet ./..."
( cd exporter && go vet ./... ) || fail "the demo's own vet must pass"
beat 1
run "pose validate --strict"
if VAL="$("$BIN" validate --strict 2>&1)"; then
  [ "$VERIFY" -eq 1 ] || printf '%s\n' "$VAL" | tail -4
else
  fail "pose validate --strict must pass in the demo scenario, or the point is lost: $(tail -3 <<<"$VAL")"
fi
beat 2

say "And yet the delivery does not close."
run "pose close spec:customer-export --apply"
if OUT="$("$BIN" close spec:customer-export --apply --reviewer agent:reviewer 2>&1)"; then
  fail "close went ahead with R1 untraced; the demo has nothing to show"
fi
grep -q "R1 has no trace entry" <<<"$OUT" \
  || fail "the expected blocking reason changed: $(head -3 <<<"$OUT")"
[ "$VERIFY" -eq 1 ] || printf '%s\n' "$OUT" | grep -E "R1 has no trace entry" | head -1 | cut -c1-110
beat 3

say "The code is not in question. The promise is unconnected to evidence."
beat 3

say "Connect it — or waive it, or withdraw it. Silence is not an option."
python3 - "$SPEC" <<'PY'
import sys, io
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
m = "### Requirement trace"
i = s.index(m)
j = s.index("\n### ", i + len(m))
s = s[:i] + m + "\n- R1 [satisfied] test:TestCSVJoinsRecords\n" + s[j:]
io.open(p, 'w', encoding='utf-8').write(s)
PY
commit "Trace R1 to its test"
beat 2

say "Now it stops for a review, which the agent cannot give itself."
run "pose close spec:customer-export --apply"
code=0
OUT="$("$BIN" close spec:customer-export --apply --reviewer agent:reviewer 2>&1)" || code=$?
[ "$code" = 3 ] || fail "close did not stop for the reviewer (exit $code): $(tail -3 <<<"$OUT")"
ATTEST="$(sed -n 's/^closeout_plan.attest=//p' <<<"$OUT")"
[ -n "$ATTEST" ] || fail "close no longer prints the filled attest command"
beat 2

say "A reviewer reads the evidence and concludes."
run "pose review attest ..."
ATTEST="${ATTEST//<id>/agent:reviewer}"
ATTEST="${ATTEST//<your conclusion>/demo reviewer: checked R1 against the CSV test}"
ATTEST="${ATTEST/#pose /\"$BIN\" }"
eval "$ATTEST" >/dev/null 2>&1 || fail "the printed attest command was refused: $ATTEST"
beat 2

run "pose close spec:customer-export --resume"
OUT="$("$BIN" close spec:customer-export --resume 2>&1)" || fail "close --resume failed: $(tail -3 <<<"$OUT")"
grep -q "closeout_plan.result=closed spec:customer-export" <<<"$OUT" || fail "close did not report the spec closed"
[ "$VERIFY" -eq 1 ] || printf '%s\n' "$OUT" | grep -E "closeout_plan.result" | cut -c1-70
grep -q "^status: done" "$SPEC" || fail "the spec is not done after the close"
VERIFIED="$("$BIN" review verify spec:customer-export 2>&1)" || fail "review verify refused the closed spec"
grep -q "approved=true" <<<"$VERIFIED" || fail "the closed spec does not verify as approved"
beat 2

say '"Done" is not evidence. This is.'
beat 2

cd "$ROOT"
rm -rf "$WORK"

if [ "$VERIFY" -eq 1 ]; then
  printf '\033[32mScenario verified: checks green, delivery blocked, then reviewed and closed through the gate.\033[0m\n'
fi
