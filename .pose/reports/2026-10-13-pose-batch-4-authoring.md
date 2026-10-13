# POSE — escrita do lote 4, 2026-10-13

Baseline: `66df0bba3ddee07e757f6c458b8fc51f9fcf4ac3`, merge do [#162](https://github.com/oseiaspereira88/pose/pull/162).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec `pose-consolidation-batch-4-authoring` atribui os doze artefatos desta escrita.

## Escopo escrito

| Unidade / candidatos | Spec draft | Resultado planejado |
|---|---|---|
| B12 / F07-02, F07-03, F07-09, PERF-07, CONS-13 | [Resultados CLI/MCP](../specs/2026-10-13-pose-cli-mcp-operation-results.md) | JSON único, parsing estrito e outcome distinto de verdict/coverage/efeitos. |
| B13 / CONS-02, CONS-07, CONS-12 | [Recovery íntegro](../specs/2026-10-13-pose-journal-checkpoint-and-lock-recovery.md) | EOF/schema/seq completos; checkpoint falho não é absent; lock vivo não é quebrado por idade. |
| B30 / F07-05 | [Facts e binding](../specs/2026-10-13-pose-spec-facts-evidence-binding.md) | Evidência associada distinta de candidato; freshness e reuso honestos. |

Manter os três deltas draft. Código, testes, start e changelogs funcionais
listados nas specs são prospectivos; reconciliar paths antes de executar.
A spec administrativa não é uma quarta unidade de implementação. Nenhum motor,
policy, adoção, release, disposition ou spec histórica foi alterado.

## Ordem e limites da execução futura

B12 é raiz com fundação done. Definir schema/exit mapping e parsers por rota
antes do código. Query success não é readiness/coverage; waiting exit 3 não é
failed genérico. Deadline/cancelamento torna execução inconclusiva, sem inventar
verdict negativo. Erro após publicação deve revelar efeitos confirmados, não
prometer rollback. Humano/JSON e CLI/MCP derivam do mesmo resultado.

B13 depende de B12 e das fundações de closeout/atomic-start. Exigir consumo
integral de journal e distinguir tail interrompido de corrupção; repair conserva
bytes/posição/hash e exige proposta autorizada. Checkpoint corrupt/unavailable
não inicia operação nova silenciosamente; resume revalida intenção/inputs e
progresso comprovado. Owner vivo ou desconhecido impede limpeza automática;
TTL/mtime/PID isolados não provam abandono. Declarar diferenças entre sentinel
e lock de kernel e limites por plataforma/filesystem/writer não cooperante.

Restart/fallback deve manter intenção/geração do executor existente. O runtime
futuro consumirá esse contrato na sua qualificação; esta escrita não cria daemon
nem antecipa o lote de B35. CAS/publicação geral, geração de índices e supersession
específica continuam B14/B15/B10, sem duplicar implementação.

B30 é independente com progressive surface done. Indexado/fresh não prova
adequação à spec. Explicitar binding, freshness e candidatos; preservar reuso
comprovado sem rebuild/rerun em toda leitura e sem gerar intent/rationale/judgment.
Se o index não possuir dados suficientes, declarar unknown/limitation.

Classificar compatibilidade e registrar ADRs estruturais adicionais antes do
código pertinente. Realizar assessment freshness, readiness, plano exato de
artefatos/evidência e start quando a execução funcional for autorizada.

## Roadmaps e matriz

Continuar [consistência/recovery](../roadmaps/pose-state-consistency-and-recovery.md)
com operation-results e journal-checkpoint-lock-recovery. Reconciliar seus três
members atuais, mantendo os demais deltas futuros e a proibição de encerramento
pelo recorte parcial. Continuar [governance-efficiency](../roadmaps/governance-efficiency.md)
com spec-facts-binding, preservando os milestones históricos.

O [coordenador](../roadmaps/pose-consolidation-program.md) possui a spec documental
e consome os três resultados por refs qualificadas sem duplicar ownership.
Preservar 96 candidatos, 44 unidades, 17 lotes, aceites, DAG e oito specs reutilizadas.
Doze specs de implementação estão em draft; 25 propostas continuam ausentes,
incluindo a migração condicionada. Registrar merge do lote 3 sem fabricar closeout
da spec administrativa. Manter relatórios anteriores como snapshots.
Próxima escrita: lote 5, B11/B29.

## Conferência reproduzível do snapshot do lote 4

Executar na raiz do checkout correspondente a este lote:

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==25
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==4 and p["authorization"]["next_authoring_batch"]==5
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][4]["authoring_spec"]=="pose-consolidation-batch-4-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==12 and p["counts"]["unmaterialized_implementation_specs"]==25
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
print("PASS: 96 candidatos/44 unidades/17 lotes; doze drafts; 25 propostas ausentes; bases, aceites, DAG e ownership preservados; execução funcional pendente")
```

## Trace e evidência da escrita

R1: três drafts, R-IDs e coverage dos nove candidatos deste lote na matriz.
R2: catálogo, contagens, lotes e propostas futuras conferidos.
R3: ownership nos responsáveis; coordenador consome resultados com refs existentes.
R4: B12/B30 independentes e B13 após B12.
R5: registrar checks efetivamente executados; sucesso documental não qualifica runtime.

A conferência reproduzível e diff --check passaram. Lint-spec passou nas quatro
specs, com aviso de seção Tasks opcional ausente. Artifact-check --strict passou
com doze claims correspondentes aos doze paths atribuídos e zero errors.
Roadmap-check --strict recusou encerramento como esperado no snapshot corrigido:
68 refs resolvidas, terminal false e 42 blockers dos members não concluídos.
O target de B12 usa closeout_plan.go existente; operation_result.go permanece
artefato prospectivo. Links relativos, preservação das unidades/camadas anteriores
e ausência de ciclos nas novas dependências nativas também foram conferidos.
A spec administrativa permanece in-progress até revisão/closeout nativo.
CI/Security do head publicado são acompanhados no PR: governança executa
check --strict, e Security executa validate --tolerant para SARIF. Sucesso do job
SARIF não equivale a validate --strict sem findings. O binário local 7.4.1 é usado
somente para checks estruturais; não executa novos comportamentos destes deltas.
