# Workflow: Release baseada em evidências

Prepare um snapshot imutável antes da tag, valide-o com `pose release check
--strict`, publique somente uma tag nova e importe evidências separadas de
publicação e verificação. Nunca trate a tag como prova de publicação, nunca
sobrescreva tags e nunca use force-push. Uma falha do provedor mantém o estado
honesto como tagged/failed até reconciliação.
