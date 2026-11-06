---
type: handoff
slug: pose-b34-implementation
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-11-04
last_reviewed_at: 2026-11-06
expires_at: 2026-12-06
source_refs:
  spec: "spec:pose-governance-read-view-generation"
  workflow: .pose/workflows/bugfix.md
  commands:
    - "go -C pose-mcp test -json ./... -count=1 -timeout=10m"
    - "pose lint-spec pose-governance-read-view-generation --strict"
  external_sources: []
---

# handoff: pose-b34-implementation

## Context

B34 corrige a identidade de leituras de governança que não distinguia duas
edições sob uma árvore já dirty. A autoridade é a spec do motor POSE; a adoção
no projeto hospedeiro permanece fora do escopo. O ADR de views limitadas por
operação está aceito e o start nativo foi aplicado.

## Current state

Atualização de fechamento em 2026-11-06: o código `5c9f1a96` passou 189/189
checks; índice real, artifact-check e surface-check passaram. O check estrito
global seguinte falhou por agregação das closures de closeout e diagnósticos
em cascata. D7 e a emenda aceita no ADR definem coordenador auditado e views
limitadas por escopo; a investigação nativa `trn-78de82fe660bb649` está concluída
e a task de implementação `b34-strict-scope-fix` está em execução com Sol medium.
Consultar `report:2026-11-06-pose-b34-global-review-view-budget.md`. Ainda não há
bundle final, revisão Claude, atestação ou closeout. Os registros abaixo conservam
os checkpoints anteriores e não qualificam a correção D7.

Última revisão: 2026-11-06. Spec `in-progress`, clone persistente
`/home/go/IdeaProjects/harne8/.worktrees/pose-b34-recovery`, branch
`work/governance-read-view-recovery`, base publicada `7079eb86`.

O clone temporário desapareceu. Patches e saídas literais das sessões permitiram
recuperar o código, o ADR e este handoff. O plano recuperado está em `22f777e9`;
o novo start prospectivo está em `b2eb2d34`. Ele não afirma precedência sobre o
código recuperado. Os recibos históricos completos de start/amendments e os
commits `56f5c150`, `3565be8a`, `dcffc886` não foram transportados ou fabricados.
O protocolo D4 está no ADR aceito e na baseline do novo start.

Compilação, `go vet` e seleção B34 com race detector passaram. A segunda suíte
completa da recuperação aprovou domínio e MCP; a CLI teve duas negativas cuja
fixture usava modo 0444 como barreira contra substituição atômica. A recusa foi
movida para a fronteira real do publisher, preservando bytes e asserções de efeito;
os quatro controles positivos/negativos relacionados passaram depois da correção.
A matriz completa passou 189/189 em `f54e9285`. Esse resultado é histórico para
os patches posteriores, não aprovação atual. O índice real encontrou dois
limites que os testes anteriores não reproduziam: profundidade federada oito
aplicada a toda cadeia local e capacidade insuficiente para a operação completa.
D5, commit `74f5762f`, separou as arestas locais dos cruzamentos federados e
incluiu redirects e deduplicação na fronteira. D6 recalibrou os limites com
medição versionada; seus testes normais focados e a integração ampliada com race
passaram. O novo índice real, a matriz atual e a revisão final continuam pendentes.

As correções integradas cobrem seleção do Store/resolver qualificado, lease
canônica dos resultados, diretórios staged por Open, separação de leases de
seeding/indexação, identidade explícita das fontes federadas e diagnóstico Git.
Nada disso torna prova histórica em autorização atual.

Notas locais de recuperação e logs estão em `recovery-notes/`, excluídos do Git.
Use os resultados nativos atuais e os commits com trailer da spec como evidência
publicável; não promova os relatos históricos deste handoff a testes atuais.

## Next checks

1. Registrar D6 e compilar seu CLI exato; repetir o índice real antes da matriz.
2. Executar a matriz nativa completa, incluindo Go, integração com `-race`,
   build, vet e checks de segurança aplicáveis.
3. Reconciliar artifacts e trace, commitar a implementação com o trailer
   `POSE-Spec`, compilar o CLI atual e gerar a evidência nativa da matriz.
4. Passar artifact-check, surface-check e check estrito. Só então selar o bundle
   e obter revisão de uma execução separada que não implementou o código final.
5. Aplicar o closeout nativo, repetir os gates pós-fechamento e publicar sem
   forçar o histórico. O CI remoto está dispensado nesta tarefa.

Para Go nesta máquina, usar o cache padrão e `GOTMPDIR`/`TMPDIR` em
`/var/tmp/pose-work-go` com a autorização de filesystem aplicável. O cache
descartável `/tmp/pose-b34-go-cache` esgotou a quota e foi limpo; não recriá-lo.

## Risks

- Não converter controles positivos de `current` em `unknown` para esconder
  falhas; tampouco promover índice legado ou prova histórica a autorização atual.
- Preservar o contrato de verificação histórica: metadado atual desconhecido
  não equivale automaticamente à invalidação de uma aprovação histórica;
  autoridade, trust e fontes necessárias ilegíveis ou revogadas continuam
  bloqueando o que seu contrato exigir.
- Um diretório comprovadamente sem Git pode sustentar um build de índices
  exclusivamente de filesystem; inputs materiais de refs/histórico continuam
  exigindo a prova Git correspondente.
- Limites recalibrados em D6: 25.000 registros agregados, 512 MiB de inputs,
  8 MiB de metadados, 16 roots, profundidade federada oito e uma repetição.
  D5 conserva 64 arestas de dependência e 256 identidades/entradas por grafo/lista.
  A calibração reteve 18.905 registros e cerca de 353,5 MiB em memória; não publicou
  índices e não equivale aos gates normais. Nenhum truncamento certifica completude.
- Writers externos e uma edição A→B→A que desaparece antes da auditoria não
  recebem garantia transacional. O fencing só cobre participantes do protocolo.
- O assessment de dívida lido nesta tarefa foi incompleto: duas specs históricas
  inválidas não puderam ser lidas. Não reportar esse scan como completo.

## Next owner

@pose-maintainers. A coordenação mantém integração e gates. Use `pose delegate`
para trabalho secundário e somente `gpt-6.1-sol` com reasoning médio, conforme
a orientação do usuário. Sucesso de task não é revisão governada ou atestação.

## References

- [Spec B34](../specs/2026-10-13-pose-governance-read-view-generation.md)
- [ADR aceito](../adr/2026-11-03-governance-reads-use-bounded-operation-scoped-views.md)
- [Start atual](../starts/pose-governance-read-view-generation.json)
