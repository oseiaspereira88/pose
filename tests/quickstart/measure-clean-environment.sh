#!/usr/bin/env bash
# Automated command-path timing, not a human reading/development budget.
set -euo pipefail
root="$(git rev-parse --show-toplevel)"
image="${QUICKSTART_IMAGE:-golang:1.26.6-bookworm}"
output="${1:-$root/.pose/reports/2026-10-02-clean-quickstart.json}"
work="$(mktemp -d)"
container=""
trap 'if [ -n "$container" ]; then docker rm -f "$container" >/dev/null; fi; rm -rf "$work"' EXIT
docker pull "$image" >/dev/null
image_id="$(docker image inspect "$image" --format '{{.Id}}')"
container="$(docker create \
  -e MEASURE_IMAGE="$image_id" \
  "$image" bash -c '
    set -euo pipefail
    apt-get update -qq && apt-get install -y -qq python3 >/dev/null
    test ! -d /go/pkg/mod || test -z "$(ls -A /go/pkg/mod)"
    mkdir -p /evidence
    mkdir /tmp/project && cd /tmp/project
    git init -q && git config user.email measurement@example.invalid && git config user.name Measurement
    git commit --allow-empty -qm initial
    python3 - <<"PY"
import datetime,json,os,pathlib,subprocess,time
commands=[
 ["bash","-c","curl -fsSLO https://github.com/oseiaspereira88/pose/releases/latest/download/install.sh && bash install.sh"],
 ["/root/.local/bin/pose","doctor"],
 ["bash","/measure-loop.sh","--binary","/root/.local/bin/pose"],
]
rows=[]
for command in commands:
 start=time.monotonic(); result=subprocess.run(command,capture_output=True,text=True)
 rows.append({"command":command,"seconds":round(time.monotonic()-start,3),"exit_code":result.returncode,"output":result.stdout+result.stderr})
 if result.returncode: raise SystemExit(rows[-1]["output"])
version=subprocess.check_output(["/root/.local/bin/pose","version"],text=True).splitlines()[0]
record={"schema_version":1,"measured_at":datetime.datetime.now(datetime.timezone.utc).isoformat(),"image":os.environ["MEASURE_IMAGE"],"version":version,"scope":"automated installer and documented gate loop; excludes prerequisite image/package setup and human reading/development","source_checkout":False,"warm_module_cache":False,"total_seconds":round(sum(r["seconds"] for r in rows),3),"steps":rows}
pathlib.Path("/evidence/result.json").write_text(json.dumps(record,indent=2)+"\n")
print("PASS:",version,record["total_seconds"],"seconds (automated command path)")
PY
  ')"
# Copy only the harness; no host bind mounts or source checkout are needed.
docker cp "$root/tests/quickstart/first-governed-loop.sh" "$container:/measure-loop.sh"
docker start -a "$container"
docker cp "$container:/evidence/result.json" "$work/result.json"
cp "$work/result.json" "$output"
