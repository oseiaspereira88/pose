# POSE — escrita do lote 6, 2026-10-13

Baseline: `916513340afb701ef6f921c0177e4ef6c100dce4`, merge do [#164](https://github.com/oseiaspereira88/pose/pull/164).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec administrativa `pose-consolidation-batch-6-authoring` atribui onze artefatos.

## Escopo escrito

| Unidade / fonte | Spec draft | Resultado prospectivo |
|---|---|---|
| B28 / D11, F07-06 | [Instruções distribuídas](../specs/2026-10-13-pose-distributed-instructions-parity.md) | Freshness/aliases/disponibilidade reais em root/scaffold/pt-BR; migrate explica apply ausente. |
| B44 / reconciliação da frente 10 | [Narrativa atual](../specs/2026-10-13-pose-planning-and-release-narrative-reconciliation.md) | Separar história, contrato aceito, entrega, adoção, eficácia e planejamento. |
| B38 / F06-04, F06-06 | [Manifest e tag](../specs/2026-10-13-pose-release-manifest-tag-integrity.md) | Schema/projeções/digests e snapshot verificados no tree da tag. |

Os três deltas estão draft. B28/B38 planejam alterações de comportamento e B44
é reconciliação documental futura; todos pertencem às contagens originais de
specs substanciais do programa. A spec administrativa é distinta desses deltas.
Os paths de execução são prospectivos, a reconciliar antes de start. Nenhuma
instrução operacional, narrativa histórica, motor, policy, adoção, tag, manifest
ou release foi alterado nesta escrita. Nenhum teste funcional dos deltas foi criado.

## Execução e compatibilidade

B28/B44/B38 são raízes independentes com fundações done. Não impor cadeia artificial
entre instruções, narrativa e release. Realizar assessment freshness, readiness,
plano exato de artefatos/evidência e start quando o escopo de execução for autorizado.

B28 deve inventariar mirrors/locales e testar o que realmente é distribuído.
Freshness existente não exige rescan universal; descoberta de regra não implica
instalação. Mensagem atual de migrate deve divulgar ausência de apply, preservando
B43 como decisão condicionada independente. Não anunciar uma próxima versão por
texto histórico nem implementar migração apply neste delta.

B44 preserva snapshots/ADRs aceitos e acrescenta camada atual de reconciliação.
Seu relatório futuro de entrega/adoção é diferente deste relatório de escrita.
Correção de texto não comprova eficácia de piloto nem capability ausente; patches
em specs iniciadas devem respeitar R-IDs/baselines e fluxo de amendments pertinente.
Conservar programas 6.x/7.0 como contexto e qualificar outcomes antes de fechar.

B38 deve ler manifest/archive/notes/version source do tree da tag, inclusive tag
anotada; worktree atual não substitui snapshot. Recompor campos redundantes e
qualificar schema/compatibilidade antes do código. Separar target head de commit
que contém snapshot; não impor igualdade impossível na preparação. Evidence
posterior e próximo ciclo não staleiam cut historicamente correto. Checker não
move tags ou corrige história; fixtures devem permanecer isoladas.

## Ownership e materialização dos roadmaps

O [coordenador](../roadmaps/pose-consolidation-program.md) passa a possuir B28/B44,
além das specs administrativas de escrita. A matriz deixa de marcá-lo como
owns_authoring_only. Não criar outro roadmap para duplicar essas duas units.

Criar [suficiência e recovery de releases](../roadmaps/pose-release-sufficiency-and-recovery.md)
com apenas B38 e consumir seu milestone pelo coordenador. B37/B39/B40/B41 continuam
nos lotes originais. Roadmap parcial, refs resolvidas e PR documental mergeado não
comprovam release suficiente nem autorizam encerramento do programa.

Preservar 96 candidatos, 44 unidades, 17 lotes, aceites/DAG e oito specs reutilizadas.
B44 não inventa candidate ID: sua source_acceptance continua vazia e seu aceite
consolidado é coberto pelos R1–R8. Agora dezesseis specs propostas estão materializadas
em draft e 21 seguem ausentes, incluindo migração condicionada. Registrar merge
do lote 5 sem fabricar closeout de sua spec administrativa. Manter relatórios
anteriores como snapshots. Próxima escrita: lote 7, B14/B15/B32.

## Conferência reproduzível do snapshot do lote 6

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==21
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==6 and p["authorization"]["next_authoring_batch"]==7
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][6]["authoring_spec"]=="pose-consolidation-batch-6-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==16 and p["counts"]["unmaterialized_implementation_specs"]==21
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
print("PASS: 96 candidatos/44 unidades/17 lotes; dezesseis drafts; 21 propostas ausentes; bases, aceites, DAG e ownership preservados")
```

## Trace e evidência da escrita

R1: três drafts, requisitos/casos e coverage dos quatro candidatos técnicos; B44 usa aceite próprio da reconciliação.
R2: catálogo/contagens/lotes/futuras propostas preservados.
R3: ownership B28/B44 no coordenador e B38 no responsável, com consumo qualificado.
R4: raízes independentes e roadmap de release materializado apenas para B38.
R5: registrar checks efetivamente executados; checks documentais não qualificam comportamento futuro.

Conferência reproduzível e diff --check passaram. Lint-spec passou nas quatro
specs, com aviso de Tasks opcional ausente. Artifact-check --strict atribuiu
exatamente onze paths, com zero errors. Roadmap-check --strict recusou encerramento
como esperado: 91 refs resolvidas, terminal false e 60 blockers de members não
concluídos. Links relativos, preservação de unidades/camadas anteriores e ausência
de ciclos nas novas dependências nativas também foram conferidos. A spec administrativa permanece
in-progress até revisão/closeout nativo. CI/Security são acompanhados no PR:
check --strict é gate estrutural; validate --tolerant gera SARIF e não equivale
a validate --strict sem findings. O binário local 7.4.1 é usado somente nos checks
estruturais, sem executar os novos comportamentos destes deltas.
