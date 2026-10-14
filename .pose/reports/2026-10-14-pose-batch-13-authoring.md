# Lote documental 13 — B39/B41/B40

Escrita autorizada em 2026-10-14, America/Sao_Paulo, após merge do
[PR #171](https://github.com/oseiaspereira88/pose/pull/171).
Base `655940a47d5622080dd213f1247b638704be6c43`; manter a
[matriz autoritativa](../results/pose-implementation-plan.json). A correção de
Security #172 já integra esta base; seu token é fornecido ao caller de validate,
sem bypass do check de instalação/upgrade. Não alterar workflows neste lote.

## Escopo e aceites prospectivos

| Unidade | Fontes | Spec draft | Pré-requisitos de execução | Ownership |
|---|---|---|---|---|
| B39 | F06-05/F06-09/F06-10 | [Replay](../specs/2026-10-14-pose-release-publication-evidence-replay.md) | B04/B38 | release/publication-evidence-replay |
| B41 | CONS-08 | [Recovery](../specs/2026-10-14-pose-release-prepare-record-recovery.md) | B12/B13/B14/B38 | release/prepare-record-recovery |
| B40 | F06-07/F06-08 | [Seleção](../specs/2026-10-14-pose-release-scope-selection-and-exclusivity.md) | B23/B24/B37/B38/B39/B41 | release/scope-selection-exclusivity |

Release na coluna ownership designa pose-release-sufficiency-and-recovery; o
coordenador consome somente milestones qualificados. Seis candidatos têm mapping
para R-IDs/casos prospectivos; não modificar source_acceptance/decisive_acceptance
ou dependências do catálogo histórico.

B39 exige hashes/identity/assets/coverage revalidados, providers por papel,
proof/assurance observado e disclosure por plataforma/canal. P05 separa candidato
stale de história íntegra; P07–P09 não viram verified total. Policy knobs têm status
de suporte e lineage/retries invalidam prova ligada à publicação superseded.
Open-next consome resultado revalidado; import factual não exige autorização
retrospectiva do cut, mas isso não dispensa prova pertinente nem autoriza mutação
de assets/tags. A integração de writers é B41; não criar ciclo B39/B41.

B41 preserva inventário/fases e resultados de operação: crash por fase, concorrência,
retry de append sem duplicação e compensação incompleta com material mantido.
Retomar intenção comprovada não equivale a backfill de manifest/proof faltantes.
O probe da frente 9 observou fase intermediária não retomável; perda de material
por Rename de retorno negado é hipótese fundada no ramo, ainda requer teste de
injeção. Crash de processo não qualifica power-loss safety. B12/B13/B14 fornecem
resultado/checkpoint/coordenação; não criar outro lifecycle/ledger de satisfação.

B40 integra os resultados após seus seis pré-requisitos: subconjunto coerente com
scope/refs/digests e preservação dos unselected, exclusão/deferral/outcome próprios,
P10 recusado em plan/apply antes de mover fila e descoberta compartilhada com check
global. Perda de leitura não prova ausência de assignment; escape-hatch de foreign
moved claims não autoriza duplicação. Roadmap alheio não cria gate global e seleção
não quita compromisso vigente nem afirma entrega de todos os canais.

## Proposta de contrato e preservação

Registrar o [ADR temporal/lineage](../adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md)
como Proposed e a [nota de revisão](../knowledge/2026-10-14-decision-log-adr-release-temporal-evidence-review.md),
conforme a exigência de decisão prospectiva de F06-09. Manter os ADRs aceitos de
manifest imutável, archival por ledger, independent verification e obligations.
A proposta distingue modos/lineage/receipts/seleção; não fixa schema/API, adota
policy ou cria gate executável. Reconciliar e resolver a decisão, os decoders,
producer/consumer inventory, compatibilidade/migration preview e limites de
coordenação/durabilidade antes do start funcional. Reusar proof/authority e
avaliadores existentes; preservar subjects/specs/tags/journals históricos.

Nenhum arquivo de código, schema, script, workflow, policy ou release real foi
alterado. Os paths de implementação/testes e ADR modified nas specs são planos
prospectivos, com inventário a completar antes de start; não são evidência de
execução. O admin atribui a escrita efetiva dos treze paths documentais.

## Integridade de planejamento

Preservar 96 candidatos/44 unidades/17 lotes, 37 propostas (uma condicional),
agora 32 materializadas e cinco ausentes; oito specs únicas reutilizadas e
fundações mantêm bytes/status/R-IDs/baselines. As camadas e delivery_complete=false
não mudam. Merge #171 é registrado sem closeout nativo inventado. B19 → B20 → B21
permanece igual. Somente o admin foi iniciado; três novas funcionais continuam
draft, sem start/changelog próprio. Os cinco deltas do roadmap de release agora
estão escritos, mas seus resultados seguem pendentes e o programa não é encerrado.
Próximo lote documental: 14, B31/B33/B35, sem implementação futura autorizada.

## Conferência reproduzível

Executar na raiz. Verifica catálogo/digests, candidatos/R-IDs/contagens, ownership,
links, DAG nativo e overlays/fases anteriores, status da proposta e ausência de
starts funcionais. A conferência não qualifica casos funcionais ou adoção do ADR.

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23","B34","B36","B18","B27","B24","B25","B26","B37","B39","B41","B40"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==5
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md","B34":"pose-state-consistency-and-recovery.md","B36":"pose-state-consistency-and-recovery.md","B18":"delegated-review.md","B27":"governance-efficiency.md","B24":"governance-efficiency.md","B25":"governance-efficiency.md","B26":"governance-efficiency.md","B37":"pose-release-sufficiency-and-recovery.md","B39":"pose-release-sufficiency-and-recovery.md","B41":"pose-release-sufficiency-and-recovery.md","B40":"pose-release-sufficiency-and-recovery.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==13 and p["authorization"]["next_authoring_batch"]==14
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][13]["authoring_spec"]=="pose-consolidation-batch-13-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==32 and p["counts"]["unmaterialized_implementation_specs"]==5
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
assert p["last_authoring_revision"]=="655940a47d5622080dd213f1247b638704be6c43"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==171
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
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=14)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-14-pose-consolidation-batch-13-authoring.md",
            ".pose/specs/2026-10-14-pose-release-publication-evidence-replay.md",
            ".pose/specs/2026-10-14-pose-release-prepare-record-recovery.md",
            ".pose/specs/2026-10-14-pose-release-scope-selection-and-exclusivity.md",
            ".pose/roadmaps/pose-release-sufficiency-and-recovery.md",
            ".pose/roadmaps/pose-consolidation-program.md", ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-14-pose-batch-13-authoring.md",
            ".pose/adr/2026-10-14-release-temporal-evidence-and-attempt-lineage.md",
            ".pose/knowledge/2026-10-14-decision-log-adr-release-temporal-evidence-review.md"]:
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

assert p["batches"][11]["authoring_merged"] and p["batches"][11]["merged_pr"]==170 and p["batches"][11]["native_closeout_pending"]
assert units["B25"]["depends_on"]==["B01","B14","B24"]
assert units["B26"]["depends_on"]==["B10","B24"]
assert units["B37"]["depends_on"]==["B01","B02","B08","B30"]
assert set(units["B25"]["native_spec_dependencies"])=={"pose-action-answers-reverify-on-read","pose-review-disclosure-requires-proof","pose-writer-coordination-and-atomic-publication","pose-residual-nature-commitment-and-effect","pose-open-backlog-reconciliation"}
assert set(units["B26"]["native_spec_dependencies"])=={"pose-action-request-recovery-and-preflight","pose-residual-nature-commitment-and-effect","pose-governance-wait-rework-observability"}
assert set(units["B37"]["native_spec_dependencies"])=={"pose-action-answers-reverify-on-read","pose-review-disclosure-requires-proof","pose-review-approval-domain-preflight","pose-action-target-and-node-scoping","pose-spec-facts-evidence-binding","pose-release-lifecycle-closure"}
for uid in ["B25","B26","B37"]:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()

assert p["batches"][12]["authoring_merged"] and p["batches"][12]["merged_pr"]==171 and p["batches"][12]["native_closeout_pending"]
assert units["B39"]["depends_on"]==["B04","B38"]
assert units["B41"]["depends_on"]==["B12","B13","B14","B38"]
assert units["B40"]["depends_on"]==["B23","B24","B37","B38","B39","B41"]
assert set(units["B39"]["native_spec_dependencies"])=={"pose-historical-trust-and-review-reuse","pose-release-manifest-tag-integrity","pose-release-lifecycle-closure","pose-reproducible-release-verification"}
assert set(units["B41"]["native_spec_dependencies"])=={"pose-cli-mcp-operation-results","pose-journal-checkpoint-and-lock-recovery","pose-writer-coordination-and-atomic-publication","pose-release-manifest-tag-integrity","pose-release-lifecycle-closure"}
assert set(units["B40"]["native_spec_dependencies"])=={"pose-followup-metadata-and-routing-integrity","pose-residual-nature-commitment-and-effect","pose-release-shared-sufficiency-contract","pose-release-manifest-tag-integrity","pose-release-publication-evidence-replay","pose-release-prepare-record-recovery","pose-release-lifecycle-closure","pose-release-prepare-guards-claimed-fragments"}
assert "pose-release-prepare-record-recovery" not in units["B39"]["native_spec_dependencies"]
assert "pose-release-publication-evidence-replay" not in units["B41"]["native_spec_dependencies"]
for uid in ["B39","B41","B40"]:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
    q=units[uid]["contract_decision_plan"]
    assert q["status"]=="proposed" and q["requires_resolution_before_functional_start"] and not q["schema_or_adoption_delivered"]
    assert (root/q["review_log_path"]).exists()
    assert "Proposed — 2026-10-14" in (root/q["proposal_path"]).read_text()
    text=(root/units[uid]["proposed_specs"][0]["path"]).read_text()
    assert "knowledge:adr-release-temporal-evidence-review" in text
    assert "[satisfied]" not in text
assert sum(len(units[uid]["source_candidates"]) for uid in ["B39","B41","B40"])==6
# Publication/recovery are predecessors of selection, never each other's predecessors.
t=(root/".pose/roadmaps/pose-release-sufficiency-and-recovery.md").read_text()
def release_milestone(name):
    return t.split(f"## Milestone: {name}\n",1)[1].split("## Milestone:",1)[0]
for name,member in [("publication-evidence-replay","pose-release-publication-evidence-replay"),("prepare-record-recovery","pose-release-prepare-record-recovery"),("scope-selection-exclusivity","pose-release-scope-selection-and-exclusivity")]:
    assert re.search(r"^- specs: *(.*)$",release_milestone(name),re.M)[1]==member
selected=re.search(r"^- after: *(.*)$",release_milestone("scope-selection-exclusivity"),re.M)[1].split(", ")
assert {"publication-evidence-replay","prepare-record-recovery","cut-sufficiency","manifest-tag-integrity"} <= set(selected)
assert "prepare-record-recovery" not in re.search(r"^- after: *(.*)$",release_milestone("publication-evidence-replay"),re.M)[1].split(", ")
assert "publication-evidence-replay" not in re.search(r"^- after: *(.*)$",release_milestone("prepare-record-recovery"),re.M)[1].split(", ")
print("PASS: 96 candidatos/44 unidades/17 lotes; 32 drafts, cinco propostas ausentes; oito reutilizadas preservadas; ADR Proposed; DAG/ownership/links e fases íntegros")

```

## Trace e evidência da escrita

| Requisito documental | Evidência |
|---|---|
| R1 | Três drafts com mapping dos seis candidatos para R-IDs/casos; ADR Proposed e nota de revisão, sem satisfação funcional. |
| R2 | Conferência reproduzível: 96 candidatos/44 unidades/17 lotes; 32 propostas materializadas, cinco ausentes; oito reutilizadas/fundações intactas e merge #171 sem closeout fabricado. |
| R3 | Ownership única dos três deltas no roadmap de release e consumo qualificado pelo coordenador; referências nativas resolvidas. |
| R4 | B39/B41 sem ciclo, B40 após ambos e seus outros pré-requisitos; DAG/camadas/overlays e B19 → B20 → B21 preservados. |
| R5 | Quatro lint-spec, knowledge-check, artifact-check e check --strict passaram; roadmap-check recusou closeout pendente. |

Conferência contra a baseline confirmou somente B39/B40/B41 alterados entre as
unidades; catálogo/policies/layers e condições originais dos ADR candidates
preservados. Cada path prospectivo modified existe; o inventário de handlers/schema
continua gate antes de start. Diff --check passou. Os quatro lint-spec --strict
passaram na revisão final; Tasks opcional ausente é aviso. O target governance
prospectivo de B39 foi alinhado ao frontmatter delivers após um lint intermediário
reprovar a divergência; a rodada final de gates usa a revisão corrigida. Nenhum
trace funcional foi marcado satisfied, e fixtures/testes futuros não foram executados.

Knowledge-check --strict passou: 10 artefatos checked, zero schema errors/warnings,
zero overdue e ref_failures=0. A nota tem owner, expiry/TTL de 90 dias justificado
e last_reviewed_at; a memória operacional expira, enquanto a proposta durável fica
no ADR. Escrever ou referenciar a proposta não a aceita/adota.

Artifact-check --strict passou (exit 0): 13 claims/13 paths observados, zero erros
e 607 warnings orphan de outros paths governados fora da spec filtrada. Preservar
os avisos; não fazer backfill de proveniência fora deste lote.
Roadmap-check --strict recusou closeout (exit 1), como esperado: 190 refs resolved,
terminal=false e 121 blockers de members/aceites pendentes. Resolução de refs não
comprova replay, recovery, suficiência/seleção ou contrato temporal entregue.
Check --strict passou (exit 0): `Result: SUCCESS — (strict mode) with 30 warning(s).`.
Avisos históricos de changelog, freshness e instance stamp foram preservados, sem
update/reassessment fora do escopo. Resultados registrados em 2026-10-14.

Comandos locais na raiz, com POSE 7.4.1:

```bash
pose lint-spec pose-release-publication-evidence-replay --strict
pose lint-spec pose-release-prepare-record-recovery --strict
pose lint-spec pose-release-scope-selection-and-exclusivity --strict
pose lint-spec pose-consolidation-batch-13-authoring --strict
pose knowledge-check --strict
pose artifact-check --spec pose-consolidation-batch-13-authoring --from 655940a47d5622080dd213f1247b638704be6c43 --to HEAD --strict --json
pose roadmap-check pose-consolidation-program --strict --json
pose check --strict
```

O binário local é 7.4.1, como nos lotes anteriores; a falha de compilador Go 1.26.0
não foi corrigida/retestada nesta escrita. Não alegar build fonte local ou testes
funcionais novos. CI/Security verificarão o head do PR; validate --tolerant/SARIF
não equivale a validate --strict sem findings. A spec administrativa fica
in-progress até revisão e closeout nativo; três funcionais ficam draft e ADR Proposed.

[CI](https://github.com/oseiaspereira88/pose/actions/runs/38517693148) e
[Security](https://github.com/oseiaspereira88/pose/actions/runs/38517693173) do #171
concluíram success no head `77d3306377f6064fea95c91cf61cd6b672f82e52`, com 131/131
checks e a jornada de instalação/upgrade aprovada. São evidência do lote 12
mergeado após correção #172, não do head do lote 13 ou de seus requisitos funcionais.
Nenhuma operação real de release, policy/adoption ou migração foi executada.
