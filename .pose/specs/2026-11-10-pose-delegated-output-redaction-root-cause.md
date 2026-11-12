---
slug: pose-delegated-output-redaction-root-cause
status: in-progress
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-delegate-picker-and-claude-runs
remediates:
priority: 2
components: pose-mcp
task_type: bugfix
delivers: governance:delegated-output-redaction-diagnostics
---

# Spec: Causa raiz do 'agent's output could not be redacted' nas tarefas Claude

## 1. Intent

### Goal

Uma tarefa delegada ao Claude no harne8 não falha mais com 'agent's output could not be redacted', e a falha, quando ocorre, nomeia o perfil e o motivo.

### Business value

No harne8 o perfil antigo do Claude produzia essa falha e foi contornado trocando-o pelo do repositório do motor; a causa raiz na redação nunca foi isolada, e a mensagem não diz qual perfil falhou.

### Constraints

Preservar limites de bytes, retenção privada e recusa de sessão ambígua. Não publicar texto bruto, regex, credenciais, IDs de sessão ou erros de filesystem no diagnóstico.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Reproduzir a falha com o perfil antigo e isolar o ponto da redação que a produz.
- R2: A redação trata a saída que a causou sem descartar o registro; ou a falha nomeia o perfil, o evento e o motivo.
- R3: Um teste com a saída que a causou falha antes e passa depois.

## 3. Technical Plan

O perfil anterior à correção harne8 `5752ec08` usa uma regex global de `session_id`; o perfil atual restringe a captura a `system/init`. Reproduzir ambos com o mesmo stream sintético, sem consumir quota do fornecedor. A nota knowledge:delegation-observation-round4 registra a diferença entre observação live e retenção final; esta correção não amplia a garantia live.

### Artifacts

- modified: .pose/specs/2026-11-10-pose-delegated-output-redaction-root-cause.md
- modified: pose-mcp/internal/pose/delegation_task.go
- created: pose-mcp/internal/pose/delegation_capture_failure.go
- created: pose-mcp/internal/pose/delegation_capture_failure_test.go
- created: pose-mcp/internal/cli/delegate_redaction_failure_test.go
- created: .pose/investigations/pose-delegated-output-redaction-root-cause/2026-11-12-findings.md
- created: .pose/changelogs/unreleased/pose-delegated-output-redaction-root-cause.md
- created: .pose/starts/pose-delegated-output-redaction-root-cause.json
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:delegated-output-redaction-diagnostics module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

## 4. Tasks

- [x] Completar a spec (artefatos, alvos de entrega, estratégia de validação) e iniciar
- [x] Implementar e testar
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Selected option: Manter a captura única por canal e diagnosticar com vocabulário fechado.
- Minimal option: Manter a mensagem genérica.
- Rationale: Uma captura ambígua não pode determinar com segurança a sessão a ocultar; o perfil atual já elimina a ambiguidade.
- Consequences: Regex ampla continua recusada; erros desconhecidos não expõem detalhes privados.
- Falsifier: A mensagem voltar a ocultar a causa reproduzida ou expor dados privados.

## 6. Validation

Executar primeiro o teste de regressão contra o código anterior; ele deve falhar por ausência de diagnóstico. Depois executar o mesmo teste, os testes de tasks/continuidade/eventos e a matriz strict de `pose-mcp` (build, testes completos, vet e integrações). Conferir lint-spec, artifact-check, revisão independente e closeout-check antes de fechar. Rollback: reverter apenas a mensagem e sua classificação; a política de retenção não muda.

Executar a matriz com `LC_ALL=C LANGUAGE=C`. A execução inicial passou 206 checks, mas um teste existente dependia do fallback ASCII e a suíte completa atingiu o orçamento de 600s. O teste de locale passou isoladamente com `LC_ALL=C`. O orçamento da suíte completa passa a 1800s, tanto no runner quanto no Go; nenhum teste, asserção ou check é removido. Não executar outra suíte completa concorrente com a matriz.

### Requirement trace

- R1 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure reproduz perfil histórico e atual; relatório de investigação registra o limite da evidência histórica.
- R2 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure test:TestDelegationCaptureFailureWithholdsPrivateErrorDetails
- R3 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure falhou antes por ausência do diagnóstico e passou depois; os testes de tasks passaram (14.051s).

---

## 7. Final Report

### Delivered scope
Diagnóstico implementado e testes focados passando. Fechamento e revisão independente pendentes.
