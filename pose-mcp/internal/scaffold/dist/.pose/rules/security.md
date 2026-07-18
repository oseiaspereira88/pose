# Rule: Security

## When to consult

Consult this guide em tarefas with autenticação/autorização, dados sensíveis, integrações externas, dependências e superfícies de ataque.

## Required patterns

- Princípio do menor privilégio para acesso a recursos e credenciais.
- Segredos must ser armazenados apenas em mecanismos apropriados (nunca em código).
- Entrada externa must ser validada/sanitizada conforme contexto.
- Logs e métricas não must conter dados pessoais/sigilosos em texto claro.
- Dependências novas must ser avaliadas por manutenção, licença e vulnerabilidades conhecidas.
- Controles de autenticação e autorização must ter testes cobrindo casos positivos e negativos.

## Blocking anti-patterns

- Commit de credenciais, tokens, chaves ou segredos em qualquer artifact versionado.
- Desativar TLS/verificações de certificado without mitigação formal documentada.
- Confiar em input do cliente para decisões de autorização.
- Executar comandos dinâmicos without validation/escape adequado.
- Ignorar alertas críticos de security without registro de exceção aprovada.

## Minimum checks

- Execução de scanner de segredos no diff/escopo alterado.
- Execução de scanner de vulnerabilidades de dependências aplicável ao módulo.
- Testes de autenticação/autorização relevantes passando.
- Revisão de exposição de dados sensíveis em logs/configs concluída.

## Precedência em conflito multi-domínio

- Em conflito with outras `rules`, apply a alternativa mais restritiva para security, contrato e operação.
- When houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
