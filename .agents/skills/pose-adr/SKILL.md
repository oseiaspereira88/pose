---
name: pose-adr
description: Use when registrar uma decision arquitetural sob POSE — escolha entre opções with impacto estrutural, mudança de contrato público, ou trade-off relevante a posterior reavaliação. Trigger keywords - ADR, architecture decision, decision arquitetural, contrato estrutural, technical decision, trade-off, design choice.
when_to_use: Há uma decision técnica cujo "porquê" needs to sobreviver à memória do autor original. Typically when: mudança de stack/biblioteca, contrato HTTP/schema, padrão de organização inter-módulos, ou descarte de alternativa que outros podem propor de novo.
---

# Skill: pose-adr

Fluxo POSE to registrar decisions arquiteturais.

## Required reading

1. [AGENTS.md](../../../AGENTS.md) — obrigatoriedade de ADR.
2. ADRs existentes em [`.pose/adr/`](../../../.pose/adr/) — to evitar duplicação e identificar decisions que esta supera.
3. Rules relevantes ao escopo arquitetural em `.pose/rules/`.

## Steps

1. Confirmar que é decision arquitetural (not tática). Critério: outras pessoas vão querer saber "por quê?" daqui a 6 meses.
2. Verificar se ADR anterior já cobre o tema:
   ```bash
   ls .pose/adr/ | grep -i <palavra-chave>
   ```
3. Criar ADR with título conciso e descritivo:
   ```bash
   ./pose new-adr "<título da decision>"
   ```
4. Preencher sections no arquivo gerado em `.pose/adr/<data>-<slug>.md`:
   - **Status**: `Proposed` | `Accepted` | `Superseded by <adr>`
   - **Context**: problema, restrições, forças que motivam a decision
   - **Decision**: o que foi decidido (not a discussão — a conclusão)
   - **Consequences**: impactos positivos, negativos, neutros; o que muda na operação/manutenção
5. Linkar módulos/serviços impactados e trade-offs descartados.
6. Se a decision cria gatilho de revisão futura (ex.: "rever em 6m"), create decision-log to rastreio:
   ```bash
   ./pose new-knowledge decision-log adr-<slug>-revisita --owner @<squad> --ttl-days 90
   ```
7. Atualizar spec relacionada (se houver) referenciando a ADR em `Decisions`.

## Output requirements

- ADR em `.pose/adr/<data>-<slug>.md` with 4 sections filled.
- Trade-offs descartados with 1 linha explicando o motivo.
- Decision-log optional when houver gatilho de revisão.
- Referência cruzada na spec (when a decision deriva de implementação ativa).
