#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

ROOT_DIR="$(pose_repo_root)"
INDEX_DIR="$ROOT_DIR/.pose/indexes"
TMP_DIR="$INDEX_DIR/.tmp"

mkdir -p "$INDEX_DIR" "$TMP_DIR"

python3 - "$ROOT_DIR" "$TMP_DIR" "$INDEX_DIR/module-metadata.json" <<'PY'
import json
import os
import pathlib
import sys

root = pathlib.Path(sys.argv[1]).resolve()
tmp_dir = pathlib.Path(sys.argv[2]).resolve()
metadata_file = pathlib.Path(sys.argv[3]).resolve()

IGNORE_DIRS = {
    ".git", "node_modules", ".gradle", "build", "dist", "target", "vendor", "__pycache__"
}
MANIFESTS = ("package.json", "go.mod", "pom.xml", "Cargo.toml")


def should_skip(path: pathlib.Path) -> bool:
    return any(part in IGNORE_DIRS for part in path.parts)


def detect_language(module_dir: pathlib.Path) -> str:
    if (module_dir / "go.mod").exists():
        return "go"
    if (module_dir / "Cargo.toml").exists():
        return "rust"
    if (module_dir / "pom.xml").exists():
        return "java"
    if (module_dir / "package.json").exists():
        return "javascript"
    return "unknown"

def has_helm_chart(module_dir: pathlib.Path) -> bool:
    if (module_dir / "Chart.yaml").exists():
        return True
    charts_dir = module_dir / "charts"
    if charts_dir.is_dir():
        for p in charts_dir.rglob("Chart.yaml"):
            if not should_skip(p):
                return True
    return False


ALLOWED_CRITICALITY = {"low", "medium", "high", "critical"}


def load_metadata():
    schema_version = 1
    defaults = {
        "owner": "unknown",
        "criticality": "medium",
        "domain": "unknown",
        "validationProfile": "baseline",
    }
    modules = {}

    if not metadata_file.exists():
        return schema_version, defaults, modules

    raw = json.loads(metadata_file.read_text(encoding="utf-8"))
    if isinstance(raw.get("schemaVersion"), int) and raw["schemaVersion"] > 0:
        schema_version = raw["schemaVersion"]
    file_defaults = raw.get("defaults", {})
    for key in defaults:
        value = file_defaults.get(key)
        if isinstance(value, str) and value.strip():
            defaults[key] = value.strip()

    for module_path, meta in raw.get("modules", {}).items():
        if not isinstance(meta, dict):
            continue
        clean = {}
        for key in defaults:
            value = meta.get(key)
            if isinstance(value, str) and value.strip():
                clean[key] = value.strip()

        criticality = clean.get("criticality")
        if criticality and criticality not in ALLOWED_CRITICALITY:
            clean["criticality"] = defaults["criticality"]

        modules[module_path.strip("/")] = clean

    if defaults["criticality"] not in ALLOWED_CRITICALITY:
        defaults["criticality"] = "medium"

    return schema_version, defaults, modules


def build_module_metadata(rel_path: str, defaults: dict, declared: dict) -> tuple[dict, dict]:
    merged = defaults.copy()
    merged.update(declared)
    if merged["criticality"] not in ALLOWED_CRITICALITY:
        merged["criticality"] = defaults["criticality"]

    missing_fields = sorted(key for key in defaults if key not in declared)
    has_declared_entry = len(declared) > 0
    status = {
        "isComplete": len(missing_fields) == 0 and has_declared_entry,
        "source": "declared" if len(missing_fields) == 0 and has_declared_entry else ("partial" if has_declared_entry else "defaulted"),
        "missingFields": missing_fields,
    }
    return merged, status

def classify_module(rel_path: str, name: str) -> str:
    lowered = f"{rel_path}/{name}".lower()
    if "service" in name.lower() or "/services/" in lowered or name.lower().endswith("-svc"):
        return "service"
    if any(token in lowered for token in ("/app", "/apps/", "-ui", "-portal", "mobile", "web")):
        return "app"
    return "package"

module_dirs = set()
manifest_files = []
dockerfiles = []
readmes = []
helm_charts = []

for dirpath, dirnames, filenames in os.walk(root):
    current = pathlib.Path(dirpath)
    rel_dir = current.relative_to(root)

    dirnames[:] = [d for d in dirnames if d not in IGNORE_DIRS and not d.startswith('.venv')]

    if should_skip(rel_dir):
        continue

    for filename in filenames:
        file_path = current / filename
        if filename in MANIFESTS:
            manifest_files.append(str(file_path.relative_to(root)))
            module_dirs.add(current)
        elif filename == "Dockerfile" or filename.startswith("Dockerfile."):
            dockerfiles.append(str(file_path.relative_to(root)))
        elif filename.lower().startswith("readme"):
            readmes.append(str(file_path.relative_to(root)))
        elif filename == "Chart.yaml":
            helm_charts.append(str(file_path.relative_to(root)))

metadata_schema_version, metadata_defaults, metadata_by_module = load_metadata()

items = []
for module_dir in sorted(module_dirs):
    rel = str(module_dir.relative_to(root))
    name = module_dir.name
    module_type = classify_module(rel, name)
    declared_metadata = metadata_by_module.get(rel, {})
    module_metadata, metadata_status = build_module_metadata(rel, metadata_defaults, declared_metadata)

    item = {
        "name": name,
        "path": rel,
        "language": detect_language(module_dir),
        "hasDockerfile": any(pathlib.Path(d).parent == pathlib.Path(rel) for d in dockerfiles),
        "hasHelmChart": has_helm_chart(module_dir),
        "owner": module_metadata["owner"],
        "criticality": module_metadata["criticality"],
        "domain": module_metadata["domain"],
        "validationProfile": module_metadata["validationProfile"],
        "metadata": module_metadata,
        "metadataStatus": metadata_status,
    }
    items.append((module_type, item))

services = [item for kind, item in items if kind == "service"]
packages = [item for kind, item in items if kind == "package"]
apps = [item for kind, item in items if kind == "app"]

repo_map = {
    "root": str(root),
    "apps": apps,
    "services": services,
    "packages": packages,
    "manifests": sorted(manifest_files),
    "dockerfiles": sorted(dockerfiles),
    "helmCharts": sorted(helm_charts),
    "readmes": sorted(readmes),
    "moduleMetadata": {
        "schemaVersion": metadata_schema_version,
        "source": str(metadata_file.relative_to(root)) if metadata_file.exists() else None,
        "defaults": metadata_defaults,
    },
}

(tmp_dir / "repo-map.json").write_text(json.dumps(repo_map, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
(tmp_dir / "services.json").write_text(json.dumps(services, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
(tmp_dir / "packages.json").write_text(json.dumps(packages, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

# Grafo de dependências entre specs (pose-spec-dependencies) — cache para
# consumidores externos (pose-mcp/Conductor); o frontmatter segue autoritativo.
SPEC_GRAPH="$ROOT_DIR/.pose/scripts/pose-spec-graph.py"
if [ -d "$ROOT_DIR/.pose/specs" ] && [ -x "$SPEC_GRAPH" ]; then
  python3 "$SPEC_GRAPH" --specs-dir "$ROOT_DIR/.pose/specs" --emit > "$TMP_DIR/spec-graph.json"
  # Roadmaps governados (pose-roadmap-artifact) — mesmo contrato de cache.
  if [ -d "$ROOT_DIR/.pose/roadmaps" ]; then
    python3 "$SPEC_GRAPH" --specs-dir "$ROOT_DIR/.pose/specs" --emit-roadmaps > "$TMP_DIR/roadmaps.json"
  fi
fi

mv -f "$TMP_DIR"/*.json "$INDEX_DIR/"
rmdir "$TMP_DIR" 2>/dev/null || true

echo "Índices POSE atualizados em $INDEX_DIR"
