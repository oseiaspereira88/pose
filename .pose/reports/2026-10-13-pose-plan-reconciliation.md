# POSE — reconciliação e planejamento do lote 0, 2026-10-13

Formalizar o plano aceito de escrita incremental. Baseline: `1e3c61c480379dd2d57b7cbcbb77bcae30bdd5ac`. Reutilizar o
[catálogo da frente 10](../results/2026-10-12-pose-front-10-consolidated-backlog.json)
sem substituir seu diagnóstico histórico. A [matriz atual](../results/pose-implementation-plan.json)
é a fonte autoritativa dos lotes, propostas de specs, responsáveis e dependências.
O [coordenador](../roadmaps/pose-consolidation-program.md) organiza o programa.

## Escopo e autorização

Escrever somente o lote 0. Não iniciar código dos lotes 1–16 nesta interação.
Separar criação de artefatos, execução, qualificação e publicação. A aprovação do
planejamento não adota capabilities nem atribui automaticamente uma release.
Manter o bootstrap documental in-progress até sua própria revisão e closeout.

## Reconciliar entregas existentes

| Unidade | Evidência mergeada | O que continua devido |
|---|---|---|
| B01 | [#157](https://github.com/oseiaspereira88/pose/pull/157) e [#158](https://github.com/oseiaspereira88/pose/pull/158), suíte Go/CI/Security nos heads registrados na matriz | Qualificar o aceite completo e fechar as duas specs pelos gates; não marcar a unidade done por merge. |
| B04 | #157: tempo de aceitação de action answers | Validade histórica de reviews, autorização presente e proveniência de reuso, com limites de ciclos/profundidade/custo. |

Preservar `pose-action-answers-reverify-on-read` e `pose-review-disclosure-requires-proof`.
Continuar os drafts de amendment history, attempt ledger, run provenance, capability
e action fulfillment; qualificar a implementação de update dry-run. São oito specs
existentes únicas. B19/B21 compartilham a mesma capability; não criar duas cópias.
Reconciliar os drafts de delegação com o ADR já aceito: vendor OU modelo diferente,
vendor diferente preferido, participação humana conforme policy/exceção.

## Responsabilidade e materialização

Atribuir cada Bxx a um roadmap planejado único. Referenciar frentes secundárias
sem duplicar execução. O coordenador possui apenas a spec de bootstrap neste lote.
Criar os roadmaps de consistência, releases e runtime nos lotes 7, 12 e 14.
Preservar os programas históricos e os roadmaps done como fundações; não renomear
6.4 para uma release futura. Acrescentar os deltas de forma prospectiva.

Registrar nomes futuros como propostas na matriz. Não colocá-los em `specs:` ou
`depends_on` nativos antes de seus arquivos existirem. Antes de ativar um programa,
conferir ownership, requisitos, ADRs, evidência necessária e DAG materializado.
Datas e números de release ficam para os cuts efetivamente definidos.

## Lotes de escrita

Planejar até três specs substanciais únicas por lote. Além delas, atualizar somente
os vínculos de roadmap e a matriz pertinentes. O lote 16 é condicionado à decisão
de realizar uma migração breaking com benefício demonstrado.

| Lote | Tema | Unidades | Specs existentes ou propostas |
|---|---|---|---|
| 0 | Organização | B01 | `pose-action-answers-reverify-on-read`; `pose-review-disclosure-requires-proof`; `pose-consolidation-program-bootstrap` |
| 1 | Confiança e aprovação | B04, B02, B05 | `pose-historical-trust-and-review-reuse`; `pose-policy-scope-validation`; `pose-review-approval-domain-preflight` |
| 2 | Prova, cobertura e Git | B03, B06, B07 | `pose-git-and-dispatch-snapshot-integrity`; `pose-proof-presence-channel-parity`; `pose-source-coverage-integrity` |
| 3 | Requests corretas | B08, B09, B10 | `pose-action-request-recovery-and-preflight`; `pose-action-target-and-node-scoping`; `pose-action-waiver-and-supersession-authority` |
| 4 | Resultados e recuperação | B12, B13, B30 | `pose-cli-mcp-operation-results`; `pose-journal-checkpoint-and-lock-recovery`; `pose-spec-facts-evidence-binding` |
| 5 | Fechamento e qualificação | B11, B29 | `pose-amendment-history-holds-before-close`; `pose-closeout-terminal-verification`; `pose-update-dry-run-reports-the-whole-update` |
| 6 | Instruções e integridade documental | B28, B44, B38 | `pose-distributed-instructions-parity`; `pose-planning-and-release-narrative-reconciliation`; `pose-release-manifest-tag-integrity` |
| 7 | Escrita e geração | B14, B15, B32 | `pose-governance-operation-performance`; `pose-index-generation-integrity`; `pose-writer-coordination-and-atomic-publication` |
| 8 | Ledger e residual íntegro | B16, B22, B23 | `pose-delegated-review-attempt-ledger`; `pose-followup-metadata-and-routing-integrity`; `pose-residual-material-coverage-and-identity` |
| 9 | Leitura coerente | B34, B36 | `pose-generation-bound-query-cursors`; `pose-governance-read-view-generation` |
| 10 | Delegação confiável | B17, B18, B19, B21 | `pose-delegated-adapter-containment`; `pose-delegated-review-capability`; `pose-delegated-review-run-provenance` |
| 11 | Fulfillment e proporcionalidade | B20, B27, B24 | `pose-delegated-review-action-fulfillment`; `pose-proportional-next-action`; `pose-residual-nature-commitment-and-effect` |
| 12 | Triagem, métricas e suficiência | B25, B26, B37 | `pose-condition-response-and-work-metrics`; `pose-release-shared-sufficiency-contract`; `pose-residual-triage-and-risk-acceptance` |
| 13 | Release recuperável | B39, B41, B40 | `pose-release-prepare-record-recovery`; `pose-release-publication-evidence-replay`; `pose-release-scope-selection-and-exclusivity` |
| 14 | Domínio e runtime | B31, B33, B35 | `pose-bounded-isolated-graph-caches`; `pose-common-domain-and-cli-mcp-handoff`; `pose-optional-worktree-runtime` |
| 15 | Qualificação integrada | B42 | `pose-integrated-agency-and-runtime-pilot` |
| 16 | Migração condicionada | B43 | `pose-legacy-migration-apply-with-recovery` |

São 36 novas specs de implementação propostas, uma condicionada e oito existentes reutilizadas. A spec documental de bootstrap é adicional.
Este lote não cria as 37 specs futuras. Ajustar a divisão somente com razão,
rastreabilidade e verificação de cobertura, sem alterar silenciosamente o catálogo.

## Matriz de responsabilidade e dependências

| Unidade | Lote | Resultado | Roadmap responsável planejado | Dependências técnicas |
|---|---|---|---|---|
| B01 | 0 | Prova persistida e disclosure coerentes | `record-truthfulness` | B04 |
| B02 | 1 | Preflight semântico de aprovação no domínio | `record-truthfulness` | B01 |
| B03 | 2 | Presença consistente entre emitters | `record-truthfulness` | B01, B04 |
| B04 | 1 | Tempo de confiança e proveniência de reuso | `record-truthfulness` | — |
| B05 | 1 | Scopes de policy e efetividade válidos | `record-truthfulness` | — |
| B06 | 2 | Leitura completa ou perda explicitada por fonte | `agency-readiness` | — |
| B07 | 2 | Git desconhecido e snapshot de dispatch verificável | `pose-state-consistency-and-recovery` | — |
| B08 | 3 | Restrições por target, fase e identidade de nó | `agency-readiness` | — |
| B09 | 3 | Autoridade de waiver e supersession | `agency-readiness` | B01, B05 |
| B10 | 3 | Recuperação e preflight de requests | `agency-readiness` | B09 |
| B11 | 5 | Fechamento semântico e histórico de amendments | `governance-efficiency` | B08 |
| B12 | 4 | Resultado e parsing comuns CLI/MCP | `pose-state-consistency-and-recovery` | — |
| B13 | 4 | Journals, checkpoints e recovery de locks | `pose-state-consistency-and-recovery` | B12 |
| B14 | 7 | Publisher exclusivo e coordenação/CAS entre writers | `pose-state-consistency-and-recovery` | B12, B13 |
| B15 | 7 | Geração completa dos índices | `pose-state-consistency-and-recovery` | B14 |
| B16 | 8 | Ledger e reserva durável de tentativas delegadas | `delegated-review` | B06, B14 |
| B17 | 10 | Run assinado e independência observada | `delegated-review` | B01, B04, B16 |
| B18 | 10 | Isolamento, recursos e retenção do adapter | `delegated-review` | B07, B12 |
| B19 | 10 | Conclusões governadas de review, adjudication e smoke | `delegated-review` | B02, B16, B17, B18 |
| B20 | 11 | Fulfillment por runner local e remoto | `delegated-review` | B08, B09, B10, B19 |
| B21 | 10 | Adoção e eficácia da delegação completa | `delegated-review` | B19, B20 |
| B22 | 8 | Cobertura material e IDs de residual/marker | `governance-efficiency` | B06 |
| B23 | 8 | Metadata válida e destino de encaminhamento reconciliado | `governance-efficiency` | B06, B22 |
| B24 | 11 | Natureza, compromisso e efeito do trabalho residual | `governance-efficiency` | B22, B23 |
| B25 | 12 | Triagem revisável e risco aceito | `governance-efficiency` | B01, B14, B24 |
| B26 | 12 | Métricas de condição, resposta e trabalho | `governance-efficiency` | B10, B24 |
| B27 | 11 | Próxima ação proporcional e autorização preservada | `governance-efficiency` | B08, B10, B12 |
| B28 | 6 | Instruções distribuídas executáveis | `pose-consolidation-program` | — |
| B29 | 5 | Qualificar e reconciliar update dry-run | `governance-efficiency` | — |
| B30 | 4 | Facts vinculados à spec e freshness | `governance-efficiency` | — |
| B31 | 14 | API comum e handoff CLI/MCP | `pose-shared-runtime` | B12, B14, B34 |
| B32 | 7 | Algoritmos por operação e benchmark seguro | `pose-shared-runtime` | — |
| B33 | 14 | Vistas de grafo e caches limitados/isolados | `pose-shared-runtime` | B15, B34 |
| B34 | 9 | Read views, geração e invalidação de dependências | `pose-state-consistency-and-recovery` | B06, B07, B14, B15 |
| B35 | 14 | Runtime opcional por worktree e budgets de concorrência | `pose-shared-runtime` | B07, B12, B14, B15, B31, B32, B33, B34 |
| B36 | 9 | Cursor e consulta ligados à geração | `pose-state-consistency-and-recovery` | B34 |
| B37 | 12 | Suficiência compartilhada do cut e critérios tipados | `pose-release-sufficiency-and-recovery` | B01, B02, B08, B30 |
| B38 | 6 | Manifest íntegro e snapshot da tag | `pose-release-sufficiency-and-recovery` | — |
| B39 | 13 | Replay temporal da evidência de publicação | `pose-release-sufficiency-and-recovery` | B04, B38 |
| B40 | 13 | Seleção de escopo e exclusividade antes do freeze | `pose-release-sufficiency-and-recovery` | B23, B24, B37, B38, B39, B41 |
| B41 | 13 | Prepare/record de release com recovery | `pose-release-sufficiency-and-recovery` | B12, B13, B14, B38 |
| B42 | 15 | Piloto integrado de UX e contratos adversariais | `governance-efficiency` | B21, B25, B26, B27, B37, B40 |
| B43 | 16 | Migração breaking somente após benefício e prova | `epistemic-lifecycle` | B14, B34 |
| B44 | 6 | Reconciliar narrativa documental e promessas de releases | `pose-consolidation-program` | — |

Preservar todas as dependências técnicas do catálogo. As camadas na matriz indicam
elegibilidade, não simultaneidade, prazo ou execução autorizada. Priorizar B04,
qualificação de B01 e B02; as demais raízes continuam elegíveis conforme escopo.
Completar consistência antes de reuso entre operações e runtime residente.
Completar prova/ledger/conclusões antes da adoção da delegação inteira.
A qualificação final de B42 também aguarda B31/B33/B35 para exercitar o runtime;
isso está explícito em `final_qualification_after`, separado dos edges históricos.

## Aceite por frente

| Frente | Aceite que permanece devido |
|---|---|
| 1 | Reconciliar diagnóstico, implementação, adoção e eficácia; qualificar update dry-run e narrativa atual. |
| 2 | Completar B01–B06: prova, tempo/reuso, preflight, presença, scopes e cobertura. |
| 3 | Completar ledger, run assinado, contenção e conclusões de review/adjudication/smoke com fulfillment e adoção. |
| 4 | Verificar destino/fase/nó, autoridade, preflight/recovery, próxima ação e invalidação. |
| 5 | Demonstrar perdas explícitas, cobertura material, IDs, encaminhamento, taxonomia e triagem governada. |
| 6 | Demonstrar suficiência compartilhada, manifest/tag, replay, recovery e seleção exclusiva de escopo. |
| 7 | Qualificar closeout, resultados CLI/MCP, facts, instruções e retomada proporcional por jornadas. |
| 8 | Medir algoritmos e qualificar domínio comum, caches limitados, paginação coerente e runtime opcional. |
| 9 | Exercitar corrupção, concorrência, crash/retry e geração em writers, ledger, readers e releases. |
| 10 | Conferir cobertura integral, revisão, evidências e claims de todas as unidades do escopo aprovado. |

## Critério de consolidação

Exigir aceite das unidades, evidência no consumidor, revisão, closeout, documentação
e adoção/piloto quando aplicáveis. Não usar apenas contagem de specs, campos presentes
ou CI verde como prova de eficácia. Não chamar B43 de implementado quando for adiado.
Materializar os milestones restantes antes de encerrar o coordenador: o seu único
milestone atual qualifica o planejamento. O check estrutural não prova completude
dos artefatos ausentes; a revisão precisa confrontar a matriz inteira.

## Validação documental

Executar a conferência reproduzível abaixo na raiz do checkout. Ela confere cobertura,
aceites, dependências, DAG, limites de lote, fontes e reutilização sem escrever arquivos.
Lint-spec e artifact-check --strict passaram em 2026-10-13. A execução de check --strict
e o head de CI serão registrados no PR documental. Roadmap-check --strict --json
retornou exit 1, terminal false e member spec is not terminal para o bootstrap
in-progress, como esperado. O aviso de ausência de cut criteria limita essa
avaliação aos members atuais; não certifica as etapas futuras. Isso não é aprovação
do programa nem uma falha de estrutura. Os quatro runs de CI/Security de #157/#158
foram conferidos novamente com conclusão success e os heads registrados na matriz.

```python
import hashlib, json, re
from collections import Counter
from pathlib import Path

root = Path(".")
p = json.loads((root/".pose/results/pose-implementation-plan.json").read_text())
cpath = root/p["historical_catalog"]["path"]
c = json.loads(cpath.read_text())
assert hashlib.sha256(cpath.read_bytes()).hexdigest() == p["historical_catalog"]["sha256"]
assert p["candidate_ownership"] == c["candidate_ownership"]
assert len(p["candidate_ownership"]) == 96
original = {u["id"]: u for u in c["units"]}
units = {u["id"]: u for u in p["units"]}
assert len(p["units"]) == len(units) == 44 and units.keys() == original.keys()
assert Counter(u for b in p["batches"] for u in b["units"]) == Counter(units.keys())
assert [b["id"] for b in p["batches"]] == list(range(17))
assert Counter(u for r in p["roadmap_plan"] for u in r["units_owned"]) == Counter(units.keys())
for b in p["batches"]:
    assert len(set(b["spec_slugs"])) == b["unique_substantive_specs"] <= 3
    assert all(units[u]["authoring_batch"] == b["id"] for u in b["units"])
reused, proposed = {}, {}
for uid, u in units.items():
    for field in ["source_candidates", "source_acceptance", "decisive_acceptance", "depends_on"]:
        assert u[field] == original[uid][field], (uid, field)
    assert u["delivery_complete"] is False
    assert u["responsible_roadmap_planned"] in {r["slug"] for r in p["roadmap_plan"] if uid in r["units_owned"]}
    for s in u["foundation_specs"] + u["reused_specs"]:
        raw = (root/s["path"]).read_bytes()
        assert hashlib.sha256(raw).hexdigest() == s["sha256"], s["path"]
        assert re.search(r"^slug:\s*(\S+)", raw.decode(), re.M)[1] == s["slug"]
        assert re.search(r"^status:\s*(\S+)", raw.decode(), re.M)[1] == s["status"]
    reused.update({s["slug"]: s for s in u["reused_specs"]})
    proposed.update({s["planned_slug"]: s for s in u["proposed_specs"]})
assert len(reused) == 8 and len(proposed) == 37
assert sum(s["conditional"] for s in proposed.values()) == 1
present = {re.search(r"^slug:\s*(\S+)", f.read_text(), re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(not s["materialized"] and slug not in present for slug, s in proposed.items())
for field, extra in [("technical_execution_layers", False), ("final_qualification_layers", True)]:
    seen = set()
    for layer in p[field]:
        assert not seen.intersection(layer)
        for uid in layer:
            deps = units[uid]["depends_on"] + (units[uid]["final_qualification_after"] if extra else [])
            assert set(deps) <= seen, (uid, deps)
        seen.update(layer)
    assert seen == units.keys()
assert all(x["merged"] and x["merge_commit_sha"] and x["head_sha"] for x in p["merged_pull_requests"])
assert [x["number"] for x in p["merged_pull_requests"]] == [157, 158]
assert all(x["tested_head"] == next(r["head_sha"] for r in p["merged_pull_requests"] if r["number"] == x["pr"]) for x in p["ci_evidence"])
assert p["authorization"]["current_authoring_batch"] == 0
assert p["authorization"]["next_authoring_batch"] == 1
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert all(s["status"] == "in-progress" for s in units["B01"]["reused_specs"])
assert p["closure_policy"]["materialize_remaining_milestones_before_program_close"]
print("PASS: 96 candidatos; 44 unidades; 17 lotes; DAGs; 36 novas + 1 condicionada; 8 reutilizadas; nenhuma spec futura criada")
```

Resultado em 2026-10-13: PASS — 96 candidatos, 44 unidades, 17 lotes, DAGs íntegros, 36 novas + 1 condicionada, oito reutilizadas e nenhuma spec futura criada.

## Próxima interação

Escrever o lote 1: B04, B02 e B05, com referências às correções existentes e ao
roadmap responsável. A implementação será tratada por escopo autorizado depois.
Não criar automaticamente os lotes seguintes ou iniciar a migração condicionada.
