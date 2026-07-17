#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-report.sh --task "descrição" [opções]

Opções:
  --spec "<ref>"             Spec relacionada (caminho ou slug).
  --workflow "<ref>"         Workflow aplicado (caminho relativo).
  --rules "<r1,r2>"          Rules aplicadas (lista separada por vírgula).
  --risk "<texto>"           Risco residual a registrar.
  --validate-output "<file>" Log de ./pose validate a parsear; default detecta
                             .pose/reports/pose-validate.latest.log.
  --type "<standard|doc-audit>"
                             Tipo do relatório (default: standard).
  --context "<texto>"        Contexto de execução (ex.: ci, manual, auto-validate).
  --validation-profile "<p>" Perfil de validação efetivo.
  --outcome "<pass|fail|partial|skipped|unknown>"
                             Outcome final do relatório. Quando omitido e houver
                             --validate-output, deduz a partir da linha
                             "Resultado: SUCESSO|FALHA|FALHA_TOLERADA".
  --since "<git-ref>"        Mostra arquivos alterados entre <ref> e working tree
                             via `git diff --name-only`; default usa o working
                             tree atual (`git status --porcelain`).
  --git-stage                Após escrever o JSONL de history, executa
                             `git add` nele (mantém history versionável sem
                             depender de devs lembrarem). Silencioso se git
                             indisponível.
  -h, --help                 Mostra esta ajuda.

Gera um relatório markdown versionável em .pose/reports/ e anexa registro
em .pose/reports/history/<type>-<task-slug>.jsonl.
USAGE
}

TASK=""
SPEC=""
RISK=""
VALIDATE_OUTPUT=""
REPORT_TYPE="standard"
WORKFLOW_REF=""
RULES_APPLIED=""
EXECUTION_CONTEXT=""
VALIDATION_PROFILE=""
OUTCOME=""
OUTCOME_SOURCE="manual"
SINCE_REF=""
GIT_STAGE=false

require_flag_value() {
  local flag="$1"
  local remaining_args="$2"
  if (( remaining_args < 2 )); then
    echo "Erro: $flag requer um valor." >&2
    usage
    exit 2
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --task)               require_flag_value "$1" "$#"; TASK="${2:-}"; shift 2 ;;
    --spec)               require_flag_value "$1" "$#"; SPEC="${2:-}"; shift 2 ;;
    --risk)               require_flag_value "$1" "$#"; RISK="${2:-}"; shift 2 ;;
    --workflow)           require_flag_value "$1" "$#"; WORKFLOW_REF="${2:-}"; shift 2 ;;
    --rules)              require_flag_value "$1" "$#"; RULES_APPLIED="${2:-}"; shift 2 ;;
    --validate-output)    require_flag_value "$1" "$#"; VALIDATE_OUTPUT="${2:-}"; shift 2 ;;
    --type)               require_flag_value "$1" "$#"; REPORT_TYPE="${2:-}"; shift 2 ;;
    --context)            require_flag_value "$1" "$#"; EXECUTION_CONTEXT="${2:-}"; shift 2 ;;
    --validation-profile) require_flag_value "$1" "$#"; VALIDATION_PROFILE="${2:-}"; shift 2 ;;
    --outcome)            require_flag_value "$1" "$#"; OUTCOME="${2:-}"; OUTCOME_SOURCE="manual"; shift 2 ;;
    --since)              require_flag_value "$1" "$#"; SINCE_REF="${2:-}"; shift 2 ;;
    --git-stage)          GIT_STAGE=true; shift ;;
    -h|--help)            usage; exit 0 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

if [[ "$REPORT_TYPE" != "standard" && "$REPORT_TYPE" != "doc-audit" ]]; then
  echo "Erro: --type deve ser 'standard' ou 'doc-audit'." >&2
  usage
  exit 2
fi

if [[ -z "$TASK" ]]; then
  echo "Erro: --task é obrigatório." >&2
  usage
  exit 2
fi

if [[ -n "$OUTCOME" ]]; then
  case "$OUTCOME" in
    pass|fail|partial|skipped|unknown) ;;
    *) echo "Erro: --outcome inválido: $OUTCOME (use pass|fail|partial|skipped|unknown)." >&2; exit 2 ;;
  esac
fi

ROOT_DIR="$(pose_repo_root)"
REPORTS_DIR="$ROOT_DIR/.pose/reports"
HISTORY_DIR="$REPORTS_DIR/history"
mkdir -p "$REPORTS_DIR" "$HISTORY_DIR"

DATE_STR="$(date +%F)"
GENERATED_AT="$(date -u +%FT%TZ)"
TASK_SLUG="$(pose_slugify "$TASK")"
REPORT_PATH="$REPORTS_DIR/${DATE_STR}-${REPORT_TYPE}-${TASK_SLUG}.md"
HISTORY_FILE="$HISTORY_DIR/${REPORT_TYPE}-${TASK_SLUG}.jsonl"

if [[ -z "$VALIDATE_OUTPUT" ]]; then
  if [[ -f "$REPORTS_DIR/pose-validate.latest.log" ]]; then
    VALIDATE_OUTPUT="$REPORTS_DIR/pose-validate.latest.log"
  elif [[ -f "$ROOT_DIR/.pose/pose-validate.log" ]]; then
    VALIDATE_OUTPUT="$ROOT_DIR/.pose/pose-validate.log"
  fi
fi

VALIDATION_COMMANDS="- _Preencher manualmente_"
RESULTS="- _Sem saída de validação detectada automaticamente_"
DERIVED_OUTCOME=""

if [[ -n "$VALIDATE_OUTPUT" && -f "$VALIDATE_OUTPUT" ]]; then
  mapfile -t cmd_lines < <(sed -nE 's/^  ->[[:space:]]+(.+)/\1/p' "$VALIDATE_OUTPUT")
  if [[ ${#cmd_lines[@]} -gt 0 ]]; then
    VALIDATION_COMMANDS=""
    for line in "${cmd_lines[@]}"; do
      VALIDATION_COMMANDS+='- `'"$line"'`'$'\n'
    done
    VALIDATION_COMMANDS="${VALIDATION_COMMANDS%$'\n'}"
  fi

  mapfile -t result_lines < <(sed -nE 's/^ - \[([^]]+)\] (.+)$/- [\1] \2/p' "$VALIDATE_OUTPUT")
  final_line="$(sed -nE 's/^(Resultado: .+)/- \1/p' "$VALIDATE_OUTPUT" | tail -n1 || true)"
  if [[ ${#result_lines[@]} -gt 0 || -n "$final_line" ]]; then
    RESULTS=""
    for line in "${result_lines[@]}"; do
      RESULTS+="$line"$'\n'
    done
    [[ -n "$final_line" ]] && RESULTS+="$final_line"$'\n'
    RESULTS="${RESULTS%$'\n'}"
  fi

  # Deduz outcome a partir do último "Resultado:" do log.
  if [[ -n "$final_line" ]]; then
    case "$final_line" in
      *SUCESSO*)         DERIVED_OUTCOME="pass" ;;
      *FALHA_TOLERADA*)  DERIVED_OUTCOME="partial" ;;
      *FALHA*)           DERIVED_OUTCOME="fail" ;;
    esac
  fi
fi

if [[ -z "$OUTCOME" && -n "$DERIVED_OUTCOME" ]]; then
  OUTCOME="$DERIVED_OUTCOME"
  OUTCOME_SOURCE="derived"
fi
[[ -n "$OUTCOME" ]] || OUTCOME="unknown"

# Files Changed: prefere --since <ref> (git diff); fallback é working tree.
FILES_CHANGED=""
FILES_CHANGED_SOURCE=""
if [[ -n "$SINCE_REF" ]]; then
  FILES_CHANGED_SOURCE="git diff --name-only -z $SINCE_REF"
  while IFS= read -r -d '' path; do
    [[ -n "$path" ]] || continue
    FILES_CHANGED+="- $path"$'\n'
  done < <(git -C "$ROOT_DIR" diff --name-only -z "$SINCE_REF" 2>/dev/null || true)
else
  FILES_CHANGED_SOURCE="git status --porcelain=v1 -z"
  while IFS= read -r -d '' entry; do
    [[ -n "$entry" ]] || continue
    path="${entry:3}"
    case "$entry" in
      R*|C*) IFS= read -r -d '' _old || true ;;
    esac
    FILES_CHANGED+="- $path"$'\n'
  done < <(git -C "$ROOT_DIR" status --porcelain=v1 -z 2>/dev/null || true)
fi
FILES_CHANGED="${FILES_CHANGED%$'\n'}"
[[ -n "$FILES_CHANGED" ]] || FILES_CHANGED="- _Nenhum arquivo detectado em \`${FILES_CHANGED_SOURCE}\`_"

RULES_SECTION="- _Não informado_"
if [[ -n "$RULES_APPLIED" ]]; then
  IFS=',' read -ra rules_array <<< "$RULES_APPLIED"
  RULES_SECTION=""
  for rule in "${rules_array[@]}"; do
    trimmed_rule="$(echo "$rule" | sed -E 's/^[[:space:]]+|[[:space:]]+$//g')"
    [[ -n "$trimmed_rule" ]] && RULES_SECTION+="- $trimmed_rule"$'\n'
  done
  RULES_SECTION="${RULES_SECTION%$'\n'}"
  [[ -n "$RULES_SECTION" ]] || RULES_SECTION="- _Não informado_"
fi

[[ -n "$EXECUTION_CONTEXT" ]] || EXECUTION_CONTEXT="não-informado"
[[ -n "$VALIDATION_PROFILE" ]] || VALIDATION_PROFILE="não-informado"

COMPARE_SCRIPT="$(dirname "${BASH_SOURCE[0]}")/pose-report-compare.py"
COMPARE_OUTPUT="$(python3 "$COMPARE_SCRIPT" \
  --history-file "$HISTORY_FILE" \
  --task "$TASK" \
  --task-slug "$TASK_SLUG" \
  --spec "$SPEC" \
  --report-type "$REPORT_TYPE" \
  --workflow "$WORKFLOW_REF" \
  --rules "$RULES_APPLIED" \
  --validation-profile "$VALIDATION_PROFILE" \
  --risk "$RISK" \
  --context "$EXECUTION_CONTEXT")"

COMPARE_STATUS="first-run"
PREVIOUS_AT=""
SEQUENCE="1"
STABLE_HASH=""
DIFF_LINES=""
while IFS= read -r line; do
  case "$line" in
    status=*)      COMPARE_STATUS="${line#status=}" ;;
    prev=*)        PREVIOUS_AT="${line#prev=}" ;;
    count=*)       SEQUENCE="${line#count=}" ;;
    stable_hash=*) STABLE_HASH="${line#stable_hash=}" ;;
    change=*)      DIFF_LINES+="- ${line#change=}"$'\n' ;;
  esac
done <<< "$COMPARE_OUTPUT"
DIFF_LINES="${DIFF_LINES%$'\n'}"
[[ -n "$DIFF_LINES" ]] || DIFF_LINES="- _Sem alterações em campos estáveis_"

cat > "$REPORT_PATH" <<EOF_MD
# POSE Report - $DATE_STR

## Report Type
- $REPORT_TYPE

## Task
- $TASK
- Task slug: $TASK_SLUG
$( [[ -n "$SPEC" ]] && echo "- Spec: $SPEC" )
$( [[ -n "$WORKFLOW_REF" ]] && echo "- Workflow: $WORKFLOW_REF" )

## Outcome
- Outcome: $OUTCOME (source: $OUTCOME_SOURCE)

## Rules Applied
$RULES_SECTION

## Files Changed
$FILES_CHANGED

## Validation Commands
$VALIDATION_COMMANDS

## Results
$RESULTS

## Execution Metadata
- Generated at (UTC): $GENERATED_AT
- Context: $EXECUTION_CONTEXT
- Validation profile: $VALIDATION_PROFILE
- Sequence for task/spec: $SEQUENCE
- Stable comparison hash: $STABLE_HASH

## Historical Comparison
- Previous execution: $( [[ -n "$PREVIOUS_AT" ]] && echo "$PREVIOUS_AT" || echo "_Nenhuma execução anterior_" )
- Status: $COMPARE_STATUS
- Stable field diffs:
$DIFF_LINES

## Risks
$( [[ -n "$RISK" ]] && echo "- $RISK" || echo "- _Sem riscos informados_" )

## Follow-ups
- _Adicionar próximos passos, se necessário._

## Human Review Needed
- [ ] Revisar impacto funcional
- [ ] Revisar cobertura de validação
- [ ] Aprovar merge
EOF_MD

# Escape mínimo para JSONL: \ e ". Campos não devem conter quebras de linha.
json_escape() {
  local s="${1//\\/\\\\}"
  s="${s//\"/\\\"}"
  printf '%s' "$s"
}

printf '{"generated_at":"%s","sequence":%s,"task":"%s","task_slug":"%s","report_type":"%s","spec":"%s","workflow":"%s","rules":"%s","validation_profile":"%s","context":"%s","risk":"%s","outcome":"%s","outcome_source":"%s","stable_hash":"%s","report_path":"%s"}\n' \
  "$GENERATED_AT" \
  "$SEQUENCE" \
  "$(json_escape "$TASK")" \
  "$(json_escape "$TASK_SLUG")" \
  "$REPORT_TYPE" \
  "$(json_escape "$SPEC")" \
  "$(json_escape "$WORKFLOW_REF")" \
  "$(json_escape "$RULES_APPLIED")" \
  "$(json_escape "$VALIDATION_PROFILE")" \
  "$(json_escape "$EXECUTION_CONTEXT")" \
  "$(json_escape "$RISK")" \
  "$OUTCOME" \
  "$OUTCOME_SOURCE" \
  "$STABLE_HASH" \
  "$(json_escape "$REPORT_PATH")" \
  >> "$HISTORY_FILE"

if $GIT_STAGE; then
  if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$ROOT_DIR" add -- "$HISTORY_FILE" >/dev/null 2>&1 || true
  fi
fi

printf '%s\n' "$REPORT_PATH"
