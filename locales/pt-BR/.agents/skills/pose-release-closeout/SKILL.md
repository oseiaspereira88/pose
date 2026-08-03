---
name: pose-release-closeout
description: Use para preparar, publicar, reconciliar e verificar uma release POSE sem notas mutáveis, sobrescrita de tag ou estado externo fabricado. Trigger keywords - release, publicar, tag, corte de changelog, fechar release, versão.
when_to_use: Specs e roadmap revisados estão terminais e uma nova versão imutável precisa ser publicada.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, release-write
---

# Skill: pose-release-closeout

Leia o workflow e a rule de release. Planeje, prepare com apply explícito,
revise e versione o snapshot. Exija worktree limpo e tag inexistente, publique
sem force, importe evidência do provedor e verificação independente vinculada
aos assets. Só declare sucesso quando `pose release status` projetar `verified`.
