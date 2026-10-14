# Lote documental 15 — piloto integrado B42

Data: 2026-10-14. Base: `32ffafbc3cecba93711bf1782199d6d8a7e6f118`, merge #174.
Escopo: somente escrita de B42, roadmaps/matriz/índice e artefatos administrativos.

## Resultado documental e limites

Criar [B42](../specs/2026-10-14-pose-integrated-agency-and-runtime-pilot.md) draft
com R1–R10 cobrindo F07-10 e J1–J8: trivial, bug/regressão, contrato em dois
módulos, judgment independente, exceção humana de risco, operação externa,
retomada após source mudar e tarefa cross-project. Cada percurso deve ter pessoa
e agente separados; rehearsal não representa experiência humana. Instruções e
outputs de erro/recusa/retomada entram na regressão adversarial.

Baseline/corpus/engine/ambiente, métricas de trabalho útil/reuso e critérios
stop/go/rollback são pré-registrados antes da execução futura. Nenhuma coleta,
chamada de agente, piloto, teste novo, runtime ou benchmark funcional foi iniciado.
Sem pessoa real ou provas/baseline/thresholds resolvidos, qualificação integral
permanece pendente. Go delimitado não muda policy nem prova eficácia universal.

## Dependências e ownership

| Classe | Unidades | Efeito |
|---|---|---|
| Técnica | B21/B25/B26/B27/B37/B40 | Pré-requisitos da preparação/execução. B21 conserva B19 → B20 → B21. |
| Qualificação final | B31/B33/B35 | Provas atuais de domínio/cache/runtime opcionais exigidas antes de closeout de B42; não atrasam escrita ou preparação de fixtures. |
| Fundações | agency-readiness-pilot / mechanization-adversarial-corpus | Done preservadas byte a byte; não reinterpretar o histórico como evidência do novo escopo. |

Governance-efficiency possui B42 uma vez no milestone integrated-agency-runtime-pilot;
seu after referencia diretamente as três specs finais e ordena sua qualificação. R9/trace/review devem conferir as provas
antes de closeout; não se inventa suporte nativo a gates de fase. O coordenador
consome esse resultado e mantém sua etapa de authoring separada da implementação.

## Integridade do plano

Preservar 96 candidatos, 44 unidades, 17 lotes, catálogo/source acceptances/DAG,
execution layers, oito reutilizadas/status/R-IDs/start baselines e fundações.
Somente B42 muda entre as unidades. Registrar merge #174 e closeout nativo ainda
pendente. Agora 36 propostas materializadas e uma ausente: B43 condicional, cujo
benefício/prova ou adiamento ainda exige adjudicação. Nenhuma unidade é marcada
delivery_complete por escrita, merge ou pipeline.

## Validação documental

Registrar resultados após checks sobre o snapshot atribuído ao admin.
CI/Security ainda pendentes à criação; informar resultados do head no PR após
a publicação. Não alegar source build/Go tests locais: usa-se o binário nativo
7.4.1; a falha histórica do compiler Go local não é resolvida por este lote.

## Conferência reproduzível

Execute da raiz do checkout a conferência abaixo, além dos checks nativos.
O script verifica estrutura e preservação; não executa/qualifica o piloto.

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23","B34","B36","B18","B27","B24","B25","B26","B37","B39","B41","B40","B31","B33","B35","B42"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==1
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md","B34":"pose-state-consistency-and-recovery.md","B36":"pose-state-consistency-and-recovery.md","B18":"delegated-review.md","B27":"governance-efficiency.md","B24":"governance-efficiency.md","B25":"governance-efficiency.md","B26":"governance-efficiency.md","B37":"pose-release-sufficiency-and-recovery.md","B39":"pose-release-sufficiency-and-recovery.md","B41":"pose-release-sufficiency-and-recovery.md","B40":"pose-release-sufficiency-and-recovery.md","B31":"pose-shared-runtime.md","B33":"pose-shared-runtime.md","B35":"pose-shared-runtime.md","B42":"governance-efficiency.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==15 and p["authorization"]["next_authoring_batch"]==16
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][14]["authoring_spec"]=="pose-consolidation-batch-14-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==36 and p["counts"]["unmaterialized_implementation_specs"]==1
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
assert p["last_authoring_revision"]=="32ffafbc3cecba93711bf1782199d6d8a7e6f118"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==174
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
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=16)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-14-pose-consolidation-batch-15-authoring.md",
            ".pose/specs/2026-10-14-pose-integrated-agency-and-runtime-pilot.md",
            ".pose/roadmaps/governance-efficiency.md",
            ".pose/roadmaps/pose-consolidation-program.md",
            ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-14-pose-batch-15-authoring.md"]:
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

assert p["batches"][13]["authoring_merged"] and p["batches"][13]["merged_pr"]==173 and p["batches"][13]["native_closeout_pending"]
assert units["B31"]["depends_on"]==["B12","B14","B34"]
assert units["B33"]["depends_on"]==["B15","B34"]
assert units["B35"]["depends_on"]==["B07","B12","B14","B15","B31","B32","B33","B34"]
assert set(units["B31"]["native_spec_dependencies"])=={"pose-cli-mcp-operation-results","pose-writer-coordination-and-atomic-publication","pose-governance-read-view-generation","pose-recoverable-closeout-plan","pose-mcp-active-context","pose-safe-validate-orchestration"}
assert set(units["B33"]["native_spec_dependencies"])=={"pose-index-generation-integrity","pose-governance-read-view-generation","pose-delivery-integrity-index-compaction"}
assert set(units["B35"]["native_spec_dependencies"])=={"pose-git-and-dispatch-snapshot-integrity","pose-cli-mcp-operation-results","pose-writer-coordination-and-atomic-publication","pose-index-generation-integrity","pose-common-domain-and-cli-mcp-handoff","pose-governance-operation-performance","pose-bounded-isolated-graph-caches","pose-governance-read-view-generation","pose-attention-within-a-second"}
for uid in ["B31","B33","B35"]:
    slug=units[uid]["proposed_specs"][0]["planned_slug"]
    assert not (root/f".pose/starts/{slug}.json").exists()
    assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
    q=units[uid]["contract_decision_plan"]
    assert q["status"]=="proposed" and q["requires_resolution_before_functional_start"] and not q["schema_or_adoption_delivered"]
    assert (root/q["review_log_path"]).exists()
    assert "Proposed — 2026-10-14" in (root/q["proposal_path"]).read_text()
    text=(root/units[uid]["proposed_specs"][0]["path"]).read_text()
    assert "knowledge:adr-shared-worktree-runtime-review" in text and "[satisfied]" not in text
assert sum(len(units[uid]["source_candidates"]) for uid in ["B31","B33","B35"])==7
assert all("pose-optional-worktree-runtime" not in units[uid]["native_spec_dependencies"] for uid in ["B31","B33"])
t=(root/".pose/roadmaps/pose-shared-runtime.md").read_text()
def runtime_milestone(name):
    return t.split(f"## Milestone: {name}\n",1)[1].split("## Milestone:",1)[0]
for name,member in [("common-domain-handoff","pose-common-domain-and-cli-mcp-handoff"),("bounded-graph-caches","pose-bounded-isolated-graph-caches"),("optional-worktree-runtime","pose-optional-worktree-runtime")]:
    assert re.search(r"^- specs: *(.*)$",runtime_milestone(name),re.M)[1]==member
parents=set(re.search(r"^- after: *(.*)$",runtime_milestone("optional-worktree-runtime"),re.M)[1].split(", "))
assert {"operation-performance","common-domain-handoff","bounded-graph-caches"} <= parents
for name in ["common-domain-handoff","bounded-graph-caches"]:
    assert "optional-worktree-runtime" not in re.search(r"^- after: *(.*)$",runtime_milestone(name),re.M)[1].split(", ")

assert p["batches"][14]["authoring_merged"] and p["batches"][14]["merged_pr"]==174 and p["batches"][14]["native_closeout_pending"]
assert units["B42"]["depends_on"]==["B21","B25","B26","B27","B37","B40"]
assert units["B42"]["final_qualification_after"]==["B31","B33","B35"]
assert set(units["B42"]["native_spec_dependencies"])=={"pose-delegated-review-capability","pose-residual-triage-and-risk-acceptance","pose-condition-response-and-work-metrics","pose-proportional-next-action","pose-release-shared-sufficiency-contract","pose-release-scope-selection-and-exclusivity","pose-agency-readiness-pilot","pose-mechanization-adversarial-corpus"}
final={"pose-common-domain-and-cli-mcp-handoff","pose-bounded-isolated-graph-caches","pose-optional-worktree-runtime"}
assert set(units["B42"]["native_spec_dependencies_before_final_qualification"])==final
assert not final.intersection(units["B42"]["native_spec_dependencies"])
q=units["B42"]["qualification_dependency_plan"]
assert q["technical_units"]==units["B42"]["depends_on"] and q["final_qualification_units"]==units["B42"]["final_qualification_after"]
assert q["requires_current_runtime_evidence_before_spec_closeout"] and q["preparation_does_not_require_runtime_delivery"] and q["partial_preparation_is_not_unit_complete"] and not q["native_phase_gate_support_claimed"]
assert units["B42"]["source_acceptance_requirement_coverage"]=={"F07-10":[f"R{i}" for i in range(1,11)]}
u=units["B42"];slug=u["proposed_specs"][0]["planned_slug"];text=(root/u["proposed_specs"][0]["path"]).read_text()
assert "[satisfied]" not in text
assert not (root/f".pose/starts/{slug}.json").exists() and not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert not (root/f".pose/results/{slug}.json").exists() and not (root/f".pose/reports/{slug}.md").exists()
assert re.findall(r"^\| (J[1-8]) \|",text,re.M)==[f"J{i}" for i in range(1,9)]
for path in re.findall(r"^- modified: (.+)$",text,re.M): assert (root/path).exists(),path
assert not units["B43"]["proposed_specs"][0]["materialized"] and units["B43"]["proposed_specs"][0]["conditional"]
t=(root/".pose/roadmaps/governance-efficiency.md").read_text().split("## Milestone: integrated-agency-runtime-pilot\n",1)[1].split("## ",1)[0]
assert re.search(r"^- specs: *(.*)$",t,re.M)[1]==slug
assert {"spec:"+x for x in final} <= set(re.search(r"^- after: *(.*)$",t,re.M)[1].split(", "))
assert "xref:proj.pose-dist/milestone:governance-efficiency/integrated-agency-runtime-pilot" in (root/".pose/roadmaps/pose-consolidation-program.md").read_text()
# Compare to the immutable merge base: no existing spec/start/baseline rewritten.
import subprocess
base="32ffafbc3cecba93711bf1782199d6d8a7e6f118"
baseline=json.loads(subprocess.check_output(["git","show",base+":.pose/results/pose-implementation-plan.json"],text=True))
for key in ["candidate_ownership","historical_catalog","technical_execution_layers","final_qualification_layers","execution_policy","closure_policy"]: assert p[key]==baseline[key],key
before={x["id"]:x for x in baseline["units"]}
assert {uid for uid in units if units[uid]!=before[uid]}=={"B42"}
changed=set(subprocess.check_output(["git","diff","--name-only",base,"--",".pose/specs",".pose/starts"],text=True).splitlines())
assert changed <= {".pose/specs/2026-10-14-pose-consolidation-batch-15-authoring.md",".pose/specs/2026-10-14-pose-integrated-agency-and-runtime-pilot.md",".pose/starts/pose-consolidation-batch-15-authoring.json"},changed
assert p["authorization"]["current_authoring_batch"]==15 and p["authorization"]["next_authoring_batch"]==16 and not p["authorization"]["future_batch_implementation_authorized"]
print("PASS: 96 candidatos/44 unidades/17 lotes; 36 drafts, somente B43 condicional ausente; oito reutilizadas/fundações/baselines preservadas; B42 J1–J8, dependencies técnicas/finais distintas, DAG/ownership/links íntegros")
```
