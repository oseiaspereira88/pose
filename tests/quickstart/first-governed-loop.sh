#!/usr/bin/env bash
# Execute the documented quickstart and assert the outcomes the page shows
# (specs pose-first-governed-loop-quickstart R5, pose-quickstart-real-lifecycle
# R4).
#
# The page's value is that a reader can tell whether their run diverged, which
# only holds while the shown output is the real one. A documentation test that
# merely checks exit codes would stay green while the prose drifted, so this
# asserts the specific gate states the page promises — including the ones the
# page depends on *refusing*. Every chapter runs the commands a project runs:
# no status is edited by hand.

set -euo pipefail

if [ "${1:-}" = "--binary" ]; then
  BIN="${2:?--binary requires the native binary path}"
  ROOT="$PWD"
else
  cd "$(dirname "$0")/../.."
  ROOT="$PWD"
  BIN="$(mktemp -d)/pose"
  go -C pose-mcp build -o "$BIN" ./cmd/pose
fi

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }
ok()   { printf '\033[32m  ok\033[0m %s\n' "$1"; }
command -v ssh-keygen >/dev/null || fail "ssh-keygen is required: chapter 4 answers a decision with a signature"

WORK="$(mktemp -d)"
KEYS="$(mktemp -d)"
cd "$WORK"
git init -q
git config user.email quickstart@example.com
git config user.name  Quickstart
mkdir -p svc
printf 'module example.com/svc\n\ngo 1.26\n' > svc/go.mod
git add -A && git commit -qm "initial"
commit() { git add -A && git commit -qm "$1" -m "POSE-Spec: ${2:-customer-export}"; }

# Chapter 1 — install, then set up.
out="$("$BIN" install "$WORK" 2>&1)" || fail "pose install failed"
grep -q "next: \`pose setup\`" <<<"$out" || fail "install no longer ends with pose setup as the next step"
out="$("$BIN" setup --no-input 2>&1)"
grep -q "setup.identity.maintainer=todo" <<<"$out" || fail "setup does not ask for a maintainer on a fresh install"
ssh-keygen -q -t ed25519 -N "" -C quickstart -f "$KEYS/id_ed25519"
out="$("$BIN" identity add --key "$KEYS/id_ed25519.pub" --role maintainer --apply 2>&1)" || fail "identity add failed: $out"
grep -q "identity.suggested=human:quickstart" <<<"$out" || fail "identity add no longer suggests the git name"
"$BIN" hooks install >/dev/null 2>&1 || fail "hooks install failed"
commit "Adopt POSE" pose-onboarding
out="$("$BIN" doctor 2>&1)" || fail "doctor failed after setup"
grep -q "0 error(s), 0 warning(s)" <<<"$out" || fail "a set-up instance is not clean in doctor: $out"
ok "chapter 1: installed, maintainer registered with a key, commit gate on, doctor clean"

# Chapter 2 — the entry gate.
out="$("$BIN" new-spec customer-export 2>&1)" || fail "new-spec failed"
grep -q "status: draft" <<<"$out" || fail "new-spec no longer reports 'status: draft' as the page shows"
SPEC="$(find .pose/specs -name '*customer-export*' -type f | head -1)"
[ -n "$SPEC" ] || fail "scaffolded spec not found"
if out="$("$BIN" lint-spec customer-export --ready-check 2>&1)"; then :; fi
grep -q "spec.ready=false" <<<"$out" \
  || fail "entry gate did not refuse an empty scaffold; the page's step is wrong"
grep -qi "Intent is missing, empty, or skeletal" <<<"$out" \
  || fail "entry gate reason changed; the page quotes it verbatim"
python3 - "$SPEC" <<'PY'
import sys, io, re
p = sys.argv[1]
s = io.open(p, encoding='utf-8').read()
s = s.replace("### Goal\n<!-- What this feature delivers, in one sentence. -->",
              "### Goal\nExport customer records as CSV for audit.")
s = s.replace("### Business value\n<!-- Why it is worth doing now. -->",
              "### Business value\nAuditors currently request exports by hand.")
s = re.sub(r'- R1: \n', '- R1: The exporter shall write customer records as CSV.\n', s, count=1)
s = s.replace("- modified: path/to/file", "- created: svc/export.go\n- created: svc/export_test.go")
io.open(p, 'w', encoding='utf-8').write(s)
PY
out="$("$BIN" lint-spec customer-export --ready-check 2>&1)" \
  || fail "entry gate still refuses after Intent and R1 were filled"
grep -q "spec.ready=true" <<<"$out" || fail "entry gate did not report spec.ready=true"
ok "chapter 2: the entry gate refuses the empty scaffold, then passes once Intent and R1 exist"

# Chapter 3 — start, implement, prove.
out="$("$BIN" start spec:customer-export 2>&1)"
grep -q "start.ready=true" <<<"$out" || fail "start does not report the spec ready"
DIGEST="$(sed -n 's/^start.digest=//p' <<<"$out")"
"$BIN" start spec:customer-export --apply --digest "$DIGEST" >/dev/null 2>&1 || fail "start --apply failed"
grep -q "^status: in-progress" "$SPEC" || fail "start did not move the spec to in-progress"
commit "Start customer-export"
cat > svc/export.go <<'GO'
package svc

import "strings"

// CSV joins customer records into one CSV line.
func CSV(records []string) string { return strings.Join(records, ",") }
GO
cat > svc/export_test.go <<'GO'
package svc

import "testing"

func TestCustomerExportWritesCSV(t *testing.T) {
	if got := CSV([]string{"a", "b"}); got != "a,b" {
		t.Fatalf("got %q", got)
	}
}
GO
"$BIN" validate --strict >/dev/null 2>&1 || fail "repository checks failed"
commit "Export customers as CSV"
ok "chapter 3: started with a recorded baseline, implemented, checks pass"

# Chapter 4 — a decision the agent cannot take alone.
out="$("$BIN" action open --origin spec:customer-export --kind decision \
  --question "Include inactive customers in the audit export?" \
  --option "include=Auditors see every customer" --option "exclude=Inactive customers are left out" \
  --recommend include --recipient-role maintainer --requested-by agent:impl \
  --target requirement:R1 --effect closeout:block --apply --json 2>&1)" || fail "action open failed: $out"
ACT="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["request"]["id"])' <<<"$out")"
DIG="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["request"]["request_digest"])' <<<"$out")"
out="$("$BIN" state --attention --actor maintainer 2>&1)"
grep -q "$ACT" <<<"$out" || fail "Attention does not show the decision to the maintainer"
if out="$("$BIN" close spec:customer-export 2>&1)"; then fail "close was not refused while the decision is open"; fi
grep -q "action request(s) restrict closeout" <<<"$out" || fail "close refusal no longer names the open decision"
out="$("$BIN" action resolve "$ACT" --actor human:quickstart --answer include --request-digest "$DIG" \
  --expected-revision 1 --idempotency-key answer-1 --sign "$KEYS/id_ed25519" --apply 2>&1)" || fail "signed answer refused: $out"
grep -q "include by human:quickstart (verified)" <<<"$out" || fail "a signed answer is not recorded as verified"
out="$("$BIN" action show "$ACT" 2>&1)"
grep -q "re-verified from the journal" <<<"$out" || fail "action show does not re-verify the signature"
commit "Decide: include inactive customers"
ok "chapter 4: the decision blocks closeout until the maintainer answers, signed and re-verifiable"

# Chapter 5 — the exit gate, then review and close.
if out="$("$BIN" close spec:customer-export --apply --reviewer agent:reviewer 2>&1)"; then
  fail "close went ahead with R1 untraced"
fi
grep -q "R1 has no trace entry" <<<"$out" \
  || fail "the exit gate did not refuse an untraced requirement; the page's step is wrong"
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
commit "Trace R1 to its test"
code=0
out="$("$BIN" close spec:customer-export --apply --reviewer agent:reviewer 2>&1)" || code=$?
[ "$code" = 3 ] || fail "close did not stop for the reviewer (exit $code): $out"
ATTEST="$(sed -n 's/^closeout_plan.attest=//p' <<<"$out")"
[ -n "$ATTEST" ] || fail "close no longer prints the filled attest command"
ATTEST="${ATTEST//<id>/agent:reviewer}"
ATTEST="${ATTEST//<your conclusion>/reviewed against R1 and the CSV test}"
ATTEST="${ATTEST/#pose /\"$BIN\" }"
eval "$ATTEST" >/dev/null 2>&1 || fail "the printed attest command was refused: $ATTEST"
out="$("$BIN" close spec:customer-export --resume 2>&1)" || fail "close --resume failed: $out"
grep -q "closeout_plan.result=closed spec:customer-export" <<<"$out" || fail "close did not report the spec closed"
grep -q "^status: done" "$SPEC" || fail "the spec is not done"
"$BIN" lint-spec customer-export --strict >/dev/null 2>&1 || fail "a spec closed by pose close fails lint"
ok "chapter 5: the exit gate refuses an untraced R1, the review uses the printed command, close lands"

cd "$ROOT"
rm -rf "$WORK" "$KEYS"
printf '\n\033[32mThe documented quickstart behaves as the page describes.\033[0m\n'
