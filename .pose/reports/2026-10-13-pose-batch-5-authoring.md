# POSE — escrita do lote 5, 2026-10-13

Baseline: `7f7b23b105bfdac3abd2759b253551fc0dd5d974`, merge do [#163](https://github.com/oseiaspereira88/pose/pull/163).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec documental `pose-consolidation-batch-5-authoring` atribui nove artefatos.

## Artefatos e ownership

| Unidade / fonte | Spec de execução | Continuidade |
|---|---|---|
| B11 / D09 | [Amendments existente](../specs/2026-10-10-pose-amendment-history-holds-before-close.md) | Preservar draft/R1–R3; completar plano antes de start, sem duplicação. |
| B11 / F07-01 | [Verificação terminal nova](../specs/2026-10-13-pose-closeout-terminal-verification.md) | Delta draft consome gate de amendments e B08; done/no-op separado de verified. |
| B29 / D10 | [Update dry-run existente](../specs/2026-10-09-pose-update-dry-run-reports-the-whole-update.md) | Preservar in-progress/R1–R3 e revisões; qualificar fidelidade e reconciliar trace. |

Este lote cria uma spec de implementação, além da spec administrativa. Não cria
segunda spec de amendments ou update. Preservar byte a byte as duas specs reutilizadas,
seus R-IDs, status, start records, baselines, revisões e logs; os planos abaixo são
prospectivos para a execução futura, não alterações retroativas desses contratos.
Nenhum código, teste funcional, policy, adoção, release, disposition ou closeout
foi aplicado. Os artefatos de execução listados na nova spec continuam prospectivos.

## Plano de continuidade — amendments / B11 D09

Executar B08 primeiro, conforme DAG. Antes de start da spec existente de amendments,
reconciliar sua lista de artefatos (hoje declara apenas a própria spec), seus
depends_on nativos com B08, assessment freshness e produtores de evidência. O header
antigo vazio não dispensa a dependência do programa; não iniciar por inferência.
Preservar R1–R3 literalmente e registrar apenas decisões/metadata prospectivos.
Se essa reconciliação mudar contrato material, usar amendment/ADR pertinente,
sem editar baseline ou fabricar precedência de decisão.

| Requisito existente | Trabalho/caso planejado | Gate de aceite |
|---|---|---|
| R1 | Primeira emenda em spec com start; registrar baseline do start antes do change event, cobrindo semantic/added e schemas admitidos. | Nós anteriores reconhecidos; log append-only; retry não duplica baseline. |
| R2 | Close de história que falharia lint done; compartilhar avaliação antes da transição. | Recusa antes de alterar status ou certificar close; findings localizados. |
| R3 | Mesma história inválida em in-progress e done. | Warning antecipado e falha exigida em done, segundo contrato aplicável. |

Plano de paths a conferir antes de start: spec existente (modified), seu start
record (created), changelog funcional (created), `pose-mcp/internal/pose/amendments.go`,
`pose-mcp/internal/pose/start.go` e writer de close/lint efetivamente selecionado
(modified), testes sintéticos de amendments/close (created ou modified conforme
inventário) e validation matrix/manual pertinentes. Não declarar esses arquivos
executados nesta escrita; fixar ações/paths exatos no escopo da spec existente.

Qualificar os dois casos históricos de 2026-10-10 em fixtures genéricas, logs
já existentes, ausência de start legado e cutoffs; não copiar dados de outra
instância nem inferir baseline histórica de conteúdo atual. Revisar/fechar essa
spec pelos gates nativos antes de iniciar o delta novo de verificação terminal.
O novo delta não implementa outro seeding/gate para D09.

## Plano de continuidade — update dry-run / B29 D10

A spec existente contém código, dez rodadas de revisão e uma decisão registrada
em 2026-10-10: retirar a entrega do cut 7.4.0 e refazer a abordagem. Seu trace
histórico marca R1–R3 satisfeitos, mas o estado continua in-progress e source
discovery/permissões ainda têm limites. Esses registros são históricos, não prova
de qualificação atual ou promessa nova de versão. Não encerrar por código presente,
nem adotar como final a estratégia de cópia apenas porque sua Decision D1 existe.

1. Confrontar código/commits/testes e todos os findings contra a decisão registrada; registrar uma matriz de resolvido, reproduzido e não qualificado com evidência atual.
2. Delimitar o contrato de preview: flags e exclusão explícita de self-update binário, source discovery para seeds, arquivos/stamp/configuration requests e diferenças legitimamente não determinísticas de IDs/horário.
3. Continuar na spec existente. Antes de modificar código, reconciliar artefatos/trace/start status e registrar prospectivamente a mudança de abordagem já decidida: mesma lógica de update descreve intenção sem aplicar writes. Não construir um segundo planejador que volte a divergir do apply.
4. Qualificar fixtures descartáveis com execução real de update somente nelas, incluindo targets externos controlados. Nenhum preview deve escrever na instância original, fora do ambiente de teste ou por symlink; nenhum real update deste repositório é autorizado por esta escrita.
5. Conservar R1–R3 e evidências históricas; acrescentar/reconciliar produtores atuais por amendment quando aplicável, sem apagar rodadas ou editar baseline. Closeout depende de fidelidade demonstrada e review, não de sucesso global da CI documental.

| Requisito existente | Qualificação planejada | Resultado exigido |
|---|---|---|
| R1 | Preview versus real update com source dirs/seeds ausentes, normal/force/no-self, stamp/merges/requests; links internos/externos/broken/cíclicos e aliases. | Mesmo plano/efeitos pertinentes, IDs variáveis delimitados e zero write do preview comprovado por inventário/digests. |
| R2 | No-op verdadeiro versus ausência apenas de schema migration; erro/parcial/unknown ao ler ou planejar. | No changes apenas quando comprovado; falha não vira no-op. |
| R3 | Instance stamp antigo/atual e engine comparável, doctor/check antes/depois da fixture. | Warning e orientação coerentes; não inferir adoção pelo binário instalado. |

Adicionar casos de arquivo/diretório não legíveis, mode 000/read-only/0100 e
permissões por nome versus listagem. Executar como usuário efetivamente sujeito
às permissões; teste sob root/ACL incapaz de reproduzir a restrição deve declarar
limitação, sem false pass. Conferir tipos, links e targets externos antes/depois,
além do diff Git, pois diff vazio sozinho não prova ausência de write.

Plano de paths a reconciliar dentro da spec existente: `pose-mcp/internal/cli/update_dryrun.go`,
`maintenance.go`, `update_dryrun_test.go`, `doctor.go`, `check.go` e testes de
upgrade/compat já declarados; inventariar helpers de source discovery e writes
antes de fixar o delta exato. Reconciliar a declaração created/modified dos
arquivos já entregues, changelog retirado, matriz e targets; não recriar start
nem apagar sua baseline. Manter scope limitado ao update existente e seus
consumidores, sem novo subsistema ou nova spec de correção concorrente.

## Integração e gates

B11 exige B08, continuidade da spec de amendments e qualificação do delta terminal;
B29 é raiz independente. Decisões estruturais/compatibilidade precisam de registro
prospectivo antes do código correspondente. Os plans acima são gates de execução,
não bypass de readiness ou das baselines existentes.

[Governance-efficiency](../roadmaps/governance-efficiency.md) possui as três specs
(nova e duas reutilizadas) em milestones distintos. O
[coordenador](../roadmaps/pose-consolidation-program.md) possui a escrita e consome
os resultados por refs qualificadas. Nenhuma ownership duplicada foi criada.

A matriz preserva 96 candidatos/44 unidades/17 lotes e oito specs reutilizadas;
13 novas specs de implementação estão em draft e 24 propostas seguem ausentes.
B29 não ganha proposed_specs. Preservar relatórios anteriores como snapshots e
registrar merge do lote 4 sem fabricar closeout de sua spec administrativa.
Próxima escrita: lote 6, B28/B44/B38.

## Conferência reproduzível do snapshot do lote 5

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==24
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==5 and p["authorization"]["next_authoring_batch"]==6
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][5]["authoring_spec"]=="pose-consolidation-batch-5-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==13 and p["counts"]["unmaterialized_implementation_specs"]==24
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
print("PASS: 96 candidatos/44 unidades/17 lotes; treze drafts; 24 propostas ausentes; duas continuidades formalizadas; bases, aceites, DAG e ownership preservados")
```

## Trace e evidência da escrita

R1: delta novo e planos completos por R1–R3 das specs reutilizadas, com limites explícitos.
R2: catálogo/contagens/lotes/futuras propostas preservados.
R3: ownership nos responsáveis e consumo por refs existentes.
R4: dependências e gates de reconciliação antes de executar, preservando os bytes das duas specs.
R5: registrar checks efetivamente executados; CI documental não qualifica gaps funcionais.

Conferência reproduzível e diff --check passaram. Lint-spec passou nas duas specs
novas e nas duas reutilizadas, com aviso de Tasks opcional ausente. Artifact-check
--strict atribuiu exatamente nove paths, com zero errors. Roadmap-check --strict
recusou encerramento como esperado: 76 refs resolvidas, terminal false e 51 blockers
dos members não concluídos. Links relativos e preservação das unidades/camadas
anteriores foram conferidos. A spec administrativa permanece
in-progress até revisão/closeout nativo. CI/Security do head publicado são
acompanhados no PR: check --strict é gate estrutural; validate --tolerant gera
SARIF e não equivale a validate --strict sem findings. O binário local 7.4.1 é
usado somente nos checks estruturais, sem executar novos comportamentos deste lote.
