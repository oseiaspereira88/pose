---
name: pose-test-plan
description: Use to definir plano de teste explícito ANTES de implementar mudanças de risco médio/alto, contrato sensível ou impacto cross-service — define escopo por camada, cenários negativos, comandos determinísticos e evidência esperada. Trigger keywords - test plan, plano de teste, risk-based testing, regression strategy, contract test, cross-service, e2e plan.
when_to_use: A tarefa tem risco médio/alto (criticalidade ≥ high no module-metadata), toca contrato HTTP/schema/eventos, ou afeta múltiplos serviços. Use ANTES de codar to alinhar critério de aceite verificável e evitar "testei localmente".
---

# Skill: pose-test-plan

Fluxo POSE to construir plano de teste risk-based antes da implementação.

## Required reading

1. [`.pose/workflows/feature.md`](../../../.pose/workflows/feature.md) ou [`bugfix.md`](../../../.pose/workflows/bugfix.md), conforme o tipo.
2. Rules de domínio aplicáveis.
3. [`.pose/indexes/validation-matrix.json`](../../../.pose/indexes/validation-matrix.json) — checks já declarados to o módulo afetado.
4. [`.pose/indexes/module-metadata.json`](../../../.pose/indexes/module-metadata.json) — criticality e validationProfile do módulo.

## Steps

1. Identificar módulo(s) afetado(s) e nível de risco real (consulte `module-metadata.json` → `criticality`).
2. Definir escopo por camada with base no risco:
   - **unit** (sempre): comportamento isolado da unidade alterada.
   - **integração/contrato** (médio+): boundary entre módulos, schema/HTTP.
   - **e2e/smoke** (alto+): fluxo end-to-end no caminho crítico.
3. Mapear cenários negativos e fallbacks:
   - Input inválido, autorização negada, timeout, dependência indisponível.
   - Para cada cenário: comportamento esperado documentado.
4. Listar comandos determinísticos por camada, separando obrigatórios vs. opcionais to o risco atual:
   ```bash
   # Reusar o que já está na matriz:
   ./pose validate --module <path> --report --report-task test-plan-baseline-<slug>
   ```
5. Definir critério de evidência esperada to cada comando (output, métrica, schema).
6. Anexar o plano à seção `Validation` da spec antes de iniciar implementação.
7. Atualizar [`validation-matrix.json`](../../../.pose/indexes/validation-matrix.json) se a tarefa justifica adicionar/promover check ao módulo (caso novo cenário deva virar gate permanente). Após editar a matrix:
   ```bash
   ./pose check --strict  # valida schema da matrix
   ```

## Output requirements

- Plano em `Validation` da spec with 3 colunas: cenário, comando, evidência esperada.
- Cenários negativos cobertos explicitamente (não apenas happy path).
- Comandos copy-pasteable, without placeholders abstratos.
- Marcação clara de obrigatório vs. optional to o risco corrente.
- Eventual atualização de `validation-matrix.json` with schema válido.
