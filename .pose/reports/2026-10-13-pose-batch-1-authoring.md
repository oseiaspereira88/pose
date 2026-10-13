# POSE — escrita do lote 1, 2026-10-13

Baseline: `32b306cd61d2985de08a8338da1faef9989ae984`, merge do [#159](https://github.com/oseiaspereira88/pose/pull/159).
Continuar o [plano autoritativo](../results/pose-implementation-plan.json).
Esta entrega materializa definições e vínculos; não executa os deltas do motor.
A spec `pose-consolidation-batch-1-authoring` atribui somente esta escrita.

## Escopo escrito

| Unidade | Spec draft | Resultado funcional planejado |
|---|---|---|
| B04 | [Tempo e reuso](../specs/2026-10-13-pose-historical-trust-and-review-reuse.md) | Separar validade histórica, confiança atual e base temporal; verificar fontes ou declarar reassunção, com budgets. |
| B02 | [Preflight de aprovação](../specs/2026-10-13-pose-review-approval-domain-preflight.md) | Nova approval inválida produz zero artefatos de review em Store/CLI/import/MCP; preservar negativas legítimas. |
| B05 | [Scopes de policy](../specs/2026-10-13-pose-policy-scope-validation.md) | Taxonomia comum; scope inválido/unsupported não aparece applicable/effective. |

As três specs permanecem draft. Os testes, código, changelogs funcionais e
artefatos de start listados nelas são prospectivos. Não foram criados nesta
interação. A spec documental de atribuição é administrativa e fica fora das
contagens de specs de implementação do programa.

## Ordem e qualificação

Executar B04 primeiro quando seu escopo de código for autorizado. Qualificar
B01 no contexto temporal resultante e fechar as duas specs de #157/#158 pelos
gates correspondentes. Só então iniciar B02: seus depends_on nativos incluem
B04 e as duas specs ainda não terminais de B01, além da fundação de bundles.
Não contornar essas dependências porque os PRs já estão mergeados.

B05 depende da fundação de effective governance; é uma raiz independente.
Não criar uma dependência artificial da cadeia temporal. A implementação
precisará classificar compatibilidade, preencher evidência/artefatos exatos,
realizar assessment freshness, avaliar readiness e registrar start.

Distinguir timestamp afirmado pelo issuer de clock observado e de campos
editáveis do legado. Uma assinatura não fabrica um receipt histórico. A
definição de confiança histórica e autorização presente deve preceder o código.
Também distinguir assinatura, admissibilidade de escrita e aprovação semântica;
negativa legítima não precisa satisfazer critérios de uma approval.

## Roadmaps e matriz

O [roadmap record-truthfulness](../roadmaps/record-truthfulness.md) preserva
seus milestones históricos 6.3.x e recebe quatro milestones prospectivos:
temporal-trust, verified-proof, approval-preflight e policy-scopes.
As três specs novas e as duas correções existentes pertencem a esse responsável.

O [coordenador](../roadmaps/pose-consolidation-program.md) possui a escrita
documental e consome os quatro milestones por refs qualificadas da instância,
sem duplicar a ownership das specs. O programa e o roadmap responsável continuam
draft. A existência de members não qualifica os resultados.

A matriz preserva 96 candidatos, 44 unidades, o DAG original e os 17 lotes.
Agora materializa três das 37 specs novas propostas; 34 continuam ausentes,
incluindo a migração condicionada. As oito specs reutilizadas e suas baselines
permanecem intactas. B19/B21 seguem compartilhando a mesma capability planejada.
Merge do lote 0 é registrado sem fabricar closeout de sua spec documental.

Os lotes 2–16 não são criados aqui. A próxima escrita é lote 2: B03/B06/B07.
Execução funcional e escrita de cada lote permanecem escopos separados.

## Validação documental reproduzível

Executar o código abaixo na raiz do checkout. Conferir o snapshot do lote 1;
o relatório do lote 0 conserva a sua conferência histórica, correspondente
ao snapshot anterior à materialização destes três drafts.

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==34
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
    assert owners==["record-truthfulness.md"], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==1 and p["authorization"]["next_authoring_batch"]==2
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][1]["authoring_spec"]=="pose-consolidation-batch-1-authoring"
print("PASS: 44 unidades/96 candidatos; três drafts do lote 1; 34 propostas ausentes; oito bases reutilizadas; DAG/ownership/aceites preservados; nenhum requisito funcional declarado entregue")
```

Resultado local: PASS em 2026-10-13. A conferência cobre preservação de aceites/
dependências, hashes/status das fundações, cobertura única de unidades/candidatos,
limites de lote, materialização exata, ownership e dependências nativas existentes.

Executar lint-spec nas quatro specs, artifact-check da escrita documental e
check --strict. Conferir CI/Security do head publicado. Registrar resultados
no PR deste lote. Essas verificações não testam uma implementação funcional
que ainda não foi realizada.

Esperar recusa de roadmap-check nos resultados de confiança enquanto os
members estiverem draft/in-progress. Conferir que refs qualificadas resolvem
e que a recusa decorre da não conclusão, não de arquivo ou identidade ausente.

## Limites e próximo passo

Preservar policy, adoção, releases, históricos e estados das specs anteriores.
Nenhuma unidade Bxx é declarada entregue nesta escrita. Os requisitos R1–R8,
R1–R7 e R1–R6 dos três deltas serão qualificados na implementação e revisão
respectivas. Não usar CI verde de documentação como prova desses resultados.
