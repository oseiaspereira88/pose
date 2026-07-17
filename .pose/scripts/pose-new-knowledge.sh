#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-new-knowledge.sh <type> <slug> [--owner @owner] [--ttl-days N] [--restricted]

Cria artefato em .pose/knowledge/<YYYY-MM-DD>-<type>-<slug>.md com frontmatter
obrigatório (type, owner, sensitivity, created_at, last_reviewed_at, expires_at).

Argumentos:
  <type>           handoff | note | decision-log
  <slug>           slug curto (kebab-case) descrevendo o tema

Opções:
  --owner <ref>    Owner do artefato (default: @pose-maintainers).
  --ttl-days <N>   TTL em dias (default: 30; máximo: 90 conforme rule).
  --restricted     Marca sensitivity como restricted (default: public-internal).
  -h, --help       Mostra esta ajuda.
USAGE
}

TYPE=""
SLUG=""
OWNER="@pose-maintainers"
TTL_DAYS=30
SENSITIVITY="public-internal"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    --owner)
      [[ $# -ge 2 && -n "${2:-}" && "${2:-}" != --* ]] || { echo "Erro: --owner exige um valor." >&2; exit 2; }
      OWNER="$2"; shift 2 ;;
    --ttl-days)
      [[ $# -ge 2 && "${2:-}" =~ ^[0-9]+$ ]] || { echo "Erro: --ttl-days exige inteiro > 0." >&2; exit 2; }
      TTL_DAYS="$2"; shift 2 ;;
    --restricted)
      SENSITIVITY="restricted"; shift ;;
    --*)
      echo "Erro: opção desconhecida: $1" >&2; usage; exit 2 ;;
    *)
      if [[ -z "$TYPE" ]]; then
        TYPE="$1"
      elif [[ -z "$SLUG" ]]; then
        SLUG="$1"
      else
        echo "Erro: argumento posicional extra: $1" >&2; usage; exit 2
      fi
      shift ;;
  esac
done

case "$TYPE" in
  handoff|note|decision-log) ;;
  "") echo "Erro: <type> é obrigatório." >&2; usage; exit 2 ;;
  *) echo "Erro: <type> inválido: $TYPE (use handoff|note|decision-log)." >&2; exit 2 ;;
esac

if [[ -z "$SLUG" ]]; then
  echo "Erro: <slug> é obrigatório." >&2
  usage
  exit 2
fi
SLUG="$(pose_slugify "$SLUG")"

if (( TTL_DAYS < 1 || TTL_DAYS > 90 )); then
  echo "Erro: --ttl-days fora do intervalo permitido (1..90). Veja .pose/rules/knowledge-governance.md" >&2
  exit 2
fi

ROOT_DIR="$(pose_repo_root)"
KNOWLEDGE_DIR="$ROOT_DIR/.pose/knowledge"
TEMPLATE="$ROOT_DIR/.pose/templates/knowledge.md"

if [[ ! -f "$TEMPLATE" ]]; then
  echo "Erro: template ausente: $TEMPLATE" >&2
  exit 2
fi

mkdir -p "$KNOWLEDGE_DIR"

CREATED_AT="$(date -u +%F)"
LAST_REVIEWED_AT="$CREATED_AT"
EXPIRES_AT="$(date -u -d "+${TTL_DAYS} days" +%F 2>/dev/null || true)"
if [[ -z "$EXPIRES_AT" ]]; then
  echo "Erro: falha ao calcular expires_at (verifique 'date -d' GNU disponível)." >&2
  exit 2
fi

FILE="$KNOWLEDGE_DIR/${CREATED_AT}-${TYPE}-${SLUG}.md"
if [[ -e "$FILE" ]]; then
  echo "Erro: artefato já existe: $FILE" >&2
  exit 1
fi

# Substituições determinísticas do template usando sed com delimitador `|`
# para evitar colisões com `/` em paths.
sed \
  -e "s|<type>|$TYPE|g" \
  -e "s|<slug>|$SLUG|g" \
  -e "s|<owner>|$OWNER|g" \
  -e "s|<sensitivity>|$SENSITIVITY|g" \
  -e "s|<created_at>|$CREATED_AT|g" \
  -e "s|<last_reviewed_at>|$LAST_REVIEWED_AT|g" \
  -e "s|<expires_at>|$EXPIRES_AT|g" \
  "$TEMPLATE" > "$FILE"

echo "Artefato de knowledge criado: $FILE"
echo "Próximos passos:"
echo "  1. Preencher Contexto, Estado atual, Próximos checks, Riscos e Próximo owner."
echo "  2. Atualizar source_refs (spec, workflow, commands)."
echo "  3. Validar com: ./pose knowledge-check --strict"
