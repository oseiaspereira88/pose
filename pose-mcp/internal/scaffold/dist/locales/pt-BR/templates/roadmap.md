---
slug: <roadmap-slug>
status: draft        # draft | active | done | abandoned
created_at: <created_at>
depends_on:          # roadmaps pré-requisito, lista inline: outro-roadmap-a, outro-roadmap-b
---

# Roadmap: <roadmap-slug>

> Roadmap governado (pose-roadmap-artifact). O frontmatter é flat (contrato
> POSE); cada milestone é uma seção `## Milestone: <id>` com bullets flat.
> A ordem entre milestones vem de `- after:`; as datas são PLANEJAMENTO
> (insumo do Gantt) — o realizado deriva de eventos, nunca é editado aqui.
> Uma spec pertence a no máximo UM roadmap ativo (`./pose check` valida).
>
> Prosa livre é bem-vinda fora das seções de milestone: contexto, riscos,
> critérios de corte de release.

## Milestone: <id-do-milestone>
- after:                       # ids de milestones deste roadmap e/ou spec:<slug>, lista inline
- target_start:                # opcional, YYYY-MM-DD
- target_due:                  # opcional, YYYY-MM-DD
- specs:                       # slugs de specs, lista inline: spec-a, spec-b
