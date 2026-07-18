# Workflow: Review

## Objetivo

Validar se a mudança está correta, segura para produção e alinhada ao escopo e às specs.

## Precondições

- Diff final está disponível e legível por commits/lotes coesos.
- Contexto de requisito/spec da mudança está acessível.
- Evidências de validação do implementador estão anexadas (incluindo saída de `./pose validate`).
- Critérios de aceite e risco esperado foram definidos.

## Checklist de execução

1. Confirmar entendimento do objetivo e do escopo aprovado.
2. Selecionar explicitamente as `rules` aplicáveis para o tipo de mudança e registrar no parecer.
2.1 Resolver conflitos entre `rules` pela alternativa mais restritiva, com prioridade para `security` quando houver risco de exposição, autorização ou integridade.
3. **Consultar `.pose/knowledge/`** por handoffs ou decision-logs que contextualizem a mudança (riscos prévios, follow-ups pendentes, decisões aceitas com gatilho de revisão).
4. Revisar aderência às specs, contratos e instruções locais.
5. Checar correção funcional e consistência de casos limite.
6. Avaliar riscos de segurança, observabilidade e performance.
7. Exigir evidência do `check` `./pose validate` conforme matriz `.pose/indexes/validation-matrix.json` e cobertura proporcional ao risco.
8. Identificar regressões potenciais e impactos de rollout/rollback.
9. Classificar achados por severidade e sugerir ações objetivas.
10. **Produzir handoff** em `.pose/knowledge/` quando achados resultarem em risco aceito, monitoramento pós-merge ou ação postergada (`./pose new-knowledge handoff <slug>`); link no parecer.
11. Emitir decisão final: aprovado, aprovado com ressalvas ou reprovado.

## Seleção obrigatória de `rules` por PR/tarefa

Antes de concluir o review, preencha e anexe a seção abaixo no parecer:

```md
## Rules aplicadas no review
- Tipo de mudança: <feature|bugfix|refactor|documentation-update|misto>
- Workflow consultado: `.pose/workflows/<arquivo>.md`
- Rules selecionadas:
  - [ ] `.pose/rules/security.md`
  - [ ] `.pose/rules/backend-go.md`
  - [ ] `.pose/rules/frontend-react.md`
  - [ ] `.pose/rules/kubernetes.md`
  - [ ] `.pose/rules/documentation-style.md`
  - [ ] `.pose/rules/knowledge-governance.md` (quando houver mudança em conhecimento/processo)
- Justificativa por rule marcada: <1 linha por item>
- Rules não aplicáveis: <listar e justificar>
```

Use a seleção como evidência obrigatória de cobertura por domínio real do monorepo.



## Escalonamento de recorrência para novo workflow

- Acione `.pose/workflows/recurrence-escalation.md` quando houver retrabalho recorrente não coberto pelos workflows atuais e limiar de ativação atendido.
- Exija evidência da métrica `recurrence_rework_uncovered` e do período de 30 dias no parecer de review.
- Exija vínculo explícito entre o workflow especializado criado e as `rules` de domínio aplicáveis.
- Exija revisão pós-piloto (45 dias) com decisão formal: `manter`, `ajustar` ou `descartar`.

## Referências explícitas de rules adotadas neste workflow

- `.pose/rules/security.md`
- `.pose/rules/backend-go.md`
- `.pose/rules/frontend-react.md`
- `.pose/rules/kubernetes.md`
- `.pose/rules/documentation-style.md`
- `.pose/rules/knowledge-governance.md`

Aplique cumulativamente por domínio e, em conflito, preserve a decisão mais restritiva documentada no parecer.

## Mapeamento mínimo de cobertura por domínio

- Mudança em UI React: aplique `frontend-react` + `security` + `documentation-style`.
- Mudança em serviços Go: aplique `backend-go` + `security` + `documentation-style`.
- Mudança em deploy/infra de cluster: aplique `kubernetes` + `security` + `documentation-style`.
- Mudança em processo/spec/workflow/rule/report: aplique `documentation-style` + `knowledge-governance` + `security` (quando houver dados sensíveis/segredos).
- Mudança transversal (multi-stack): aplique cumulativamente todas as `rules` dos domínios tocados.

## Checklist de review por domínio

### Segurança

- Confirmar autenticação/autorização e princípio do menor privilégio quando aplicável.
- Verificar ausência de segredos em código, config, manifests, docs e logs.
- Exigir evidência de `check` de vulnerabilidades/segredos aplicável ao escopo.

### Contratos

- Confirmar compatibilidade de contratos públicos (HTTP, eventos, schema, CLI, arquivos).
- Validar estratégia de compatibilidade backward/forward em rollout e rollback.
- Exigir atualização de `spec` quando contrato mudar.

### Observabilidade

- Verificar logs estruturados e métricas sem dados sensíveis.
- Confirmar probes/healthchecks/alertas coerentes com o comportamento real.
- Garantir rastreabilidade mínima para diagnóstico pós-deploy.

### Validação

- Confirmar cobertura de `check` proporcional ao risco: `lint`, `typecheck`, `test`, `build`.
- Exigir evidência de execução de `./pose validate` e resultados relevantes.
- Registrar limitações de ambiente e riscos residuais de validação.

## Checklist rápido de aderência editorial

- Validar tom imperativo e instruções acionáveis.
- Confirmar bullets curtos, sem duplicação de seções.
- Checar uso consistente dos termos `check`, `spec` e `workflow`.
- Exigir referência explícita de arquivo/caminho para evitar ambiguidade.

## Saídas obrigatórias

- Parecer de review com decisão final e racional.
- Seção `Rules aplicadas no review` preenchida com justificativas.
- Lista de achados com severidade, evidência e recomendação.
- Seção de recorrência de achados por domínio/causa e ação preventiva associada.
- Confirmação explícita sobre contratos públicos e compatibilidade.
- Riscos residuais e condições para deploy seguro.
- Referência aos `checks` executados e às evidências coletadas.

## Exemplo de review completo (multi-rule)

```md
## Review Summary
- Decisão: aprovado com ressalvas
- Tipo de mudança: feature (API Go + UI React + Helm)
- Workflow: `.pose/workflows/feature.md`

## Rules aplicadas no review
- `.pose/rules/backend-go.md`: validada aderência de handlers, contexto e erros.
- `.pose/rules/frontend-react.md`: validada acessibilidade e tratamento explícito de loading/erro.
- `.pose/rules/kubernetes.md`: validados resources/probes e imagem imutável.
- `.pose/rules/security.md`: validada ausência de segredos e revisão de autorização.
- `.pose/rules/documentation-style.md`: validada consistência editorial em docs/spec.

## Checks e evidências
- `check`: `./pose validate` (ok)
- `check`: `go test ./...` no módulo backend (ok)
- `check`: `pnpm lint && pnpm test` no módulo frontend (ok)
- `check`: `helm template` + `kubectl apply --dry-run=client` (ok)

## Contratos e compatibilidade
- Contrato HTTP `POST /v1/storage` preservado.
- Campo novo `retentionDays` adicionado como opcional (backward compatible).

## Achados
- Médio: falta alerta para saturação de fila (ação: incluir métrica e alerta antes de produção).
- Baixo: mensagem de erro frontend sem contexto do request-id (ação: ajustar UX observável).

## Riscos residuais
- Carga real do cluster não simulada em ambiente de review.
- Recomendar monitoramento reforçado nas primeiras 24h.
```

## Interpretação obrigatória de falhas de CI (POSE)

- Considere `POSE check (strict)` bloqueante para merge em branches principais (`main`).
- Considere `POSE validate (strict, required gate)` bloqueante quando houver falha em check `required`.
- Baixe e anexe evidências dos artefatos da pipeline:
  - log de check (`pose-check.log`)
  - log de validação (`pose-validate.latest.log`)
  - relatório gerado por `pose-report.sh`
- Classifique falhas de check `optional` como ressalva de qualidade e registre plano de saneamento.
- Reprove o PR quando a falha impedir compliance com spec, contrato público, segurança ou rollout seguro.

## Rollout faseado para módulos não prontos

- Aplique enforcement imediato apenas ao conjunto atual de checks `required` na matriz.
- Use `moduleOverrides` para modularizar adoção sem relaxar gates globais de `check` e `required`.
- Planeje promoção de checks `optional` para `required` por módulo com janela acordada e owner definido.

### Protocolo de promoção de check (optional -> required)

- Selecione domínio piloto e mapeie checks `optional` candidatos a `required` com owner e risco explícitos.
- Meça taxa de sucesso por 4 semanas e confirme baseline >= 95% antes da promoção.
- Altere classificação do check de `optional` para `required` somente no domínio elegível via `moduleOverrides` da matriz.
- Atualize a matriz de validação e a documentação da política de qualidade no mesmo change set.
- Monitore regressões nas semanas seguintes e ajuste rollout por domínio sem remover gates globais já `required`.
- Exija atualização de spec/rules quando o rollout alterar critérios de aceite de merge.

## Critérios de pronto

- Todos os achados críticos/altos estão resolvidos ou aceitos formalmente.
- Decisão final está clara e acionável para o próximo passo.
- Evidências sustentam conclusões de qualidade e segurança.
- Escopo foi respeitado sem deriva não justificada.

## Execução — modo revisor

**Objetivo:** avaliar qualidade técnica e prontidão de produção com foco em correção, risco e aderência ao escopo.

- **Foco:** correção funcional e consistência com specs; risco de regressão, segurança e operabilidade; qualidade e suficiência das validações executadas; clareza do feedback e decisão acionável.
- **Anti-padrões:** aprovar sem evidência de validação suficiente; focar apenas em estilo e ignorar risco funcional; solicitar mudanças fora do escopo sem justificativa; bloquear progresso por preferências subjetivas.
- **Handoff:** decisão explícita (aprovado / aprovado com ressalvas / reprovado); achados com severidade, evidência e ação esperada; condições para merge/deploy seguro; riscos aceitos e monitoramento recomendado.
