#!/usr/bin/env bash
# pose-release-notes.sh — consolida os changelog fragments de
# .pose/changelogs/unreleased/ em notas de release user-facing, agrupadas por
# categoria (added/changed/fixed/removed/deprecated/security), com seção de
# breaking changes no topo. Spec: pose-release-pipeline (fecha o follow-up de
# publicação externa de pose-release-changelog).
#
# Uso: pose-release-notes.sh [--version <v>] [--dir <fragments-dir>] [--filter <prefixo>]
#   --filter limita aos fragments cujo slug de spec começa com o prefixo
#   (pré-extração do monorepo: 'pose' isola as notas do produto POSE).
# Saída: markdown em stdout (consumido por goreleaser --release-notes).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/pose-lib.sh"

ROOT_DIR="$(pose_repo_root)"
FRAGMENTS_DIR="$ROOT_DIR/.pose/changelogs/unreleased"
VERSION=""
FILTER=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="${2:?}"; shift 2 ;;
    --dir) FRAGMENTS_DIR="${2:?}"; shift 2 ;;
    --filter) FILTER="${2:?}"; shift 2 ;;
    -h|--help) sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "[ERRO] opção desconhecida: $1" >&2; exit 2 ;;
  esac
done

[[ -d "$FRAGMENTS_DIR" ]] || { echo "[ERRO] sem fragments em $FRAGMENTS_DIR" >&2; exit 1; }

python3 - "$FRAGMENTS_DIR" "$VERSION" "$FILTER" <<'PYEOF'
import sys, re
from pathlib import Path

frag_dir, version = Path(sys.argv[1]), sys.argv[2]
flt = sys.argv[3] if len(sys.argv) > 3 else ""
CATEGORIES = ["security", "added", "changed", "fixed", "deprecated", "removed"]
TITLES = {"added": "Added", "changed": "Changed", "fixed": "Fixed",
          "removed": "Removed", "deprecated": "Deprecated", "security": "Security"}
buckets = {c: [] for c in CATEGORIES}
breaking = []

for f in sorted(frag_dir.glob("*.md")):
    text = f.read_text(encoding="utf-8")
    m = re.match(r"(?s)^---\n(.*?)\n---\n(.*)$", text)
    if not m:
        continue
    front, body = m.groups()
    meta = {}
    for line in front.splitlines():
        if ":" in line:
            k, v = line.split(":", 1)
            meta[k.strip()] = v.split("#")[0].strip()
    body = re.sub(r"(?s)<!--.*?-->", "", body).strip()
    if not body:
        continue
    cat = meta.get("category", "changed")
    if cat not in buckets:
        cat = "changed"
    spec = meta.get("spec", f.stem)
    if flt and not spec.startswith(flt):
        continue
    entry = f"- {body.replace(chr(10), ' ')} (`{spec}`)"
    if meta.get("breaking", "false").lower() == "true":
        breaking.append(entry)
    buckets[cat].append(entry)

title = f"## POSE {version}" if version else "## Unreleased"
print(title)
print()
if breaking:
    print("### ⚠️ Breaking changes")
    print("\n".join(breaking))
    print()
empty = True
for cat in CATEGORIES:
    if buckets[cat]:
        empty = False
        print(f"### {TITLES[cat]}")
        print("\n".join(buckets[cat]))
        print()
if empty:
    print("_No user-facing changes recorded._")
PYEOF
