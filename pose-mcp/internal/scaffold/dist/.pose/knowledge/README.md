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

- Liste vencidos: `pose knowledge-housekeeping list-expired`
- Arquive vencidos (simulação): `pose knowledge-housekeeping archive-expired --dry-run`
- Expurgue arquivados (simulação): `pose knowledge-housekeeping purge-archived --dry-run`
