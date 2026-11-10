---
slug: pose-worktree-runtime-qos-and-measurement
status: done
created_at: 2026-11-09
completed_at: 2026-11-10
supersedes:
depends_on: pose-worktree-runtime-resident-cache
remediates:
priority: 3
components: pose-mcp
task_type: feature
surface: minimal
delivers: capability:worktree-runtime-qos
---

# Spec: Admissão por faixas e medição do runtime opcional

## 1. Intent

### Goal

O runtime opcional protege as consultas curtas dos checks pesados com duas faixas de admissão, cada uma com seus slots e sua fila limitada, expõe o que admitiu e recusou, e é medido num repositório real com amostras, cenários e limiares fixados antes de qualquer número, com o ambiente declarado.

### Business value

Fatia 3 da B35 (emenda do ADR de 2026-11-09). Depois da fatia 2 o runtime rodava um comando por vez: um `surface-check` de 10 ms esperaria os 25 s de um `review-check`. Medido aqui: `surface-check` de uma spec leva cerca de 10 ms; `review-check` cerca de 25 s; `roadmap-check <slug>` cerca de 160 s (por isso é pesado, ao contrário do que a fatia anterior supunha).

### Constraints

- A saída de um comando hospedado continua sendo a do standalone, byte a byte, em todos os cenários.
- Recusa por fila cheia é antes de esperar e leva ao fallback standalone; nada é enfileirado sem limite.
- Os limiares abaixo foram fixados no teste de medição e nesta spec antes da primeira medição; não foram ajustados depois dela nem da segunda execução (que veio de lacunas no harness achadas pela revisão).

### Non-goals

- Limites de CPU, memória ou E/S por processo ou cgroup: o runtime limita o que admite (slots e fila) e o que guarda (o memo), não o que cada comando consome.
- Um scheduler ou fila durável: a fila é a espera pelos slots de uma conexão aberta.
- Valores por plataforma além do ambiente medido.

## 2. Requirements

- R6: A admissão tem duas faixas, `short` (`surface-check`) e `heavy` (os demais comandos hospedados), cada uma com slots e fila limitada configuráveis (`--short-slots`, `--short-queue`, `--heavy-slots`, `--heavy-queue`, `--memo-mib`; padrão 2/16 e 1/4, memo de 256 MiB); uma consulta curta não espera um check pesado; com todos os slots da faixa ocupados e a fila cheia o comando é recusado antes de esperar (e roda standalone); o status e o handshake trazem por faixa os slots, o limite da fila, os enfileirados, os em execução, os admitidos, os recusados e a espera total e máxima.
- R8: A medição roda por `POSE_RUNTIME_MEASURE=<arquivo> go test ./internal/cli -run TestMeasureTheRuntimeOnTheRealRepository`, com os cenários cold standalone ({3} amostras), fallback sem runtime (3), resident first e warm (1 + 5), restart seguido de run (3) e consultas curtas ociosas e sob um check pesado (10 cada), grava as figuras e o ambiente (CPUs, versões do Go, do Git e do kernel, o HEAD e o número de commits do repositório) e falha se um limiar não for cumprido: T1 a mediana warm é no máximo 0,60 da mediana cold; T2 a mediana após restart é no mínimo 0,80 da cold (nada é herdado); T3 o p95 de uma consulta curta sob carga é no máximo 3 vezes o p95 ocioso ou 250 ms, o que for maior; T4 o pico residente do runtime é no máximo 1536 MiB; T5 toda saída hospedada é idêntica à standalone; T6 tentar um runtime que não existe custa no máximo 500 ms a mais que o standalone. Os relatórios dizem o que é suportado, configurado e efetivo.

## 3. Technical Plan

### Artifacts

- created: .pose/starts/pose-worktree-runtime-qos-and-measurement.json
- created: .pose/changelogs/unreleased/pose-worktree-runtime-qos-and-measurement.md
- created: .pose/reports/2026-11-09-pose-worktree-runtime-measurements.md
- created: .pose/reports/2026-11-09-pose-worktree-runtime-measurements.json
- modified: .pose/specs/2026-11-09-pose-worktree-runtime-qos-and-measurement.md
- modified: .pose/indexes/validation-matrix.json
- created: pose-mcp/internal/cli/runtime_lanes.go
- created: pose-mcp/internal/cli/runtime_measure_test.go
- modified: pose-mcp/internal/cli/runtime_host.go
- modified: pose-mcp/internal/cli/runtime_host_test.go
- modified: pose-mcp/internal/cli/runtime_cmd.go
- modified: pose-mcp/internal/cli/runtime_cmd_test.go
- modified: pose-mcp/internal/worktreeruntime/runtime.go
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:worktree-runtime-qos module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/cli.go

### API/contract changes

`pose runtime start|serve` ganha `--heavy-slots`, `--short-slots`, `--heavy-queue`, `--short-queue` e `--memo-mib`; as respostas de status trazem `metrics`.

## 4. Tasks

- [x] Faixas de admissão com fila limitada e métricas
- [x] Flags de admissão em `start`/`serve` e repasse ao `serve` destacado
- [x] Harness de medição com amostras, cenários e limiares fixados
- [x] Rodar a medição e registrar o relatório
- [x] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R6, R8
- Minimal option: manter um comando por vez e medir
- Selected option: duas faixas com slots e fila limitada, por comando conhecido (curto ou pesado)
- Rationale: a consulta curta de 10 ms não deve esperar o check de 25 s; os comandos são seguros em paralelo como o são em processos separados (as escritas de derivados são atômicas, spec `pose-writer-coordination-and-atomic-publication`); a classificação é uma lista declarada e medida, não uma heurística
- Consequences: um comando novo é pesado até ser medido e listado; os padrões (1 pesado, 2 curtos) foram escolhidos para a máquina medida e são configuráveis
- Falsifier: uma consulta curta que espera um check pesado, ou uma fila que cresce além do limite

## 6. Validation

### Strategy

Testes com um comando lento injetado provam o isolamento das faixas, a recusa por fila cheia e as métricas; a medição no repositório real é o teste opt-in, rodada uma vez com o resultado no relatório.

### Deterministic checks

`go test -race` dos testes do rastreio; a matriz estrita em clone limpo; a medição opt-in registrada em `.pose/reports/`.

### Requirement trace
- R6 [satisfied] test:TestAShortQueryIsNotQueuedBehindAHeavyCheckAndAFullLaneRefuses check:runtime-qos-integration
- R8 [satisfied] test:TestMeasureTheRuntimeOnTheRealRepository

### Known gaps
- A medição é de um ambiente (declarado no relatório) e de um repositório; não vale para outros tamanhos de histórico.
- Dois comandos que chegam no mesmo instante ao último slot livre podem ambos esperar: a fila pode passar do limite pelos chegados daquele instante.

---

## 7. Final Report

### Delivered scope
Duas faixas de admissão com slots e fila limitada, métricas por faixa no status, as flags de configuração (R6) e o harness de medição com limiares fixados antes dos números (R8); o relatório (`.pose/reports/2026-11-09-pose-worktree-runtime-measurements.md`) registra duas execuções (a revisão achou lacunas no harness depois da primeira; os limiares não mudaram) e os números da segunda: warm 0,42 da cold (T1), restart 1,01 da cold (T2), consulta curta sob carga p95 9 ms contra 11 ms ocioso (T3), pico residente 535 MiB (T4), saída idêntica incluindo a consulta curta (T5) e custo do fallback de 102 ms (T6); todos os limiares passaram nas duas execuções.

### Files and modules changed
- pose-mcp/internal/cli (faixas, flags, harness), worktreeruntime (métricas), os quatro POSE.md; a matriz ganha `runtime-qos-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado; a medição opt-in registrada no relatório
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Sem limite de CPU/memória por comando: um check pesado ocupa os recursos que ele ocupa standalone.
- Os padrões de slots são os da máquina medida.

### Follow-ups

- [open] Dar a `memo_hits` o valor do próprio comando com faixas concorrentes (hoje é a diferença de um contador global) e provar a concorrência com trabalho curto real ao lado de um pesado, não só com `surface-check --help`. (owner:@pose-maintainers crit:low review:2027-02-08)
- [open] Rotear os checks para o runtime quando ele existe e limites por cgroup quando a plataforma os oferecer. (owner:@pose-maintainers crit:low review:2027-02-08)
