---
slug: pose-delegated-adapter-containment
status: done
created_at: 2026-10-13
completed_at: 2026-10-26
depends_on: pose-git-and-dispatch-snapshot-integrity, pose-cli-mcp-operation-results, pose-delegated-review-dispatch
priority: 1
components: pose-mcp
task_type: bugfix
surface: minimal
delivers: capability:delegated-adapter-containment
---

# Spec: Delimitar e verificar contenção, recursos e retenção do adapter

## 1. Intent

### Goal

Executar somente adapters compatíveis com a política de isolamento e limites declarados, divulgando garantias reais por plataforma sem chamar worktree de sandbox.

### Business value

B18 / D06 e T3-06. Consultar o [plano autoritativo](../results/pose-implementation-plan.json) e a [análise de delegação](../reports/2026-10-11-pose-front-03-delegated-review.md). Completar os mecanismos existentes sem duplicar as fundações done.

### Constraints

Implementação autorizada após o merge de B17 em `a2c0237add2adc789d1a338536713ccfd5e0b33f`. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Definir perfil/policy de execução com capacidades exigidas e observadas de filesystem/raiz original, metadados Git compartilhados, rede, ambiente/segredos, filhos e recursos. Worktree é cópia de trabalho, não sandbox; verificação antes/depois de B07 detecta alterações pertinentes, sem impedir acesso externo ou provar isolamento de credenciais.
- R2: Validar schema/configuração antes de spawn: argv/executor/identidade não vazios, duração positiva e limites de runs/streams/output/memória/retention com unidades e ranges explícitos. Valores negativos/overflow/whitespace/incompatíveis devem recusar execução; zero deve ter significado admitido explícito, nunca desligar limite silenciosamente. Defaults/ausência/legacy exigem compatibilidade documentada, sem converter budget zero em ilimitado.
- R3: Resolver capacidades do adapter/backend/plataforma contra a policy efetiva antes do efeito. Quando a policy exige contenção não demonstrada ou indisponível, recusar com unsupported/unavailable e orientação; não anunciar garantia só porque o comando compila, o adapter a declara ou o processo está em worktree. Modo de adapter confiável deve divulgar seus limites e não dispensar checks de integridade.
- R4: Usar ambiente mínimo ou allowlist explícita, escopo de credenciais autorizado e representação argv que preserve argumentos sem avaliação de conteúdo de brief/notas como shell. Distinguir diretório descartável, paths de output e raízes proibidas; validar confinamento/symlinks quando aplicável. Não copiar segredos/chaves ao brief/transcript ou registrar valores sensíveis como diagnóstico.
- R5: Limitar stdout/stderr, transcript e arquivo de conclusão na captura/leitura, incluindo streams simultâneos e arquivos externos/special/symlinks. Limite apenas no tail persistido não basta para memória; overflow/truncamento pertinente deve produzir falha/incomplete e nunca conclusão validada. Documentar se memória/CPU são limites aplicados pelo backend ou indisponíveis, com recusa quando exigidos.
- R6: Propagar deadline/cancelamento e encerrar o conjunto de processos coberto pelo mecanismo efetivamente qualificado; limitar espera de pipes/descendentes, detectar falhas de término e não retornar sucesso fictício. Unix process group não prova controle de filho que escapou do grupo; Windows deve ter mecanismo verificado ou declarar unsupported onde a policy exige contenção de descendentes.
- R7: Cleanup deve ser limitado, idempotente e confinado aos recursos pertencentes à operação, preservando outputs/evidências necessários e leitores vivos. Falha de término/remoção deve aparecer no outcome/receipt de B12, separada da conclusão de domínio; não apagar paths arbitrários ou ocultar orphan que ainda executa. Não reexecutar spawn automaticamente para compensar cleanup.
- R8: Definir acesso/permissões, retenção/bytes/TTL e exportação dos artefatos, com ordem de redaction/canonicalização/digests/assinatura explícita para verificação futura. Preservar inventário, orçamento e rejeições do ledger B16 no fluxo integrado; retenção não pode apagar prova conhecida para restaurar cota/aprovação. Conteúdo removido por policy deve ter receipt/limitação, sem promover evidência ausente a válida.
- R9: Divulgar em CLI/MCP e registros profile/capacidades observadas, falha de setup/limite/cleanup, plataforma, coverage e limitations conforme B12/B07. Preservar comportamento de adapters/instâncias legadas com classificação de compatibilidade e opt-in deliberados; não ativar novo backend ou substituir policy por esta entrega.
- R10: Qualificar configurações inválidas, escrita externa controlada, Git compartilhado, ambiente/segredos sintéticos, streams/arquivos grandes, deadline/cancelamento/descendentes e retention/cleanup em fixtures isoladas por plataforma. Teste que não consegue aplicar restrição deve declarar limitação; nenhum fornecedor real, sandbox ou suporte Windows é considerado entregue por lint ou por teste Unix.

## 3. Technical Plan

### Artifacts

Declarações reconciliadas antes do start; evidência Git será verificada separadamente.

- modified: .pose/specs/2026-10-13-pose-delegated-adapter-containment.md
- modified: pose-mcp/internal/pose/review_run_continuity.go
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_dispatch_unix.go
- modified: pose-mcp/internal/pose/review_dispatch_windows.go
- modified: pose-mcp/internal/pose/review_attempt_ledger.go
- modified: pose-mcp/internal/pose/review_run_provenance_test.go
- modified: pose-mcp/internal/pose/review_run_provenance.go
- modified: pose-mcp/internal/cli/review_closeout.go
- modified: pose-mcp/internal/cli/review_run_continuity.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- modified: .pose/assessments/README.md
- modified: .pose/assessments/consolidated.md
- modified: .pose/assessments/integrations.md
- modified: .pose/assessments/technical-debt.md
- modified: .pose/state/integrations.json
- modified: .pose/state/technical-debt.json
- modified: .pose/indexes/validation-matrix.json
- modified: locales/pt-BR/POSE.md
- modified: pose-mcp/internal/scaffold/dist/POSE.md
- modified: pose-mcp/internal/scaffold/dist/locales/pt-BR/POSE.md
- modified: POSE.md
- created: .pose/starts/pose-delegated-adapter-containment.json
- created: .pose/changelogs/unreleased/pose-delegated-adapter-containment.md
- created: .pose/adr/2026-10-25-bound-trusted-reviewer-execution-and-disclose-unavailable-isolation.md
- created: pose-mcp/internal/pose/review_adapter_containment.go
- created: pose-mcp/internal/pose/review_adapter_containment_regression_test.go
- created: pose-mcp/internal/pose/review_adapter_containment_test.go
- created: pose-mcp/internal/cli/review_adapter_containment_test.go
- created: pose-mcp/internal/mcpserver/review_adapter_containment_test.go
- created: .pose/reports/2026-10-26-b18-adapter-containment.md

### Delivery targets

- capability:delegated-adapter-containment module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### Approach and execution gates

Executar B07/B12 antes do delta. Inventariar backends/plataformas realmente suportados e comparar com as garantias do ADR aceito de delegação; registrar extensão estrutural prospectiva quando necessária, sem enfraquecer unilateralmente a integridade da cópia/repo. Reutilizar timeout/process group existente e delimitar seu alcance; escolher mecanismos de isolamento/credenciais e controle de filhos por perfil qualificado, sem impor fornecedor ou container obrigatório a toda instância. Conferir callers CLI/MCP, paths, configurações e produtores de evidência antes de start. Este delta é independente de B16 na matriz: valida policy/limites do adapter, enquanto reserva/cota durável fica em B16 e prova/resultado em B17/B19. Integrar com esses contratos sem duplicar ledger ou issuer.

### Implementation plan

Reutilizar B07/B12/B16/B17, `knowledge:pose-b07-implementation` e a decisão [trusted-process](../adr/2026-10-25-bound-trusted-reviewer-execution-and-disclose-unavailable-isolation.md). Não criar sandbox ou issuer paralelo. Resolver policy e capacidades antes de reserva/spawn; capturar streams e output com limites; registrar cleanup no OperationResult dentro do run imutável antes de publicá-lo. Preservar snapshot/digests e quota. Não apagar evidência automaticamente.

## 4. Tasks

- [x] Resolver perfil/compatibilidade e validar configuração antes do efeito.
- [x] Aplicar limites de captura, ambiente e leitura; qualificar término/cleanup.
- [x] Expor a mesma evidência em registros, CLI e MCP; documentar retenção.
- [x] Executar regressões, matriz canônica e revisão separada do código; preparar closeout nativo.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Exigir compatibilidade verificável com a policy do adapter e divulgar limites por plataforma, com captura e cleanup limitados.
- Rationale: Execução em worktree e exit zero não provam contenção ou conclusão válida; exigir garantias verificáveis impede anunciar proteção que o adapter não entrega.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Plano V1–V8 definido antes do código; regressões reais correspondentes constam no trace. Validação canônica final no commit `93bd352`: 156/156 checks strict em 254.6s, incluindo `delegated-adapter-containment-integration`, no caminho de policy `.pose/results/delivery-validation.json`. Race final passou nos três consumidores; govulncheck e secret scan sem achados. [Relatório](../reports/2026-10-26-b18-adapter-containment.md).

| Caso planejado | Resultado esperado |
|---|---|
| V1 — Timeout/cota/bytes/retention negativos, overflow, zero e whitespace | Recusa ou significado zero explícito; nenhum limite desabilitado silenciosamente. |
| V2 — Policy exige filesystem/rede/segredos/descendentes sem backend apto | Unsupported/unavailable antes de spawn; worktree não divulgada como sandbox. |
| V3 — Escrita externa sintética, Git compartilhado, env allowlist e argv hostil | Garantia aplicada ou limitação fiel; sem segredo em brief/log, sem notas como comando. |
| V4 — Streams paralelos, output grande/truncado/special/symlink | Capture/leitura e espera limitadas; nenhum resultado truncado validado. |
| V5 — Deadline/cancelamento, filhos no grupo/fora dele e Windows | Término comprovado no perfil qualificado ou falha/unsupported explícitos. |
| V6 — Cleanup falho/retry, recursos alheios e leitor vivo | Outcome fiel, idempotência e confinamento; sem apagar evidência ou repetir spawn. |
| V7 — Redaction/assinatura/retention e histórico negativo conhecido | Prova permanece verificável ou limitação explícita; cota/rejeição não restauradas. |
| V8 — CLI/MCP, legado e perfil Unix/Windows | Paridade e compatibilidade delimitadas; aprovação/documentação não inventam suporte. |

### Test plan (required, before implementation)

Risco alto: execução de processo, dados sensíveis e evidência assinada. Usar apenas executáveis sintéticos em repositórios temporários. Testes de integração devem passar pelo dispatch real, ledger e consumidores CLI/MCP; nenhum vendor real será chamado. Linux qualifica grupo de processos, Windows somente compilação e recusa de garantia não disponível.

| Camada | Comando no módulo pose-mcp | Evidência exigida |
|---|---|---|
| Unit/integration | `go test ./internal/pose ./internal/cli ./internal/mcpserver -run 'TestAdapterContainment' -count=1` | Config inválida/zero/overflow, capacidades indisponíveis, allowlist e redaction, output regular limitado, timeout/grupo/cleanup e paridade. |
| Concorrência | `go test -race ./internal/pose -run 'TestAdapterContainment' -count=1` | Captura simultânea sem corrida nem crescimento ilimitado. |
| Regressão | `go test ./...` e `go vet ./...` | B07/B12/B16/B17 e legado preservados. |
| Build | `go build ./...` e `GOOS=windows GOARCH=amd64 go build ./...` | Build local e cross-build sem alegar runtime Windows. |
| Segurança | `govulncheck ./...` e secret scan do diff | Nenhuma vulnerabilidade aplicável ou segredo introduzido. |
| Governança | `pose validate --strict --json-out .pose/results/delivery-validation.json` | Todos os produtores required atuais passam; integration qualifica target composto. |

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

- R1 [satisfied] test:TestAdapterContainmentConfiguration test:TestAdapterContainmentControlledOutsideWriteIsNotIsolation
- R2 [satisfied] test:TestAdapterContainmentConfiguration test:TestAdapterContainmentCLIDenialBeforeEffectAndBoundedNotes
- R3 [satisfied] test:TestAdapterContainmentConfiguration test:TestAdapterContainmentTimeoutAndChildren test:TestAdapterContainmentEscapedFiniteChildIsDisclosedNotContained
- R4 [satisfied] test:TestAdapterContainmentEnvironmentAndSignedRetention test:TestAdapterContainmentNotesAndArgv test:TestAdapterContainmentOverlappingSessionDoesNotExposeCredentialSuffix test:TestAdapterContainmentCredentialSessionResumesWithPrivateOriginal
- R5 [satisfied] test:TestAdapterContainmentConcurrentCapture test:TestAdapterContainmentUnsafeAndOversizedOutput test:TestAdapterContainmentOverflowWithholdsCredentialPrefix test:TestAdapterContainmentShortSessionExpansionCannotGovern
- R6 [satisfied] test:TestAdapterContainmentTimeoutAndChildren test:TestAdapterContainmentCallerCancellation test:TestAdapterContainmentEscapedFiniteChildIsDisclosedNotContained
- R7 [satisfied] test:TestAdapterContainmentCleanupCapturedRootsRefuseReplacement test:TestAdapterContainmentCleanupOnlyOwnRegistrationAndIdempotence test:TestAdapterContainmentFailedCleanupContinuationRemainsFailed test:TestAdapterContainmentCLICleanupFailureIsRetainedAndNotApproval
- R8 [satisfied] test:TestAdapterContainmentEnvironmentAndSignedRetention test:TestAdapterContainmentFailedCleanupContinuationRemainsFailed
- R9 [satisfied] capability:delegated-adapter-containment evidence:integration check:delegated-adapter-containment-integration test:TestAdapterContainmentCLIRealDispatchAndDisclosure test:TestAdapterContainmentMCPReportsFailedCleanupWithoutApproval
- R10 [satisfied] test:TestAdapterContainmentConfiguration test:TestAdapterContainmentSetupDoesNotRunCheckoutHook test:TestAdapterContainmentCleanupCapturedRootsRefuseReplacement test:TestAdapterContainmentMCPReportsFailedCleanupWithoutApproval

## 7. Final Report

### Delivered scope

Implementados perfil trusted-process/legacy, capacidades honestas, configuração e budgets finitos, ambiente/credenciais explícitos, captura/redaction limitada, leitura por root descriptor, cancelamento e cleanup confinado com OperationResult imutável. Regressões reais CLI/MCP e qualificação Linux em [relatório B18](../reports/2026-10-26-b18-adapter-containment.md). Validação canônica final passou 156/156; artifact-check reconciliou 31/31 e surface-check 1 target/156 resultados/0 findings. Bundle `rvb-089d6bbf48270dc0` selou 153 evidências; revisão separada `rva-013be502c850b7c2` aprovada com 15 critérios/11 tools/mapping `SD-ebdf5e74 → D1`. Native verify confirmou fresh/approved/ready-to-close e `pose close` aplicou done em 2026-10-26 UTC. A revisão real é declarada, executada separadamente pelo mesmo fornecedor; provas sintéticas não autenticam essa identidade.

### Residual risks

Garantias dependem do backend e da plataforma, não da presença de worktree/process group. Streams/retention podem afetar assinatura e disponibilidade de evidência. O backend exige executável local confiável; isolamento de filesystem/rede/Git/credenciais em disco, memória/CPU e descendentes escapados está indisponível e requisitos correspondentes são recusados. I/O de filesystem não tem deadline absoluto. Redaction cobre valores exatos, não variantes codificadas. Windows foi somente cross-compilado, sem qualificação runtime. Evidências ficam retidas sem TTL automático; orçamento é por execução, não quota global de disco.

### Follow-ups

Nenhum follow-up residual novo desta unidade; backlogs canônicos futuros continuam com sua própria autoridade. Nenhuma disposition, adoção de policy, fornecedor real, release ou atualização do gitlink Harne8 aplicada.
