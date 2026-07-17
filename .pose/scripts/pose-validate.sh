#!/usr/bin/env bash

set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-validate.sh [--strict|--tolerant] [--stack <node|go|rust|java|contract>] [--module <path>] [--report] [--report-task <nome>]

Opções:
  --strict | --tolerant     Modo do gate (default: strict via matriz).
  --stack <s>               Filtra por stack (node|go|rust|java|contract).
  --module <path>           Filtra para um único módulo (relativo à raiz).
  --report                  Após validar, captura a saída em
                            .pose/reports/pose-validate.latest.log e dispara
                            ./pose report automaticamente com --outcome
                            deduzido (auto-validate).
  --report-task <nome>      Nome da task no relatório auto-gerado. Default:
                            validate-<filtros>-<modo> (gerado a partir dos
                            filtros aplicados).
  -h, --help                Mostra esta ajuda.

Exemplos:
  ./pose validate --strict
  ./pose validate --tolerant --stack node
  ./pose validate --module local/storage-ui
  ./pose validate --tolerant --report
USAGE
}

MODE=""
STACK_FILTER=""
MODULE_FILTER=""
AUTO_REPORT=false
REPORT_TASK=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --strict) MODE="strict"; shift ;;
    --tolerant) MODE="tolerant"; shift ;;
    --stack)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --stack exige um valor." >&2; usage; exit 2; }
      STACK_FILTER="${2:-}"
      case "$STACK_FILTER" in
        node|go|rust|java|contract) ;;
        *) echo "Erro: --stack inválido: $STACK_FILTER (use node|go|rust|java|contract)." >&2; usage; exit 2 ;;
      esac
      shift 2
      ;;
    --module)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --module exige um caminho." >&2; usage; exit 2; }
      MODULE_FILTER="${2:-}"
      shift 2
      ;;
    --report) AUTO_REPORT=true; shift ;;
    --report-task)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --report-task exige um valor." >&2; usage; exit 2; }
      REPORT_TASK="${2:-}"; shift 2
      ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
MATRIX_PATH="$REPO_ROOT/.pose/indexes/validation-matrix.json"

if [[ ! -f "$MATRIX_PATH" ]]; then
  echo "Erro: matriz de validação não encontrada em $MATRIX_PATH" >&2
  exit 2
fi

if [[ -n "$MODULE_FILTER" ]]; then
  if [[ "$MODULE_FILTER" != /* ]]; then
    MODULE_FILTER="$REPO_ROOT/$MODULE_FILTER"
  fi
  MODULE_FILTER="$(cd "$(dirname "$MODULE_FILTER")" && pwd)/$(basename "$MODULE_FILTER")"
fi

run_in_module() {
  local module="$1"; shift
  (
    cd "$module"
    bash -lc "$*"
  )
}

run_validation() {
  declare -A MODULE_STACK MODULE_MODE MODULE_STATUS MODULE_REASON REQUIRED_FAIL OPTIONAL_FAIL
  local modules=()

  local DISCOVER_SCRIPT
  DISCOVER_SCRIPT="$(dirname "${BASH_SOURCE[0]}")/pose-validate-discover.py"

  while IFS='|' read -r module stack mode severity command; do
    [[ -n "$module" ]] || continue
    if [[ -z "${MODULE_STACK[$module]:-}" ]]; then
      modules+=("$module")
    fi
    MODULE_STACK["$module"]="$stack"
    MODULE_MODE["$module"]="$mode"

    local display_module="${module#"$REPO_ROOT"/}"
    if [[ -z "${MODULE_STATUS[$module]:-}" ]]; then
      MODULE_STATUS["$module"]="SUCESSO"
      REQUIRED_FAIL["$module"]=0
      OPTIONAL_FAIL["$module"]=0
      echo "[módulo] $display_module ($stack, mode=${mode})"
    fi

    echo "  -> $command"
    if run_in_module "$module" "$command"; then
      :
    else
      MODULE_STATUS["$module"]="FALHA"
      if [[ "$severity" == "required" ]]; then
        REQUIRED_FAIL["$module"]=1
      else
        OPTIONAL_FAIL["$module"]=1
      fi
    fi
  done < <(python3 "$DISCOVER_SCRIPT" "$REPO_ROOT" "$MATRIX_PATH" "$MODE" "$STACK_FILTER" "$MODULE_FILTER")

  if [[ "${#modules[@]}" -eq 0 ]]; then
    echo "Nenhum módulo/check correspondente à matriz/filtros."
    return 0
  fi

  local overall_fail=0
  local critical_fail=0

  for module in "${modules[@]}"; do
    local req="${REQUIRED_FAIL[$module]:-0}"
    local opt="${OPTIONAL_FAIL[$module]:-0}"
    local m_mode="${MODULE_MODE[$module]:-strict}"

    if [[ "$req" -eq 0 && "$opt" -eq 0 ]]; then
      MODULE_REASON["$module"]="Todas as validações executadas com sucesso"
    elif [[ "$req" -eq 1 ]]; then
      MODULE_REASON["$module"]="Falha em check required"
    else
      MODULE_REASON["$module"]="Falha apenas em check optional"
    fi

    if [[ "$m_mode" == "strict" && "$req" -eq 1 ]]; then
      overall_fail=1
    fi
    if [[ "$m_mode" == "tolerant" && "$req" -eq 1 ]]; then
      critical_fail=1
    fi
  done

  echo
  echo "Resumo final por módulo:"
  for module in "${modules[@]}"; do
    printf ' - [%s] %s (%s, mode=%s) - %s\n' \
      "${MODULE_STATUS[$module]}" \
      "${module#"$REPO_ROOT"/}" \
      "${MODULE_STACK[$module]}" \
      "${MODULE_MODE[$module]}" \
      "${MODULE_REASON[$module]}"
  done

  echo
  if [[ "$overall_fail" -eq 1 ]]; then
    echo "Resultado: FALHA (required falhou em módulo strict)"
    return 1
  fi
  if [[ "$critical_fail" -eq 1 ]]; then
    echo "Resultado: FALHA (falha crítica: required falhou em módulo tolerant)"
    return 1
  fi
  echo "Resultado: SUCESSO"
  return 0
}

derive_report_task() {
  local parts=("validate")
  [[ -n "$STACK_FILTER" ]] && parts+=("stack-$STACK_FILTER")
  if [[ -n "$MODULE_FILTER" ]]; then
    local mod_slug
    mod_slug="$(pose_slugify "${MODULE_FILTER#"$REPO_ROOT"/}")"
    parts+=("module-$mod_slug")
  fi
  [[ -z "$STACK_FILTER" && -z "$MODULE_FILTER" ]] && parts+=("all")
  parts+=("${MODE:-strict}")
  local IFS="-"
  printf '%s' "${parts[*]}"
}

if $AUTO_REPORT; then
  AUTO_REPORT_LOG="$REPO_ROOT/.pose/reports/pose-validate.latest.log"
  mkdir -p "$(dirname "$AUTO_REPORT_LOG")"

  set +e
  run_validation 2>&1 | tee "$AUTO_REPORT_LOG"
  EXIT_CODE="${PIPESTATUS[0]}"
  set -e

  : "${REPORT_TASK:=$(derive_report_task)}"
  REPORT_SCRIPT="$(dirname "${BASH_SOURCE[0]}")/pose-report.sh"
  bash "$REPORT_SCRIPT" \
    --task "$REPORT_TASK" \
    --workflow ".pose/scripts/pose-validate.sh" \
    --context "auto-validate" \
    --validation-profile "${MODE:-strict}" \
    --validate-output "$AUTO_REPORT_LOG" > /dev/null

  exit "$EXIT_CODE"
fi

run_validation
