# POSE — escrita do lote 10, 2026-10-13

Baseline: `32626ca12258d135528421cb6b62a3a81e95df20`, merge do [#168](https://github.com/oseiaspereira88/pose/pull/168).
Data documental em America/Fortaleza; timestamps dos registros nativos permanecem UTC.
Consultar o [plano autoritativo](../results/pose-implementation-plan.json).
A spec documental `pose-consolidation-batch-10-authoring` atribui nove paths.

## Artefatos e ownership

| Unidade / fonte | Spec de execução | Continuidade |
|---|---|---|
| B17 / D04, T3-04 | [Proveniência existente](../specs/2026-10-09-pose-delegated-review-run-provenance.md) | Preservar draft/R1–R4; completar binding e reconciliação com ADR aceito. |
| B18 / D06, T3-06 | [Contenção nova](../specs/2026-10-13-pose-delegated-adapter-containment.md) | Delta draft de policy, recursos, descendentes, cleanup e retenção por plataforma. |
| B19 / T3-05, T3-07 | [Capability existente](../specs/2026-10-09-pose-delegated-review-capability.md) | Fase de conclusões governadas e inputs/authority; preservar R1–R7. |
| B21 / T3-09, T3-10 | Mesma capability de B19 | Fase posterior de adoção/eficácia integrada, após B19/B20; não duplicar spec. |

Quatro unidades, três specs substanciais: uma nova e duas reutilizadas. Esta
escrita não inicia execução, dispatch, vendor CLI, sandbox, migração, adoção,
policy, release, dispositions ou closeouts. Preservar byte a byte as duas specs
reutilizadas, R-IDs/status e registros/baselines existentes, além das outras seis
reutilizadas. Planos abaixo são prospectivos e não promovem delivery_complete.

## Contrato e gates de execução

O [ADR de delegação](../adr/2026-10-09-delegated-review-is-an-adapter.md) está aceito
desde 2026-10-09. O engine registra conclusão validada; pessoa intervém apenas
pelas exceções da policy; vendor OU modelo distinto satisfaz a regra aceita, com
vendor distinto preferido; review/adjudication/smoke pertencem à primeira entrega
completa. Não reabrir essas decisões nem substituí-las por apply manual universal.

As specs de proveniência e fulfillment ainda contêm texto de ADR provisório;
capability já incorpora as decisões. Corrigir metadata/plano/Decisions pertinentes
prospectivamente antes do start, preservando R1–R4/R1–R7 e evidências históricas.
B20 conserva a escrita no lote 11; não alterar sua spec ou formalizar sua execução
nesta interação. O roadmap reconcilia a narrativa atual com o ADR, preservando
contratos done e gates históricos, sem declarar eficácia entregue.

Antes de implementar: escopo funcional explícito, readiness e assessment freshness,
reconciliação de artefatos/dependências/targets, compatibilidade/ADRs pertinentes,
produtores de evidência atuais na matriz e baseline pelo start. Se uma extensão
mudar contrato material, registrar requirements novos/amendment/ADR antes do código,
sem apagar baseline ou atribuir decisão a data retrospectiva. Lint desta escrita
não satisfaz os requisitos funcionais ou os checkpoints planejados.

## Plano de continuidade — proveniência / B17

B17 exige B01/B04/B16 e as fundações de dispatch/issuer. Header hoje aponta somente
para dispatch/issuer; reconciliar os dois deltas de B01, tempo/proveniência de B04
e ledger de B16 antes de start. Não criar issuer paralelo.

| Requisito existente | Continuidade planejada |
|---|---|
| R1 | Run assinável por issuer pinned, com brief/transcript/adapter/vendor/model/bundle vinculados; completar P1–P3 abaixo. |
| R2 | Attestation referencia run; verificador resolve fonte, integridade, autoridade e vendor/model desse run. |
| R3 | Vendor OU modelo diferente com prova aceita; mesmo vendor/model recusa, baseline desconhecida não vira diferença provada; prefixo não satisfaz na instância adotante. |
| R4 | Disclosure machine-attested distinto de declared e confirmação assinada de pessoa; assinatura não decide mérito semântico. |

P1–P6 são IDs de extensão **deste relatório**, não R-IDs já incorporados ou trace
satisfeito. Incorporar prospectivamente à mesma spec antes da execução.

| Extensão prospectiva | Contrato/caso a incorporar |
|---|---|
| P1 — Binding integral | Vincular projeto/worktree/scope/kind/bundle, policy/config version/digest, argv/executor/adapter, identidades de implementação/revisão, template/notas/inputs, timestamps e outcome ao run. Rejeitar fonte/claim/output divergentes ou contexto não autorizado. |
| P2 — Modelo observado e limites | Divulgar fonte e assurance de vendor/model/executor. Assinatura do executor vincula sua declaração; não é receipt independente do fornecedor sem evidência dele. Policy deve delimitar confiança exigida; config declarada, whitespace ou baseline unknown não provam por si independência. |
| P3 — Canonicalização e conclusão | Distinguir brief body digest do footer e bytes transmitidos; vincular transcript e output final reais, que podem divergir. Documentar redaction/digests/assinatura e canonicalização; truncamento/incompletude não produz prova de conclusão integral. |
| P4 — Verificação atual | Reusar cadeia B01/B04/issuer: principal/pin/audience/scope/tempo/proveniência aplicáveis, fontes e integridade na leitura. Prova inválida/alterada/incompleta não satisfaz different-actor ou gate; não reconstruir prova para legado declared. |
| P5 — Linhagem e rejeição | Referenciar ledger B16 e preservar tentativas/rejeições conhecidas; assinatura válida não restaura cota ou apaga rejeição. Bundle/inputs superados não produzem efeito atual; exceptions/reason não equivalem a aprovação. |
| P6 — Matriz e paridade | Qualificar mesmo vendor/model, mesmo vendor/model diferente, vendor diferente, prefix-only, issuer não autorizado, proof/output/config adulterados e baseline desconhecida. CLI/MCP/verify/Attention divulgam a mesma assurance; machine-attested não satisfaz requisito de pessoa. |

Plano de paths a fixar antes de start: spec existente (modified), seu start record
(created se ausente), changelog funcional (created), `pose-mcp/internal/pose/review_dispatch.go`,
`review_bundle.go`, `review_assurance.go`, `review_closeout.go`, `issuer_key.go`
e callers CLI/MCP efetivamente afetados (modified), testes de proveniência e
paridade/negativos (created ou modified conforme inventário), matriz/manual.
Conferir actions/paths exatos e schemas antes do código; esta escrita não executa
nenhum deles. Não assinar outputs com chave de pessoa para simular revisão humana.

## B18 — contenção e recursos

B18 exige B07/B12, reutiliza dispatch e é raiz independente de B16 na matriz.
Verificar capabilities exigidas versus backend/plataforma; worktree não impede
filesystem/rede/credenciais externos e process group Unix não controla todo filho
que escapa. Policy que exige garantia indisponível deve recusar antes de spawn.
Modo de adapter confiável divulga limites, sem enfraquecer checks de imutabilidade
ou alterar unilateralmente o ADR aceito. Backend/controles devem ser decididos e
qualificados prospectivamente, sem impor fornecedor ou container obrigatório.

Qualificar valores inválidos/zero/ausentes sem desligar limites silenciosamente,
streams/output grandes, paths/symlinks/special files, ambiente mínimo/argv, timeout,
filhos, cleanup e retention. Não chamar teste Unix ou build Windows de garantia
Windows entregue. Falha de cleanup é outcome/receipt fiel, não retry automático de
spawn. Retenção preserva inventário/cota/rejeições e prova necessária a B16/B17;
redaction/assinatura exigem ordem explícita. Nenhum adapter real foi executado aqui.

## Plano de continuidade — capability / B19 e B21

Uma spec, duas fases de responsabilidade e um closeout final. B19 exige
B02/B16/B17/B18; B21 exige B19/B20. Manter R1–R7 literalmente e reconciliar artefatos,
metadata/ADRs e produtores antes de start. Header atual aponta somente para
proveniência/ledger; o plano nativo prospectivo acrescenta preflight B02 e contenção
B18, sem incluir a própria capability ou fulfillment como pré-condição de seu start.

| Requisito existente | Fase/qualificação prospectiva |
|---|---|
| R1 | B21: opt-in com adapter apto, data/cutoff reais e preservação de instâncias antigas; configuração nominal não prova aptidão. |
| R2 | B19/B21: contrato/examples/assurance/notas/limites seguem ADR; não anunciar integração externa não testada. |
| R3 | B19/B21: skill orienta brief gerado, rejeição respondida por mudança e exceções por policy; não cria rubber stamp ou prompter manual obrigatório. |
| R4 | B19 prova conclusões/destinos com fixtures; B21 qualifica jornadas integradas após B20, seeded defect → rejeição → mudança → novo bundle → close, além de escalations de pessoa. |
| R5 | B19: engine registra approval/rejection válidas de review após checks e preflight, sem passo manual; erro/output incompleto não produz efeito governado. |
| R6 | B19 define/classifica exceções; B20 conecta requests/fulfillment; B21 verifica eficácia. Compartilhar contrato de ActionRequests, sem implementar dois escaladores concorrentes. |
| R7 | B19: adjudication/smoke têm validadores e destinos tipados; B21 qualifica os três kinds nas jornadas completas. |

C1–C10 são extensões de **plano deste relatório**, não novos R-IDs vigentes.
Incorporar/reconciliar prospectivamente no mesmo contrato antes de execução.

| Extensão prospectiva | Contrato/caso a incorporar |
|---|---|
| C1 — Resultado tipado | Envelope versionado com kind, scope/bundle/inputs, decisão, critérios/tools, findings/dispositions/evidências/escalations e destino autorizado. Vazio/truncado/foreign/critério faltante/contradição não grava no destino nem satisfaz request/gate; tentativa/diagnóstico B16 continua registrada. |
| C2 — Engine e pré-condições | Engine aplica a conclusão válida usando preflight compartilhado de B02, prova B01/B04/B17, ledger B16 e contenção B18. Processo completed/exit zero/assinatura não equivale a approval; negativa válida também é gravada e bloqueia bundle inalterado. |
| C3 — Três destinos | Review → attestation; adjudication → decision record da questão; smoke → evidência do target. Validar schema, bindings, evidence classes e autoridade por kind, não traduzir prosa arbitrária em approval nem escolher destino pelo output do agente. |
| C4 — Efeito e recovery | Commit idempotente/receipt da conclusão e correlação da tentativa; replay da mesma chave não duplica, conflito de inputs ou bundle stale impede efeito. Setup/cancel/crash/erro parcial usa outcome fiel; não repetir agente para compensar write ou cleanup. |
| C5 — Mérito e negativos | Qualificar seeded defect, rejection/approval, conclusão incompleta/contraditória, tool/evidência errada, bundle antigo e replay; assinatura válida de conteúdo semanticamente inválido não passa o preflight. Rejeição exige mudança; reason por si só não a resolve. |
| C6 — Inputs selados | Materializar refs/bytes verificáveis do snapshot, diff/docs/evidence e smoke requirements/targets via parser canônico/layouts suportados. Mudança de inputs/template/notas altera identidade adequada; smoke não recomputa de spec corrente sem binding selado. |
| C7 — Autoridade no engine | Notas/diff/docs são dados identificados, não autorização para tool, scope, criteria ou destino. Texto adversarial não muda policy/permissions no engine; não prometer obediência absoluta do modelo ou executar notas como shell. |
| C8 — Disclosure de evidência | Expor module/provenance/subject/freshness/coverage, carried-forward/inferred/unknown e fontes resolvíveis; incompletude permanece localizada. Formatos não suportados têm diagnóstico, não smoke completo fabricado. |
| C9 — Adoção integrada | Só após B19/B20, qualificar adapter apto, opt-in explícito/off default, compatibilidade/cutoffs, rollback e journeys dos três kinds, standalone/local e runner CI pelo mesmo contrato. Harne8 requer demonstração de sua fronteira real antes de anunciar integração pronta. |
| C10 — Eficácia e exceções | Divulgar tentativas/rejeições/escalations, denominadores e unknown; verificar critérios de pessoa, conflito, decisão de negócio e cota esgotada com perguntas/opções. Run de agente não satisfaz pessoa requerida; operação ordinária validada não exige apply manual universal. |

### Fases e gate executável antes da execução

A sequência canônica é **B19 → B20 → B21**. B19 e B21 usam a mesma spec de capability;
B20 continua na spec de fulfillment existente do lote 11. Não criar uma nova spec
por fase nem mudar o DAG/camadas para resolver um problema de projeção em slugs.

Antes da execução correspondente, formalizar e qualificar no engine um checkpoint
corrente do recorte B19: resultado dos requisitos R5/R7 e dos contratos C1–C8
pertinentes, vinculado ao hash de requisitos/config/inputs/revisão e a evidência
válida/review independente. Ele demonstra prontidão da integração de B20, sem
certificar a capability inteira ou mudar seu status para done. Não usar arquivo
ad hoc, texto satisfied, boolean da matriz ou saída de processo como autorização.
Se os gates nativos atuais só exprimirem status da spec inteira, implementar e
qualificar o gate de fase dentro do escopo funcional autorizado da capability
antes de iniciar B20; este relatório não afirma que esse suporte já existe.

B20 deve exigir esse aceite específico atual e os gates adotados, além dos seus
próprios B08/B09/B10, sem exigir capability done. B21 exige B19/B20 qualificados e
as jornadas/adoção/rollback; só então o contrato completo R1–R7 pode atravessar
review e closeout nativo. Aceite intermediário não anuncia delivery_complete final
nem contorna policy/authority. Evitar a dupla espera: capability depende de
fulfillment para qualificação final, e fulfillment não pode depender da closure
global da mesma capability. Testar stale/invalid checkpoint e ordem invertida
para demonstrar que o protocolo impede avanço prematuro sem se tornar circular.

Plano de paths a fixar na spec capability antes de start: sua spec (modified),
start/changelog (created se ausentes), `pose-mcp/internal/pose/review_dispatch.go`,
`review_bundle.go`, `review_brief.go`, `review_closeout.go`, `capability_catalog.go`,
`capabilities.go`, `action_request.go`/`action_resolution.go` se pertinentes e handlers CLI/MCP/validators dos
três kinds (modified conforme inventário), checkpoint/resultado de fase e testes
(created ou modified conforme contrato escolhido), skill/docs/examples e jornadas
(created), matriz/manual. Conferir paths reais e registrar actions exatas; nenhum
desses arquivos é executado/criado sob a spec funcional nesta escrita.

## Roadmaps, contagens e próxima escrita

[Delegated-review](../roadmaps/delegated-review.md) possui proveniência em trust,
contenção em milestone próprio e capability/fulfillment em adoption uma vez. O
[coordenador](../roadmaps/pose-consolidation-program.md) possui a escrita administrativa
e consome resultados, sem copiar members. B17/B18/B19/B21 conservam seus prereqs;
B18 não depende artificialmente de ledger, e adoption não fecha antes de B20.
O gate histórico de adoption é mantido; referências existentes não provam eficácia.

Preservar 96 candidatos/44 unidades/17 lotes, aceites/DAG e oito reutilizadas.
Agora 24 propostas materializadas em draft e 13 ausentes, incluindo a condicionada.
B19/B21 não recebem proposed_specs e compartilham a mesma spec/ownership. Registrar
merge do lote 9 e closeout nativo pendente; relatórios anteriores são snapshots.
Próxima escrita: lote 11, B20/B27/B24. Nenhuma execução funcional futura autorizada.

## Conferência reproduzível do snapshot do lote 10

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
assert len(reused)==8 and len(proposed)==37 and materialized=={"B04","B02","B05","B03","B06","B07","B08","B09","B10","B12","B13","B30","B11","B28","B44","B38","B14","B15","B32","B22","B23","B34","B36","B18"}
assert sum(s["conditional"] for s in proposed.values())==1
present={re.search(r"^slug:[ \t]*(\S+)",f.read_text(),re.M)[1] for f in (root/".pose/specs").glob("*.md")}
assert all(slug in present if s["materialized"] else slug not in present for slug,s in proposed.items())
assert sum(not s["materialized"] for s in proposed.values())==13
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
    assert owners==[{"B06":"agency-readiness.md","B07":"pose-state-consistency-and-recovery.md","B08":"agency-readiness.md","B09":"agency-readiness.md","B10":"agency-readiness.md","B12":"pose-state-consistency-and-recovery.md","B13":"pose-state-consistency-and-recovery.md","B30":"governance-efficiency.md","B11":"governance-efficiency.md","B28":"pose-consolidation-program.md","B44":"pose-consolidation-program.md","B38":"pose-release-sufficiency-and-recovery.md","B14":"pose-state-consistency-and-recovery.md","B15":"pose-state-consistency-and-recovery.md","B32":"pose-shared-runtime.md","B22":"governance-efficiency.md","B23":"governance-efficiency.md","B34":"pose-state-consistency-and-recovery.md","B36":"pose-state-consistency-and-recovery.md","B18":"delegated-review.md"}.get(uid,"record-truthfulness.md")], (slug,owners)
assert p["authorization"]["current_authoring_batch"]==10 and p["authorization"]["next_authoring_batch"]==11
assert p["authorization"]["future_batch_implementation_authorized"] is False
assert p["batches"][0]["authoring_merged"] and p["batches"][0]["native_closeout_pending"]
assert p["batches"][10]["authoring_spec"]=="pose-consolidation-batch-10-authoring"
assert p["batches"][1]["authoring_merged"] and p["batches"][1]["merged_pr"]==160 and p["batches"][1]["native_closeout_pending"]
assert p["counts"]["materialized_implementation_specs"]==24 and p["counts"]["unmaterialized_implementation_specs"]==13
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
assert p["last_authoring_revision"]=="32626ca12258d135528421cb6b62a3a81e95df20"
assert p["authoring_history"][-2]["phase"]=="merged" and p["authoring_history"][-2]["pr"]==168
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
assert not any(u["proposed_specs"] and any(s["materialized"] for s in u["proposed_specs"]) for u in p["units"] if u["authoring_batch"]>=11)
# Conferir links relativos e dependências nativas sem ciclos nas specs existentes.
for rel in [".pose/specs/2026-10-13-pose-consolidation-batch-10-authoring.md",
            ".pose/specs/2026-10-13-pose-delegated-adapter-containment.md",
            ".pose/specs/2026-10-09-pose-delegated-review-run-provenance.md",
            ".pose/roadmaps/delegated-review.md",
            ".pose/roadmaps/pose-consolidation-program.md", ".pose/reports/2026-10-11-pose-analysis-index.md",
            ".pose/reports/2026-10-13-pose-batch-10-authoring.md"]:
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
for uid in ["B16","B17","B19"]:
    graph[units[uid]["reused_specs"][0]["slug"]]=units[uid]["native_spec_dependencies_before_start"]
visited.clear();active.clear()
for s in graph:visit(s)
slug=units["B18"]["proposed_specs"][0]["planned_slug"]
assert not (root/f".pose/starts/{slug}.json").exists()
assert not (root/f".pose/changelogs/unreleased/{slug}.md").exists()
assert "reuse_plan_formalized" not in units["B20"]
print("PASS: 96 candidatos/44 unidades/17 lotes; 24 drafts, 13 propostas ausentes; oito reutilizadas preservadas; fases B19/B20/B21 e DAG/ownership/links íntegros")

```

## Trace e evidência da escrita

| Requisito documental | Evidência |
|---|---|
| R1 | B18 draft e dois planos de continuidade; coverage dos oito candidatos nas quatro unidades; R1–R4/R1–R7 reutilizados preservados. |
| R2 | Conferência reproduzível: 96 candidatos/44 unidades/17 lotes; 24 propostas materializadas, 13 ausentes; oito reutilizadas e merge #168 sem closeout fabricado. |
| R3 | Ownership única em delegated-review; consumo pelo coordenador; B19/B21 compartilham spec uma vez. |
| R4 | DAG/camadas anteriores e DAG nativo prospectivo sem ciclos; fases B19 → B20 → B21 explícitas, com gate executável ainda prospectivo. |
| R5 | Registrar abaixo somente checks efetivamente executados; não satisfaz requisitos funcionais dos deltas. |

A conferência reproduzível e quatro lint-spec --strict passaram; Tasks opcional
ausente nas duas specs novas é aviso, não falha. Conferência adicional contra a
baseline confirmou apenas B17/B18/B19/B21 alterados entre as unidades da matriz,
DAG/policies/fundações e unidades anteriores preservados, bytes das duas specs
reutilizadas intactos e paths prospectivos modified existentes. B20 não ganhou
plano de continuidade ou alteração de contrato nesta escrita. Diff --check passou.

Checks locais usam o binário 7.4.1, como nos lotes 8/9; o problema de compilador
não foi corrigido/retestado nesta escrita. Não alegar build fonte local ou testes
funcionais atuais. CI/Security verificarão o head deste PR; validate --tolerant/SARIF
não equivale a validate --strict sem findings. A spec administrativa permanece
in-progress até review/closeout nativo, e as três specs de execução permanecem draft.

A CI e Security de #168 concluíram success no head `abaa326fa29f1d36289e3afc98515ca3355d249e`:
[CI #734](https://github.com/oseiaspereira88/pose/actions/runs/38484202409) e
[Security #736](https://github.com/oseiaspereira88/pose/actions/runs/38484202336).
Essa evidência é do lote 9 mergeado, não do head do lote 10 ou de suporte funcional
à delegação/checkpoint/containment. Nenhum agente/adapter externo foi executado aqui.

Artifact-check --strict da escrita passou: 9 claims/9 paths observados, zero erros
e 607 warnings orphan de outros paths governados fora da spec filtrada. Não alegar
zero findings nem efetuar backfill de proveniência externa neste lote.
Roadmap-check --strict do coordenador recusou encerramento como esperado:
161 refs resolved, terminal=false e 95 blockers de members/aceites pendentes.
A resolução das referências, inclusive adoption com B20 ainda draft, não qualifica
fulfillment, gate de fase, independência ou adoção integrada. Nenhuma dependência
circular foi introduzida no grafo atual ou no plano nativo prospectivo conferido.

Check --strict passou: `Result: SUCCESS — (strict mode) with 30 warning(s).`. Avisos históricos de changelogs, freshness e instance stamp foram preservados, sem update/reassessment fora de escopo. Diff --check passou.
