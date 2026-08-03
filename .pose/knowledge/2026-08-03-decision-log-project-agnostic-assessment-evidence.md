---
type: decision-log
slug: project-agnostic-assessment-evidence
owner: @pose-maintainers
sensitivity: public-internal
created_at: 2026-08-03
last_reviewed_at: 2026-08-03
expires_at: 2026-09-02
source_refs:
  spec: "pose-project-agnostic-assessment-engines"
  workflow: "bugfix"
  commands: ["go -C pose-mcp test ./...", "go -C pose-mcp vet ./...", "pose assess discover --update-state", "pose assess integrate --update-state", "pose assess tech-debt --update-state"]
  external_sources: []
---

# decision-log: project-agnostic-assessment-evidence

## Contexto

As implementações originais de discovery, integration e technical-debt
misturavam APIs genéricas com nomes, topologia, contratos e conclusões do
repositório que produziu o POSE. Esses valores chegaram às releases v0.10.0 a
v0.16.1 e também ao espelho gerado do Harness. Mover os mesmos valores para
configuração ou templates apenas esconderia o defeito: uma instalação ainda
afirmaria fatos que não observou no projeto selecionado.

## Decisão

Todo resultado dos assessors deve ser derivado de evidência encontrada sob a
raiz autorizada do projeto e de seus artefatos POSE ativos. O core não contém
adapters, nomes ou defaults para Harne8, GraphForge, pose-dist ou qualquer
outro consumidor/produtor. Extensões específicas de projeto, quando existirem,
devem viver fora do core e declarar sua própria evidência.

Os scanners compartilham estas invariantes:

- raiz real resolvida e paths relativos confinados, sem seguir symlinks para
  fora;
- exclusão de VCS, dependências, builds, `.pose`, testes/fixtures quando a
  evidência deve ser de produção e arquivos gerados;
- leitura limitada a 4 MiB por arquivo e deduplicação por path canônico;
- detecção contextual: contratos exigem declaração/uso do protocolo e dívida
  distingue comentários/código de strings literais;
- ordenação determinística antes de gerar contratos, gaps e identificadores.

## Alternativas descartadas

- **Adapter Harne8 embutido:** manteria acoplamento de produto no mecanismo
  open-source e poderia ser ativado acidentalmente em outro projeto.
- **Nomes configuráveis em templates:** resolveria somente o texto, não os
  contratos e conclusões fabricados.
- **Inventário estático versionado:** envelheceria fora do código observado e
  continuaria incapaz de diferenciar provider, consumer e ausência real.

## Gatilhos de revisão

Revisar esta decisão se o POSE adotar um formato público e versionado de plugin
de assessment com proveniência explícita, ou se um detector produzir taxa de
falso positivo que exija parser específico. Mesmo nesses casos, plugins não
podem alterar o comportamento genérico nem emitir evidência sem origem
rastreável.

## Estado atual

Implementação e fixture neutra concluídas na spec
`pose-project-agnostic-assessment-engines`. O dogfooding eliminou falsos
positivos de nomes JSON classificados como MCP tools e palavras de marcador
dentro de strings. O espelho Harness é regenerado somente após os testes do
standalone.

## Próximos checks

- Manter a fixture neutra e o scan de identidades produtoras no gate de release.
- Executar os três assessors no dogfooding após mudanças em detectores.
- Confirmar `pose check --strict`, `pose validate` e `knowledge-check --strict`
  antes do closeout.

## Riscos

Análise estática prova somente declarações observáveis no repositório; não prova
saúde em runtime. Relatórios e documentação devem preservar essa distinção.

## Próximo owner

`@pose-maintainers`, para revisão até `expires_at` ou no primeiro gatilho acima.
