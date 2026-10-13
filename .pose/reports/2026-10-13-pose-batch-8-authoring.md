# POSE — escrita do lote 8, 2026-10-13

Baseline: `95a36d2b945253045bc0cb65e8e0bfba543b78d2`, merge do [#166](https://github.com/oseiaspereira88/pose/pull/166).
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec documental `pose-consolidation-batch-8-authoring` atribui onze paths.

## Artefatos e ownership

| Unidade / fonte | Spec de execução | Continuidade |
|---|---|---|
| B16 / D05, T3-01, T3-03 | [Ledger existente](../specs/2026-10-09-pose-delegated-review-attempt-ledger.md) | Preservar draft/R1–R4; reconciliar cota/reserva/idempotência antes de start na mesma spec. |
| B22 / F05-02, F05-04 | [Cobertura e identidade](../specs/2026-10-13-pose-residual-material-coverage-and-identity.md) | Delta draft de associação material, ocorrências e continuidade de vínculos. |
| B23 / F05-05, F05-06 | [Metadata e encaminhamento](../specs/2026-10-13-pose-followup-metadata-and-routing-integrity.md) | Delta draft de calendário/metadata e destinos reconciliados, sem disposition automática. |

Este lote cria duas specs de implementação, além da spec administrativa; não cria
segunda spec de ledger. Preservar byte a byte a spec reutilizada, R-IDs, status,
registros e baseline existentes. Os planos são prospectivos e não equivalem a
execução ou alteração retroativa do contrato. Não executar dispatch/agentes, testes
funcionais dos deltas, policy, adoção, release, migration de identidade ou closeout.

## Plano de continuidade — ledger / B16

Executar B06/B14 primeiro e preservar a dependência de dispatch. A spec existente
assume o contrato aceito e tem R1–R4; reutilizá-la. Seu header atualmente aponta
somente para dispatch e seus artefatos declaram somente a própria spec. Isso não
dispensa B06/B14 nem o contrato de cota/reserva da unidade. Antes de start, reconciliar
prospectivamente deps com `pose-source-coverage-integrity` e
`pose-writer-coordination-and-atomic-publication`, artefatos, compatibilidade,
assessment freshness, targets e produtores de evidência na mesma spec.
Não editar baseline ou iniciar a spec por inferência de código presente.

| Requisito existente | Continuidade planejada | Aceite |
|---|---|---|
| R1 | Ledger append-only/content-addressed e history-check sobre reescrita/remoção, integrado à reserva e inventário duráveis. | Corrupção/remoção conhecida é diagnosticada; não restaura orçamento ou aprovação. |
| R2 | Regra de rerun sobre bundle inalterado e reason registrado, integrada a idempotência e cota. | Reason registra exceção; não apaga rejeição, não aumenta cota nem promove aprovação. |
| R3 | Disclosure em verify inclui rejeições e incerteza da cobertura. | Rejeição conhecida permanece visível ao lado da aprovação; fontes inválidas não completam sequência. |
| R4 | Insights conserva runs/bundle, taxa por adapter e reruns; distinguir reserva/tentativa/execução e unknown. | Métricas têm unidade/denominador explícitos, sem contar tentativa interrompida como review concluído. |

R1–R4 sozinhos não especificam o aceite completo de T3-01. As extensões E1–E8
abaixo pertencem **ao plano deste relatório**, não são R-IDs já existentes nem
trace satisfeito. Antes de start, incorporá-las prospectivamente como requisitos
novos na mesma spec, preservando literalmente R1–R4 e registrando amendment/ADR
quando o mecanismo nativo ou a mudança estrutural exigir. Não criar outra spec
concorrente nem atribuir cota à existência do contrato append-only.

| Extensão prospectiva | Contrato/casos a incorporar |
|---|---|
| E1 — Escopo e cota | Definir chave de orçamento (projeto/worktree/scope/bundle/policy), K, limites e reset autorizado; reservar atomicamente antes de qualquer spawn. K=0/1/N e dispatches concorrentes devem iniciar no máximo K execuções no mesmo orçamento. |
| E2 — Reserva durável | Persistir intento/reserva sob coordenação/CAS de B14 antes do efeito externo; vincular operation/idempotency key, bundle digest e policy snapshot. Nenhum processo é iniciado sem reserva confirmada. |
| E3 — Setup e falha | Registrar setup failure, cancelamento, interrupção e crash como tentativa recuperável. Definir consumo conservador: reserva ocupa orçamento até reconciliação comprovada; erro ou absence de resultado não o restaura automaticamente. Histórico conhecido nunca é apagado. |
| E4 — Idempotência | Repetição da mesma chave/input devolve a reserva/run conhecida e não inicia outra execução; reutilização da chave com inputs diferentes é conflito. Chaves distintas competem pelo mesmo orçamento sob processos concorrentes. |
| E5 — Ambiguidade externa | Crash entre spawn e receipt/result produz estado unknown/recovery-required; não repetir spawn por ausência de outcome. Recuperar somente com evidência do runner/receipt ou adjudicação governada. Sem protocolo do adapter que prove o efeito, declarar limite de recuperação; não prometer exactly-once universal. |
| E6 — Integridade e inventário | Reescrita, truncamento, digest inválido, remoção de run/reserva ou ledger ilegível não restaura orçamento/aprovação. Verificar inventário/proveniência e marcar coverage desconhecida quando não há prova de completude. B06 divulga perda no recorte afetado; não selecionar aprovação anterior silenciosamente. |
| E7 — Rejeição e reason | Rejeição do mesmo bundle permanece nos gates/disclosure, inclusive após retry com reason, outcome aprovado ou exceção autorizada; validar reason e sua autoridade conforme contrato vigente, sem apagar/reescrever a negativa. Reason por si só não satisfaz o critério rejeitado. |
| E8 — Paridade e compatibilidade | Qualificar CLI/MCP/verify/Attention/insights no mesmo snapshot, legado sem retroinventar reservas/provas e limites de cota/durabilidade documentados. Executar repro D05 antes de ampliar/adotar delegação; nenhuma métrica ou assinatura inventa conclusão. |

Cobrir barreiras de concorrência por processos, CAS conflitante, retry duplicado,
crash antes/depois de reserva, setup e spawn; injetar falhas e clock. Testar
remoção/corrupção com inventário conhecido, rejeição/approval/reason e legado.
Race detector é complementar e não prova limite entre processos. Usar fixtures
isoladas e adapter sintético, jamais dispatch contra agentes reais nesta escrita.

Durabilidade depende das classes e plataformas qualificadas por B14. Ledger local
não impede um writer externo de apagar toda a raiz e também todas as referências
sem deixar prova: explicitar esse limite e exigir fonte de inventário/continuidade
confiável quando a policy promete detectar tal remoção. Ausência sem prova não é
orçamento novo nem aprovação válida; recuperação deve ser governada.

Plano de paths a fixar antes de start: spec reutilizada (modified), start record
(created se ausente), changelog funcional (created), dispatch e persistência em
`pose-mcp/internal/pose/review_dispatch.go`, verify em `pose-mcp/internal/pose/review_bundle.go`,
`pose-mcp/internal/cli/historycheck.go`, `pose-mcp/internal/pose/insights.go` e callers CLI/MCP (modified conforme inventário),
novo ledger/testes de concorrência/recuperação (created), matriz/manual pertinentes.
Conferir paths existentes e actions exatas; estes arquivos não são executados por
esta escrita. B17 cuida containment, B19/B21 proveniência, B20 fulfillment e B18
adoção; B16 não substitui esses contratos.

## Dependências, integração e ownership

B22 depende de B06 e continua precisão lexical/âncoras: menção a arquivo produz
associação candidata; cobertura exige compromisso sobre a ocorrência. P02/P03/P07
qualificam falsos positivos, colisões e identidade rastreada sob mudanças de
apresentação. IDs legados permanecem consultáveis; aliases/rename/migração exigem
prova e limites explícitos. Não escolher sintaxe definitiva de ID sem reconciliação
prospectiva do contrato e não inventar equivalência de ocorrências indistinguíveis.

B23 depende de B06/B22 e continua ownership-SLA/reconciliation: P05/P06 qualificam
calendário, metadata duplicada/incompleta, destinos órfãos/abandoned/superseded e
ciclos. Status done requer adequação e evidência; SLA permanece triagem. Nenhum
caso aplica disposition automaticamente ou transforma consulta em escrita.

[Delegated-review](../roadmaps/delegated-review.md) possui B16 no milestone
`attempt-ledger`; `trust` conserva proveniência e passa a depender desse resultado,
sem perder o gate histórico. [Governance-efficiency](../roadmaps/governance-efficiency.md)
possui B22/B23 em milestones distintos. O [coordenador](../roadmaps/pose-consolidation-program.md)
possui a spec administrativa e consome os três resultados por refs qualificados.
Nenhuma ownership duplicada ou dependência de lote documental substitui o DAG técnico.

Preservar 96 candidatos/44 unidades/17 lotes, aceites/DAG e oito specs reutilizadas;
21 propostas materializadas em draft e 16 ausentes, incluindo a condicionada.
Registrar merge do lote 7 e seu closeout nativo pendente; manter relatórios anteriores
como snapshots. Próxima escrita: lote 9, B34/B36. Nenhuma execução desses lotes autorizada.

## Conferência reproduzível do snapshot do lote 8

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==16
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==8 and p["authorization"]["next_authoring_batch"]==9
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][8]["authoring_spec"]=="pose-consolidation-batch-8-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==21 and p["counts"]["unmaterialized_implementation_specs"]==16
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
assert p["last_authoring_revision"]=="95a36d2b945253045bc0cb65e8e0bfba543b78d2"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==166
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
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=9)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-13-pose-consolidation-batch-8-authoring.md",
            ".pose/specs/2026-10-13-pose-residual-material-coverage-and-identity.md",
            ".pose/specs/2026-10-13-pose-followup-metadata-and-routing-integrity.md",
            ".pose/roadmaps/delegated-review.md", ".pose/roadmaps/governance-efficiency.md",
            ".pose/roadmaps/pose-consolidation-program.md", ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-13-pose-batch-8-authoring.md"]:
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
print("PASS: 96 candidatos/44 unidades/17 lotes; 21 drafts, 16 propostas ausentes; B16 preservada; aceites/DAG/ownership/links íntegros")

```

## Trace e evidência da escrita

| Requisito documental | Evidência desta escrita |
|---|---|
| R1 | B16/R1–R4 preservada byte a byte; E1–E8 prospectivos; B22/B23 draft com coverage dos quatro candidatos. |
| R2 | Conferência reproduzível: 96 candidatos/44 unidades/17 lotes; 21 propostas materializadas, 16 ausentes; oito specs reutilizadas; merge #166 registrado sem fabricar closeout. |
| R3 | Ownership única conferida; ledger separado de proveniência e resultados consumidos pelo coordenador. |
| R4 | DAG/camadas anteriores preservados; B16 após B06/B14, B22 após B06, B23 após B06/B22; extensão antes de start explícita. |
| R5 | Registrar abaixo somente os checks executados; não satisfaz requisitos funcionais. |

A conferência reproduzível e os quatro lint-spec --strict passaram; o aviso de
Tasks opcional ausente nas três specs novas não é falha. Conferência adicional
contra a baseline confirmou somente B16/B22/B23 alterados na matriz, camadas,
policies, catálogo e unidades restantes intactos e onze paths exclusivamente documentais.

Artifact-check --strict, por trailer e por range explícito da baseline ao commit,
conferiu 11 claims/11 paths observados, zero erros; o binário local 7.4.1 emitiu
607 warnings orphan de outros paths governados fora da spec filtrada. Não alegar
zero findings ou backfill de proveniência desses paths nesta escrita. Não ampliar
o lote para corrigir o engine ou o inventário histórico.

Roadmap-check do coordenador no snapshot commitado recusou encerramento como
esperado: 146 refs resolved, terminal=false e 78 blockers de aceites/members não
concluídos. Consultas anteriores enquanto a escrita estava em alteração não são
evidência do snapshot final; usar o resultado commitado. Links e grafo nativo
flat/folder foram conferidos, sem ciclos. Não confundir recusa de closeout com
falha de materialização ou implementação iniciada.

A compilação local do engine fonte foi tentada com Go 1.26.0 e falhou no toolchain
(`error obtaining buildID for go tool compile: signal: segmentation fault`). Os
lints e checks locais usam o binário 7.4.1; CI/Security verificarão o head no PR
com o toolchain configurado pelo repositório. Não alegar build local ou testes
funcionais dos deltas aprovados. Validate --tolerant/SARIF não equivale a validate
--strict sem findings. Spec administrativa permanece in-progress até revisão e
closeout nativo; specs de implementação permanecem draft.

Check --strict passou com 30 warnings de changelogs históricos, freshness e instance stamp; nenhum erro estrutural. Diff --check passou. Não executar update ou reassessment amplo nesta escrita para eliminar esses avisos.
