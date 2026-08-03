#!/usr/bin/env bash
# Corte seguro de um candidato já preparado e revisado.
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

if [[ ! "$TARGET_TAG" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "ERRO: versão inválida: $TARGET_TAG" >&2
  exit 2
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "ERRO: working tree deve estar limpo; prepare e versione o snapshot antes do corte." >&2
  exit 1
fi

if git rev-parse -q --verify "refs/tags/$TARGET_TAG" >/dev/null; then
  echo "ERRO: tag $TARGET_TAG já existe; tags nunca são sobrescritas." >&2
  exit 1
fi

echo "=== Validando candidato POSE $TARGET_TAG ==="

# 2. Sincroniza os arquivos de distribuição incorporados em Go
echo "[1/6] Validando snapshot preparado..."
(cd pose-mcp && go run ./cmd/pose release check --version "$TARGET_TAG" --strict)

# 3. Roda a suíte interna Go
echo "[2/6] Executando testes unitários Go..."
(cd pose-mcp && go test ./...)

# 4. Valida a porta de compatibilidade de release
echo "[3/6] Executando porta de compatibilidade (compat.sh $TARGET_TAG)..."
bash tests/release/compat.sh "$TARGET_TAG"

# 5. Tag imutável no HEAD já revisado
echo "[4/6] Criando tag anotada $TARGET_TAG no HEAD..."
git tag -a "$TARGET_TAG" -m "Release $TARGET_TAG"

# 6. Push para o GitHub
echo "[5/6] Enviando somente a nova tag $TARGET_TAG..."
git push origin "$TARGET_TAG"

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

echo "ERRO: tag criada, mas publicação ainda não foi verificada. Não recrie nem force a tag; diagnostique o workflow e registre um evento failed." >&2
echo "https://github.com/oseiaspereira88/pose/actions"
exit 1
