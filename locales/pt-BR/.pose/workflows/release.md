# Workflow: Release baseada em evidências

## Objetivo

Consumir fragmentos unreleased revisados em um candidato imutável e manter preparação, etiquetagem (tag), publicação pelo provedor e verificação independente como fatos separados baseados em evidências.

## Passos

1. Confirmar se o roadmap e as specs membros estão terminais com a revisão macro atual.

2. Atualizar e revisar a versão autoritativa do projeto.

3. Executar `pose release plan --version vX.Y.Z`; resolver todos os impedimentos.

4. Executar `pose release prepare --version vX.Y.Z --apply` e versionar apenas o manifesto, fragmentos arquivados e notas canônicas gerados.

5. Executar `pose release check --version vX.Y.Z --strict`, validação completa, compatibilidade, assinatura/SBOM e gates de verificação independente.

6. Exigir um worktree limpo e criar uma nova tag anotada sem sobrescrita ou force. O CI etiquetado deve consumir `pose release notes --version vX.Y.Z`.

7. Importar evidência retida do provedor com `pose release record`; não inferir publicação a partir da presença de tag.

8. Importar verificação independente vinculada à publicação e aos digests dos assets.

9. Exigir que `pose release status --version vX.Y.Z` relate `verified` antes de planejar a próxima versão de desenvolvimento com `pose release open-next`.

## Tratamento de falhas

Uma execução que falha no provedor deixa a release etiquetada, porém não publicada. Registre um evento `failed`, corrija o workflow e re-execute contra a mesma tag imutável; nunca recrie ou faça force-push na tag. Um yank é um evento explícito e não apaga o histórico de publicação anterior.
