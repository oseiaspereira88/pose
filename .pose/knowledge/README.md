# `.pose/knowledge/`

Repositório de memória operacional de curto prazo sob governança POSE.

## Regras obrigatórias

- Siga `.pose/rules/knowledge-governance.md`.
- Siga `.pose/specs/pose-knowledge-governance.md`.
- Use front matter com `expires_at` e `owner`.

## Operação

- Cadência operacional: execute o ciclo quinzenalmente (segunda-feira, 09:00 UTC).
- Responsável primário: `@pose-maintainers`.
- Responsável de backup: `@your-team-leads`.

- Liste vencidos: `bash .pose/scripts/pose-knowledge-housekeeping.sh list-expired`
- Arquive vencidos (simulação): `bash .pose/scripts/pose-knowledge-housekeeping.sh archive-expired --dry-run`
- Expurgue arquivados (simulação): `bash .pose/scripts/pose-knowledge-housekeeping.sh purge-archived --dry-run`
