# POSE — escrita do lote 11, 2026-10-13

Baseline: `bdb514772e1edc465dac24a468301c97dda6ad2d`, merge do [#169](https://github.com/oseiaspereira88/pose/pull/169).
Data documental em America/Fortaleza; timestamps nativos permanecem UTC.
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec documental `pose-consolidation-batch-11-authoring` atribui onze paths.

## Artefatos e ownership

| Unidade / fonte | Spec de execução | Continuidade |
|---|---|---|
| B20 / T3-08 | [Fulfillment existente](../specs/2026-10-09-pose-delegated-review-action-fulfillment.md) | Preservar draft/R1–R3; completar prova/correlação e gate de fase B19 antes de start. |
| B27 / T4-07, T4-09, T4-10, F07-07, F07-08 | [Próxima ação](../specs/2026-10-13-pose-proportional-next-action.md) | Delta draft de rotas/actors/efeitos e autorização vigente, sem perguntas cosméticas. |
| B24 / D07, F05-07 | [Dimensões do residual](../specs/2026-10-13-pose-residual-nature-commitment-and-effect.md) | Delta draft de natureza, compromisso e efeito; legado não classificado e nenhum gate por rótulo. |

Três specs substanciais: duas novas e uma reutilizada. Preservar byte a byte a
spec de fulfillment, R1–R3/status/start/baselines existentes e as outras sete
reutilizadas. Artefatos/testes dos deltas são prospectivos; esta escrita não
executa runner/agente, classificação/triagem/apply, migração, mudança de policy,
adoção, release ou closeout. Nenhuma delivery_complete foi promovida.

## Plano de continuidade — fulfillment / B20

B20 exige B08/B09/B10 e aceite corrente da fase B19. Consultar o
[plano B19 → B20 → B21 do lote 10](2026-10-13-pose-batch-10-authoring.md).
O suporte nativo ao checkpoint de fase ainda é prospectivo: implementar/qualificar
a verificação corrente do recorte B19 antes de B20; não tratar texto/boolean da
matriz ou status de processo como evidência de autorização. B21 qualifica a
capability integrada após fulfillment, sem obrigar capability done para iniciar B20.

Header de fulfillment atualmente aponta somente para dispatch; reconciliar
prospectivamente os deltas de scoping/authority/recovery e produtores de prova,
ledger, contenção/preflight pertinentes. Registrar dependência do recorte B19 no
gate executável qualificado, sem dependência circular de closure global da
capability compartilhada. Não inventar ref nativo para checkpoint inexistente.
Conservar as dependências de unidades/camadas exatamente como no catálogo.

| Requisito existente | Continuidade e qualificação planejadas |
|---|---|
| R1 | Request com brief digest/accepted adapters; completar binding de origem/scope/kind/targets/policy/revision, pergunta e authority. |
| R2 | Run correlacionado por brief é necessário, não suficiente; exigir prova/conclusão/destino admitidos no mesmo contexto e idempotência. |
| R3 | Local e runner não local usam o mesmo contrato/verificador; demonstrar CI e documentar fronteira Harne8 sem anunciar integração real não qualificada. |

O ADR de delegação está aceito desde 2026-10-09: engine registra conclusão
validada, pessoas por exceção de policy, vendor OU modelo distinto e os três
kinds. Reconciliar texto provisório de fulfillment prospectivamente antes de start,
sem reabrir essas decisões ou alterar R1–R3. F1–F8 abaixo são extensões **do plano
neste relatório**, não R-IDs já vigentes ou trace satisfeito; incorporar antes do
código com amendment/ADR quando pertinente, preservando baselines.

| Extensão prospectiva | Contrato/caso a incorporar |
|---|---|
| F1 — Request material | Correlacionar projeto/worktree/actor autorizado, origin/scope/targets/kind, bundle/brief/input digests, policy e revision à pergunta e efeitos. Reutilizar ActionRequest existente; configuração nominal de adapter não prova que está apto. |
| F2 — Run válido | Verificar prova B17, ledger B16, contenção B18, preflight/resultado tipado B19 e destino autorizado; brief digest ou completed isolados não satisfazem fulfillment. Fonte/claim/target estrangeiros, resultado incompleto ou prova inválida preservam condição/diagnóstico. |
| F3 — Serviço versus aprovação | Distinguir pedido de execução/review e condição de approval. Review rejeitado válido pode cumprir o serviço solicitado se o contrato o admite, enquanto approval/gates permanecem insatisfeitos e rejeição visível; negative não vira approval nem exige reabrir pergunta apenas para buscar um sim. |
| F4 — Pessoa e exceções | Critério de pessoa por policy não é satisfeito por run de agente. Cota esgotada, conflito entre reviewers e decisão de negócio produzem pergunta/opções/contexto e authority pertinente, com pergunta material e autorização anterior considerada; nenhuma aplicação manual genérica é imposta. |
| F5 — Corrente e idempotente | Vincular request digest/revision/subject ao run e efeito no destino; replay de mesma chave/input é no-op fiel, mudança/conflito recusa ou invalida de modo explícito. Bundle/subject/policy superados exigem revalidação, sem satisfazer request antiga com conclusão de outro contexto. |
| F6 — Paridade de runner | Local, CI e fronteira Harne8 submetem o mesmo envelope/verificador, com principal e escopo autorizados. Runner não escolhe authority/destino nem registra pessoa fictícia; autenticação/digests/prova seguem o mesmo contrato. |
| F7 — Recovery e rejeições | Fault injection antes/depois de conclusão/answer/receipt não duplica efeito ou spawn. Outcome parcial/deadline/cleanup é distinto de rejection válida; ledger conserva cota/tentativas/negativas. Retry/alias/reason não apagam bloqueio de bundle inalterado nem restauram orçamento. |
| F8 — Jornada e limites | Qualificar request → runner → conclusão/prova → efeito → retomada por kind, com negativo, pessoa required, quota/conflict/business e stale checkpoint B19. Demonstrar CLI/MCP e CI sintético; integração Harne8 real só é anunciada com evidência da fronteira. |

Plano de paths a fixar antes de start: spec existente (modified), start record
(created se ausente), changelog funcional (created), `pose-mcp/internal/pose/action_request.go`,
`action_resolution.go`, `review_dispatch.go`, `review_bundle.go`, callers CLI/MCP
pertinentes (modified), fulfillment/verificador/integração e testes de runner,
correlação/idempotência/negativos (created ou modified conforme inventário),
exemplo CI/contrato de fronteira/documentação e validation matrix/manual.
Conferir actions/paths exatos e schemas; nada disso foi implementado nesta escrita.

## B27 — próximo passo proporcional

B27 exige B08/B10/B12 e reutiliza presentation/progressive surface. Query literal
por actor é comportamento atual: uma futura consulta por holder/capacidade deve
ter semântica explícita e membership conhecido, sem grant no writer. Recipient,
executor, pessoa requerida e advisory são diferentes. A apresentação não altera
satisfaction, effects, IDs ou coverage para acomodar layout; ausência de blocker
conhecido no recorte não concede autorização universal.

Projetar fase/target/responsável/condição/autorização observada e source refs.
Rotas trivial/bug/contract/decision são condicionais; reutilizar spec pertinente e
criar handoff só quando necessário, preservando request/digest/revision/efeitos.
Não repetir confirmação ao retomar sem mudança material; não fabricar assinatura
ou observação humana para reutilizar autorização que a policy exige provar.
Dedup semântico permanece candidato, e perguntas agrupadas conservam answers
independentes. Negativa válida permanece rastreável, sem loop para obter aprovação.

As quatro jornadas — decisão humana, review de agente, input tipado e operação
externa — devem retomar fase/target corretos com evidência pertinente. B27 pode
apresentar capacidade indisponível honestamente antes de B20 estar entregue;
não depende artificialmente de delegação completa para toda orientação/advisory.
Qualificação integrada posterior usa B20/B21 e o piloto B42. Inventariar também
instruções root/scaffold/locales ao alterar workflows/skills na implementação,
sem escrever instruções operacionais ou skill nova nesta interação.

## B24 — natureza, compromisso e efeito do residual

B24 exige B22/B23 e continua o contrato de obligations/reconciliation. Classificar
com fundamento: dívida técnica, defeito, compromisso incompleto, evidência faltante,
adoção/operação, melhoria opcional ou não classificado. Marker, open, path citado,
criticidade e atraso de triagem isolados não definem natureza nem satisfação.
Uma dívida high adiada pode ser consultiva; requirement low obrigatório conserva
seu efeito no escopo. Encaminhar ou renomear não retira compromisso vigente.

Natureza, compromisso, efeito por fase/escopo, urgência/impacto, dono de triagem
versus implementação, SLA e destino permanecem eixos separados. Fratura de release
é relação com entrega/consumidor específicos; suficiência/release B37 não é
antecipada nesta taxonomia. Origem/destino e aliases preservam identidade e
adequação, sem contar a mesma implementação devida duas vezes ou aceitar done
isolado como prova. Legado continua útil e não classificado/unknown onde necessário.

Reusar o [ADR de obligations](../adr/2026-10-04-obligations-are-projected-action-requests-are-persisted.md):
read model/fonte canônica, lifecycle pequeno, efeitos por escopo, diagnostics
nos produtores e autorização já observada. Não criar outro backlog persistido,
arquivo por TODO ou estado resolved no agregador. O candidato de ADR deste lote
é condicionado: não se selecionou sintaxe de metadata, storage ou nova versão
de schema nesta escrita, então não há ADR concorrente nem alteração do aceito.
Antes do código, registrar somente a extensão estrutural necessária para essa
escolha, se os ADRs atuais não a cobrirem; declarar versionamento/migração e paths
exatos. O path atual de schema v1 na spec é inventário prospectivo, não autorização
para redefinir o significado de v1 silenciosamente.

B24 não aplica triagem/risco de B25, não mede condições de B26 e não aplica
classificação ou disposition nas queries. Candidato não é fato classificado.
Coverage/perdas permanecem explícitas; unknown não vira ausência de obrigação ou
bloqueio global fabricado. Gate nasce da autoridade/policy/compromisso vigente,
e não do rótulo que torna a tela mais fácil de agrupar.

## Roadmaps e sequência

[Delegated-review](../roadmaps/delegated-review.md) possui B20 no milestone
fulfillment; adoption possui a capability compartilhada e exige fulfillment.
Preservar o gate histórico de adoption e B19 → B20 → B21, com gate B19 ainda
prospectivo. [Governance-efficiency](../roadmaps/governance-efficiency.md) possui
B27/B24 em milestones separados. O [coordenador](../roadmaps/pose-consolidation-program.md)
possui a spec administrativa e consome resultados, sem duplicar ownership.

Preservar 96 candidatos/44 unidades/17 lotes, aceites/DAG e oito specs reutilizadas.
Agora 26 propostas materializadas em draft e 11 ausentes, incluindo a condicionada.
Registrar merge do lote 10 sem fabricar closeout nativo. Não editar relatórios
anteriores; os snapshots descrevem a ownership vigente na sua revisão fonte.
Próxima escrita: lote 12, B25/B26/B37. Nenhuma execução funcional futura autorizada.

## Conferência reproduzível do snapshot do lote 11

Executar na raiz do checkout correspondente ao lote:

```python
import hashlib, json, re
from collections import Counter
from pathlib import Path
root = Path(".")
p = json.loads((root/".pose/results/pose-implementation-plan.json").read_text())
cpath = root/p["historical_catalog"]["path"]
c = json.loads(cpath.read_text())
assert hashlib.sha256(cpath.read_bytes()).hexdigest() == p["historical_catalog"]["sha256"]
assert p["candidate_ownership"] == c["candidate_ownership"] and len(p["candidate_ownership"]) == 96
original = {u["id"]:u for u in c["units"]}
units = {u["id"]:u for u in p["units"]}
assert len(p["units"]) == len(units) == 44 and units.keys() == original.keys()
assert Counter(u for b in p["batches"] for u in b["units"]) == Counter(units.keys())
assert Counter(u for r in p["roadmap_plan"] for u in r["units_owned"]) == Counter(units.keys())
assert [b["id"] for b in p["batches"]] == list(range(17))
for b in p["batches"]:
    assert len(set(b["spec_slugs"])) == b["unique_substantive_specs"] <= 3
    assert all(units[u]["authoring_batch"] == b["id"] for u in b["units"])
reused, proposed, materialized = {}, {}, set()
for uid,u in units.items():
    for key in ["source_candidates","source_acceptance","decisive_acceptance","depends_on"]:
        assert u[key] == original[uid][key], (uid,key)
    assert u["delivery_complete"] is False
    for s in u["foundation_specs"] + u["reused_specs"]:
        raw=(root/s["path"]).read_bytes()
        assert hashlib.sha256(raw).hexdigest() == s["sha256"]
        assert re.search(r"^status:[ \t]*(\S+)",raw.decode(),re.M)[1] == s["status"]
    reused.update({s["slug"]:s for s in u["reused_specs"]})
    for s in u["proposed_specs"]:
        proposed[s["planned_slug"]]=s
        if s["materialized"]:
            materialized.add(uid)
            raw=(root/s["path"]).read_bytes(); text=raw.decode()
            assert hashlib.sha256(raw).hexdigest() == s["sha256"]
            assert re.search(r"^status:[ \t]*(\S+)",text,re.M)[1] == s["status"] == "draft"
            reqs=re.findall(r"^- (R\d+):",text,re.M)
            assert len(reqs) == len(set(reqs)) >= 6
            assert set(r for rows in u["source_acceptance_requirement_coverage"].values() for r in rows) <= set(reqs)
            assert set(u["source_acceptance_requirement_coverage"]) == set(u["source_candidates"])
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23","B34","B36","B18","B27","B24"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==11
for field,extra in [("technical_execution_layers",False),("final_qualification_layers",True)]:
    seen=set()
    for layer in p[field]:
        assert not seen.intersection(layer)
        for uid in layer:
            assert set(units[uid]["depends_on"]+(units[uid]["final_qualification_after"] if extra else [])) <= seen
        seen.update(layer)
    assert seen==units.keys()
assert set(units["B02"]["native_spec_dependencies"]) == {"pose-historical-trust-and-review-reuse","pose-action-answers-reverify-on-read","pose-review-disclosure-requires-proof","pose-review-bundle-convergence"}
assert set(units["B04"]["native_spec_dependencies"]) == {"pose-material-equivalence-reuse","pose-native-attestation-issuer"}
assert units["B05"]["native_spec_dependencies"] == ["pose-effective-governance-projection"]
assert all(d in present for uid in materialized for d in units[uid]["native_spec_dependencies"])
for uid in materialized:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    owners=[]
    for f in (root/".pose/roadmaps").glob("*.md"):
        for line in f.read_text().splitlines():
            if line.startswith("- specs:") and slug in [v.strip() for v in line.split(":",1)[1].split(",")]:
                owners.append(f.name)
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md","B34":"pose-state-consistency-and-recovery.md","B36":"pose-state-consistency-and-recovery.md","B18":"delegated-review.md","B27":"governance-efficiency.md","B24":"governance-efficiency.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==11 and p["authorization"]["next_authoring_batch"]==12
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][11]["authoring_spec"]=="pose-consolidation-batch-11-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==26 and p["counts"]["unmaterialized_implementation_specs"]==11
for uid in materialized:
    text=(root/units[uid]["proposed_specs"][0]["path"]).read_text()
    assert re.search(r"^depends_on: *(.*)$",text,re.M)[1].split(", ")==units[uid]["native_spec_dependencies"]
assert set(units["B03"]["native_spec_dependencies"]) >= {"pose-historical-trust-and-review-reuse","pose-action-answers-reverify-on-read","pose-review-disclosure-requires-proof"}
assert set(units["B06"]["native_spec_dependencies"])=={"pose-obligation-projection","pose-debt-marker-lexical-precision"}
assert set(units["B07"]["native_spec_dependencies"])=={"pose-recoverable-closeout-plan","pose-delegated-review-dispatch"}
assert not units["B06"]["depends_on"] and not units["B07"]["depends_on"]
for uid in {"B03","B06","B07"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert p["batches"][2]["authoring_merged"] and p["batches"][2]["merged_pr"]==161 and p["batches"][2]["native_closeout_pending"]
assert units["B08"]["native_spec_dependencies"]==["pose-phase-scoped-readiness","pose-governed-effect-enforcement"]
assert set(units["B09"]["native_spec_dependencies"]) >= {"pose-action-answers-reverify-on-read","pose-review-disclosure-requires-proof","pose-policy-scope-validation"}
assert "pose-action-waiver-and-supersession-authority" in units["B10"]["native_spec_dependencies"]
assert not units["B08"]["depends_on"]
for uid in {"B08","B09","B10"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert p["batches"][3]["authoring_merged"] and p["batches"][3]["merged_pr"]==162 and p["batches"][3]["native_closeout_pending"]
assert units["B12"]["native_spec_dependencies"]==["pose-recoverable-closeout-plan"]
assert "pose-cli-mcp-operation-results" in units["B13"]["native_spec_dependencies"]
assert units["B30"]["native_spec_dependencies"]==["pose-progressive-spec-surface"]
assert not units["B12"]["depends_on"] and not units["B30"]["depends_on"]
for uid in {"B12","B13","B30"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert p["batches"][4]["authoring_merged"] and p["batches"][4]["merged_pr"]==163 and p["batches"][4]["native_closeout_pending"]
assert set(units["B11"]["native_spec_dependencies"])=={"pose-action-target-and-node-scoping","pose-amendment-history-holds-before-close","pose-recoverable-closeout-plan"}
assert units["B29"]["proposed_specs"]==[] and not units["B29"]["depends_on"]
for uid in {"B11","B29"}:
    assert units[uid]["reuse_plan_formalized"] and (root/units[uid]["reuse_continuation_plan_path"]).exists()
    for s in units[uid]["reused_specs"]:
        owners=[]
        for f in (root/".pose/roadmaps").glob("*.md"):
            for line in f.read_text().splitlines():
                if line.startswith("- specs:") and s["slug"] in [v.strip() for v in line.split(":",1)[1].split(",")]: owners.append(f.name)
        assert owners==["governance-efficiency.md"],(s["slug"],owners)
slug=units["B11"]["proposed_specs"][0]["planned_slug"]
assert not (root/f".pose/starts/{slug}.json").exists()
assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert p["batches"][5]["authoring_merged"] and p["batches"][5]["merged_pr"]==164 and p["batches"][5]["native_closeout_pending"]
assert units["B28"]["native_spec_dependencies"]==["pose-adaptive-assessment-freshness","pose-v7-legacy-cleanup-plan"]
assert units["B38"]["native_spec_dependencies"]==["pose-release-lifecycle-closure"]
assert units["B44"]["native_spec_dependencies"]==["pose-delegated-review-contract","pose-v7-legacy-cleanup-plan"]
for uid in {"B28","B44","B38"}:
    assert not units[uid]["depends_on"]
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert units["B44"]["source_candidates"]==[] and units["B44"]["source_acceptance_requirement_coverage"]=={}
assert p["batches"][6]["authoring_merged"] and p["batches"][6]["merged_pr"]==165 and p["batches"][6]["native_closeout_pending"]
assert set(units["B14"]["native_spec_dependencies"]) >= {"pose-cli-mcp-operation-results","pose-journal-checkpoint-and-lock-recovery"}
assert "pose-writer-coordination-and-atomic-publication" in units["B15"]["native_spec_dependencies"]
assert not units["B32"]["depends_on"]
for uid in {"B14","B15","B32"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()

assert p["batches"][7]["authoring_merged"] and p["batches"][7]["merged_pr"]==166 and p["batches"][7]["native_closeout_pending"]
assert p["last_authoring_revision"]=="bdb514772e1edc465dac24a468301c97dda6ad2d"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==169
assert set(units["B22"]["native_spec_dependencies"]) >= {"pose-source-coverage-integrity"}
assert set(units["B23"]["native_spec_dependencies"]) >= {"pose-source-coverage-integrity","pose-residual-material-coverage-and-identity"}
assert units["B16"]["proposed_specs"]==[] and units["B16"]["reuse_plan_formalized"]
assert (root/units["B16"]["reuse_continuation_plan_path"]).exists()
assert units["B16"]["native_spec_dependencies_before_start"]==["pose-delegated-review-dispatch","pose-source-coverage-integrity","pose-writer-coordination-and-atomic-publication"]
s=units["B16"]["reused_specs"][0]
t=(root/s["path"]).read_text()
reqs=set(re.findall(r"^- (R\d+):",t,re.M))
assert reqs=={"R1","R2","R3","R4"}
assert re.search(r"^depends_on: *(.*)$",t,re.M)[1]=="pose-delegated-review-dispatch"
assert not (root/".pose/starts/pose-delegated-review-attempt-ledger.json").exists()
assert set(units["B16"]["reuse_extension_acceptance_coverage"])==set(units["B16"]["source_candidates"])
assert set(e for rows in units["B16"]["reuse_extension_acceptance_coverage"].values() for e in rows)=={f"E{i}" for i in range(1,9)}
assert set(e for rows in units["B16"]["reused_spec_requirement_coverage"][s["slug"]].values() for e in rows)<=reqs
owners=[]
for f in (root/".pose/roadmaps").glob("*.md"):
    for line in f.read_text().splitlines():
        if line.startswith("- specs:") and s["slug"] in [v.strip() for v in line.split(":",1)[1].split(",")]: owners.append(f.name)
assert owners==["delegated-review.md"],owners
for uid in {"B22","B23"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=12)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-13-pose-consolidation-batch-11-authoring.md",
            ".pose/specs/2026-10-13-pose-proportional-next-action.md",
            ".pose/specs/2026-10-13-pose-residual-nature-commitment-and-effect.md",
            ".pose/roadmaps/delegated-review.md", ".pose/roadmaps/governance-efficiency.md",
            ".pose/roadmaps/pose-consolidation-program.md", ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-13-pose-batch-11-authoring.md"]:
    f=root/rel
    for target in re.findall(r"\[[^\]]*\]\(([^)]+)\)",f.read_text().split("```python")[0]):
        if "://" not in target and not target.startswith("#"):
            assert (f.parent/target.split("#",1)[0]).resolve().exists(),(rel,target)
graph={}
for f in list((root/".pose/specs").glob("*.md"))+list((root/".pose/specs").glob("*/spec.md")):
    t=f.read_text(); slug=re.search(r"^slug: *(\S+)",t,re.M)
    if not slug: continue
    dep=re.search(r"^depends_on: *(.*)$",t,re.M)
    graph[slug[1]]=[d.strip() for d in dep[1].split("#",1)[0].split(",") if d.strip() and ":" not in d] if dep else []
visited=set(); active=set()
def visit(s):
    assert s not in active,("cycle",s)
    if s in visited: return
    active.add(s)
    for d in graph[s]:
        assert d in graph,(s,d)
        visit(d)
    active.remove(s); visited.add(s)
for s in graph: visit(s)

assert p["batches"][8]["authoring_merged"] and p["batches"][8]["merged_pr"]==167 and p["batches"][8]["native_closeout_pending"]
assert set(units["B34"]["native_spec_dependencies"]) >= {"pose-source-coverage-integrity","pose-git-and-dispatch-snapshot-integrity","pose-writer-coordination-and-atomic-publication","pose-index-generation-integrity"}
assert units["B36"]["native_spec_dependencies"]==["pose-governance-read-view-generation","pose-obligation-projection"]
for uid in {"B34","B36"}:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()

assert p["batches"][9]["authoring_merged"] and p["batches"][9]["merged_pr"]==168 and p["batches"][9]["native_closeout_pending"]
assert units["B18"]["native_spec_dependencies"]==["pose-git-and-dispatch-snapshot-integrity","pose-cli-mcp-operation-results","pose-delegated-review-dispatch"]
assert units["B18"]["depends_on"]==["B07","B12"]
assert units["B19"]["reused_specs"][0]["slug"]==units["B21"]["reused_specs"][0]["slug"]=="pose-delegated-review-capability"
assert units["B19"]["shared_spec_execution_plan"]["execution_phases"]==["B19","B20","B21"]
assert units["B19"]["shared_spec_execution_plan"]["native_phase_gate_support_claimed"] is False
assert units["B19"]["shared_spec_execution_plan"]["intermediate_acceptance_is_not_spec_done"]
assert units["B21"]["shared_spec_execution_plan_ref"]["unit"]=="B19"
assert units["B21"]["depends_on"]==["B19","B20"] and units["B20"]["depends_on"]==["B08","B09","B10","B19"]
for uid in ["B17","B19","B21"]:
    u=units[uid]
    assert not u["proposed_specs"] and u["reuse_plan_formalized"]
    assert (root/u["reuse_continuation_plan_path"]).exists()
    s=u["reused_specs"][0]; text=(root/s["path"]).read_text()
    reqs=set(re.findall(r"^- (R\d+):",text,re.M))
    assert reqs=={f"R{i}" for i in range(1,5 if uid=="B17" else 8)}
    assert set(u["reuse_extension_acceptance_coverage"])==set(u["source_candidates"])
    assert set(u["reused_spec_requirement_coverage"][s["slug"]])==set(u["source_candidates"])
    assert set(q for rows in u["reused_spec_requirement_coverage"][s["slug"]].values() for q in rows)<=reqs
    assert all(d in present and d!=s["slug"] for d in u["native_spec_dependencies_before_start"])
    assert "pose-delegated-review-action-fulfillment" not in u["native_spec_dependencies_before_start"]
    assert not (root/f".pose/starts/{s['slug']}.json").exists()
    owners=[]
    for f in (root/".pose/roadmaps").glob("*.md"):
        for line in f.read_text().splitlines():
            if line.startswith("- specs:") and s["slug"] in [v.strip() for v in line.split(":",1)[1].split(",")]:owners.append(f.name)
    assert owners==["delegated-review.md"],(uid,owners)
assert set(q for rows in units["B17"]["reuse_extension_acceptance_coverage"].values() for q in rows)=={f"P{i}" for i in range(1,7)}
assert set(q for uid in ["B19","B21"] for rows in units[uid]["reuse_extension_acceptance_coverage"].values() for q in rows)=={f"C{i}" for i in range(1,11)}
# Prospectively reconciled native dependencies must also be acyclic, without a capability/fulfillment status cycle.
for uid in ["B16","B17","B19","B20"]:
    graph[units[uid]["reused_specs"][0]["slug"]]=units[uid]["native_spec_dependencies_before_start"]
visited.clear();active.clear()
for s in graph:visit(s)
slug=units["B18"]["proposed_specs"][0]["planned_slug"]
assert not (root/f".pose/starts/{slug}.json").exists()
assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert units["B20"]["reuse_plan_formalized"]

assert p["batches"][10]["authoring_merged"] and p["batches"][10]["merged_pr"]==169 and p["batches"][10]["native_closeout_pending"]
assert units["B27"]["depends_on"]==["B08","B10","B12"] and units["B24"]["depends_on"]==["B22","B23"]
assert set(units["B27"]["native_spec_dependencies"]) >= {"pose-action-target-and-node-scoping","pose-action-request-recovery-and-preflight","pose-cli-mcp-operation-results"}
assert set(units["B24"]["native_spec_dependencies"]) >= {"pose-residual-material-coverage-and-identity","pose-followup-metadata-and-routing-integrity"}
s=units["B20"]["reused_specs"][0];t=(root/s["path"]).read_text()
assert set(re.findall(r"^- (R\d+):",t,re.M))=={"R1","R2","R3"}
assert re.search(r"^depends_on: *(.*)$",t,re.M)[1]=="pose-delegated-review-dispatch"
assert (root/units["B20"]["reuse_continuation_plan_path"]).exists()
assert units["B20"]["phase_acceptance_dependency_plan"]["unit"]=="B19"
assert units["B20"]["phase_acceptance_dependency_plan"]["requires_shared_spec_done"] is False
assert units["B20"]["phase_acceptance_dependency_plan"]["native_phase_gate_support_claimed"] is False
assert "pose-delegated-review-capability" not in units["B20"]["native_spec_dependencies_before_start"]
assert set(q for rows in units["B20"]["reuse_extension_acceptance_coverage"].values() for q in rows)=={f"F{i}" for i in range(1,9)}
assert set(units["B20"]["reuse_extension_acceptance_coverage"])==set(units["B20"]["source_candidates"])
assert units["B20"]["reused_spec_requirement_coverage"][s["slug"]]=={"T3-08":["R1","R2","R3"]}
assert not (root/f".pose/starts/{s['slug']}.json").exists()
owners=[]
for f in (root/".pose/roadmaps").glob("*.md"):
    for line in f.read_text().splitlines():
        if line.startswith("- specs:") and s["slug"] in [v.strip() for v in line.split(":",1)[1].split(",")]:owners.append(f.name)
assert owners==["delegated-review.md"],owners
for uid in ["B27","B24"]:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
# Roadmap placement avoids a status cycle while preserving the scoped B19 gate as a future execution requirement.
t=(root/".pose/roadmaps/delegated-review.md").read_text()
def milestone(name):
    return t.split(f"## Milestone: {name}\n",1)[1].split("## Milestone:",1)[0]
assert "fulfillment" in re.search(r"^- after: *(.*)$",milestone("adoption"),re.M)[1].split(", ")
assert re.search(r"^- specs: *(.*)$",milestone("adoption"),re.M)[1]=="pose-delegated-review-capability"
assert re.search(r"^- specs: *(.*)$",milestone("fulfillment"),re.M)[1]=="pose-delegated-review-action-fulfillment"
assert "adoption" not in re.search(r"^- after: *(.*)$",milestone("fulfillment"),re.M)[1].split(", ")
print("PASS: 96 candidatos/44 unidades/17 lotes; 26 drafts, 11 propostas ausentes; oito reutilizadas preservadas; fulfillment antes de adoption e DAG/ownership/links íntegros")

```

## Trace e evidência da escrita

| Requisito documental | Evidência |
|---|---|
| R1 | B27/B24 draft e continuidade de fulfillment, com coverage dos oito candidatos; R1–R3 reutilizados intactos. |
| R2 | Conferência reproduzível: 96 candidatos/44 unidades/17 lotes; 26 propostas materializadas, 11 ausentes; oito reutilizadas e merge #169 sem closeout fabricado. |
| R3 | Ownership única: fulfillment separado e anterior a adoption; B27/B24 nos responsáveis; consumo pelo coordenador. |
| R4 | DAG/camadas atuais e plano nativo prospectivo sem ciclos; B19 → B20 → B21 mantido; natureza, compromisso, efeito e autorização separados. |
| R5 | Registrar abaixo apenas checks executados; nenhum teste funcional dos deltas satisfeito pelo lint. |

Conferência reproduzível e quatro lint-spec --strict passaram. Tasks opcional
ausente nas três specs novas é aviso, não falha. Conferência adicional contra
baseline confirmou apenas B20/B27/B24 alterados entre as unidades, DAG/policies
anteriores intactos, fulfillment byte a byte e existência dos paths prospectivos
modified. Diff --check passou. Os gates nativos e as refs usam o slug existente
`pose-action-request-recovery-and-preflight` de B10.

Checks locais usam o binário 7.4.1, como nos lotes anteriores; o problema de
compilador não foi corrigido/retestado por esta escrita. Não alegar build fonte
local ou testes funcionais novos. CI/Security verificarão o head atual; validate
--tolerant/SARIF não equivale a validate --strict sem findings. Spec administrativa
permanece in-progress até review/closeout nativo; as três specs de execução
permanecem draft. Nenhum runtime/runner/checkpoint de fase foi entregue aqui.

A CI e Security de #169 concluíram success no head `bdd1d4b4a482dc4aac3233702eb2c27fa21411e2`:
[CI #736](https://github.com/oseiaspereira88/pose/actions/runs/38492490375) e
[Security #738](https://github.com/oseiaspereira88/pose/actions/runs/38492490239).
São evidência do lote 10 mergeado, não do head do lote 11 nem dos requisitos
funcionais de fulfillment, próxima ação ou classificação residual.

Artifact-check --strict da escrita passou: 11 claims/11 paths observados, zero
erros e 607 warnings orphan de outros paths governados fora da spec filtrada.
Não alegar zero findings nem efetuar backfill de proveniência externa neste lote.
Roadmap-check --strict do coordenador recusou encerramento como esperado:
168 refs resolved, terminal=false e 103 blockers de members/aceites pendentes.
A resolução das refs não qualifica gate B19, fulfillment/runner, taxonomia ou
next action; o grafo atual e o plano nativo prospectivo conferidos não têm ciclos.

Check --strict passou: `Result: SUCCESS — (strict mode) with 30 warning(s).`. Avisos históricos de changelogs, freshness e instance stamp foram preservados, sem update/reassessment fora de escopo. Diff --check passou.
