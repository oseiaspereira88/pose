# Rule: Garantia de delivery surface

## Aplicar quando

Uma spec altera uma superfície de produto configurada ou raiz de composição, declara um alvo de entrega tipado, ou participa de critérios de corte de roadmap.

## Obrigatório

- Manter os conjuntos de referências `delivers` e `### Delivery targets` idênticos.
- Usar apenas checagens registradas na `validation-matrix` e classes de evidência fechadas.
- Exigir `reachability` mais `integration` ou `e2e` para superfícies de entrega.
- Exigir `integration` para capacidades compostas e contratos.
- Vincular resultados aprovados ao digest de procedência atual; re-executar quando obsoleto.
- Manter critérios de roadmap declarativos: apenas refs tipados, nomes de checagem ou relatórios manuais confinados.

## Bloquear

- Comandos brutos em specs ou critérios de roadmap.
- Sucesso de build/unit apresentado como prova de composição.
- Evidência manual usada para satisfazer reachability obrigatório.
- Uma raiz de entrega alterada sem declaração tipada.
