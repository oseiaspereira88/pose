# Auditoria da autoria do programa de consolidação

Data: 2026-11-03. Base da triagem: `f3f31bb7765866260280dafb92416e31e0208a9c`.

## Escopo e resultado

Os 16 reproduzidores documentais do bootstrap e dos lotes 1–15 passaram nos
commits históricos abaixo. O SHA-256 do programa executado é igual ao do bloco
original em todos os casos; nenhum script histórico foi corrigido. As 16 specs
são candidatas a fechamento da autoria. A revisão e a transição de lifecycle
continuam pertencendo aos gates nativos de cada spec.

O primeiro extrator desta auditoria terminava em um literal de cerca Markdown
dentro do programa e produziu um falso diagnóstico de sintaxe nos lotes 8–15.
A extração ancorada por linha eliminou o falso positivo antes de qualquer
alteração de relatório histórico. Não há erratum dos scripts originais.

## Proveniência e reprodução

Consultar o [resultado congelado](../results/pose-consolidation-authoring-audit.json)
para os pins, digests, exit codes e stdout/stderr. O
[método executável](2026-11-03-pose-consolidation-authoring-audit.py) lê esses pins,
confere o programa no Git e usa um clone descartável local por snapshot.
Não redescobre os escopos pelo status atual; fechar uma spec não remove sua prova.

Executar a partir da raiz do projeto, escolhendo um resultado externo ao checkout:

```sh
python3 -B .pose/reports/2026-11-03-pose-consolidation-authoring-audit.py "$PWD" /var/tmp/pose-authoring-rerun.json
```

Não consultar CI remoto. As asserções antigas sobre `ci_evidence` conferem somente
os metadados retidos no snapshot, sem observar novamente o provedor. O snapshot
é o último commit do conteúdo do relatório, incluindo a correção documental do
lote 2 em `4c22c3d608f570c18b66fed82701db23943c5c48`, não apenas sua criação.
O lote 15 consulta o Git local; por isso a reprodução conserva `.git`, com
checkout destacado, configuração global desligada e hooks desabilitados.

| Spec de autoria | Snapshot reproduzido | Exit code |
|---|---|---|
| `pose-consolidation-program-bootstrap` | `e4ca6f572c6bb877c51f2f0a0b6109cc5db35862` | 0 |
| `pose-consolidation-batch-1-authoring` | `9703ba27e03e6261f0293bb7dca29e66d9ab48cd` | 0 |
| `pose-consolidation-batch-2-authoring` | `4c22c3d608f570c18b66fed82701db23943c5c48` | 0 |
| `pose-consolidation-batch-3-authoring` | `5b8ad1e8fd8f65d23fd3a0c45cd878e128565d68` | 0 |
| `pose-consolidation-batch-4-authoring` | `8ea7a4596be89dc9d5266778588db98cbe64798c` | 0 |
| `pose-consolidation-batch-5-authoring` | `2ca1b24c7b9d947a2f2d1ad2ded10f07180cf14a` | 0 |
| `pose-consolidation-batch-6-authoring` | `b4f1eaecb63c6e255064cf0cf48c960fa3d43c4f` | 0 |
| `pose-consolidation-batch-7-authoring` | `5f78ef8bd0d90b88933b5478ddf0b8f2bccfef3f` | 0 |
| `pose-consolidation-batch-8-authoring` | `7f68bb41a0284ce619d676f17749fd555064c918` | 0 |
| `pose-consolidation-batch-9-authoring` | `abaa326fa29f1d36289e3afc98515ca3355d249e` | 0 |
| `pose-consolidation-batch-10-authoring` | `bdd1d4b4a482dc4aac3233702eb2c27fa21411e2` | 0 |
| `pose-consolidation-batch-11-authoring` | `077e7b88c1d6ffd441fc87f52fd28ba1271a6237` | 0 |
| `pose-consolidation-batch-12-authoring` | `77d3306377f6064fea95c91cf61cd6b672f82e52` | 0 |
| `pose-consolidation-batch-13-authoring` | `ffe3192fb82076ebca2f27da495760822ceefeec` | 0 |
| `pose-consolidation-batch-14-authoring` | `068533ae5d469cb25d0c832811bcad44758383b4` | 0 |
| `pose-consolidation-batch-15-authoring` | `d1bd2a3229096cb360c516372bcdc1610b391b9f` | 0 |

## Reconciliação do bootstrap R4

Os objetos de merge locais registram exatamente os heads retidos na matriz:

| PR | Merge | Segundo parent / head registrado |
|---|---|---|
| 157 | `e351fcaf794fe0ff9d590be1ff074e9999f51699` | `1fa999c3cb4a3648cd791937918c56e808eab18b` |
| 158 | `1e3c61c480379dd2d57b7cbcbb77bcae30bdd5ac` | `93bbd53057c218f3529d8eaa50d6b515fbb0ef9a` |

Isso comprova a relação merge/head no histórico local. Não comprova uma nova
execução de CI, eficácia funcional, adoção no Harne8 ou conclusão das unidades.

## Gates atuais e disposição

Preencher o trace de cada spec com seu relatório e commit históricos. Reconciliar
os artefatos atribuídos, executar lint/check estritos atuais e gerar a evidência
no `results_path` antes de selar. Exigir revisão separada dos critérios de cada
bundle e usar o fechamento nativo. Nenhuma spec deve ser abandonada só por idade.

As frases “manter draft”, contagens de propostas ausentes e hashes da matriz
são afirmações do snapshot de escrita. Não aplicá-las ao checkout atual nem
regredir implementações concluídas para reproduzir o passado. Preservar os
relatórios antigos e os requisitos; a presente auditoria fornece a referência
temporal que faltava no trace.

Fechar autoria não fecha as 44 unidades, o programa de consolidação ou os roadmaps
de implementação. A matriz histórica continua sendo plano, com qualificação
funcional nos respectivos contratos. Não escrever dispositions de backlog por
inferência desta auditoria. Consultar o frontmatter das specs e suas atestações
para o estado de lifecycle efetivo.
