---
name: pose-release-closeout
description: Use para preparar, publicar, reconciliar e verificar uma release POSE sem notas mutáveis, sobrescrita de tag ou estado externo fabricado. Trigger keywords - release, publicar, tag, corte de changelog, fechar release, versão.
when_to_use: Specs e roadmap revisados estão terminais e uma nova versão imutável precisa ser publicada.
pose_schema_range: "1-1"
clients: agents-skills, mcp, claude-code
capabilities: read, release-write
---

# Skill: pose-release-closeout

## Leitura obrigatória

1. [Workflow de Release](../../../.pose/workflows/release.md).
2. [Regra de integridade de release](../../../.pose/rules/release-integrity.md).
3. [Política de release](../../../.pose/release-policy.json).

## Procedimento

1. Exigir escopo de entrega revisado e terminal, e uma versão alvo explícita.
2. Planejar, depois preparar com apply explícito; revisar e versionar o snapshot congelado.
3. Executar gates estritos de release, compatibilidade, segurança e validação completa.
4. Exigir worktree limpo e tag inexistente; nunca fazer staging amplo ou force.
5. Publicar a nova tag e monitorar a conclusão no provedor.
6. Importar evidência de publicação retida e evidência de verificação independente.
7. Não parar na criação da tag: o sucesso terminal exige a projeção verified da política. Em caso de falha, registre-a e preserve a tag imutável.

## Requisitos de saída

- Manifesto imutável, notas e fragmentos arquivados.
- Nova tag não sobrescrita vinculada ao commit revisado.
- Evidência do provedor com identidade minimizada e digests dos assets.
- Verificação independente vinculada a essa publicação.
- Fila pendente pós-corte vazia, exceto trabalhos genuinamente mais recentes.
