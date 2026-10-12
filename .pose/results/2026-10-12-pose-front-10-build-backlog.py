"""Rebuild the proposed cross-front backlog from versioned analyses.

Read-only except --output. Validates exhaustive candidate ownership, the DAG,
existing spec identities and baseline source hashes. Does not adjudicate issues,
close specs, adopt policy, or prepare releases. Run from the repository root.
"""
import argparse
import collections
import hashlib
import json
import pathlib
import re
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
a = parser.parse_args()
root = pathlib.Path.cwd()
baseline = '35ffb4a66817803c4cc42f56e41bf2e667e52da9'

# id | outcome | original candidate owners | wave | priority | existing foundations | prerequisites | decisive acceptance
ROWS = '''
B01|Prova persistida e disclosure coerentes|T2-01,T2-03|R1|P0|pose-action-request-resolution,pose-review-assurance-disclosure|B04|Adulteração de answer/actor/digest/proof não satisfaz condição; JSON/MCP/gate consomem a mesma prova sem promover legado.
B02|Preflight semântico de aprovação no domínio|T2-02|R1|P1|pose-review-bundle-convergence|B01|Approval inválida grava zero artefatos em Store/CLI/import/MCP; negativa legítima permanece registrável.
B03|Presença consistente entre emitters|T2-04|R2|P1|pose-native-attestation-issuer|B01,B04|Policy delimita presença e nenhum canal alternativo a dispensa; matriz SSH/SK/issuer verifica o contrato.
B04|Tempo de confiança e proveniência de reuso|T2-06,T2-07|R1|P1|pose-material-equivalence-reuse,pose-native-attestation-issuer||Separar validade histórica de autorização atual; clock injetável e fonte do reuso verificada, com ciclos/custo limitados.
B05|Scopes de policy e efetividade válidos|T2-08|R1|P1|pose-effective-governance-projection||Typo de scope não se apresenta como proteção efetiva; extensão desconhecida tem diagnóstico/versionamento.
B06|Leitura completa ou perda explicitada por fonte|T2-05,D01,D13,F05-01,F05-03|R1|P1|pose-obligation-projection,pose-debt-marker-lexical-precision||Journal/scan/parser ilegível ou descartado não vira zero completo; paridade de follow-ups e cobertura por recorte.
B07|Git desconhecido e snapshot de dispatch verificável|D02,T3-02,CONS-03|R1|P1|pose-recoverable-closeout-plan,pose-delegated-review-dispatch||Status com erro/assume-unchanged não certifica árvore intacta; paths e inputs selados são verificados antes/depois.
B08|Restrições por target, fase e identidade de nó|T4-01,T4-04|R1|P1|pose-phase-scoped-readiness,pose-governed-effect-enforcement||Cross-target e roadmap só restringem o destino correto; requests de origem terminal seguem visíveis; kind/ID resolvem exatamente.
B09|Autoridade de waiver e supersession|T4-02|R1|P1|pose-action-request-resolution,pose-action-requests|B01,B05|Dispensa e substituição não retiram condição de terceiro sem autoridade/prova exigidas; cancellation mantém a semântica própria.
B10|Recuperação e preflight de requests|T4-03,T4-05,CONS-01|R1|P1|pose-action-requests,pose-action-request-presentation|B09|Retry do mesmo ID completa supersession uma vez; invalidated/missing-subject/decline têm próximo passo suportado; preview verifica admissibilidade sem escrever.
B11|Fechamento semântico e histórico de amendments|D09,F07-01|R1|P1|pose-amendment-history-holds-before-close,pose-recoverable-closeout-plan|B08|Done/no-op não certifica review ausente; close antecipa gate de amendments e preserva baseline append-only e cutoffs.
B12|Resultado e parsing comuns CLI/MCP|F07-02,F07-03,F07-09,PERF-07,CONS-13|R1|P1|pose-recoverable-closeout-plan||Um JSON por chamada; flags/modos inválidos recusados; waiting/deadline/conflict/partial-commit distintos de verdict de domínio.
B13|Journals, checkpoints e recovery de locks|CONS-02,CONS-07,CONS-12|R1|P1|pose-recoverable-closeout-plan,pose-abm-atomic-start|B12|EOF/schema/seq integrais; checkpoint corrupt/unavailable não vira absent; owner vivo nunca tem lock quebrado por idade.
B14|Publisher exclusivo e coordenação/CAS entre writers|CONS-04,CONS-06,CONS-09|R2|P1|pose-spec-authority-transfer,pose-abm-atomic-start|B12,B13|Sem staging compartilhado ou lost update; curated preservada; efeito antes de erro tem receipt e precondições verificáveis.
B15|Geração completa dos índices|CONS-05|R2|P1|pose-delivery-integrity-index-compaction|B14|Leitor recebe conjunto coerente ou unavailable; falha/crash não certifica mistura de gerações; migração preserva consumers atuais.
B16|Ledger e reserva durável de tentativas delegadas|D05,T3-01,T3-03|R2|P1|pose-delegated-review-attempt-ledger|B06,B14|Cota K inicia no máximo K runs; setup/crash registrados; retry não duplica; rejeição/corrupção não restaura orçamento.
B17|Run assinado e independência observada|D04,T3-04|R3|P1|pose-delegated-review-run-provenance|B01,B04,B16|Bundle/prompt/output/executor/config vinculados; vendor OU modelo distinto conforme ADR; machine-attested não é pessoa.
B18|Isolamento, recursos e retenção do adapter|D06,T3-06|R3|P1|pose-delegated-review-dispatch|B07,B12|Worktree não é anunciada como sandbox; recursos/streams/descendentes/retention têm limites e qualificação por plataforma.
B19|Conclusões governadas de review, adjudication e smoke|T3-05,T3-07|R3|P1|pose-delegated-review-capability|B02,B16,B17,B18|Conclusão completa validada vai ao destino tipado pelo engine; seeded defect rejeitado; notas não modificam authority/criteria.
B20|Fulfillment por runner local e remoto|T3-08|R3|P1|pose-delegated-review-action-fulfillment|B08,B09,B10,B19|Request satisfeita só por run válido correlacionado; local/CI/Harne8 têm mesmo contrato; pessoa apenas por policy/exceção.
B21|Adoção e eficácia da delegação completa|T3-09,T3-10|R3|P2|pose-delegated-review-capability|B19,B20|Opt-in delimitado e rollback; três kinds atravessam jornada; rejeição exige mudança; docs/drafts seguem ADR aceito.
B22|Cobertura material e IDs de residual/marker|F05-02,F05-04|R2|P1|pose-debt-marker-lexical-precision,pose-covered-followup-anchor-verification|B06|Menção a arquivo é associação candidata; ocorrências distintas têm IDs distintos e vínculos sobrevivem rewrap/reordenação.
B23|Metadata válida e destino de encaminhamento reconciliado|F05-05,F05-06|R4|P1|pose-followup-ownership-sla,pose-followup-reconciliation-candidates|B06,B22|Datas reais; destino abandoned/órfão/superseded/cíclico sinalizado; done não aplica disposition sem adequação.
B24|Natureza, compromisso e efeito do trabalho residual|D07,F05-07|R4|P2|pose-obligation-contract,pose-followup-reconciliation-candidates|B22,B23|Dívida/defeito/incompletude/evidência/adoção/opcional distintos; legado não classificado; categoria não cria gate sozinha.
B25|Triagem revisável e risco aceito|F05-08,F05-09|R4|P2|pose-open-backlog-reconciliation|B01,B14,B24|108 itens reconciliados por diff/autoridade com razão e gatilho; preview/apply idempotentes sem disposition automática.
B26|Métricas de condição, resposta e trabalho|T4-08,F05-10|R4|P2|pose-governance-wait-rework-observability|B10,B24|Decline não encerra restrição; intervalos não duplicados; desconhecido não vira zero ou timestamp fabricado.
B27|Próxima ação proporcional e autorização preservada|T4-07,T4-09,T4-10,F07-07,F07-08|R4|P2|pose-action-request-presentation,pose-progressive-spec-surface|B08,B10,B12|Holder/executor/pessoa/advisory distintos; autorização vigente reaproveitada; quatro kinds retomam a fase pertinente.
B28|Instruções distribuídas executáveis|D11,F07-06|R1|P1|pose-adaptive-assessment-freshness,pose-v7-legacy-cleanup-plan||Workflow/skill/scaffold/pt-BR usam freshness comum; suggest revela regra ausente; migrate explica apply não disponível sem prometer 7.0 futura.
B29|Qualificar e reconciliar update dry-run|D10|R1|P1|pose-update-dry-run-reports-the-whole-update||Confrontar copy/real update, source discovery, symlink/permissão e trace; código presente não significa novo subsistema nem fechamento automático.
B30|Facts vinculados à spec e freshness|F07-05|R1|P1|pose-progressive-spec-surface||Check de outro módulo/digest é candidato não vinculado; facts não inventam conclusão semântica; reuso adequado preservado.
B31|API comum e handoff CLI/MCP|F07-04,PERF-06|R5|P2|pose-recoverable-closeout-plan|B12,B14,B34|Mesmos resultados/erros/authority pelo domínio; wrappers deixam de spawnar onde extraídos; stdio de cliente não é IPC compartilhável.
B32|Algoritmos por operação e benchmark seguro|D12,PERF-01,PERF-02,PERF-03,PERF-05|R2|P2|pose-check-spawns-fewer-git-processes,pose-attention-within-a-second||Perfil por produtor; catálogo/bundle/Git reusados; clustering preserva semântica; benchmark isolado preserva verdict e arquivos locais.
B33|Vistas de grafo e caches limitados/isolados|PERF-04,PERF-10,CONS-11|R5|P2|pose-delivery-integrity-index-compaction|B15,B34|Mutation/race tests profundos; bytes e misses coordenadas limitados; sem cópia/expansão total desnecessária ou contaminação cruzada.
B34|Read views, geração e invalidação de dependências|D03,T4-06,PERF-09,CONS-10|R2|P1|pose-obligation-projection,pose-spec-authority-transfer|B06,B07,B14,B15|Answer/subject/policy/tests/index/refs relevantes mudam geração; watcher é hint; interleaving causa retry/unknown, não autorização stale.
B35|Runtime opcional por worktree e budgets de concorrência|PERF-08,PERF-12|R5|P2|pose-attention-within-a-second|B07,B12,B14,B15,B31,B32,B33,B34|Handshake engine/schema/root, start/stop/restart, fallback CI/standalone e QoS testados; RAM/CPU/latência medidos por cenário.
B36|Cursor e consulta ligados à geração|PERF-11|R2|P1|pose-obligation-projection|B34|Mutação entre páginas retorna snapshot consistente ou cursor stale explícito; filtros/ordem vinculados; limit reduz trabalho quando indexável.
B37|Suficiência compartilhada do cut e critérios tipados|D08,F06-01,F06-02,F06-03|R2|P1|pose-release-lifecycle-closure|B01,B02,B08,B30|Plan/apply avaliam mesmo escopo; delivery sem evidência não passa; heading/ref não reconhecido é explicitado; path não é approval.
B38|Manifest íntegro e snapshot da tag|F06-04,F06-06|R1|P1|pose-release-lifecycle-closure||Schema/projeções redundantes recompostos; manifest/archive/notes/version source no tree da tag; história legítima não é invalidada por próximo ciclo.
B39|Replay temporal da evidência de publicação|F06-05,F06-09,F06-10|R2|P1|pose-release-lifecycle-closure,pose-reproducible-release-verification|B04,B38|Hashes/identidade/assets/coverage revalidados; candidate stale distinto de história íntegra; open-next só consome resultado validado.
B40|Seleção de escopo e exclusividade antes do freeze|F06-07,F06-08|R4|P1|pose-release-lifecycle-closure|B23,B24,B37,B38,B39,B41|Subconjunto coerente explícito; dupla assignment recusada antes de mover fila; exclusão/adiamento/outcome têm significado próprio.
B41|Prepare/record de release com recovery|CONS-08|R2|P1|pose-release-lifecycle-closure|B12,B13,B14,B38|Fases/inventário persistidos; retry evento não duplica; compensação falha preserva material; não fabricar manifest por backfill.
B42|Piloto integrado de UX e contratos adversariais|F07-10|R4|P2|pose-agency-readiness-pilot,pose-mechanization-adversarial-corpus|B21,B25,B26,B27,B37,B40|Oito percursos pessoa/agente com baseline/stop-go/rollback; medir trabalho útil/reuso; presença de campo não equivale a eficácia.
B43|Migração breaking somente após benefício e prova||R6|P3|pose-v7-legacy-cleanup-plan|B14,B34|Inventário atual por campo; dry-run sem write; apply com guard/recovery; significado/assinatura de história preservados; major não é reforma geral.
B44|Reconciliar narrativa documental e promessas de releases||R1|P2|pose-delegated-review-contract,pose-v7-legacy-cleanup-plan||História 6.x/7.0 preservada como contexto; relatório atual distingue entregue/adoção/eficácia; drafts não reabrem ADR aceito nem anunciam capability ausente.
'''

reports = {}
candidates = {}
for n in range(1, 10):
    p = next((root / '.pose/reports').glob(f'*front-{n:02}*.md'))
    reports[n] = p
    text = p.read_text()
    if n in (2, 3, 4):
        companion = next((root / '.pose/results').glob(f'*front-{n:02}*.json'))
        for item in json.loads(companion.read_text())['items']:
            candidates[item['id']] = {'id': item['id'], 'front': n, 'report': str(p.relative_to(root)),
                'title': item['title'], 'priority': item['priority'], 'evidence_level': item['evidence_level'],
                'acceptance_criteria': item['acceptance_criteria']}
    else:
        for line in text.splitlines():
            m = re.match(r'\| (D\d+|F0[567]-\d+|PERF-\d+|CONS-\d+) \|', line)
            if not m:
                continue
            cells = [s.strip() for s in line.strip('|').split('|')]
            key = cells[0]
            title_index = 2 if key.startswith(('F05-', 'F06-', 'F07-')) else 1
            candidates[key] = {'id': key, 'front': n, 'report': str(p.relative_to(root)),
                'title': cells[title_index], 'source_row': line,
                'evidence_level': 'mixed; consult source report', 'acceptance_criteria': [cells[-1]]}
assert len(candidates) == 96, len(candidates)

specs = {}
for p in (root / '.pose/specs').rglob('*.md'):
    text = p.read_text()
    slug = re.search(r'^slug:\s*(\S+)\s*$', text, re.M)
    status = re.search(r'^status:\s*(\S+)\s*$', text, re.M)
    if slug and status:
        key = slug.group(1)
        if key in specs:
            raise ValueError('spec identity conflict: ' + key)
        specs[key] = {'slug': key, 'status': status.group(1), 'path': str(p.relative_to(root))}

units = []
owned = {}
for row in ROWS.strip().splitlines():
    key, title, aliases, wave, priority, slugs, deps, acceptance = row.split('|')
    aliases = [s for s in aliases.split(',') if s]
    foundations = [specs[s] for s in slugs.split(',')]
    nonterminal = [s for s in foundations if s['status'] in ('draft', 'in-progress', 'blocked')]
    route = 'continue-existing-plus-scoped-delta' if nonterminal else 'new-scoped-delta-on-existing-foundation'
    if key == 'B29':
        route = 'reconcile-existing-implementation-and-close-only-with-evidence'
    if key == 'B43':
        route = 'defer-breaking-apply; inventory-already-exists'
    if key == 'B44':
        route = 'editorial-reconciliation; preserve-historical-records'
    for alias in aliases:
        assert alias in candidates, alias
        assert alias not in owned, alias
        owned[alias] = key
    units.append({'id': key, 'title': title, 'status': 'proposed-not-adjudicated',
        'source_candidates': aliases, 'wave': wave, 'priority_proposed': priority,
        'foundations': foundations, 'implementation_route': route,
        'depends_on': [s for s in deps.split(',') if s], 'decisive_acceptance': acceptance,
        'source_acceptance': {s: candidates[s]['acceptance_criteria'] for s in aliases},
        'proposed_owner_role': 'maintainer/domain-owner; executor assigned at spec start',
        'compatibility': 'classify actual diff and consumers before choosing final SemVer; no automatic adoption'})
assert set(owned) == set(candidates), sorted(set(candidates) - set(owned))
by_id = {u['id']: u for u in units}
seen, active = set(), set()
def visit(key):
    assert key in by_id, key
    assert key not in active, 'dependency cycle: ' + key
    if key in seen:
        return
    active.add(key)
    for dep in by_id[key]['depends_on']:
        visit(dep)
    active.remove(key)
    seen.add(key)
for u in units:
    visit(u['id'])
    assert all(int(by_id[d]['wave'][1:]) <= int(u['wave'][1:]) for d in u['depends_on']), u['id']

previous = json.loads((root / '.pose/results/2026-10-11-pose-front-01-reconciliation.json').read_text())
old = []
for item in previous['items']:
    current = specs[item['existing_spec']['slug']]
    old.append({'previous_id': item['previous_id'], 'title': item['title'],
        'baseline_classification': item['classification'], 'existing_spec_now': current,
        'treatment': 'retain delivered foundation; residual/new gaps are separately mapped, not reopened wholesale',
        'limits': item['residual_or_limit']})

wave_titles = {'R1': 'Verdade e correções compatíveis', 'R2': 'Consistência e contratos comuns',
    'R3': 'Revisão delegada completa', 'R4': 'Residual, cut e experiência comprovados',
    'R5': 'Domínio compartilhado e runtime opcional', 'R6': 'Migração breaking condicionada'}
versions = {'R1': '7.4.2 only if compatible; otherwise first corrective minor', 'R2': '7.5.0 candidate',
    'R3': '7.6.0 candidate', 'R4': '7.7.0 candidate', 'R5': '7.8.0 candidate', 'R6': '8.0.0 only if justified; unscheduled'}
waves = [{'id': w, 'outcome': title, 'version_hint': versions[w], 'status': 'proposal-not-release-manifest',
          'units': [u['id'] for u in units if u['wave'] == w]} for w, title in wave_titles.items()]
paths = {str(p.relative_to(root)) for p in reports.values()}
paths.update(c['report'] for c in candidates.values())
paths.update(s['path'] for u in units for s in u['foundations'])
paths.update(s['path'] for s in specs.values() if s['status'] in ('draft', 'in-progress', 'blocked'))
paths.update(['.pose/results/2026-10-11-pose-front-01-reconciliation.json',
    '.pose/results/2026-10-12-pose-front-05-residual-work.json',
    '.pose/adr/2026-10-09-delegated-review-is-an-adapter.md',
    '.pose/roadmaps/record-truthfulness.md', '.pose/roadmaps/agency-readiness.md',
    '.pose/roadmaps/governance-efficiency.md', '.pose/roadmaps/epistemic-lifecycle.md',
    '.pose/roadmaps/delegated-review.md', '.pose/roadmaps/pose-v7-onboarding-and-consolidation.md',
    'pose-mcp/internal/cli/migrate_dryrun.go', 'scripts/bench-governance.sh',
    '.pose/workflows/feature.md', 'docs-site/docs/obligations.md'])
sources = []
for path in sorted(paths):
    raw = subprocess.check_output(['git', 'show', baseline + ':' + path], cwd=root)
    assert hashlib.sha256(raw).hexdigest() == hashlib.sha256((root / path).read_bytes()).hexdigest(), path
    sources.append({'path': path, 'sha256': hashlib.sha256(raw).hexdigest(),
        'url': f'https://github.com/oseiaspereira88/pose/blob/{baseline}/{path}'})
assert subprocess.run(['git', 'diff', '--quiet', 'v7.4.1', baseline, '--', 'pose-mcp'], cwd=root).returncode == 0

output = {'schema_version': 1, 'document_type': 'proposed-consolidated-backlog', 'front': 10,
    'source_revision': baseline, 'engine_version': '7.4.1', 'status': 'proposed-not-adjudicated',
    'counts': {'source_candidates': len(candidates), 'canonical_units': len(units), 'old_items_reconciled': len(old),
        'by_wave': dict(collections.Counter(u['wave'] for u in units)),
        'old_classifications': dict(collections.Counter(i['baseline_classification'] for i in old))},
    'source_candidates': list(candidates.values()), 'candidate_ownership': owned, 'units': units,
    'waves': waves, 'previous_33_reconciliation': old,
    'nonterminal_specs': sorted([s for s in specs.values() if s['status'] in ('draft', 'in-progress', 'blocked')], key=lambda s: s['slug']),
    'sources': sources, 'checks': {'exhaustive_unique_ownership': True, 'acyclic_dependencies': True,
        'dependencies_do_not_require_later_wave': True, 'foundation_paths_statuses_verified': True,
        'engine_sources_unchanged_since_published_7_4_1': True},
    'limits': ['Proposals are not adopted policy, adjudicated dispositions, issues or prepared releases',
        'No Go build/suite/race detector, external vendor execution or rollout in this front',
        'Historical report findings retain their original evidence level and environment',
        'No fabricated estimate, owner assignment or due date; wave order is a dependency proposal']}
pathlib.Path(a.output).write_text(json.dumps(output, ensure_ascii=False, indent=2) + '\n')
print(json.dumps(output['counts'], ensure_ascii=False))
