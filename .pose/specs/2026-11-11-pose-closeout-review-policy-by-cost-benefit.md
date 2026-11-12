---
slug: pose-closeout-review-policy-by-cost-benefit
status: draft
created_at: 2026-11-11
completed_at:
supersedes:
depends_on:
remediates:
priority: 2
components: pose-mcp
task_type: feature
---

# Spec: Política de revisão do fechamento de milestone e de roadmap por custo e ganho

## 1. Intent

### Goal

A instância escolhe, por tipo de escopo, se o fechamento de um milestone e o de um roadmap exigem revisão independente: `never`, `when-justified` ou `always`. O `pose close` do milestone e o do roadmap continuam obrigatórios em todos os casos; muda só a exigência da revisão do escopo.

### Business value

Dados do pose-dist em 2026-11-11: 7 runs de revisão de milestone e 6 de roadmap, todas de `pose-shared-runtime`. As revisões acharam coisa de substância (a chave do memo ignorava a configuração global do Git, um pedido do runtime sem identidade, a medição que o exit gate exige não existia como descrita) e deriva entre ADR, roadmap e specs, mas só em um roadmap com ADR e vários membros. Num milestone de um só membro a revisão repete a da spec, e o loop do B31 foi parado por isso em 2026-11-08. Hoje a revisão é sempre exigida; o custo (runs, rodadas, bundle próprio) não se justifica em escopos pequenos e o ganho existe em escopos compostos.

### Constraints

- O padrão da instância é `never` para os dois tipos (decisão do mantenedor); `always` reproduz o comportamento de hoje.
- A revisão de cada membro (spec) continua exigida como hoje; a dispensa só vale para a revisão do escopo agregado.
- Fechamentos já selados continuam válidos pela regra com que foram selados: a política vale para escopos fechados depois dela.
- Dispensar nunca é silencioso: o fechamento registra que a revisão não foi exigida, por qual política e por qual regra.

### Non-goals

- Mudar a revisão de spec.
- Remover o `pose close` de milestone e de roadmap.

## 2. Requirements

- R1: ADR primeiro: onde a política vive (arquivo de política versionado da instância), a semântica dos três valores, o padrão `never`, a compatibilidade com escopos já selados, e por que `when-justified` é uma regra determinística e não um juízo do agente.
- R2: Chaves `milestone_review_required` e `roadmap_review_required`, com os valores `never`, `when-justified` e `always`, lidas pelo motor; valor desconhecido é recusado.
- R3: `when-justified` para milestone: a revisão é exigida quando o milestone tem mais de dois membros. Para roadmap: exigida quando contém mais de uma spec e está associado a pelo menos uma ADR (campo do roadmap ou ADR que o cita). Caso contrário a revisão do escopo é dispensável e a dos membros vale.
- R4: Quando a revisão não é exigida (`never`, ou `when-justified` sem os critérios), `pose close` do milestone e do roadmap continua exigido e passa sem a revisão do escopo, e o resultado registra `review: not-required`, a chave, o valor e o motivo (por exemplo `members=1, threshold>2`).
- R5: Uma revisão feita mesmo quando não exigida continua válida e é verificada como hoje (a instância pode revisar por escolha).
- R6: `pose setup` e o comando de configuração mostram e permitem mudar as duas chaves; `pose doctor` avisa quando um escopo aberto passa a exigir revisão por causa de uma mudança de política.
- R7: Os fechamentos existentes do pose-dist e das instâncias não mudam de estado ao instalar a política (nenhum `pose check` novo falha).

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start, depois da ADR.

### Artifacts

- modified: .pose/specs/2026-11-11-pose-closeout-review-policy-by-cost-benefit.md
- created: .pose/adr/2026-11-12-closeout-review-policy-by-cost-and-benefit.md

Esta lista cobre a decisão em rascunho; os artefatos de implementação serão fixados antes do start.

## 4. Tasks

- [x] ADR (R1) e aprovação explícita do mantenedor em 2026-11-12
- [ ] Completar a spec (artefatos, alvos de entrega, estratégia de validação) e iniciar
- [ ] Implementar e testar
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2, R3, R4, R5, R6, R7
- Selected option: ADR [Closeout review policy by cost and benefit](../adr/2026-11-12-closeout-review-policy-by-cost-and-benefit.md), aprovada explicitamente pelo mantenedor em 2026-11-12.
- Rationale: separar o fechamento obrigatório da revisão opcional com limiares determinísticos e preservação do histórico selado.
- Consequences: a adoção é explícita; uma policy legada sem as chaves mantém a semântica atual, enquanto instalações/adotantes novas recebem `never`.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start; inclui um teste por valor e por critério do `when-justified`, e um teste de que o fechamento sem revisão registra a dispensa.

---

## 7. Final Report

### Delivered scope
Rascunho: nada entregue ainda.
