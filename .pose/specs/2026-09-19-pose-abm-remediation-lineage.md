---
slug: pose-abm-remediation-lineage
status: draft
created_at: 2026-09-19
completed_at:
supersedes:
depends_on: pose-abm-governance-outcomes
priority: 2
components: pose-mcp
task_type: feature
delivers:
---

# Spec: Lineage explícita de remediação

## 1. Intent

### Objetivo
Registrar e validar um vínculo `remediates` explícito entre entregas, e calcular
remediação numa janela madura apenas quando a população observável e a censura
forem conhecidas.

### Valor de negócio
Distinguir uma entrega que conserta outra de uma que apenas veio depois. Sem
isso, qualquer taxa de remediação mede ordem cronológica, não causa.

### Restrições
- Esta spec recebe R4 e R5 de [governance-outcomes](2026-09-19-pose-abm-governance-outcomes.md),
  que fechou com os dois diferidos para cá. Nada aqui foi implementado ainda.
- O vínculo é declarado, nunca inferido por similaridade lexical. Depender de
  parecença textual entre títulos foi rejeitado no desenho do programa.
- `depends_on` não é `remediates`: dependência não implica conserto.
- Janela incompleta é inconclusiva, e a censura precisa ser reportada junto com
  a taxa; ausência de dado não vira zero.

### Não-objetivos
Não inferir causalidade a partir de proximidade temporal, não ranquear pessoas,
agentes ou modelos, e não transformar a taxa observada num score agregado.

## 2. Requirements

### Funcionais e critérios de aceite
- R1: Registrar `remediates` apontando para spec ou finding, com categoria de um
  conjunto fechado — defect-fix, simplification, revert, requirement-change,
  planned-evolution — recusando refs órfãs e ciclos.
- R2: Calcular remediação numa janela de 30 dias apenas sobre a população que
  completou a janela, reportando censura e entregas excluídas por imaturidade.
- R3: Somente as categorias pertinentes entram na taxa; mudança de requisito não
  é defeito, e aceitação de risco é categoria própria, não defeito corrigido.

### Não-funcionais
Determinístico e offline. Nenhum serviço, índice ou banco novo.

### Segurança
Conteúdo do repositório é input não confiável. A projeção agrega por projeto,
processo ou banda, e nunca retorna principal, unidade de negócio ou identidade.

### Compatibilidade
Aditivo e opt-in. Specs sem `remediates` continuam válidas e fora da população.

## 3. Technical Plan

### Áreas afetadas
`pose-mcp/internal/pose` e a projeção de `pose stats governance` entregue por
governance-outcomes.

### Artifacts
- created: .pose/specs/2026-09-19-pose-abm-remediation-lineage.md

Inventário inicial. Os paths de implementação entram por amendment quando o
trabalho começar; declarar agora arquivos que ninguém escreveu seria contrato
falso.

### Delivery targets
Nenhum alvo declarado: nada foi implementado. O alvo tipado será declarado no
incremento que tocar uma raiz de entrega, porque a policy o exige de quem a
altera.

### Mudanças de API/contrato
Um campo `remediates` no contrato de spec e a extensão da projeção existente.
Sem DSL, sem motor paralelo.

### Dados e operação
Nenhuma persistência nova; a projeção continua reconstruível a partir dos
artefatos existentes.

### Riscos técnicos
A tentação de inferir o vínculo quando ele não é declarado. Um vínculo ausente
é desconhecido, e a spec precisa manter esse estado visível em vez de preenchê-lo.

### Rollout e reversão
Advisory primeiro, com a taxa publicada ao lado da cobertura. Reverter é deixar
de exigir o campo, sem reescrever dado histórico.

## 4. Tasks

- [ ] Definir o contrato de `remediates` e seus casos negativos antes de codar.
- [ ] Implementar registro, validação e projeção.
- [ ] Cobrir refs órfãs, ciclos, categorias inválidas e janela imatura.
- [ ] Declarar alvo de entrega e fechar pelo gate POSE.

## 5. Decisions

- Data: 2026-09-19.
- Contexto: governance-outcomes precisava fechar com R4 e R5 sem evidência, e a
  própria spec instruía manter-se aberta até eles ou abrir sucessora.
- Opções consideradas: manter governance-outcomes aberta indefinidamente;
  retirar R4/R5 dela; abrir esta sucessora.
- Decisão: abrir esta sucessora e apontar o trace de lá para cá.
- Racional: manter a spec aberta parava um incremento pronto por causa de outro
  que nem começou, e retirar os requisitos apagaria a intenção registrada.
- Consequências: a taxa de remediação continua indisponível até esta spec ser
  entregue, e isso fica visível em vez de implícito.

## 6. Validation

### Estratégia
Planejamento anterior à implementação. Os comandos abaixo são entregas a criar.

| Cenário/requisitos | Comando planejado | Evidência esperada |
| --- | --- | --- |
| Refs órfãs, ciclos e categorias; R1 | `cd pose-mcp && go test ./internal/pose -run 'RemediationLineage' -count=1` | Cada caso negativo com blocker estável |
| Janela madura e censura; R2/R3 | `pose stats governance --json` | População, censura e exclusões reportadas |

### Checks de governança
- `pose lint-spec pose-abm-remediation-lineage --ready-check` antes de iniciar.
- `pose validate --strict --module pose-mcp` no escopo aplicável.
- `pose review verify spec:pose-abm-remediation-lineage` antes da transição.

### Log de execução
2026-09-19: spec criada para receber R4 e R5 de governance-outcomes no
fechamento daquela. Nenhum requisito foi implementado ou declarado satisfeito.

### Requirement trace
Pendente de implementação: cada R-ID exige evidência nominal no closeout.

### Gaps conhecidos
A população observável de remediação ainda não existe: o corpus histórico das
duas instâncias não contém nenhum finding, então não há classe positiva a medir
até que o contrato de julgamento explícito produza uma.

---

## 7. Final Report

### Escopo entregue
Nada. Esta spec é o destino declarado de dois requisitos diferidos.

### Riscos residuais
Os riscos da seção Technical Plan não foram aceitos como entrega.

### Follow-ups

Nenhum desdobramento adicional nesta fase: o escopo pendente são os próprios
requisitos desta spec.
