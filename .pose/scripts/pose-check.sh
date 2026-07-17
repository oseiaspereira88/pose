#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

ROOT_DIR="$(pose_repo_root)"
MODE="strict"

case "${1:-}" in
  ""|--strict) MODE="strict" ;;
  --tolerant) MODE="tolerant" ;;
  -h|--help)
    cat <<'USAGE'
Uso: pose-check.sh [--strict|--tolerant]

Valida integridade estrutural do POSE: paths obrigatórios, scripts-chave
e referências em AGENTS.md / POSE.md.
USAGE
    exit 0
    ;;
  *)
    echo "Erro: argumento inválido: ${1:-}" >&2
    echo "Uso: $0 [--strict|--tolerant]" >&2
    exit 2
    ;;
esac

ERRORS=0
WARNINGS=0

report_issue() {
  local level="$1"
  local message="$2"

  if [[ "$level" == "ERRO" ]]; then
    echo "[ERRO] $message"
    ERRORS=$((ERRORS + 1))
  else
    echo "[AVISO] $message"
    WARNINGS=$((WARNINGS + 1))
  fi
}

fail_or_warn() {
  local message="$1"
  if [[ "$MODE" == "tolerant" ]]; then
    report_issue "AVISO" "$message"
  else
    report_issue "ERRO" "$message"
  fi
}

# Gate de versão do contrato (pose-schema-versioning): instância sem
# schema-version falha em strict (aviso em tolerant); instância com versão
# maior que o motor é erro sempre — motor antigo não opera contrato novo.
check_schema_version() {
  local version_file="$ROOT_DIR/.pose/schema-version"
  if [[ ! -f "$version_file" ]]; then
    fail_or_warn "schema: instância sem .pose/schema-version — rode './pose upgrade' (contrato v$POSE_SCHEMA_VERSION)"
    return 0
  fi
  local instance
  instance="$(tr -d '[:space:]' <"$version_file")"
  if ! [[ "$instance" =~ ^[0-9]+$ ]]; then
    report_issue "ERRO" "schema: .pose/schema-version inválido ('$instance')"
    return 0
  fi
  if (( instance > POSE_SCHEMA_VERSION )); then
    report_issue "ERRO" "schema: instância v$instance é mais nova que o motor (v$POSE_SCHEMA_VERSION) — atualize o motor POSE"
  elif (( instance < POSE_SCHEMA_VERSION )); then
    fail_or_warn "schema: instância v$instance atrás do motor (v$POSE_SCHEMA_VERSION) — rode './pose upgrade'"
  fi
}

check_required_path() {
  local path="$1"
  if [ ! -e "$path" ]; then
    report_issue "ERRO" "Path obrigatório ausente: $path"
  fi
}

check_required_file() {
  local path="$1"
  if [ ! -f "$path" ]; then
    report_issue "ERRO" "Arquivo obrigatório ausente: $path"
  fi
}

check_script_integrity() {
  local script="$1"

  check_required_file "$script"
  if [ ! -f "$script" ]; then
    return
  fi

  if [ ! -x "$script" ]; then
    report_issue "ERRO" "Script sem permissão de execução (rode chmod +x): $script"
  fi

  local first_line
  first_line="$(head -n 1 "$script" || true)"
  if [[ ! "$first_line" =~ ^#! ]]; then
    report_issue "ERRO" "Script com shebang inválido ou ausente: $script"
  fi
}

extract_referenced_paths() {
  local source_file="$1"
  # awk POSIX (compatível com mawk): match() de 2 argumentos + strip manual
  # dos delimitadores — o match() de 3 argumentos com array é extensão gawk.
  awk '
    {
      s = $0
      while (match(s, /(^|[[:space:]`"\047(])(\.pose\/[[:alnum:]_.\/-]*|\.agents\/skills\/[[:alnum:]_.\/-]*|local\/[[:alnum:]_.\/-]*)\/?([`"\047),.:;!?]|$)/)) {
        ref = substr(s, RSTART, RLENGTH)
        s = substr(s, RSTART + RLENGTH)
        sub(/^[[:space:]`"\047(]/, "", ref)
        sub(/[`"\047),.:;!?]$/, "", ref)
        sub(/\/$/, "", ref)
        if (ref != "") print ref
      }
    }
  ' "$source_file" | sort -u
}

validate_references_in_file() {
  local source_file="$1"
  local found_any=0

  while IFS= read -r ref; do
    [[ -z "$ref" ]] && continue
    found_any=1
    if [ ! -e "$ROOT_DIR/$ref" ]; then
      fail_or_warn "Referência quebrada: '$ref' (origem: ${source_file#$ROOT_DIR/})"
    fi
  done < <(extract_referenced_paths "$source_file")

  if [[ "$found_any" -eq 0 ]]; then
    report_issue "ERRO" "Nenhuma referência POSE encontrada para validar em ${source_file#$ROOT_DIR/}"
  fi
}

required_paths=(
  "$ROOT_DIR/AGENTS.md"
  "$ROOT_DIR/POSE.md"
  "$ROOT_DIR/.pose"
  "$ROOT_DIR/.pose/workflows"
  "$ROOT_DIR/.pose/templates"
  "$ROOT_DIR/.pose/rules"
  "$ROOT_DIR/.pose/scripts"
)

for path in "${required_paths[@]}"; do
  check_required_path "$path"
done

check_schema_version

required_files=(
  "$ROOT_DIR/.pose/workflows/feature.md"
  "$ROOT_DIR/.pose/workflows/review.md"
  "$ROOT_DIR/.pose/workflows/bugfix.md"
  "$ROOT_DIR/.pose/templates/spec.md"
)

for file in "${required_files[@]}"; do
  check_required_file "$file"
done

key_scripts=(
  "$ROOT_DIR/.pose/scripts/pose-lib.sh"
  "$ROOT_DIR/.pose/scripts/pose-init.sh"
  "$ROOT_DIR/.pose/scripts/pose-check.sh"
  "$ROOT_DIR/.pose/scripts/pose-index.sh"
  "$ROOT_DIR/.pose/scripts/pose-validate.sh"
  "$ROOT_DIR/.pose/scripts/pose-report.sh"
  "$ROOT_DIR/.pose/scripts/pose-new-spec.sh"
  "$ROOT_DIR/.pose/scripts/pose-new-adr.sh"
  "$ROOT_DIR/.pose/scripts/pose-new-knowledge.sh"
  "$ROOT_DIR/.pose/scripts/pose-knowledge-check.sh"
  "$ROOT_DIR/.pose/scripts/pose-knowledge-housekeeping.sh"
  "$ROOT_DIR/.pose/scripts/pose-reports-housekeeping.sh"
  "$ROOT_DIR/.pose/scripts/pose-recurrence-check.sh"
  "$ROOT_DIR/.pose/scripts/pose-hooks.sh"
  "$ROOT_DIR/.pose/scripts/pose-lint-spec.sh"
  "$ROOT_DIR/.pose/scripts/pose-suggest.sh"
  "$ROOT_DIR/.pose/scripts/pose-stats.sh"
  "$ROOT_DIR/.pose/scripts/pose-history-check.sh"
)

for script in "${key_scripts[@]}"; do
  check_script_integrity "$script"
done

for source in "$ROOT_DIR/AGENTS.md" "$ROOT_DIR/POSE.md"; do
  if [ -f "$source" ]; then
    validate_references_in_file "$source"
  else
    report_issue "ERRO" "Não foi possível validar referências porque o arquivo está ausente: $source"
  fi
done

# Schema check da validation-matrix.json — pega typos como `severty` que
# silenciosamente downgradam o comportamento do validate.
MATRIX_PATH="$ROOT_DIR/.pose/indexes/validation-matrix.json"
MATRIX_VALIDATOR="$ROOT_DIR/.pose/scripts/pose-matrix-validate.py"
if [ -f "$MATRIX_PATH" ] && [ -x "$MATRIX_VALIDATOR" ]; then
  matrix_tmp="$(mktemp)"
  if python3 "$MATRIX_VALIDATOR" --matrix-path "$MATRIX_PATH" >"$matrix_tmp" 2>&1; then
    :
  else
    while IFS= read -r line; do
      case "$line" in
        \[ERRO\]*)  fail_or_warn "validation-matrix.json: ${line#\[ERRO\] }" ;;
        \[AVISO\]*) report_issue "AVISO" "validation-matrix.json: ${line#\[AVISO\] }" ;;
      esac
    done < "$matrix_tmp"
  fi
  rm -f "$matrix_tmp"
fi

# Sync check de task-map.json — workflows/skills/rules referenciados devem existir.
TASKMAP_PATH="$ROOT_DIR/.pose/indexes/task-map.json"
if [ -f "$TASKMAP_PATH" ]; then
  taskmap_tmp="$(mktemp)"
  if python3 - "$TASKMAP_PATH" "$ROOT_DIR" >"$taskmap_tmp" 2>&1 <<'PY'
import json, os, sys
path, root = sys.argv[1:3]
try:
    data = json.loads(open(path, encoding='utf-8').read())
except (OSError, json.JSONDecodeError) as exc:
    print(f"[ERRO] falha ao parsear: {exc}")
    sys.exit(1)
errors = 0
tasks = data.get("tasks", {})
if not isinstance(tasks, dict):
    print("[ERRO] tasks: deve ser objeto")
    sys.exit(1)
for name, task in tasks.items():
    if not isinstance(task, dict):
        print(f"[ERRO] tasks.{name}: deve ser objeto")
        errors += 1
        continue
    workflow = task.get("workflow", "")
    if workflow and not os.path.exists(os.path.join(root, workflow)):
        print(f"[ERRO] tasks.{name}.workflow inexistente: {workflow}")
        errors += 1
    skill = task.get("skill", "")
    if skill:
        skill_path = os.path.join(root, ".agents/skills", skill, "SKILL.md")
        if not os.path.exists(skill_path):
            print(f"[ERRO] tasks.{name}.skill ausente: {skill}")
            errors += 1
    for rule in task.get("rules", []):
        rule_path = os.path.join(root, ".pose/rules", f"{rule}.md")
        if not os.path.exists(rule_path):
            print(f"[ERRO] tasks.{name}.rules: rule inexistente: {rule}")
            errors += 1
sys.exit(1 if errors else 0)
PY
  then
    :
  else
    while IFS= read -r line; do
      case "$line" in
        \[ERRO\]*) fail_or_warn "task-map.json: ${line#\[ERRO\] }" ;;
      esac
    done < "$taskmap_tmp"
  fi
  rm -f "$taskmap_tmp"
fi

# Enum de status no frontmatter das specs — pega `completed`/`in_progress` e afins
# que o lint-spec reprova mas antes passavam silenciosamente pelo check. Fonte única
# do enum: VALID_STATUS em .pose/scripts/pose-lint-spec.py (mantenha em sincronia).
SPECS_DIR="$ROOT_DIR/.pose/specs"
if [ -d "$SPECS_DIR" ]; then
  status_tmp="$(mktemp)"
  if python3 - "$SPECS_DIR" >"$status_tmp" 2>&1 <<'PY'
import glob, os, sys
specs_dir = sys.argv[1]
VALID = ("draft", "in-progress", "done", "blocked", "superseded", "abandoned")
errors = 0
for spec in sorted(glob.glob(os.path.join(specs_dir, "*", "spec.md"))):
    slug = os.path.basename(os.path.dirname(spec))
    status = None
    try:
        with open(spec, encoding="utf-8") as fh:
            if fh.readline().strip() != "---":
                continue  # sem frontmatter (formato legado) → não dispara o gate
            for line in fh:
                if line.strip() == "---":
                    break
                if line.startswith("status:"):
                    status = line.split(":", 1)[1].split("#", 1)[0].strip()
                    break
    except OSError as exc:
        print(f"[ERRO] {slug}: falha ao ler spec.md: {exc}")
        errors += 1
        continue
    if status is not None and status not in VALID:
        print(f"[ERRO] {slug}: status inválido no frontmatter: '{status}' (use {'|'.join(VALID)})")
        errors += 1
sys.exit(1 if errors else 0)
PY
  then
    :
  else
    while IFS= read -r line; do
      case "$line" in
        \[ERRO\]*) fail_or_warn "spec status: ${line#\[ERRO\] }" ;;
      esac
    done < "$status_tmp"
  fi
  rm -f "$status_tmp"
fi

# Grafo de dependências entre specs (pose-spec-dependencies) — valida refs de
# depends_on (sintaxe tipada, specs existentes), priority e aciclicidade.
SPEC_GRAPH="$ROOT_DIR/.pose/scripts/pose-spec-graph.py"
if [ -d "$SPECS_DIR" ] && [ -x "$SPEC_GRAPH" ]; then
  graph_tmp="$(mktemp)"
  if python3 "$SPEC_GRAPH" --specs-dir "$SPECS_DIR" --check >"$graph_tmp" 2>&1; then
    :
  else
    while IFS= read -r line; do
      case "$line" in
        \[ERRO\]*)  fail_or_warn "spec deps: ${line#\[ERRO\] }" ;;
        \[AVISO\]*) report_issue "AVISO" "spec deps: ${line#\[AVISO\] }" ;;
      esac
    done < "$graph_tmp"
  fi
  rm -f "$graph_tmp"
fi

# Changelog fragments (pose-release-changelog): fragment órfão/inválido = erro;
# spec done (pós-adoção, sem isenção) sem fragment nem menção em consolidado = aviso.
CHANGELOG_DIR="$ROOT_DIR/.pose/changelogs"
if [ -d "$SPECS_DIR" ] && [ -d "$CHANGELOG_DIR" ]; then
  changelog_tmp="$(mktemp)"
  if python3 - "$ROOT_DIR" >"$changelog_tmp" 2>&1 <<'PY'
import glob, json, os, re, sys
root = sys.argv[1]
unreleased = os.path.join(root, ".pose", "changelogs", "unreleased")
specs_dir = os.path.join(root, ".pose", "specs")
VALID_CATEGORIES = ("added", "changed", "fixed", "removed", "deprecated", "security")
errors = 0

def frontmatter(path):
    fields, body_lines, in_fm, seen_end = {}, [], False, False
    with open(path, encoding="utf-8") as fh:
        lines = fh.read().split("\n")
    if lines and lines[0].strip() == "---":
        for i, line in enumerate(lines[1:], start=1):
            if line.strip() == "---":
                body_lines = lines[i + 1:]
                seen_end = True
                break
            if ":" in line and not line.lstrip().startswith("#"):
                k, _, v = line.partition(":")
                fields[k.strip()] = re.sub(r"\s+#.*$", "", v).strip()
    if not seen_end:
        body_lines = lines
    body = re.sub(r"<!--.*?-->", "", "\n".join(body_lines), flags=re.DOTALL).strip()
    return fields, body

spec_slugs = {os.path.basename(os.path.dirname(p)) for p in glob.glob(os.path.join(specs_dir, "*", "spec.md"))}
covered = set()
for frag in sorted(glob.glob(os.path.join(unreleased, "*.md"))):
    name = os.path.basename(frag)
    if name.lower() == "readme.md":
        continue
    fields, body = frontmatter(frag)
    slug = fields.get("spec", os.path.splitext(name)[0])
    covered.add(slug)
    if slug not in spec_slugs:
        print(f"[ERRO] fragment {name}: spec inexistente: '{slug}'")
        errors += 1
    if fields.get("category") not in VALID_CATEGORIES:
        print(f"[ERRO] fragment {name}: category inválida: '{fields.get('category')}' (use {'|'.join(VALID_CATEGORIES)})")
        errors += 1
    if not body:
        print(f"[ERRO] fragment {name}: corpo vazio (escreva o resumo user-facing)")
        errors += 1

# menção em consolidados libera a cobertura
for released in glob.glob(os.path.join(root, ".pose", "changelogs", "*.md")):
    try:
        content = open(released, encoding="utf-8").read()
    except OSError:
        continue
    for slug in spec_slugs:
        if slug in content:
            covered.add(slug)

policy_path = os.path.join(root, ".pose", "policy", "changelog.json")
adopted_at = ""
if os.path.exists(policy_path):
    try:
        adopted_at = json.load(open(policy_path, encoding="utf-8")).get("adopted_at", "")
    except (OSError, json.JSONDecodeError):
        adopted_at = ""
if adopted_at:
    for spec_path in sorted(glob.glob(os.path.join(specs_dir, "*", "spec.md"))):
        slug = os.path.basename(os.path.dirname(spec_path))
        fields, _ = frontmatter(spec_path)
        if fields.get("status") != "done":
            continue
        if fields.get("changelog") == "none":
            continue
        completed = fields.get("completed_at", "")
        if not completed or completed < adopted_at:
            continue
        if slug not in covered:
            print(f"[AVISO] spec done sem changelog fragment: {slug} (crie .pose/changelogs/unreleased/{slug}.md ou marque 'changelog: none')")
sys.exit(1 if errors else 0)
PY
  then
    while IFS= read -r line; do
      case "$line" in
        \[AVISO\]*) report_issue "AVISO" "changelog: ${line#\[AVISO\] }" ;;
      esac
    done < "$changelog_tmp"
  else
    while IFS= read -r line; do
      case "$line" in
        \[ERRO\]*)  fail_or_warn "changelog: ${line#\[ERRO\] }" ;;
        \[AVISO\]*) report_issue "AVISO" "changelog: ${line#\[AVISO\] }" ;;
      esac
    done < "$changelog_tmp"
  fi
  rm -f "$changelog_tmp"
fi

# Definition of Ready (pose-definition-of-ready): a transição de uma spec para
# in-progress na árvore de trabalho exige o ready-check (gate de ENTRADA — o
# acervo não é reavaliado retroativamente; specs já in-progress no HEAD passam).
DOR_LINTER="$ROOT_DIR/.pose/scripts/pose-lint-spec.py"
if [ -d "$SPECS_DIR" ] && [ -f "$DOR_LINTER" ] && git -C "$ROOT_DIR" rev-parse --verify HEAD >/dev/null 2>&1; then
  while IFS= read -r rel; do
    case "$rel" in
      .pose/specs/*/spec.md) ;;
      *) continue ;;
    esac
    [ -f "$ROOT_DIR/$rel" ] || continue
    new_status="$(awk -F': *' '/^status:/{print $2; exit}' "$ROOT_DIR/$rel" | sed 's/ *#.*//')"
    [ "$new_status" = "in-progress" ] || continue
    old_status="$(git -C "$ROOT_DIR" show "HEAD:$rel" 2>/dev/null | awk -F': *' '/^status:/{print $2; exit}' | sed 's/ *#.*//')"
    [ "$old_status" = "in-progress" ] && continue
    if ! python3 "$DOR_LINTER" --spec "$ROOT_DIR/$rel" --ready-check >/dev/null 2>&1; then
      dor_slug="$(basename "$(dirname "$rel")")"
      fail_or_warn "DoR: transição para in-progress sem Definition of Ready: $dor_slug (detalhes: ./pose lint-spec $dor_slug --ready-check)"
    fi
  done < <({ git -C "$ROOT_DIR" diff --name-only HEAD -- .pose/specs 2>/dev/null; git -C "$ROOT_DIR" ls-files --others --exclude-standard -- .pose/specs 2>/dev/null; } | sort -u)
fi

if [ "$ERRORS" -gt 0 ]; then
  echo "Resultado: FALHA — estrutura POSE com $ERRORS erro(s)."
  exit 1
fi

if [ "$WARNINGS" -gt 0 ]; then
  echo "Resultado: SUCESSO (modo tolerant) com $WARNINGS aviso(s)."
  exit 0
fi

echo "Resultado: SUCESSO — estrutura POSE válida (modo $MODE)."
