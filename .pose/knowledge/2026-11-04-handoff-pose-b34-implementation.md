---
type: handoff
slug: pose-b34-implementation
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-11-04
last_reviewed_at: 2026-11-05
expires_at: 2026-12-05
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

Última revisão: 2026-11-05. Spec `in-progress`, clone persistente
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
A matriz nativa completa e a revisão final ainda devem qualificar o checkpoint.

As correções integradas cobrem seleção do Store/resolver qualificado, lease
canônica dos resultados, diretórios staged por Open, separação de leases de
seeding/indexação, identidade explícita das fontes federadas e diagnóstico Git.
Nada disso torna prova histórica em autorização atual.

Notas locais de recuperação e logs estão em `recovery-notes/`, excluídos do Git.
Use os resultados nativos atuais e os commits com trailer da spec como evidência
publicável; não promova os relatos históricos deste handoff a testes atuais.

## Next checks

1. Registrar o checkpoint de implementação e compilar seu CLI exato.
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
- Limites aceitos: 10.000 registros, 128 MiB de inputs, 8 MiB de metadados,
  16 roots, profundidade 8 e uma repetição. Nenhum truncamento certifica completude.
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
