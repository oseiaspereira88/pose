#!/usr/bin/env bash
set -euo pipefail

# shellcheck source=pose-lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/pose-lib.sh"

usage() {
  cat <<'USAGE'
Uso: pose-hooks.sh <comando> [opções]

Comandos:
  install [--force]   Instala hooks gerenciados de .pose/hooks/ em .git/hooks/
                      via symlink relativo. Em conflito com hook pré-existente,
                      use --force para sobrescrever (salvando .backup).
  uninstall           Remove os symlinks gerenciados de .git/hooks/, restaurando
                      .backup quando existir.
  status              Lista o estado de cada hook gerenciado (instalado,
                      ausente, conflitando).

Hooks gerenciados atualmente:
  - pre-commit  → roda ./pose check --tolerant
  - post-merge  → roda ./pose index
USAGE
}

CMD="${1:-}"
if [[ -z "$CMD" || "$CMD" == "-h" || "$CMD" == "--help" ]]; then
  usage
  [[ -z "$CMD" ]] && exit 2 || exit 0
fi
shift

FORCE=false
while [[ $# -gt 0 ]]; do
  case "$1" in
    --force) FORCE=true; shift ;;
    *) echo "Erro: argumento inválido: $1" >&2; usage; exit 2 ;;
  esac
done

REPO_ROOT="$(pose_repo_root)"
HOOKS_SRC_DIR="$REPO_ROOT/.pose/hooks"
GIT_HOOKS_DIR="$REPO_ROOT/.git/hooks"
MANAGED_HOOKS=(pre-commit post-merge)

if [[ ! -d "$GIT_HOOKS_DIR" ]]; then
  echo "Erro: .git/hooks/ não encontrado em $REPO_ROOT (este é um git repo?)" >&2
  exit 2
fi

# Detecta se o destino já é um link gerenciado por nós.
is_managed_link() {
  local target="$1"
  [[ -L "$target" ]] || return 1
  local resolved
  resolved="$(readlink "$target")"
  [[ "$resolved" == *.pose/hooks/* ]]
}

case "$CMD" in
  install)
    for hook in "${MANAGED_HOOKS[@]}"; do
      src="$HOOKS_SRC_DIR/${hook}.sh"
      dst="$GIT_HOOKS_DIR/$hook"

      if [[ ! -x "$src" ]]; then
        echo "[ERRO] hook fonte ausente ou não executável: $src" >&2
        continue
      fi

      if [[ -e "$dst" || -L "$dst" ]]; then
        if is_managed_link "$dst"; then
          ln -snf "../../.pose/hooks/${hook}.sh" "$dst"
          echo "[OK] $hook: symlink já gerenciado, atualizado."
          continue
        fi
        if ! $FORCE; then
          echo "[AVISO] $hook: já existe em .git/hooks/ (não-gerenciado). Use --force para sobrescrever." >&2
          continue
        fi
        backup="$dst.backup.$(date +%s)"
        mv "$dst" "$backup"
        echo "[INFO] $hook: backup criado em $backup"
      fi

      ln -s "../../.pose/hooks/${hook}.sh" "$dst"
      echo "[OK] $hook: instalado."
    done
    ;;
  uninstall)
    for hook in "${MANAGED_HOOKS[@]}"; do
      dst="$GIT_HOOKS_DIR/$hook"
      if is_managed_link "$dst"; then
        rm -f "$dst"
        echo "[OK] $hook: removido."

        # Restaura backup mais recente, se existir.
        latest_backup="$(ls -t "$GIT_HOOKS_DIR/$hook".backup.* 2>/dev/null | head -n1 || true)"
        if [[ -n "$latest_backup" ]]; then
          mv "$latest_backup" "$dst"
          echo "[INFO] $hook: backup restaurado de $latest_backup"
        fi
      elif [[ -e "$dst" ]]; then
        echo "[AVISO] $hook: presente mas não-gerenciado; preservado." >&2
      else
        echo "[INFO] $hook: já ausente."
      fi
    done
    ;;
  status)
    for hook in "${MANAGED_HOOKS[@]}"; do
      dst="$GIT_HOOKS_DIR/$hook"
      if is_managed_link "$dst"; then
        echo "[INSTALADO] $hook -> $(readlink "$dst")"
      elif [[ -e "$dst" ]]; then
        echo "[CONFLITO]  $hook (existe mas não é symlink gerenciado)"
      else
        echo "[AUSENTE]   $hook"
      fi
    done
    ;;
  *)
    echo "Erro: comando desconhecido: $CMD" >&2
    usage
    exit 2
    ;;
esac
