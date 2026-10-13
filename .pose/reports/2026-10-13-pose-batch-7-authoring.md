# POSE — escrita do lote 7, 2026-10-13

Baseline: `4a60ccc4713efe18566ea72a52e2ae04f6a49c03`, merge do [#165](https://github.com/oseiaspereira88/pose/pull/165).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec administrativa `pose-consolidation-batch-7-authoring` atribui doze artefatos documentais.

## Escopo escrito

| Unidade / fonte | Spec draft | Resultado prospectivo |
|---|---|---|
| B14 / CONS-04, CONS-06, CONS-09 | [Coordenação e publicação](../specs/2026-10-13-pose-writer-coordination-and-atomic-publication.md) | Staging exclusivo, CAS, coordenação entre writers e efeitos parciais recuperáveis. |
| B15 / CONS-05 | [Geração de índices](../specs/2026-10-13-pose-index-generation-integrity.md) | Geração completa ou indisponibilidade explícita, sem mistura após retry/crash. |
| B32 / D12, PERF-01, PERF-02, PERF-03, PERF-05 | [Performance por operação](../specs/2026-10-13-pose-governance-operation-performance.md) | Algoritmos e Git em lote com fidelidade; benchmark isolado e métricas por produtor. |

As três specs permanecem draft. Requisitos, artefatos e testes de execução são
prospectivos. Esta escrita não muda motor, policy, adoção ou estado de specs antigas.
Não executar o benchmark atual: seu reset do worktree e tolerância a erros devem
ser substituídos por fixtures descartáveis e verdict explícito durante B32.

## Dependências e execução futura

B14 depende dos resultados tipados de B12 e recovery de B13, preservando fundações
de start/transferência. Reconciliar artefatos antes do start e qualificar staging,
coordenação por recurso, CAS de preimages, criação imutável e durabilidade por classe
nas plataformas suportadas. Writers externos exigem limites honestos de proteção;
efeitos anteriores ao erro precisam de receipt verificável e retry idempotente.
Curadoria e histórico concorrentes não podem ser perdidos na recuperação.

B15 depende de B14 e da fundação de compaction. Validar o conjunto completo antes
de publicar sua geração; readers devem fixar uma geração e tratar partes ausentes,
incompatíveis ou corrompidas como indisponíveis. Planejar crash/retry, migração de
índices legados e coleta limitada que proteja readers e preserve proveniência.

B32 é raiz independente, com fundações de redução de Git e atenção rápida.
Comparar baseline e otimização no mesmo corpus isolado, incluindo entradas hostis.
Medir tempo, CPU, RSS, leituras/parses/cópias e processos/bytes Git por produtor,
sem fabricar atribuições. Preservar verdict, ordem, identidade, layouts, conflitos
e histórico completo. Clustering exige equivalência ou aproximação contratada;
caches são limitados à operação e não autorizam daemon ou aliases ilimitados.
Checks desta escrita não qualificam esses resultados funcionais.

## Ownership e roadmaps

O [roadmap de consistência](../roadmaps/pose-state-consistency-and-recovery.md)
possui B14/B15 junto de B07/B12/B13. B34/B36 continuam planejados para o lote 9.
O novo [roadmap de runtime](../roadmaps/pose-shared-runtime.md) possui somente B32;
B31/B33/B35 seguem no lote 14, sem refs nativas para specs ausentes.
O [coordenador](../roadmaps/pose-consolidation-program.md) possui a escrita
administrativa e consome os três resultados por refs qualificados, sem ownership duplicada.

Preservar 96 candidatos, 44 unidades, 17 lotes, aceites/DAG e oito specs reutilizadas.
Dezenove propostas estão materializadas em draft e 18 seguem ausentes, incluindo
migração condicionada. Registrar merge do lote 6 sem fabricar closeout nativo.
Relatórios anteriores permanecem snapshots. Próxima escrita: lote 8, B16/B22/B23.

## Conferência reproduzível do snapshot do lote 7

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==18
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==7 and p["authorization"]["next_authoring_batch"]==8
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][7]["authoring_spec"]=="pose-consolidation-batch-7-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==19 and p["counts"]["unmaterialized_implementation_specs"]==18
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
print("PASS: 96 candidatos/44 unidades/17 lotes; dezenove drafts; 18 propostas ausentes; bases, aceites, DAG e ownership preservados")
```

## Trace e evidência da escrita

R1: três drafts e coverage dos nove candidatos de origem.
R2: catálogo, contagens, lotes e propostas futuras preservados.
R3: ownership única em consistência/runtime, consumida pelo coordenador.
R4: B14 após B12/B13, B15 após B14 e B32 independente; runtime parcial explícito.
R5: registrar somente checks executados; encerramento exige qualificação futura.

Conferência reproduzível e diff --check passaram. Lint-spec passou nas quatro
specs, com aviso de Tasks opcional ausente. Artifact-check --strict atribuiu
exatamente doze paths, com zero findings. Check --strict passou. Roadmap-check
--strict recusou encerramento como esperado: 107 refs resolvidas, terminal false
e 69 blockers de members não concluídos. Links relativos, preservação de unidades
/camadas anteriores e ausência de ciclos nas novas dependências nativas foram
conferidos. A spec administrativa permanece in-progress até
revisão e closeout nativo. CI/Security verificarão o head no PR. Check --strict é
gate estrutural; validate --tolerant gera SARIF e não equivale a validate --strict
sem findings. O binário local 7.4.1 serve apenas aos checks estruturais.
