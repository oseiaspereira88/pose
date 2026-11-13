---
slug: pose-claude-reviewer-runs-tests-in-sandbox
status: in-progress
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-claude-sandboxed-profile
remediates:
priority: 2
components: pose-mcp
task_type: feature
delivers: capability:claude-sandboxed-review
---

# Spec: O revisor Claude roda go test dentro do sandbox

## 1. Intent

### Goal

O perfil de revisão do Claude permite `go test` e `go vet` dentro do sandbox do sistema operacional, recusa-se a iniciar quando o sandbox não é imposto e preserva o que o perfil restrito faz hoje.

### Business value

O perfil de revisão do Claude teve de abrir mão de `go test`, `go vet`, `git log|diff|show` e `lint-spec` porque uma lista por prefixo não exclui flags que gravam ou executam; a spec `pose-claude-sandboxed-profile` deu ao Claude um perfil `sandboxed` imposto pelo sistema operacional, com checagem de dependências, mas a revisão ainda é read.

### Constraints

Preservar o perfil restrito `read`, a integridade do checkout selado, a prova de execução, os budgets e a limpeza existente. Qualificar Linux com Claude real; não atribuir isolamento ao processo confiável do adaptador. Não usar fallback sem sandbox.

### Non-goals

Sandbox de todo o processo Claude, sigilo de todo o filesystem, mudança de adjudicação/smoke/tasks, instalação automática de dependências ou adoção automática em outras instâncias.

## 2. Requirements

- R1: Um perfil de revisão com sandbox permite os comandos de teste e leitura e nega gravação fora do diretório do run, medido por uma tentativa real.
- R2: O perfil não inicia sem bubblewrap e socat, com a dica de instalação.
- R3: Uma revisão com o novo perfil reexecuta os testes do rastreio da spec.
- R4: Seleção é exclusiva de review, mantém compatibilidade na ausência do perfil e aplica a mesma contenção na retomada; comandos inválidos são recusados antes da execução.

## 3. Technical Plan

Adicionar `review_profile.command` e `review_profile.resume_command` opcionais por agente. Projetar somente no adaptador de review; clonar Resume antes de substituir seu comando. A policy atual adota o perfil Claude com sandbox obrigatório e Bash sem Edit/Write. O wrapper deriva scratch privado do output do motor, configura Go/TMPDIR e nega escrita no cwd selado e no Git comum físico. O motor prepara apenas diretórios vazios de staging Claude no checkout descartável antes do snapshot; arquivos continuam sujeitos à mesma checagem. A limpeza Go fica dentro do sandbox. O módulo é de criticidade high. Assessment de 2026-11-12: 114333 LOC de produção, 109213 de testes, um TODO; conteúdo reaproveitado. knowledge:delegated-conclusions-round4 orienta preservar recusas e provas, sem transformar task em atestação.

### Artifacts

- modified: .pose/specs/2026-11-10-pose-claude-reviewer-runs-tests-in-sandbox.md
- created: .pose/adr/2026-11-12-sandboxed-claude-review-commands.md
- created: .pose/starts/pose-claude-reviewer-runs-tests-in-sandbox.json
- modified: .pose/policy/delegation.json
- modified: .pose/indexes/validation-matrix.json
- modified: pose-mcp/internal/pose/delegation_policy.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/delegation_events.go
- created: pose-mcp/internal/pose/claude_sandboxed_review_test.go
- created: .pose/reports/claude-review-sandbox-qualification.md
- created: .pose/changelogs/unreleased/pose-claude-reviewer-runs-tests-in-sandbox.md
- modified: POSE.md
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md

### Delivery targets

- capability:claude-sandboxed-review module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/cli/delegate.go

### API/contract changes

Schema 2 ganha `review_profile` opt-in. Perfil ausente não muda o adaptador; parsers antigos estritos recusam a chave. Review continua com acesso read ao sujeito e scratch descartável para execução. Não ampliar as capacidades declaradas do backend trusted-process.

O codec Claude configurado canonicaliza somente init tipado repetido para a mesma identidade capturada, antes da classificação da sessão; identidades diferentes e regex genérica ambígua continuam recusadas. Bytes originais continuam usados em proveniência, redação e reconhecimento de limites/reset. A ADR registra o gatilho e o owner da requalificação quando o fornecedor muda.

## 4. Tasks

- [x] Completar a spec (artefatos, alvos de entrega, estratégia de validação) e iniciar
- [x] Implementar e testar
- [x] Revisão independente
- [ ] Fechamento nativo

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4
- Selected option: perfil específico de review, conforme ADR `2026-11-12-sandboxed-claude-review-commands`.
- Minimal option: ampliar a allow-list do perfil restrito.
- Rationale: flags executáveis de Go não podem ser isoladas por prefixo.
- Consequences: sandbox e dependências são obrigatórios, inclusive no resume; testes que gravam no source falham.
- Falsifier: uma execução selecionada escrever fora do scratch autorizado ou voltar a executar sem sandbox.

## 6. Validation

Executar obrigatoriamente as camadas abaixo, em série nas operações que capturam Git/índices. Fakes comprovam seleção/recusa, nunca isolamento do SO. Rollback: retirar `review_profile` explicitamente da policy e reverter a extensão opcional; não reescrever runs nem tratar revisão restrita como reexecução de testes.

| Cenário | Comando obrigatório | Evidência esperada |
| --- | --- | --- |
| Seleção review, compatibilidade task/adjudicate/smoke e resume sem mutação | `LC_ALL=C go -C pose-mcp test ./internal/pose -run 'TestSandboxedReview' -count=1` | Seleção correta, ausência preservada, inválidos recusados e ambos comandos protegidos |
| bwrap ausente, socat ausente, ambos ausentes e falha de inicialização | mesmo comando | Sem Claude iniciado ou fallback; exit e dica de instalação |
| Linux real, scratch permitido, source/sentinel/Git recusados, subprocessos | `pose delegate task /tmp/pose-sandbox-qualification-prompt.md --agent claude-opus-5-5 --access read --cd /tmp/pose-sandbox-qualification --json` em instância descartável com o comando de revisão projetado | Tool calls reais e erros do SO; hashes preservados; relatório público sanitizado |
| Review governado reexecuta rastreio e vet | `pose delegate review spec:pose-claude-reviewer-runs-tests-in-sandbox --agent claude-opus-5-5 --json` após bundle/evidência válidos | Run nativo com comandos, resultados e conclusão independente; sem alterações no checkout |
| Build, suite completa, vet e integração | `LC_ALL=C LANGUAGE=C pose validate --strict --module pose-mcp --jobs 4 --json-out .pose/results/delivery-validation.json` | Todos os checks obrigatórios passam; não rodar suite completa concorrente |
| Segurança e governança | `govulncheck ./...` no módulo; `pose lint-spec pose-claude-reviewer-runs-tests-in-sandbox --strict`; `pose artifact-check --spec pose-claude-reviewer-runs-tests-in-sandbox --strict`; `pose surface-check --spec pose-claude-reviewer-runs-tests-in-sandbox --strict` | Sem vulnerabilidade alcançável nova, diff sem segredos, claims reconciliados e composição comprovada |

### Requirement trace

- R1 [satisfied] test:TestSandboxedReviewClaudeEnforcesFreshAndResumedShellBoundaries test:TestSandboxedReviewStagingIsMaterializedBeforeIntegrityAndFilesStillCount check:claude-sandboxed-review-integration report:claude-review-sandbox-qualification
- R2 [satisfied] test:TestSandboxedReviewDependencyFailuresNeverLaunchClaude test:TestSandboxedReviewWrapperKeepsScratchPrivateAndPropagatesStartupFailure check:claude-sandboxed-review-integration
- R3 [satisfied] check:claude-sandboxed-review-integration report:claude-review-sandbox-qualification
- R4 [satisfied] test:TestSandboxedReviewSelectsOnlyReviewAndKeepsResumeConfiguration test:TestSandboxedReviewOmissionPreservesLegacyAdapter test:TestSandboxedReviewRefusesIncompleteOrUnsafeCommands test:TestSandboxedReviewRepeatedTypedInitKeepsOneIdentityButRejectsConflicts test:TestSandboxedReviewSessionCodecPreservesUnambiguousChannelsExactly check:claude-sandboxed-review-integration

2026-11-13: matriz do implementador 209/209 PASS em 372,6s sobre o código 37a96344, selada no bundle rvb-6b28e970d696e950. O run governado rrn-c65b2904a61d6055 terminou com captura e limpeza completas e decisão approved: Claude Opus 5.5 (Anthropic), independente da implementação OpenAI, reexecutou os 11 testes TestSandboxedReview (0.094s) e go vet (exit 0), confirmou por probes não mutantes a recusa de escrita no source, Git comum e gitdir do worktree, e passou artifact-check e surface-check. A suite completa dentro do sandbox apresentou falhas de ambiente/ownership/rede e timeouts; sua aprovação da matriz completa cita a evidência selada do host, sem alegar execução integral bem-sucedida no sandbox. Govulncheck: nenhuma vulnerabilidade alcançável; dois avisos em módulos requeridos não chamados. O scan de dívida permanece incompleto por duas specs históricas inválidas. Runs anteriores recusados e suas evidências estão preservados.

---

## 7. Final Report

### Delivered scope

Perfil Claude exclusivo de review, com sandbox de shell obrigatório, scratch privado, negação de escrita no checkout e Git comum físico, mesmas proteções no resume e recusa explícita sem dependências. Staging vazio é preparado antes do snapshot; arquivos continuam contados pela integridade. A captura aceita init tipado repetido da mesma sessão, preservando bytes originais para proveniência e redação. Qualificação real Linux, testes determinísticos e revisão governada aprovados. Não há alegação de isolamento do processo Claude inteiro nem qualificação de outras plataformas.

### Review

Bundle rvb-6b28e970d696e950; run rrn-c65b2904a61d6055; attempt rat-26c766961e9166a238e29681bf37d434ef3df7956743baff7e6887e074727e06. Reviewer claude-opus-5-5, Anthropic; implementação OpenAI. Decisão approved, com três riscos baixos aceitos pelo revisor. Assinatura do executor b34-local-executor não é recibo criptográfico do vendor.

### Follow-ups

- [open] F1: derivar o staging Claude de um sinal explícito do adaptador projetado no DispatchPlan, vinculado à proveniência, em vez de reler a policy no dispatch; o comportamento atual falha fechado. (owner:@pose-maintainers crit:low review:2026-12-13)
- [open] F2: usar erro sentinel e errors.Is para ambiguidade de captura, evitando dependência da mensagem textual; alteração dessa mensagem hoje volta a recusar captura. (owner:@pose-maintainers crit:low review:2026-12-13)
- [open] F3: adicionar regressão e documentação para IDs init conflitantes em canal originalmente não ambíguo, fixando a recusa mais estrita já implementada. (owner:@pose-maintainers crit:low review:2026-12-13)
