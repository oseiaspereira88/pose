#!/usr/bin/env bash
# Regenerate .github/action-runtimes.json from the workflows as they stand.
#
# The runtime-currency gate requires every referenced action to have a record
# pinned at the same ref, so a bump — by hand or from a Dependabot PR — fails CI
# until the record is refreshed. That two-file edit is deliberate: it is what
# stops the manifest from describing a version the workflows no longer use.
# This makes the second file a one-command edit rather than a manual one.
#
# The deprecated-runtime list is preserved: it is a human input with owners and
# announcement dates, not something to regenerate.
#
# Usage: bash scripts/refresh-action-runtimes.sh
# Requires: gh (authenticated read), python3.
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
manifest="$repo_root/.github/action-runtimes.json"

python3 - "$repo_root" "$manifest" <<'PY'
import base64, glob, json, os, re, subprocess, sys

root, manifest = sys.argv[1], sys.argv[2]

refs = {}
for path in sorted(glob.glob(os.path.join(root, ".github/workflows/*.yml"))):
    with open(path, encoding="utf-8") as fh:
        for m in re.finditer(r"^\s*(?:-\s+)?uses:\s*([^\s#]+)", fh.read(), re.M):
            ref = m.group(1)
            if ref.startswith("./"):
                continue
            action, _, pinned = ref.partition("@")
            if pinned:
                refs[action] = pinned

def runs_using(action, pinned):
    parts = action.split("/")
    base, sub = "/".join(parts[:2]), "/".join(parts[2:])
    candidates = [f"{sub}/action.yml", f"{sub}/action.yaml"] if sub else ["action.yml", "action.yaml"]
    for candidate in candidates:
        proc = subprocess.run(
            ["gh", "api", f"repos/{base}/contents/{candidate}?ref={pinned}", "--jq", ".content"],
            capture_output=True, text=True)
        if proc.returncode == 0 and proc.stdout.strip():
            body = base64.b64decode(proc.stdout).decode("utf-8", "replace")
            found = re.search(r"^\s*using:\s*['\"]?([\w.-]+)", body, re.M)
            if found:
                return found.group(1)
    return None

existing = {}
if os.path.exists(manifest):
    with open(manifest, encoding="utf-8") as fh:
        existing = json.load(fh)

runtimes = {}
for action in sorted(refs):
    pinned = refs[action]
    using = runs_using(action, pinned)
    if using is None:
        print(f"refresh-action-runtimes: FAIL: cannot read action.yml for {action} at {pinned}", file=sys.stderr)
        sys.exit(1)
    runtimes[action] = {"ref": pinned, "using": using}
    print(f"  {action:<42} {using}")

doc = {
    "schema_version": 1,
    "comment": existing.get("comment", ""),
    "deprecated_runtimes": existing.get("deprecated_runtimes", []),
    "runtimes": runtimes,
}
with open(manifest, "w", encoding="utf-8") as fh:
    json.dump(doc, fh, indent=2)
    fh.write("\n")
print(f"refresh-action-runtimes: recorded {len(runtimes)} action(s)")
PY
