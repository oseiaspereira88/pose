# POSE — escrita do lote 9, 2026-10-13

Baseline: `9ed5b74e4784cad8193d89ed16b7e037d9554edf`, merge do [#167](https://github.com/oseiaspereira88/pose/pull/167).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec documental `pose-consolidation-batch-9-authoring` atribui dez paths.

## Artefatos e ownership

| Unidade / fonte | Spec de execução | Aceite prospectivo |
|---|---|---|
| B34 / D03, T4-06, PERF-09, CONS-10 | [Read view/generation](../specs/2026-10-13-pose-governance-read-view-generation.md) | Closure de inputs observados e revalidados; interleaving não vira autorização stale. |
| B36 / PERF-11 | [Queries/cursors](../specs/2026-10-13-pose-generation-bound-query-cursors.md) | Mesma geração identificada ou stale/restart explícito; filtros/ordem vinculados, custo honesto. |

Os dois deltas permanecem draft, com artefatos/testes prospectivos. Esta escrita
não implementa geração, watcher, paginação, reuso entre operações ou runtime.
Não executar benchmark, policy/adoption, migration, release ou closeout funcional.
Preservar byte a byte fundações e oito specs reutilizadas, incluindo o plano de
continuidade de B16 do lote anterior. Registrar merge sem fabricar seu closeout.

## Dependências e execução futura

B34 depende de B06/B07/B14/B15. B06 divulga perdas por fonte/recorte, B07 vincula
Git/dispatch aos inputs pertinentes, B14 coordena publicação/CAS e B15 fixa conjunto
de índices completo. B34 não reimplementa esses publishers nem substitui a prova
histórica de review: acrescenta closure/read view por operação e revalidação para
certificar leitura ou efeito atual. B36 depende de B34 e consome esse contrato.

O snapshot atual de obligations cobre parte dos inputs e antecede producers
concorrentes; isso não prova mistura em toda execução nem demonstra cache atual
servindo stale. O delta responde aos gaps de identidade/revalidação identificados
nas análises, sem transformar um achado por inspeção em reprodução já executada
nesta interação. Tests/matriz, journals/answers, sujeitos, proofs/results, policy,
curadoria, índices, roots/refs e untracked pertinentes devem entrar na closure real.

| Dimensão | Contrato a qualificar |
|---|---|
| Coerência | Resultados derivam da mesma read view/closure pertinente, com protocolo comprovado; relabel final não corrige mistura anterior. |
| Completude | Coverage efetiva por produtor/recorte; discard/read failure impede Complete no escopo afetado. |
| Frescor | Bytes/identidade relevantes revalidados; mtime/tamanho/HEAD/dirty/horário e watcher isolados não bastam. |
| Autoridade atual | Gate/write revalida closure e authority perto do efeito, com CAS/fencing pertinente; snapshot histórico não autoriza por si. |

Escolher/provar protocolo de captura/revalidação e compatibilidade antes de start;
reconciliar ADRs existentes de obligations, bundles/federação/authority e publicação,
sem fabricar decisão arquitetural já aceita. Declarações abaixo são gates de execução,
não um novo ADR aceito nem implementação implícita de cache/daemon.

Captura com hashes antes/depois não observa necessariamente transição ABA ou
writer externo hostil. Definir quando equivalência de conteúdo basta e quando
versão monotônica/coordenação é necessária. Preservar bytes imutáveis observados,
validar closure após producers e antes do efeito; limitar retry/custo e devolver
incoherence/unknown quando não há prova. Writers externos que ignoram o protocolo
exigem limites honestos, sem promessa de transação universal sobre filesystem.
Watcher sugere invalidação; perda/overflow/restart exige reconciliação por fonte.
Federação exige bindings por root e limites/vetor de consistência explícitos,
sem supor um contador global compartilhado entre repos independentes.

B36 deve escolher por tool/query entre snapshot retido e recusa stale/restart;
ambas precisam de geração/filtros/ordem/escopo/autorização vinculados. Definir
proteção/versionamento dos tokens, TTL/bytes/eviction, limites e compatibilidade
com chamadas antigas e tokens de offset. Base64 não prova integridade. Cada
página revalida acesso; nenhuma query expande autoridade por possuir cursor.
Snapshot histórico pode servir navegação identificada, nunca gate atual sem B34.

Paginar só a resposta depois de construir toda a lista não prova menor custo.
Aplicar query sobre catálogo/índice adequado quando disponível e medir leituras,
parses/cópias, bytes e latência; total exato pode exigir scan integral e deve
ter custo explícito/opcional. Limitar exibição não limita histórico exigido por
um gate. Cursor de eventos de HTTP/Redis não é cursor de catálogo nem checkpoint
de operação. CLI one-shot deve funcionar sem depender de runtime B35 ausente.

Antes de executar: autorização funcional delimitada, readiness/assessment freshness,
reconciliação de artefatos e compatibilidade/ADRs, produtores de evidência atuais
na matriz e baseline pelo start. Qualificar interleavings, same-mtime/size, dirty
preexistente, untracked, rename/replace/checkout, perda de watcher e CAS em fixtures
isoladas. Para B36, qualificar mutações entre páginas, empates, query alterada,
tokens hostis, expiração/eviction e corpus indexável/não indexável. Nenhum desses
testes foi criado ou executado por esta escrita; suites da CI não satisfazem deltas
funcionais ainda não implementados.

## Roadmaps e sequência

[Consistência/recovery](../roadmaps/pose-state-consistency-and-recovery.md) possui
B34/B36 em milestones distintos; seus sete members planejados agora existem,
ainda draft e sem qualificação funcional. O [coordenador](../roadmaps/pose-consolidation-program.md)
possui a spec administrativa e consome os dois resultados por refs qualificados,
sem ownership duplicada. As camadas técnicas não são substituídas pela ordem de escrita.

Preservar 96 candidatos/44 unidades/17 lotes, aceites/DAG e oito reutilizadas.
Agora 23 propostas materializadas em draft e 14 ausentes, incluindo a condicionada;
nenhuma delivery_complete foi promovida. Lotes 10–16 permanecem futuros.
Próxima escrita: lote 10, B17/B18/B19/B21, com três specs substanciais conforme matriz.
Não contar as quatro unidades como quatro novas specs ou autorização funcional.

## Conferência reproduzível do snapshot do lote 9

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23","B34","B36"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==14
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md","B34":"pose-state-consistency-and-recovery.md","B36":"pose-state-consistency-and-recovery.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==9 and p["authorization"]["next_authoring_batch"]==10
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][9]["authoring_spec"]=="pose-consolidation-batch-9-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==23 and p["counts"]["unmaterialized_implementation_specs"]==14
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
assert p["last_authoring_revision"]=="9ed5b74e4784cad8193d89ed16b7e037d9554edf"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==167
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
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=10)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-13-pose-consolidation-batch-9-authoring.md",
            ".pose/specs/2026-10-13-pose-governance-read-view-generation.md",
            ".pose/specs/2026-10-13-pose-generation-bound-query-cursors.md",
            ".pose/roadmaps/pose-state-consistency-and-recovery.md",
            ".pose/roadmaps/pose-consolidation-program.md", ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-13-pose-batch-9-authoring.md"]:
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
print("PASS: 96 candidatos/44 unidades/17 lotes; 23 drafts, 14 propostas ausentes; oito reutilizadas preservadas; aceites/DAG/ownership/links íntegros")

```

## Trace e evidência da escrita

| Requisito documental | Evidência |
|---|---|
| R1 | Dois deltas draft com coverage dos cinco candidatos, artefatos/targets e casos prospectivos. |
| R2 | Conferência reproduzível: 96 candidatos/44 unidades/17 lotes; 23 propostas materializadas, 14 ausentes; oito specs reutilizadas; merge #167 registrado sem closeout fabricado. |
| R3 | Ownership única no responsável; resultados consumidos pelo coordenador; links relativos e grafo flat/folder conferidos. |
| R4 | Dependências/camadas preservadas; closure/revalidação, navegação histórica e autorização atual distintas. |
| R5 | Conferir os resultados efetivamente executados abaixo; lints não satisfazem requisitos funcionais. |

Conferência reproduzível e três lint-spec --strict passaram; Tasks opcional ausente
é aviso, não falha. A conferência adicional contra a baseline confirmou somente
B34/B36 alterados entre as unidades da matriz, camadas/policies/catalogo anteriores
intactos, dez paths documentais e existência dos paths prospectivos marcados modified.
Diff --check passou. Os checks locais usam o binário 7.4.1 identificado no lote 8;
o problema de compilador daquele ambiente não foi corrigido nem retestado por
esta escrita. Não alegar build fonte local ou entrega funcional atual.

A CI e Security de #167 concluíram success no head `7f68bb41a0284ce619d676f17749fd555064c918`:
[CI #732](https://github.com/oseiaspereira88/pose/actions/runs/38482776552) e
[Security #734](https://github.com/oseiaspereira88/pose/actions/runs/38482776493).
Essa evidência é do lote 8 mergeado, não qualifica o head do lote 9 nem os requisitos
funcionais de B34/B36. CI/Security verificarão este PR no head correspondente.
Validate --tolerant/SARIF não equivale a validate --strict sem findings.
A spec administrativa permanece in-progress até revisão e closeout nativo;
os dois deltas permanecem draft. Nenhuma delivery_complete foi inferida.

Artifact-check --strict da escrita (baseline ao commit documental) passou:
10 claims/10 paths observados, zero erros e 607 warnings orphan de paths governados
fora da spec filtrada. Não alegar zero findings nem corrigir/backfill de proveniência
externa neste lote. Roadmap-check --strict do coordenador recusou encerramento como
esperado: 152 refs resolved, terminal=false, 85 blockers de members/aceites pendentes.
Esses resultados validam atribuição/refs; não qualificam comportamento de read view,
watcher, cursores ou gates com reuso entre operações.

Check --strict passou: `Result: SUCCESS — (strict mode) with 30 warning(s).`. Avisos de changelogs históricos, freshness e instance stamp não foram apagados por update/reassessment fora de escopo. Diff --check passou.
