---
name: pose-adr
description: Use ao registrar uma decisão arquitetural sob POSE — escolha entre opções com impacto estrutural, mudança de contrato público, ou trade-off relevante a posterior reavaliação. Trigger keywords - ADR, architecture decision, decisão arquitetural, contrato estrutural, technical decision, trade-off, design choice.
when_to_use: Há uma decisão técnica cujo "porquê" precisa sobreviver à memória do autor original. Tipicamente quando: mudança de stack/biblioteca, contrato HTTP/schema, padrão de organização inter-módulos, ou descarte de alternativa que outros podem propor de novo.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, adr-write
---

# Skill: pose-adr

Fluxo POSE para registrar decisões arquiteturais.

## Required reading

1. [AGENTS.md](../../../AGENTS.md) — obrigatoriedade de ADR.
2. ADRs existentes em [`.pose/adr/`](../../../.pose/adr/) — para evitar duplicação e identificar decisões que esta supera.
3. Rules relevantes ao escopo arquitetural em `.pose/rules/`.

## Steps

1. Confirmar que é decisão arquitetural (não tática). Critério: outras pessoas vão querer saber "por quê?" daqui a 6 meses.
2. Verificar se ADR anterior já cobre o tema:
   ```bash
   ls .pose/adr/ | grep -i <palavra-chave>
   ```
3. Criar ADR com título conciso e descritivo:
   ```bash
   ./pose new-adr "<título da decisão>"
   ```
4. Preencher seções no arquivo gerado em `.pose/adr/<data>-<slug>.md`:
   - **Status**: `Proposed` | `Accepted` | `Superseded by <adr>`
   - **Context**: problema, restrições, forças que motivam a decisão
   - **Decision**: o que foi decidido (não a discussão — a conclusão)
   - **Consequences**: impactos positivos, negativos, neutros; o que muda na operação/manutenção
5. Linkar módulos/serviços impactados e trade-offs descartados.
6. Se a decisão cria gatilho de revisão futura (ex.: "rever em 6m"), criar decision-log para rastreio:
   ```bash
   ./pose new-knowledge decision-log adr-<slug>-revisita --owner @<squad> --ttl-days 90
   ```
7. Atualizar spec relacionada (se houver) referenciando a ADR em `Decisions`.

## Output requirements

- ADR em `.pose/adr/<data>-<slug>.md` com 4 seções preenchidas.
- Trade-offs descartados com 1 linha explicando o motivo.
- Decision-log opcional quando houver gatilho de revisão.
- Referência cruzada na spec (quando a decisão deriva de implementação ativa).
