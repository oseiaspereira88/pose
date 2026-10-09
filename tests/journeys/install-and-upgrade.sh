#!/usr/bin/env bash
# The two ways a project meets this engine, walked end to end (spec
# pose-install-and-upgrade-journeys): a fresh install, and an instance
# installed by the latest published release and then updated. Each journey
# asserts the states the onboarding promises — a clean doctor, one next step,
# nothing adopted without an answer — not only exit codes.
#
# Usage: install-and-upgrade.sh [--binary <pose>] [--previous <version>]

set -euo pipefail

BIN="" PREVIOUS="${POSE_PREVIOUS_RELEASE:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --binary) BIN="${2:?}"; shift 2 ;;
    --previous) PREVIOUS="${2:?}"; shift 2 ;;
    *) echo "usage: $0 [--binary <pose>] [--previous <version>]" >&2; exit 2 ;;
  esac
done
cd "$(dirname "$0")/../.."
ROOT="$PWD"
if [ -z "$BIN" ]; then
  BIN="$(mktemp -d)/pose"
  go -C pose-mcp build -o "$BIN" ./cmd/pose
fi

fail() { printf '\033[31mFAIL:\033[0m %s\n' "$1" >&2; exit 1; }
ok()   { printf '\033[32m  ok\033[0m %s\n' "$1"; }
command -v ssh-keygen >/dev/null || fail "ssh-keygen is required: the upgrade journey answers a review request"

KEYS="$(mktemp -d)"
ssh-keygen -q -t ed25519 -N "" -C journey -f "$KEYS/id_ed25519"
export HOME="$KEYS"   # the machine's ~/.ssh and git config never leak in

new_repo() {
  local dir; dir="$(mktemp -d)"
  git -C "$dir" init -q
  git -C "$dir" config user.email ada@example.com
  git -C "$dir" config user.name Ada
  mkdir -p "$dir/svc"
  printf 'module example.com/svc\n\ngo 1.26\n' > "$dir/svc/go.mod"
  git -C "$dir" add -A && git -C "$dir" commit -qm initial
  echo "$dir"
}
commit() { git -C "$1" add -A && git -C "$1" commit -qm "$2" --no-verify; }
field() { sed -n "s/^$1=//p"; }

# Journey 1 — a fresh install.
REPO="$(new_repo)"
cd "$REPO"
out="$("$BIN" install . 2>&1)" || fail "install failed: $out"
grep -q 'next: `pose setup`' <<<"$out" || fail "install does not end with pose setup"
commit "$REPO" "Adopt POSE"
"$BIN" check --strict >/dev/null 2>&1 || fail "a fresh install fails the strict check"
out="$("$BIN" doctor 2>&1)" || fail "doctor errors on a fresh install"
grep -q "0 error(s), 0 warning(s)" <<<"$out" || fail "a fresh install is not clean in doctor: $(grep '\[!\]\|\[✗\]' <<<"$out")"
plan="$("$BIN" setup --json)"
python3 - "$plan" <<'PY' || fail "setup on a fresh install"
import json, sys
plan = json.loads(sys.argv[1])
assert plan["next"]["command"] == "pose start spec:pose-onboarding", plan["next"]
assert plan["new"] == [], plan["new"]
steps = {s["id"]: s["state"] for s in plan["steps"]}
assert steps["identity.maintainer"] == "todo", steps
PY
"$BIN" lint-spec pose-onboarding >/dev/null 2>&1 || fail "the onboarding spec does not lint"
ok "journey 1: fresh install — strict check, clean doctor, onboarding spec first, nothing to review"

# The onboarding spec is taken to done by the steps setup names.
out="$("$BIN" start spec:pose-onboarding 2>&1)"
"$BIN" start spec:pose-onboarding --apply --digest "$(field start.digest <<<"$out")" >/dev/null 2>&1 || fail "start pose-onboarding failed"
commit "$REPO" "Start onboarding" && git commit -q --amend -m "Start onboarding" -m "POSE-Spec: pose-onboarding" --no-verify
"$BIN" identity add --key "$KEYS/id_ed25519.pub" --role maintainer --apply >/dev/null 2>&1 || fail "identity add failed"
"$BIN" hooks install >/dev/null 2>&1 || fail "hooks install failed"
git add -A && git commit -qm "Register the maintainer" -m "POSE-Spec: pose-onboarding" --no-verify
ONBOARDING="$(find .pose/specs -name '*-pose-onboarding.md' | head -1)"
python3 - "$ONBOARDING" <<'PY'
import sys
p = sys.argv[1]
s = open(p, encoding="utf-8").read()
trace = "".join(f"- R{i} [satisfied] doc:{d}\n" for i, d in enumerate(
    [".pose/project.json", ".pose/policy/actions.json", ".git/hooks/pre-commit", ".pose/policy/adoption-decisions.json", ".pose/project.json"], 1))
s = s.replace("### Requirement trace\n", "### Requirement trace\n" + trace, 1)
open(p, "w", encoding="utf-8").write(s)
PY
git add -A && git commit -qm "Trace onboarding" -m "POSE-Spec: pose-onboarding" --no-verify
next="$("$BIN" setup --json | python3 -c 'import json,sys; print(json.load(sys.stdin)["next"]["command"])')"
[ "$next" = "pose close spec:pose-onboarding" ] || fail "with every step done, setup names $next instead of closing the onboarding spec"
code=0
out="$("$BIN" close spec:pose-onboarding --apply --reviewer agent:reviewer 2>&1)" || code=$?
[ "$code" = 3 ] || fail "closing the onboarding spec did not stop for the reviewer (exit $code): $out"
ATTEST="$(field closeout_plan.attest <<<"$out")"
ATTEST="${ATTEST//<id>/agent:reviewer}"
ATTEST="${ATTEST//<your conclusion>/checked against the onboarding requirements}"
ATTEST="$(sed -E 's/<decision that reaches a requirement, e.g. D1>/D2/g; s/<why [^>]*>/configuration only, as the onboarding requirements state/g' <<<"$ATTEST")"
ATTEST="${ATTEST/#pose /\"$BIN\" }"
eval "$ATTEST" >/dev/null 2>&1 || fail "the printed attest command for the onboarding spec was refused: $ATTEST"
out="$("$BIN" close spec:pose-onboarding --resume 2>&1)" || fail "closing the onboarding spec failed: $out"
grep -q "^status: done" "$ONBOARDING" || fail "the onboarding spec is not done"
git add -A && git commit -qm "Close onboarding" -m "POSE-Spec: pose-onboarding" --no-verify
out="$("$BIN" check --strict 2>&1)" || fail "the instance fails the strict check after onboarding: $out"
grep -q "warning" <<<"$out" && fail "closing the onboarding spec leaves a warning: $(grep -i warning <<<"$out")"
"$BIN" lint-spec pose-onboarding --strict >/dev/null 2>&1 || fail "the closed onboarding spec fails lint"
grep -q "nothing to set up" <<<"$("$BIN" setup --no-input 2>&1)" || fail "setup still has steps after onboarding"
ok "journey 1b: the onboarding spec is started, driven by setup and closed through review"

# Journey 2 — installed by the newest published release older than both the
# engine under test and its newest governed capability, then updated. "Older"
# matters: an update between releases that offer the same capabilities has
# nothing to review, and the journey would test nothing (specs
# pose-upgrade-journey-starts-below-current and
# pose-upgrade-journey-starts-below-the-newest-capability).
if [ -z "$PREVIOUS" ]; then
  CURRENT="$("$BIN" version | sed -n '1s/^pose \([0-9][0-9.]*\).*/\1/p')"
  [ -n "$CURRENT" ] || fail "could not read the version of the engine under test"
  NEWEST_CAPABILITY="$("$BIN" adopt --list --json | python3 -c 'import json,sys; v=[tuple(int(x) for x in c["introduced_in"].split(".")) for c in json.load(sys.stdin) if c.get("introduced_in")]; print(".".join(map(str,max(v))) if v else "")')"
  [ -n "$NEWEST_CAPABILITY" ] || fail "could not read when the newest governed capability was introduced"
  if python3 -c 'import sys; a,b=(tuple(int(x) for x in v.split(".")) for v in sys.argv[1:3]); sys.exit(0 if b < a else 1)' "$CURRENT" "$NEWEST_CAPABILITY"; then
    CURRENT="$NEWEST_CAPABILITY"
  fi
  RELEASES="$(mktemp)"
  curl -fsSL -o "$RELEASES" "https://api.github.com/repos/oseiaspereira88/pose/releases?per_page=50" || fail "listing the published releases"
  PREVIOUS="$(python3 - "$CURRENT" "$RELEASES" <<'PY'
import json, re, sys
current = tuple(int(x) for x in sys.argv[1].split("."))
older = []
for release in json.load(open(sys.argv[2])):
    if release.get("draft") or release.get("prerelease"):
        continue
    m = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)", release.get("tag_name", ""))
    if m and tuple(map(int, m.groups())) < current:
        older.append(tuple(map(int, m.groups())))
print(".".join(map(str, max(older))) if older else "")
PY
)"
  [ -n "$PREVIOUS" ] || fail "no published release is older than $CURRENT, the engine under test or its newest governed capability"
fi
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$(uname -m)" in x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; *) fail "unsupported architecture $(uname -m)" ;; esac
OLD="$(mktemp -d)"
ASSET="pose_${PREVIOUS}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/oseiaspereira88/pose/releases/download/v${PREVIOUS}"
curl -fsSL -o "$OLD/$ASSET" "$BASE/$ASSET" || fail "downloading $ASSET"
curl -fsSL -o "$OLD/checksums.txt" "$BASE/checksums.txt" || fail "downloading checksums.txt"
(cd "$OLD" && grep " ${ASSET}\$" checksums.txt | sha256sum -c - >/dev/null) || fail "checksum mismatch for $ASSET"
tar -xzf "$OLD/$ASSET" -C "$OLD"

REPO="$(new_repo)"
cd "$REPO"
"$OLD/pose" install . >/dev/null 2>&1 || fail "pose $PREVIOUS install failed"
commit "$REPO" "POSE $PREVIOUS"
out="$("$BIN" update --no-self 2>&1)" || fail "update from $PREVIOUS failed: $out"
grep -q "configuration review: .pose/specs/" <<<"$out" || fail "the update did not open a configuration review"
commit "$REPO" "Update POSE"
"$BIN" check --strict >/dev/null 2>&1 || fail "the updated instance fails the strict check"
out="$("$BIN" doctor 2>&1)" || fail "doctor errors after the update"
grep -q "0 error(s), 0 warning(s)" <<<"$out" || fail "the updated instance is not clean in doctor: $(grep '\[!\]\|\[✗\]' <<<"$out")"
grep -q "setup.capabilities" <<<"$out" || fail "doctor does not name the pending decisions"
plan="$("$BIN" setup --json)"
read -r REQUEST CAP <<<"$(python3 - "$plan" <<'PY'
import json, sys
plan = json.loads(sys.argv[1])
assert plan["new"], "an instance from an older release has nothing to review"
steps = {s["id"]: s for s in plan["steps"]}
assert steps["identity.maintainer"]["state"] == "todo", steps["identity.maintainer"]
asked = [s for s in plan["steps"] if s["id"].startswith("capability:") and "asked in act-" in s["summary"]]
assert len(asked) == len(plan["new"]), (len(asked), len(plan["new"]))
target = [s for s in asked if s["id"] == "capability:contract-nodes"] or asked
print(target[0]["summary"].rsplit("asked in ", 1)[1].split()[0].rstrip(";"), target[0]["id"].split(":", 1)[1])
PY
)"
[ -n "${CAP:-}" ] || fail "setup after the update"
out="$("$BIN" update --no-self 2>&1)" || fail "second update failed"
grep -q "configuration review: .pose/specs/" <<<"$out" && fail "a second update asked again"
[ -z "$(git status --porcelain -- .pose/specs .pose/actions)" ] || fail "a second update wrote review files"
ok "journey 2a: $PREVIOUS → current — strict check, clean doctor, one request per new capability, asked once"

"$BIN" identity add --key "$KEYS/id_ed25519.pub" --role maintainer --apply >/dev/null 2>&1 || fail "identity add failed"
show="$("$BIN" action show "$REQUEST" 2>&1)"
DIGEST="$(field action.request_digest <<<"$show")"
"$BIN" action resolve "$REQUEST" --actor human:ada --answer adopt --request-digest "$DIGEST" --expected-revision 1 \
  --idempotency-key journey --sign "$KEYS/id_ed25519" --apply >/dev/null 2>&1 || fail "the signed answer was refused"
out="$("$BIN" adopt --request "$REQUEST" --apply 2>&1)" || fail "adopt --request failed: $out"
"$BIN" adopt --list | grep -q "^adopt.$CAP=on" || fail "$CAP is not on after the answer was applied"
commit "$REPO" "Adopt $CAP"
"$BIN" check --strict >/dev/null 2>&1 || fail "the instance fails the strict check after applying an answer"
ok "journey 2b: a signed answer to a review request is applied with pose adopt --request"

cd "$ROOT"
printf '\n\033[32mThe install and upgrade journeys behave as onboarding promises.\033[0m\n'
