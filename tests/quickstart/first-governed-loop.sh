#!/usr/bin/env bash
# Execute the documented first governed loop and assert the outcomes the
# quickstart page shows (spec pose-first-governed-loop-quickstart, R5).
#
# The page's value is that a reader can tell whether their run diverged, which
# only holds while the shown output is the real one. A documentation test that
# merely checks exit codes would stay green while the prose drifted, so this
# asserts the specific gate states the page promises — including the two the
# page depends on *failing*.

set -euo pipefail

cd "$(dirname "$0")/../.."
ROOT="$PWD"

BIN="$(mktemp -d)/pose"
go -C pose-mcp build -o "$BIN" ./cmd/pose

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }
ok()   { printf '\033[32m  ok\033[0m %s\n' "$1"; }

WORK="$(mktemp -d)"
cd "$WORK"
git init -q
git config user.email quickstart@example.com
git config user.name  Quickstart
mkdir -p svc
printf 'module example.com/svc\n\ngo 1.26\n' > svc/go.mod
git add -A && git commit -qm "initial"

"$BIN" install "$WORK" >/dev/null 2>&1 || fail "pose install failed"

# Step 1 — scaffold
out="$("$BIN" new-spec customer-export 2>&1)" || fail "new-spec failed"
grep -q "status: draft" <<<"$out" || fail "new-spec no longer reports 'status: draft' as the page shows"
SPEC="$(find .pose/specs -name '*customer-export*' -type f | head -1)"
[ -n "$SPEC" ] || fail "scaffolded spec not found"
ok "step 1: spec scaffolded as draft"

# Step 2 — the entry gate must REFUSE the empty scaffold.
if out="$("$BIN" lint-spec customer-export --ready-check 2>&1)"; then :; fi
grep -q "spec.ready=false" <<<"$out" \
  || fail "entry gate did not refuse an empty scaffold; the page's step 2 is wrong"
grep -qi "Intent is missing, empty, or skeletal" <<<"$out" \
  || fail "entry gate reason changed; the page quotes it verbatim"
ok "step 2: entry gate refuses the empty scaffold, with the documented reason"

# Step 3 — fill Intent and one acceptance criterion, exactly as the page says.
python3 - "$SPEC" <<'PY'
import sys, io, re
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
s = s.replace("### Goal\n<!-- What this feature delivers, in one sentence. -->",
              "### Goal\nExport customer records as CSV for audit.")
s = s.replace("### Business value\n<!-- Why it is worth doing now. -->",
              "### Business value\nAuditors currently request exports by hand.")
s = re.sub(r'- R1: \n', '- R1: The exporter shall write customer records as CSV.\n', s, count=1)
io.open(p, 'w', encoding='utf-8').write(s)
PY
out="$("$BIN" lint-spec customer-export --ready-check 2>&1)" \
  || fail "entry gate still refuses after Intent and R1 were filled"
grep -q "spec.ready=true" <<<"$out" || fail "entry gate did not report spec.ready=true"
ok "step 3: entry gate passes once Intent and R1 exist"

# Step 4 — the trail resolves.
"$BIN" suggest feature >/dev/null 2>&1 || fail "pose suggest feature failed"
ok "step 4: suggest resolves the applicable trail"

# Step 6 — declaring done must be REFUSED while R1 has no trace entry. This is
# the beat the whole page is built around.
python3 - "$SPEC" <<'PY'
import sys, io, re
from datetime import date
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
s = re.sub(r'^status: draft', 'status: done', s, count=1, flags=re.M)
# Today, not a literal: the scaffold stamps created_at with the current date, so
# a fixed completion date fails the lifecycle gate with
# "completed_at is earlier than created_at" on every run after that date.
s = re.sub(r'^completed_at:\s*.*$', 'completed_at: ' + date.today().isoformat(), s, count=1, flags=re.M)
io.open(p, 'w', encoding='utf-8').write(s)
PY
if out="$("$BIN" lint-spec customer-export --strict 2>&1)"; then :; fi
grep -q "R1 has no trace entry" <<<"$out" \
  || fail "closeout gate did not refuse a done spec with an untraced requirement; the page's step 6 is wrong"
ok "step 6: closeout gate refuses 'done' while R1 has no evidence"

# Step 7 — connecting the promise to the proof closes it.
python3 - "$SPEC" <<'PY'
import sys, io
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
marker = "### Requirement trace"
i = s.index(marker)
j = s.index("\n### ", i + len(marker))
s = s[:i] + marker + "\n- R1 [satisfied] test:TestCustomerExportWritesCSV\n" + s[j:]
io.open(p, 'w', encoding='utf-8').write(s)
PY
out="$("$BIN" lint-spec customer-export --strict 2>&1)" \
  || fail "closeout gate still refuses after the requirement trace was declared"
grep -q "spec.trace.missing=0" <<<"$out" || fail "trace not reported as complete"
ok "step 7: closeout passes once R1 points at evidence"

cd "$ROOT"
rm -rf "$WORK"
printf '\n\033[32mThe documented first governed loop behaves as the page describes.\033[0m\n'
