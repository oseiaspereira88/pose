#!/usr/bin/env bash
# pose-lib.sh — funções compartilhadas dos scripts POSE.
#
# Convenções:
# - Todas as funções com prefixo `pose_`.
# - Mensagens humanas em português; chaves técnicas em JSON/JSONL em ASCII estável.
# - `set -euo pipefail` deve ser definido no script chamador, não aqui.

# Evita duplicação ao ser sourced múltiplas vezes na mesma sessão.
if [[ -n "${__POSE_LIB_LOADED:-}" ]]; then
  return 0
fi
__POSE_LIB_LOADED=1

# Caminho absoluto da raiz do repositório (git ou cwd como fallback).
pose_repo_root() {
  if [[ -n "${__POSE_REPO_ROOT_CACHE:-}" ]]; then
    printf '%s' "$__POSE_REPO_ROOT_CACHE"
    return 0
  fi
  local root
  root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
  __POSE_REPO_ROOT_CACHE="$root"
  printf '%s' "$root"
}

# Slugify determinístico: minúsculas, separa por `-`, remove pontuação.
pose_slugify() {
  local text="${1:-}"
  text="$(printf '%s' "$text" | tr '[:upper:]' '[:lower:]')"
  text="$(printf '%s' "$text" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//; s/-+/-/g')"
  [[ -n "$text" ]] || text="task"
  printf '%s' "$text"
}

# Parsing canônico de --strict/--tolerant. Imprime o modo escolhido.
# Uso: MODE="$(pose_parse_mode "${1:-}")" || exit 2
pose_parse_mode() {
  case "${1:-}" in
    ""|--strict) printf 'strict' ;;
    --tolerant) printf 'tolerant' ;;
    *)
      echo "Erro: modo inválido: ${1:-} (use --strict ou --tolerant)" >&2
      return 2
      ;;
  esac
}

# Logging consistente. Escreve em stderr para não poluir stdout de scripts que
# imprimem dados estruturados (slug, path, etc.).
pose_log_info()  { printf '[INFO] %s\n'  "$*" >&2; }
pose_log_ok()    { printf '[OK] %s\n'    "$*" >&2; }
pose_log_warn()  { printf '[AVISO] %s\n' "$*" >&2; }
pose_log_error() { printf '[ERRO] %s\n'  "$*" >&2; }

# Valida que `--flag` veio acompanhada de um valor (próximo arg não-flag).
# Uso: pose_require_flag_value --task "$#" "${2:-}" || exit 2
pose_require_flag_value() {
  local flag="$1"
  local remaining="$2"
  local next="${3:-}"
  if (( remaining < 2 )) || [[ -z "$next" ]] || [[ "$next" == --* ]]; then
    echo "Erro: $flag exige um valor." >&2
    return 2
  fi
  return 0
}

# Versão corrente do contrato .pose/ (spec pose-schema-versioning).
# Instâncias declaram a sua em .pose/schema-version; `pose upgrade` migra.
POSE_SCHEMA_VERSION=1
