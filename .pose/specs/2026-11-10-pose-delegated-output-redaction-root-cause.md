---
slug: pose-delegated-output-redaction-root-cause
status: done
created_at: 2026-11-10
completed_at: 2026-11-12
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

Alterar a captura única de sessão, recuperar saída histórica descartada, mudar a policy do fornecedor ou ampliar a garantia dos eventos live.

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
- [x] Revisão independente e fechamento

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

Executar a matriz com `LC_ALL=C LANGUAGE=C`. A execução inicial passou 206 checks, mas um teste existente dependia do fallback ASCII e a suíte completa atingiu o orçamento de 600s enquanto havia outra suíte completa concorrente. O teste de locale passou isoladamente com `LC_ALL=C`; duas matrizes isoladas passaram em 418.0s e 416.8s. O aumento experimental do orçamento foi retirado do change set final: manter os limites originais de Go/runner, sem remover testes, asserções ou checks. Não executar outra suíte completa concorrente com a matriz. Cobrir também o canal stdout quando a expansão da redação excede o limite, sem preservar canais inseguros.

### Requirement trace

- R1 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure reproduz perfil histórico e atual; relatório de investigação registra o limite da evidência histórica.
- R2 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure test:TestDelegateTaskDiagnosesRedactionGrowthWithoutKeepingTheCapture test:TestDelegationCaptureFailureWithholdsPrivateErrorDetails
- R3 [satisfied] test:TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure falhou antes por ausência do diagnóstico e passou depois; os testes de tasks passaram (14.051s).

---

## 7. Final Report

### Delivered scope
Diagnóstico implementado: agente, formato de eventos, etapa e motivo seguro, com dica para regex Claude ampla. A recusa e o descarte da captura ambígua permanecem. A reprodução sintética compara o perfil histórico e o atual. Revisão independente aprovada no run `rrn-e0ff67e5650fa813`, atestação `rva-e5f68c7fb514f381`, bundle `rvb-dacc920864b44d67`; fechamento nativo aplicado em 2026-11-12.

### Validation executed

- `go test ./internal/cli -run 'TestDelegateTask|TestDelegate.*Session|TestDelegate.*Capture' -count=1`: passou (14.051s).
- `go test ./internal/pose -run 'DelegationCaptureFailure|ReviewCapture|Review.*Continu|DelegationEvents' -count=1`: passou.
- `go test ./internal/cli -run 'TestDelegateTaskExplainsTheLegacyClaudeSessionCaptureFailure|TestDelegateTaskDiagnosesRedactionGrowth' -count=1`: passou (0.345s), cobrindo perfil histórico/corrigido e falha por expansão no stdout com todos os canais retidos vazios.
- `LC_ALL=C LANGUAGE=C pose validate --strict --module pose-mcp --jobs 4 --json-out .pose/results/delivery-validation.json`: 208/208 checks passaram em 2026-11-12, incluindo suíte completa, build, vet e integrações. O resultado atual a ser selado é o do arquivo estruturado; a primeira execução verde precedeu a finalização editorial desta spec.
- `pose lint-spec pose-delegated-output-redaction-root-cause --strict`: passou.
- `pose review conclude rrn-e0ff67e5650fa813 --apply`: state recorded, decision approved; `pose review-check spec:pose-delegated-output-redaction-root-cause`: fresh=true, approved=true, different-vendor.
- `pose close spec:pose-delegated-output-redaction-root-cause`: exit 0, Closed. O material de código validado está no commit `1af0142e`; este relatório final registra o resultado posterior da revisão e da transição.
- `pose artifact-check --spec pose-delegated-output-redaction-root-cause --strict`: passou após corrigir os caminhos literais; 9 claims, 17 caminhos observados (incluindo saídas derivadas), sem erros.
- `pose assess tech-debt`: scan incompleto por duas specs históricas inválidas; nenhuma garantia de completude é inferida.
- A primeira matriz teve 206 passes, uma falha de locale e um timeout. A execução final acima substitui esses resultados para o fechamento, sem apagar as falhas históricas dos logs de execução.

### Residual risks

- O stream bruto da ocorrência histórica não existe mais; a investigação confirma o mecanismo com fixture sintética, não a composição exata daquele stream.

### Follow-ups

- [open] Fazer TestCheckRefusesAnIndexSetThatItsDescriptorDoesNotDescribe reconhecer o símbolo de warning tanto no perfil UTF-8 quanto no ASCII; hoje a matriz precisa de LC_ALL=C para esse teste existente. (owner:@pose-maintainers crit:low review:2027-02-12)
- [open] Distinguir ambiguidade de session/reset e padrões com múltiplos grupos antes de sugerir system/init, com testes dos casos; finding:rva-e5f68c7fb514f381/F1. (owner:@pose-maintainers crit:low review:2027-02-12)
- [open] Usar sentinels compartilhados ou testar todos os motivos a partir dos produtores reais, para que mudanças no texto não degradem o diagnóstico silenciosamente; finding:rva-e5f68c7fb514f381/F2. (owner:@pose-maintainers crit:low review:2027-02-12)
