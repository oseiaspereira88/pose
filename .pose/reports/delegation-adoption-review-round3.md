# Delegated review capability — cross-vendor source review round 3

Source: Claude Opus 5.5 separate execution `20261029T120257-claude-1416301`. This source approval covers 17eb6e25..b4cdf0dd; it is not yet a sealed delivery attestation.

# Revisão independente de fonte do B21 delegated-review (`17eb6e25..b4cdf0dd`)

**Decisão: aprovado na fonte, sem bloqueadores.** Encontrei dois pontos de baixa severidade que não bloqueiam. Isto não aprova a entrega: falta a validação registrada passar, e falta a evidência nativa selada.

**Quem revisou:** sou Claude Opus 5.5 (`claude-opus-5-5`), da Anthropic. Isso dá independência de fornecedor em relação ao Codex/GPT-6, mas nenhum recibo de fornecedor nem atestado. Esta revisão não tem assinatura, não selei nada e não registrei atestado.

## Correções verificadas

**R2-F01, prova de run assinado no review da mudança que desliga o delegated review.** Conferi cada ponto onde a regra precisa valer:
- **Plano:** `applyProtectedPolicyBaseline` (`review_policy_baseline.go:246-254`) liga `DelegatedRunProofRequired` só quando o baseline tinha a delegação ligada e a política atual a desliga.
- **Digests:** o flag entra no `digestReviewPlan` (`review_plan.go:1350`) e no `reviewBundlePlanDigest` (`review_bundle.go:1643`) com `omitempty`. Bundles antigos mantêm o digest.
- **Admissão e verificação:** `validateBundleAttestationWith` (`review_bundle.go:3186`) usa uma única condição `policy.DelegatedReview || bundle…DelegatedRunProofRequired`. Ela vale para gravação e leitura.
- **Assurance:** `review_assurance.go:163` usa a mesma condição.
- **Runtime:** roteamento (`delegate.go:366`) e auto-conclusão (`review_conclusions.go:656`, via `DelegatedReviewExecutionRequired`) seguem o flag selado. Isso não religa a delegação de forma global: `review_plan.go:274` e `effective_governance.go` continuam lendo só a política live.
- **Rebaixamento:** o rebaixamento de `review_plan.go:368` é suprimido quando o baseline é protegido.
- **Barreira B17:** continua live para aprovações históricas, sem grandfathering.

**R1-F01..F04:** as quatro correções estão presentes:
- o piso da delegação é restaurado;
- a escalação via MCP tem preview tipado e um apply autorizado;
- a ativação valida os bytes capturados (`delegated_review_adoption.go:155-176`) e faz a comparação pós-publicação sobre esses mesmos bytes;
- o roteamento pula candidatos do mesmo modelo antes da seleção.

**Correções adicionais:**
- **Reuso do issuer:** `init_delegation.go:86-93` faz `LoadIssuerKey` primeiro. `CreateIssuerKey` usa `O_EXCL` (`issuer_key.go:165`), então nunca sobrescreve uma chave existente e combina os dois erros.
- **Identidade estável da escalação:** a chave é `Scope@BundleID#Purpose@PolicyDigest` e não depende mais de `requested_at`.
- **Init:** a ativação pelo init passa por `PolicyDocs.Write` → `publishDelegationActivation`, que tem lock, validação do issuer exato e desliga a decisão se a configuração for trocada no meio.

**Docs e skills:** as cópias de `.agents/skills`, `pose-delegate`, README e pt-BR batem byte a byte com o scaffold `dist`. A única diferença no `POSE.md` raiz é a seção Contributor Mode, que pertence à instância. O symlink `.claude/skills/pose-delegate` está correto.

**Mutações do round 2:** as quatro falham na asserção comportamental esperada, sem erro de compilação.

## Achados

**L1 — Baixo (não bloqueante): a escalação do review protegido depende de já existir um bundle selado e atual.**
- **Gatilho:** a mudança que desliga a delegação também deixa a mudança sem adapters/agentes, ou ainda não existe bundle selado para o HEAD atual. Nesse caso `routeDelegateAgent` retorna `ErrNoReviewerAdapter` antes de `RenderReviewBrief` (`delegate.go:343-349`).
- **Consequência:** `DelegatedReviewExecutionRequired(ref)` (`delegated_review_adoption.go:29-30`) usa `CurrentReviewBundle`, que só encontra bundles selados. O resultado é `false`.
  - O dry-run não informa `escalation_required`, e a chamada não-dry-run não abre a decisão (`delegate.go:477-497`). O caminho termina em `delegateNoAgent`.
  - É fail-closed: uma aprovação só com prefixo continua rejeitada e um humano ainda pode revisar. Mas a mensagem da entrega diz que a escalação respeita o review protegido, e isso só vale quando já existe um bundle selado.
  - O R11 só exige isso "while the capability is on", então não viola nenhum requisito, e nenhum teste cobre esse caso.
- **Correção sugerida:** derivar o flag de `PrepareReviewBundle(ref).Payload.Plan.DelegatedRunProofRequired`, que não exige selagem, e adicionar um teste de disabling sem adapters.

**L2 — Baixo (nit): ruído de "conclusion was not recorded" em projetos sem a capability.**
- **Gatilho:** `AutoConcludeReviewRun` agora propaga o erro de `DelegatedReviewExecutionRequired` (`review_conclusions.go:656-659`). Um run concluído cujo `BundleID` está vazio ou sumiu cai em `CurrentReviewBundle`/`LoadReviewBundle` e falha.
- **Consequência:** `review_conclude.go:99-102` mostra uma dica de falha onde antes ficava em silêncio. É só cosmético.

**Observação (não é defeito):** a projeção de obrigações em `obligation_projection.go:479` não expõe a assurance do review protegido quando ele não tem run reference. Afeta só a exibição.

## Limitações

- **Validação registrada não confirmada:** `registered-validation.log` para em `-> test go test ./...`. `.pose/results/delivery-validation.json` é das 00:35, anterior ao commit head (10:37).
- **Comando do race não verificável:** `adoption-race-verified.log` (10:35) mostra três `ok` e nenhum FAIL, mas não registra o comando usado. Não consigo confirmar que foi `-race` nem o filtro aplicado.
- **Fronteira de confiança:** assinatura de executor local não prova cognição nem é recibo de fornecedor. O caso de corpus `canned-executor-approval-proves-cognition` protege só a divulgação dessa limitação, não detecta aprovações prontas.
- **Fora da proteção do baseline:**
  - a proteção só dispara quando o próprio escopo revisado toca o contrato de review (`guard.Paths`). Um desligamento feito em commit não atribuído a nenhum escopo fica fora, o que é o design B17 já existente;
  - sob assurance `declared`, um revisor `human:` continua aceito por declaração.
- **O que eu fiz:** li a fonte e os logs retidos. Não rodei suítes, não fiz edições nem escritas Git, não chamei vendors nem outros agentes, e não toquei em issuers.

O lead pode fechar e dar push da spec quando a validação registrada passar. Não deve começar a spec de paridade nem trabalho de roadmap seguinte.


## Lead dispositions and subsequent validation

- L1 is retained as open low follow-up F2 of the capability spec; it is fail-closed and does not violate the on-capability R11.
- L2 is retained as open low follow-up F3; any refinement must preserve truthful source-loss diagnostics.
- The registered matrix then completed: 158/159 checks passed. The sole failure was the existing real-release install/upgrade journey counting a newer needs-setup catalog entry in its version bound, although CapabilitiesToReview excludes that state. The bound now selects the newest reviewable capability; no production adoption decision is invented. The real-release journey passes after that correction. D4 and the next native review cover this post-source-review test change.
- The capability integration check now records -race in the native validation command itself, so its command and result are observable in the registered evidence rather than inferred from an external plain log.
- Final validation and the sealed native review remain required.
