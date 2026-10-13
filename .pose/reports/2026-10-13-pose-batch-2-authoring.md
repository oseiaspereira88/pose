# POSE — escrita do lote 2, 2026-10-13

Baseline: `6dba937748644d8f00419f136c5e84a57c8cf12b`, merge do [#160](https://github.com/oseiaspereira88/pose/pull/160).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec `pose-consolidation-batch-2-authoring` atribui estes treze artefatos documentais.

## Escopo escrito

| Unidade / candidatos | Spec draft | Resultado planejado |
|---|---|---|
| B03 / T2-04 | [Presença entre canais](../specs/2026-10-13-pose-proof-presence-channel-parity.md) | Policy explícita; issuer não dispensa prova de presença exigida. |
| B06 / T2-05, D01, D13, F05-01, F05-03 | [Coverage por fonte](../specs/2026-10-13-pose-source-coverage-integrity.md) | Perdas de leitura/parse/scan não viram zero complete; paridade de follow-ups e histórico íntegro. |
| B07 / D02, T3-02, CONS-03 | [Git e snapshot](../specs/2026-10-13-pose-git-and-dispatch-snapshot-integrity.md) | Git desconhecido não certifica clean; bytes, paths e inputs pertinentes verificados antes/depois. |

Manter os três deltas draft. Os paths de código/testes/start/changelog nas specs
são prospectivos, a reconciliar antes da execução. Nenhum motor, teste funcional,
policy, adoção, release ou disposition foi alterado. A spec administrativa da
escrita fica fora das contagens de implementação.

## Dependências e limites

Executar B03 somente após qualificar/encerrar B04 e as duas correções de B01,
além das fundações declaradas. B02 não é dependência artificial de B03.
Decidir escopo universal versus SSH e reconciliar R5/D2 antes do código;
não converter grant humano ou login de Portal em hardware touch comprovado.

B06 e B07 são raízes independentes com fundações done. Coverage incompleta deve
ser vinculada a produtor/recorte; header ilegível não permite associação precisa
inventada. Descarte de obligation deve afetar Complete. Dispatch precisa de
outputs autorizados e consumo protegido; comparação antes/depois não é sandbox
nem transação geral. Classificar mudanças de schema/compatibilidade e decisões
estruturais antes de implementar, com assessment freshness/readiness/start.

## Roadmaps e reaproveitamento

Continuar [record-truthfulness](../roadmaps/record-truthfulness.md) com B03 e
[agency-readiness](../roadmaps/agency-readiness.md) com B06. Preservar seus programas
históricos. Antecipar somente a criação de
[consistência e recuperação](../roadmaps/pose-state-consistency-and-recovery.md)
do lote 7 para o lote 2, para possuir B07 desde sua escrita. Manter B12/B13,
B14/B15 e B34/B36 nos lotes 4, 7 e 9. Essa antecipação não cria specs futuras.

O [coordenador](../roadmaps/pose-consolidation-program.md) possui a spec de escrita
e consome os milestones dos responsáveis por refs qualificadas, sem duplicar
ownership. Não fechar o roadmap parcial pelo aceite isolado de B07: os demais
members obrigatórios e sua qualificação continuam pendentes.

A matriz conserva 96 candidatos, 44 unidades, 17 lotes, os aceites, o DAG e as
oito specs reutilizadas. Seis specs de implementação estão materializadas em
draft; 31 propostas continuam ausentes, incluindo a migração condicionada.
Registrar merge do lote 1 sem fabricar closeout nativo de sua spec documental.
Preservar os relatórios dos lotes 0/1 como snapshots históricos.

A próxima escrita é lote 3: B08/B09/B10. A execução funcional permanece uma etapa
separada, a seguir na ordem formalizada e mediante escopo correspondente.

## Conferência reproduzível do snapshot do lote 2

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==31
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==2 and p["authorization"]["next_authoring_batch"]==3
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][2]["authoring_spec"]=="pose-consolidation-batch-2-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==6 and p["counts"]["unmaterialized_implementation_specs"]==31
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
print("PASS: 96 candidatos/44 unidades/17 lotes; seis drafts; 31 propostas ausentes; bases, aceites, DAG e ownership preservados; execução funcional pendente")
```

## Evidência e trace da escrita

R1: três drafts com R-IDs e coverage dos nove candidatos deste lote na matriz.
R2: conferência do catálogo, 44 unidades, contagens, lotes e ausência dos próximos drafts.
R3: ownership nos responsáveis e refs existentes; coordinator consome resultados.
R4: dependências nativas explícitas; roadmap de B07 antecipado sem seus futuros members.
R5: registrar os checks efetivamente executados abaixo; não declarar resultados funcionais por sucesso documental.

Conferência reproduzível e diff --check passaram. Lint-spec passou nas quatro
specs, com aviso de seção Tasks opcional ausente. Artifact-check --strict passou
com treze claims correspondentes aos treze paths atribuídos e zero errors.
Roadmap-check --strict recusou encerramento como esperado: 43 refs resolvidas,
terminal false e blockers de members não concluídos. A conferência foi repetida
no snapshot commitado final após acrescentar o trace dos casos planejados.

CI/Security do head publicado serão acompanhados no PR; o gate de governança
executa check --strict e validate --strict. O binário local 7.4.1 foi usado nos
checks estruturais, sem representar execução de novos comportamentos do motor.
A spec documental permanece in-progress até revisão/closeout nativo; resultados
funcionais permanecem draft.
