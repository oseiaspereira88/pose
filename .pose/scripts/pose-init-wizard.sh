#!/usr/bin/env bash
# pose-init-wizard.sh — onboarding assistido (spec pose-init-wizard):
# detecta stacks do repositório e semeia moduleOverrides na
# validation-matrix.json, para o time sair de "instalado" para "validando"
# sem editar JSON à mão.
#
# Uso: ./pose init --wizard [--yes]
#   --yes  aceita todas as sugestões sem prompt (CI/scripts)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/pose-lib.sh"

ROOT_DIR="$(pose_repo_root)"
MATRIX="$ROOT_DIR/.pose/indexes/validation-matrix.json"
ASSUME_YES=0
[[ "${1:-}" == "--yes" ]] && ASSUME_YES=1

echo "== POSE init wizard =="
bash "$SCRIPT_DIR/pose-init.sh"

[[ -f "$MATRIX" ]] || { echo "[ERRO] validation-matrix.json ausente — instale o POSE primeiro." >&2; exit 1; }

# Descoberta de módulos por marcador de stack (mesmas convenções do validate).
mapfile -t found < <(python3 - "$ROOT_DIR" <<'PYEOF'
import os, sys, json
root = sys.argv[1]
PRUNE = {".git", "node_modules", "vendor", ".venv", ".pnpm-store",
         "dist", "build", ".next", "target", "coverage", ".pose"}
MARKERS = [("go.mod", "go"), ("package.json", "node"),
           ("Cargo.toml", "rust"), ("pom.xml", "java"), ("build.gradle", "java")]
seen = {}
for dirpath, dirnames, filenames in os.walk(root):
    dirnames[:] = [d for d in dirnames if d not in PRUNE and not d.startswith(".")]
    rel = os.path.relpath(dirpath, root)
    if rel == ".":
        rel = ""
    for marker, stack in MARKERS:
        if marker in filenames:
            key = rel or "."
            seen.setdefault(key, stack)
            break
for module, stack in sorted(seen.items()):
    print(f"{module}|{stack}")
PYEOF
)

if [[ ${#found[@]} -eq 0 ]]; then
  echo "[INFO] nenhum módulo com marcador de stack (go.mod/package.json/Cargo.toml/pom.xml) encontrado."
  echo "[INFO] edite .pose/indexes/validation-matrix.json manualmente quando o primeiro módulo existir."
  exit 0
fi

echo
echo "Módulos detectados:"
for entry in "${found[@]}"; do
  echo "  - ${entry%%|*}  (stack: ${entry##*|})"
done
echo

accepted=()
for entry in "${found[@]}"; do
  module="${entry%%|*}"
  stack="${entry##*|}"
  if [[ "$ASSUME_YES" -eq 1 ]]; then
    accepted+=("$entry")
    continue
  fi
  read -r -p "Incluir '$module' ($stack) na matriz de validação? [Y/n] " answer </dev/tty || answer="y"
  case "${answer,,}" in
    n|no|nao|não) echo "  pulado: $module" ;;
    *) accepted+=("$entry") ;;
  esac
done

if [[ ${#accepted[@]} -eq 0 ]]; then
  echo "[INFO] nada aceito; matriz intocada."
  exit 0
fi

python3 - "$MATRIX" "${accepted[@]}" <<'PYEOF'
import json, sys
matrix_path = sys.argv[1]
with open(matrix_path, encoding="utf-8") as f:
    matrix = json.load(f)
overrides = matrix.setdefault("moduleOverrides", {})
added, kept = [], []
for line in sys.argv[2:]:
    if not line.strip():
        continue
    module, stack = line.rsplit("|", 1)
    if module in overrides:
        kept.append(module)
        continue
    # Onboarding seguro: módulos novos entram em tolerant; promova a strict
    # quando os checks estabilizarem (POSE.md §7, rollout faseado).
    overrides[module] = {"stack": stack, "mode": "tolerant"}
    added.append(module)
with open(matrix_path, "w", encoding="utf-8") as f:
    json.dump(matrix, f, ensure_ascii=False, indent=2)
    f.write("\n")
for m in added:
    print(f"[OK] moduleOverrides + {m}")
for m in kept:
    print(f"[INFO] já existia (mantido): {m}")
PYEOF

echo
echo "[INFO] módulos entram em modo 'tolerant' — promova a 'strict' quando estabilizar (POSE.md §7)."
bash "$SCRIPT_DIR/pose-index.sh" >/dev/null
echo "[INFO] índices regenerados. Rode: ./pose validate --tolerant"
