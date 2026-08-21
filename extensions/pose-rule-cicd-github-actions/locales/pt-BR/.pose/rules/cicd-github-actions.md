# Regra: CI/CD GitHub Actions

## Quando consultar

Consulte este guia para workflows do GitHub Actions (`.github/workflows/*.yml`), composite actions, workflows reutilizáveis e configurações de segurança de CI/CD.

## Padrões obrigatórios

- Trave actions de terceiros em commit SHAs imutáveis completos com comentário indicando a tag de release (ex: `uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2`).
- Aplique princípio de menor privilégio em `permissions:` no topo do workflow (ex: `permissions: contents: read`) e eleve permissões apenas nos jobs estritamente necessários.
- Use variáveis de ambiente intermediárias para manipular valores de contexto não confiáveis (`github.event.issue.title`, `github.head_ref`) em scripts shell inline a fim de prevenir injeção de script.
- Mascare segredos e evite expor tokens sensíveis em outputs de steps, logs ou upload de artefatos.
- Restrinja workflows com trigger `pull_request_target` a operações estritamente de leitura, nunca executando código do PR com tokens de escrita.

## Anti-padrões bloqueantes

- Usar referências de branch ou tags mutáveis (`@main`, `@master`, `@v1`) em actions de terceiros.
- Definir permissões amplas como `permissions: write-all` nos workflows.
- Injetar variáveis de contexto controladas por usuários diretamente dentro de scripts `run:` (ex: `run: echo "${{ github.event.comment.body }}"`).
- Imprimir segredos ou credenciais em base64 diretamente nos logs de execução do GitHub Actions.
- Disparar execução de código não confiável em workflows que possuem acesso a segredos de deploy.

## Checagens mínimas

- Rodar `actionlint` ou validador de schema de workflows nos arquivos `.github/workflows/*.yml`.
- Validar a sintaxe dos workflows usando ferramentas de simulação (ex: `act` ou validações do GitHub CLI).
- Garantir que os nomes dos checks obrigatórios correspondam exatamente às regras de branch protection.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
