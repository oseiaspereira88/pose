# Reports POSE

## Convenção de naming

- Use `--task` com frase curta e estável por contexto funcional.
- Reutilize o mesmo task slug para execuções do mesmo fluxo.
- Evite incluir data, ticket efêmero ou branch no `--task`.
- Exemplo recomendado: `--task "comparison-history-pose-report"`.

## Comparação temporal

- O report padrão grava histórico mínimo em `.pose/reports/history/<type>-<task-slug>.jsonl`.
- A comparação usa campos estáveis: `task_slug`, `spec`, `report_type`, `workflow`, `rules`, `validation_profile`, `context`.
- O relatório mostra `status` (`first-run`, `stable`, `changed`) e lista diffs de campos estáveis.
- Metadados de rastreabilidade incluem `generated_at`, `sequence`, `validation_profile`, `context`, `risk` e `stable_hash`.

## Gatilho de ativação

- Ative obrigatoriamente leitura temporal quando 3 ou mais módulos críticos exigirem comparação histórica por task/spec.
- Mantenha uso opcional fora desse cenário, sem bloquear geração padrão.
