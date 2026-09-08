#!/usr/bin/env bash
# The launch proof scenario (spec pose-launch-proof-demo).
#
# Builds a disposable repository where every conventional check passes — tests,
# vet, build — and the delivery still cannot close, then closes it legitimately.
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

BIN="$(mktemp -d)/pose"
go -C pose-mcp build -o "$BIN" ./cmd/pose >/dev/null

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }

# Pacing exists only for the recording. --verify runs flat out.
beat() { [ "$VERIFY" -eq 1 ] || sleep "${1:-1}"; }
say()  { [ "$VERIFY" -eq 1 ] || printf '\n\033[1;36m%s\033[0m\n' "$1"; }
run()  { [ "$VERIFY" -eq 1 ] || printf '\033[1m$ %s\033[0m\n' "$*"; }

WORK="$(mktemp -d)"
cd "$WORK"
git init -q
git config user.email demo@example.com
git config user.name  Demo

# A real module with a real passing test. The demo's credibility depends on
# the green checks being genuinely green.
mkdir -p exporter
cat > exporter/go.mod <<'EOF'
module example.com/exporter

go 1.26
EOF
cat > exporter/exporter.go <<'EOF'
package exporter

import "strings"

// CSV renders records as a single CSV line per record.
func CSV(records [][]string) string {
	lines := make([]string, 0, len(records))
	for _, r := range records {
		lines = append(lines, strings.Join(r, ","))
	}
	return strings.Join(lines, "\n")
}
EOF
cat > exporter/exporter_test.go <<'EOF'
package exporter

import "testing"

func TestCSVJoinsRecords(t *testing.T) {
	got := CSV([][]string{{"a", "b"}, {"c", "d"}})
	if got != "a,b\nc,d" {
		t.Fatalf("got %q", got)
	}
}
EOF
git add -A && git commit -qm "feat(exporter): render records as CSV"

"$BIN" install "$WORK" >/dev/null 2>&1 || fail "pose install failed"
"$BIN" new-spec customer-export >/dev/null 2>&1 || fail "new-spec failed"
SPEC="$(find .pose/specs -name '*customer-export*' -type f | head -1)"
[ -n "$SPEC" ] || fail "spec not scaffolded"

# A finished-looking spec: intent stated, one acceptance criterion, marked done
# by whoever did the work. Exactly the state an agent leaves behind when it
# reports success.
python3 - "$SPEC" <<'PY'
import sys, io, re
from datetime import date
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
s = s.replace("### Goal\n<!-- What this feature delivers, in one sentence. -->",
              "### Goal\nExport customer records as CSV for audit.")
s = s.replace("### Business value\n<!-- Why it is worth doing now. -->",
              "### Business value\nAuditors currently request exports by hand.")
s = re.sub(r'- R1: \n', '- R1: The exporter shall write customer records as CSV.\n', s, count=1)
s = re.sub(r'^status: draft', 'status: done', s, count=1, flags=re.M)
# Today, not a literal: the scaffold stamps created_at with the current date, so
# a fixed completion date fails the lifecycle gate with
# "completed_at is earlier than created_at" on every run after that date.
s = re.sub(r'^completed_at:\s*.*$', 'completed_at: ' + date.today().isoformat(), s, count=1, flags=re.M)
io.open(p, 'w', encoding='utf-8').write(s)
PY

say "The agent implemented the feature and reported it finished."
beat 2

say "Every conventional check agrees."
run "go test ./..."
( cd exporter && go test ./... ) || fail "the demo's own tests must pass"
beat 1
run "go vet ./..."
( cd exporter && go vet ./... ) || fail "the demo's own vet must pass"
beat 1
run "go build ./..."
( cd exporter && go build ./... ) || fail "the demo's own build must pass"
beat 2

say "So does POSE, on the checks it runs."
run "pose validate --strict"
if VAL="$("$BIN" validate --strict 2>&1)"; then
  [ "$VERIFY" -eq 1 ] || printf '%s\n' "$VAL" | tail -6
else
  fail "pose validate --strict must pass in the demo scenario, or the point is lost"
fi
beat 2

say "And yet the delivery does not close."
run "pose lint-spec customer-export --strict"
if OUT="$("$BIN" lint-spec customer-export --strict 2>&1)"; then
  fail "the closeout gate passed; the demo has nothing to show"
fi
grep -q "R1 has no trace entry" <<<"$OUT" \
  || fail "the expected blocking reason changed: $(head -3 <<<"$OUT")"
[ "$VERIFY" -eq 1 ] || printf '%s\n' "$OUT" | grep -E "ERROR|Resultado" | head -3
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
run "pose lint-spec customer-export --strict"
if OUT2="$("$BIN" lint-spec customer-export --strict 2>&1)"; then
  [ "$VERIFY" -eq 1 ] || printf '%s\n' "$OUT2" | grep -E "trace|Resultado" | head -4
else
  fail "the closeout gate still refuses after the trace was declared"
fi
grep -q "spec.trace.missing=0" <<<"$OUT2" || fail "trace not reported complete"
beat 2

say '"Done" is not evidence.'
beat 2

cd "$ROOT"
rm -rf "$WORK"

if [ "$VERIFY" -eq 1 ]; then
  printf '\033[32mScenario verified: checks green, delivery blocked, then legitimately closed.\033[0m\n'
fi
