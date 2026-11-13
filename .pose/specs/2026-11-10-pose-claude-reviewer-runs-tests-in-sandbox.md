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
- [ ] Revisão independente e fechamento

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

### Planned coverage

- R1: testes `TestSandboxedReview` e qualificação real em `.pose/reports/claude-review-sandbox-qualification.md`: fresh e forma retomada passaram, após duas falhas preservadas.
- R2: testes de dependência em fresh/resume e failIfUnavailable passaram; jq é dependência adicional para codificar caminhos absolutos sem interpolação insegura.
- R3: review governado com reexecução do rastreio; ainda não executado.
- R4: testes de projeção, continuidade e compatibilidade passaram, incluindo regressão do read restrito e suite de policy.

2026-11-12: regressão de seleção falhou antes por `unknown field review_profile` e passou após a implementação. Testes focados de sandbox/policy/read restrito passaram (0.569s). Qualificação fresh `trn-43c6174032cb0775` e continuação nativa `trn-a1c0193704f5f055` passaram em Linux; os testes executaram a tentativa de escrita em subprocesso, com recusa do SO. O scan de dívida é incompleto pelas mesmas duas specs históricas inválidas, com dois marcadores e um descoberto sem cobertura. Matrizes do implementador passaram 209/209, a última em 390s. O run governado rrn-41cc27dbae7fdd47 reexecutou os oito testes então existentes e vet, mas foi recusado por mutação de diretórios e limpeza incompleta; suas críticas F1/F2/F3 motivaram a negação do Git comum, qualificação em worktree real e recusa de jq ausente no teste. A nova versão com staging controlado passou nove testes focados (0.489s) e qualificação real de Git comum trn-8d24f721bc87ba11. Ainda requer matriz e revisão frescas.

---

## 7. Final Report

### Delivered scope
Implementado e verificado com testes focados e qualificação sintética real no Linux. A matriz completa, a reexecução do rastreio pela revisão governada e o fechamento ainda não ocorreram. Relatório de qualificação preserva as duas falhas anteriores, o limite do sandbox de shell e a ausência de qualificação de outros sistemas.
