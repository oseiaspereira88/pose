---
slug: pose-delegated-adapter-containment
status: draft
created_at: 2026-10-13
completed_at:
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

Esta escrita não implementa o comportamento. Preservar fontes canônicas, layouts flat/folder, história, autorizações e evidências existentes. Não aplicar dispositions, gerar compromisso de release, adotar policy ou fechar itens automaticamente. Decidir compatibilidade/schema e reconciliar os ADRs pertinentes antes do código; registrar somente extensão estrutural necessária. Reconciliar artefatos e produtores reais antes de start, executar assessment freshness e readiness e aceitar separadamente o escopo funcional.

## 2. Requirements

- R1: Definir perfil/policy de execução com capacidades exigidas e observadas de filesystem/raiz original, metadados Git compartilhados, rede, ambiente/segredos, filhos e recursos. Worktree é cópia de trabalho, não sandbox; verificação antes/depois de B07 detecta alterações pertinentes, sem impedir acesso externo ou provar isolamento de credenciais.
- R2: Validar schema/configuração antes de spawn: argv/executor/identidade não vazios, duração positiva e limites de runs/streams/output/memória/retention com unidades e ranges explícitos. Valores negativos/overflow/whitespace/incompatíveis devem recusar execução; zero deve ter significado admitido explícito, nunca desligar limite silenciosamente. Defaults/ausência/legacy exigem compatibilidade documentada, sem converter budget zero em ilimitado.
- R3: Resolver capacidades do adapter/backend/plataforma contra a policy efetiva antes do efeito. Quando a policy exige contenção não demonstrada ou indisponível, recusar com unsupported/unavailable e orientação; não anunciar garantia só porque o comando compila, o adapter a declara ou o processo está em worktree. Modo de adapter confiável deve divulgar seus limites e não dispensar checks de integridade.
- R4: Usar ambiente mínimo ou allowlist explícita, escopo de credenciais autorizado e representação argv que preserve argumentos sem avaliação de conteúdo de brief/notas como shell. Distinguir diretório descartável, paths de output e raízes proibidas; validar confinamento/symlinks quando aplicável. Não copiar segredos/chaves ao brief/transcript ou registrar valores sensíveis como diagnóstico.
- R5: Limitar stdout/stderr, transcript e arquivo de conclusão na captura/leitura, incluindo streams simultâneos e arquivos externos/special/symlinks. Limite apenas no tail persistido não basta para memória; overflow/truncamento pertinente deve produzir falha/incomplete e nunca conclusão validada. Documentar se memória/CPU são limites aplicados pelo backend ou indisponíveis, com recusa quando exigidos.
- R6: Propagar deadline/cancelamento e encerrar o conjunto de processos coberto pelo mecanismo efetivamente qualificado; limitar espera de pipes/descendentes, detectar falhas de término e não retornar sucesso fictício. Unix process group não prova controle de filho que escapou do grupo; Windows deve ter mecanismo verificado ou declarar unsupported onde a policy exige contenção de descendentes.
- R7: Cleanup deve ser limitado, idempotente e confinado aos recursos pertencentes à operação, preservando outputs/evidências necessários e leitores vivos. Falha de término/remoção deve aparecer no outcome/receipt de B12, separada da conclusão de domínio; não apagar paths arbitrários ou ocultar orphan que ainda executa. Não reexecutar spawn automaticamente para compensar cleanup.
- R8: Definir acesso/permissões, retenção/bytes/TTL e exportação dos artefatos, com ordem de redaction/canonicalização/digests/assinatura explícita para verificação futura. Preservar inventário, orçamento e rejeições do ledger B16 no fluxo integrado; retenção não pode apagar prova conhecida para restaurar cota/aprovação. Conteúdo removido por policy deve ter receipt/limitação, sem promover evidência ausente a válida.
- R9: Divulgar em CLI/MCP e registros profile/capacidades observadas, falha de setup/limite/cleanup, plataforma, coverage e limitations conforme B12/B07. Preservar comportamento de adapters/instâncias legadas com classificação de compatibilidade e opt-in deliberados; não ativar novo backend ou substituir policy nesta entrega documental.
- R10: Qualificar configurações inválidas, escrita externa controlada, Git compartilhado, ambiente/segredos sintéticos, streams/arquivos grandes, deadline/cancelamento/descendentes e retention/cleanup em fixtures isoladas por plataforma. Teste que não consegue aplicar restrição deve declarar limitação; nenhum fornecedor real, sandbox ou suporte Windows é considerado entregue por lint ou por teste Unix.

## 3. Technical Plan

### Artifacts

Os paths são prospectivos. Somente esta spec é criada neste lote, atribuída a `pose-consolidation-batch-10-authoring`. Inventariar callers/testes e fixar ações exatas antes do start; testes abaixo ainda não existem nem foram executados.

- created: .pose/specs/2026-10-13-pose-delegated-adapter-containment.md
- created: .pose/starts/pose-delegated-adapter-containment.json
- created: .pose/changelogs/unreleased/pose-delegated-adapter-containment.md
- modified: pose-mcp/internal/pose/review_dispatch.go
- modified: pose-mcp/internal/pose/review_dispatch_unix.go
- modified: pose-mcp/internal/pose/review_dispatch_windows.go
- modified: pose-mcp/internal/cli/review_dispatch.go
- created: pose-mcp/internal/pose/review_adapter_containment.go
- created: pose-mcp/internal/pose/review_adapter_containment_test.go
- created: pose-mcp/internal/cli/review_adapter_containment_test.go
- created: pose-mcp/internal/mcpserver/review_adapter_containment_test.go

- modified: .pose/indexes/validation-matrix.json
- modified: POSE.md

### Delivery targets

- capability:delegated-adapter-containment module:pose-mcp profile:composed-capability entrypoint:pose-mcp/internal/pose/review_dispatch.go

### Approach and execution gates

Executar B07/B12 antes do delta. Inventariar backends/plataformas realmente suportados e comparar com as garantias do ADR aceito de delegação; registrar extensão estrutural prospectiva quando necessária, sem enfraquecer unilateralmente a integridade da cópia/repo. Reutilizar timeout/process group existente e delimitar seu alcance; escolher mecanismos de isolamento/credenciais e controle de filhos por perfil qualificado, sem impor fornecedor ou container obrigatório a toda instância. Conferir callers CLI/MCP, paths, configurações e produtores de evidência antes de start. Este delta é independente de B16 na matriz: valida policy/limites do adapter, enquanto reserva/cota durável fica em B16 e prova/resultado em B17/B19. Integrar com esses contratos sem duplicar ledger ou issuer.

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7, R8
- Decision: Exigir compatibilidade verificável com a policy do adapter e divulgar limites por plataforma, com captura e cleanup limitados.
- Rationale: Execução em worktree e exit zero não provam contenção ou conclusão válida; exigir garantias verificáveis impede anunciar proteção que o adapter não entrega.

## 6. Validation

### Strategy

Usar fixtures sintéticas e snapshot fixo. Os casos abaixo são planejamento de execução, não evidência de satisfação.

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

### Deterministic checks

Executar suíte Go do módulo e integração CLI/MCP/Attention dos produtores alterados, incluindo controles positivos; artifact-check, surface-check, lint/check e segurança pertinentes. Registrar evidência atual na matriz e no trace. Usar race detector se houver writers concorrentes ou estruturas compartilhadas. Lint da escrita não qualifica o target funcional.

### Requirement trace

| Requisito | Casos planejados |
|---|---|
| R1 | V2, V3 |
| R2 | V1 |
| R3 | V2, V5 |
| R4 | V3, V4 |
| R5 | V4 |
| R6 | V5 |
| R7 | V5, V6 |
| R8 | V6, V7 |
| R9 | V1–V8 |
| R10 | V1–V8 |

## 7. Final Report

### Delivered scope

Spec draft escrita no lote 10. Implementação, testes funcionais, revisão e closeout pendentes; consultar [relatório da escrita](../reports/2026-10-13-pose-batch-10-authoring.md).

### Residual risks

Garantias dependem do backend e da plataforma, não da presença de worktree/process group. Streams/retention podem afetar assinatura e disponibilidade de evidência. Antes da implementação, decidir perfil exigido e política de incompatibilidade sem reabrir decisões aceitas ou fingir proteção universal.

### Follow-ups

Manter unidades e dependências restantes na matriz. Nenhuma disposition ou migração aplicada nesta escrita.
