# POSE — escrita do lote 3, 2026-10-13

Baseline: `7a7b87b28d95bd99da169852d56050bfb3bcdc71`, merge do [#161](https://github.com/oseiaspereira88/pose/pull/161).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec `pose-consolidation-batch-3-authoring` atribui os onze artefatos desta escrita.

## Escopo escrito

| Unidade / candidatos | Spec draft | Resultado planejado |
|---|---|---|
| B08 / T4-01, T4-04 | [Target e nó](../specs/2026-10-13-pose-action-target-and-node-scoping.md) | Destino/fase corretos; membership e artifact/kind/ID exatos. |
| B09 / T4-02 | [Autoridade de lifecycle](../specs/2026-10-13-pose-action-waiver-and-supersession-authority.md) | Waiver e supersession não retiram condição de terceiro sem autoridade/prova. |
| B10 / T4-03, T4-05, CONS-01 | [Recovery e preflight](../specs/2026-10-13-pose-action-request-recovery-and-preflight.md) | Next step suportado, preview read-only e retry da mesma intenção convergente. |

Manter os três deltas draft. Código, testes, start e changelogs funcionais
listados nas specs são prospectivos; reconciliar seus paths antes de executar.
A spec administrativa de escrita não é uma quarta unidade de implementação.
Nenhum motor, policy, adoção, release, disposition ou spec histórica foi alterado.

## Ordem e limites da implementação futura

B08 é raiz independente com duas fundações done. Separar origem, target,
effect scope e subject; scope vazio segue contrato explícito, não vira restrição
parcial implicitamente. Requests de origem terminal não somem da leitura.
Conferir artifact/projeto, kind e ID completos; não herdar hash de outro nó.

B09 depende da qualificação/closeout de B01 e B05, além das fundações de actions.
As duas specs de B01 e a spec de B05 são depends_on nativos. B04 entra pela
qualificação temporal de B01, preservando o DAG histórico. Definir matriz de
answer/waiver/cancellation/supersession/invalidation e bindings de prova de cada
operação. Assinatura de answer não é autorização de waiver; observação mecânica
do engine não equivale a dispensa arbitrária por ator. Cancelamento legítimo
não apaga obrigação de outro producer.

B10 depende de B09. Preview não escreve nem aciona assinatura/toque; revela
checks dependentes de prova ainda ausente. Apply revalida admissibilidade e
precondições protegidas. Retomar supersession após criação da substituta e
falha no append do predecessor com mesma intenção/ID; existência da substituta
não comprova retirada da anterior. Preservar conflito, linhagem, fase parcial
e limites de retry. Reconciliar auditoria de refusal com fatos realmente gravados.

Classificar compatibilidade e registrar decisões estruturais adicionais em ADR
antes do código pertinente. Realizar assessment freshness, readiness, plano de
artefatos/evidência e start quando a execução funcional for autorizada. Não ampliar
estes deltas para publisher CAS geral (B14), todos os journals (B13), snapshots
completos (B11) ou membership do release (lote próprio).

## Roadmaps e matriz

Continuar [agency-readiness](../roadmaps/agency-readiness.md) com três milestones
prospectivos: action-target-scoping, action-lifecycle-authority e
action-recovery-preflight. Preservar o programa histórico 6.4 e seus gates.
Reconciliar o aviso de lote 3 antes ausente sem alterar o aceite de B06.

O [coordenador](../roadmaps/pose-consolidation-program.md) possui a spec documental
e consome os três resultados por refs qualificadas, sem duplicar ownership.
Não encerrar o programa pelo merge de docs ou pela resolução de suas refs.

Preservar 96 candidatos, 44 unidades, 17 lotes, aceites, DAG e oito specs reutilizadas.
Nove specs de implementação estão materializadas em draft; 28 propostas seguem
ausentes, incluindo a migração condicionada. Registrar merge do lote 2 sem
fabricar closeout de sua spec administrativa. Relatórios anteriores conservam
seus snapshots. Próxima escrita: lote 4, B12/B13/B30.

## Conferência reproduzível do snapshot do lote 3

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==28
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==3 and p["authorization"]["next_authoring_batch"]==4
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][3]["authoring_spec"]=="pose-consolidation-batch-3-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==9 and p["counts"]["unmaterialized_implementation_specs"]==28
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
print("PASS: 96 candidatos/44 unidades/17 lotes; nove drafts; 28 propostas ausentes; bases, aceites, DAG e ownership preservados; execução funcional pendente")
```

## Trace e evidência da escrita

R1: três drafts, R-IDs e coverage dos seis candidatos deste lote na matriz.
R2: catálogo, contagens, lotes e propostas futuras conferidos.
R3: ownership em agency-readiness; coordenador consome resultados com refs existentes.
R4: B08 independente, B09 após B01/B05 e B10 após B09.
R5: registrar checks efetivamente executados; sucesso documental não qualifica runtime.

A conferência reproduzível e diff --check passaram. Lint-spec passou nas quatro
specs, com aviso de seção Tasks opcional ausente. Artifact-check --strict passou
com onze claims correspondentes aos onze paths atribuídos e zero errors.
Roadmap-check --strict recusou encerramento como esperado: 56 refs resolvidas,
terminal false e 33 blockers dos members não concluídos. Os links relativos,
a preservação das unidades/camadas anteriores e ausência de ciclos nas novas
dependências nativas também foram conferidos. A spec administrativa permanece
in-progress até revisão/closeout nativo.
CI/Security do head publicado são acompanhados no PR: governança executa
check --strict, e Security executa validate --tolerant para SARIF. Sucesso do job
SARIF não equivale a validate --strict sem findings. O binário local 7.4.1 é usado
somente para checks estruturais; não executa novos comportamentos destes deltas.
