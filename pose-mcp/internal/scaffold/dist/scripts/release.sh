#!/usr/bin/env bash
# Script de automação de release do POSE (spec pose-release-pipeline).
# Executa a suíte de compatibilidade, gera os scaffolds, empurra o commit e tag vX.Y.Z para o GitHub
# e monitora o workflow de publicação de release no GitHub Actions.
#
# Uso: bash scripts/release.sh [vX.Y.Z]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

# 1. Determina a versão do release
TARGET_TAG="${1:-}"

if [[ -z "$TARGET_TAG" ]]; then
  VER="$(grep -E 'var Version =' pose-mcp/internal/version/version.go | sed -E 's/.*"([^"]+)".*/\1/' | sed 's/-dev//')"
  TARGET_TAG="v${VER}"
fi

if [[ "$TARGET_TAG" != v* ]]; then
  TARGET_TAG="v${TARGET_TAG}"
fi

VERSION_NUM="${TARGET_TAG#v}"

echo "=== Iniciando processo de release POSE $TARGET_TAG ==="

# 2. Sincroniza os arquivos de distribuição incorporados em Go
echo "[1/6] Sincronizando scaffolds (go generate)..."
(cd pose-mcp && go generate ./internal/scaffold)

# 3. Roda a suíte interna Go
echo "[2/6] Executando testes unitários Go..."
(cd pose-mcp && go test ./...)

# 4. Valida a porta de compatibilidade de release
echo "[3/6] Executando porta de compatibilidade (compat.sh $TARGET_TAG)..."
bash tests/release/compat.sh "$TARGET_TAG"

# 5. Commit e Tag
echo "[4/6] Efetuando commit e tag $TARGET_TAG..."
git add .
if ! git diff --cached --quiet; then
  git commit -m "chore(release): cut $TARGET_TAG"
fi

git tag -fa "$TARGET_TAG" -m "Release $TARGET_TAG"

# 6. Push para o GitHub
echo "[5/6] Enviando branch main e tag $TARGET_TAG para o GitHub..."
git push origin main
git push origin "$TARGET_TAG" --force

# 7. Acompanhamento no GitHub Actions
echo "[6/6] Aguardando publicação do release no GitHub Actions..."
echo "Monitore a execução com: gh run list --repo oseiaspereira88/pose"
echo "URL do release: https://github.com/oseiaspereira88/pose/releases/tag/$TARGET_TAG"

for i in {1..30}; do
  sleep 10
  if gh release view "$TARGET_TAG" --repo oseiaspereira88/pose >/dev/null 2>&1; then
    echo "✅ SUCESSO: Release $TARGET_TAG publicado com sucesso no GitHub!"
    gh release view "$TARGET_TAG" --repo oseiaspereira88/pose
    exit 0
  fi
  echo "Aguardando publicação do $TARGET_TAG ($((i * 10))s)..."
done

echo "⚠️ O release $TARGET_TAG foi empurrado. Verifique a conclusão no GitHub Actions:"
echo "https://github.com/oseiaspereira88/pose/actions"
