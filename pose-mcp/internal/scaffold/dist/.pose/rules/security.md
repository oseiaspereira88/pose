# Rule: Security

## When to consult

Consult this guide em tarefas com autenticação/autorização, dados sensíveis, integrações externas, dependências e superfícies de ataque.

## Required patterns

- Princípio do menor privilégio para acesso a recursos e credenciais.
- Segredos devem ser armazenados apenas em mecanismos apropriados (nunca em código).
- Entrada externa deve ser validada/sanitizada conforme contexto.
- Logs e métricas não devem conter dados pessoais/sigilosos em texto claro.
- Dependências novas devem ser avaliadas por manutenção, licença e vulnerabilidades conhecidas.
- Controles de autenticação e autorização devem ter testes cobrindo casos positivos e negativos.

## Blocking anti-patterns

- Commit de credenciais, tokens, chaves ou segredos em qualquer artifact versionado.
- Desativar TLS/verificações de certificado sem mitigação formal documentada.
- Confiar em input do cliente para decisões de autorização.
- Executar comandos dinâmicos sem validation/escape adequado.
- Ignorar alertas críticos de security sem registro de exceção aprovada.

## Minimum checks

- Execução de scanner de segredos no diff/escopo alterado.
- Execução de scanner de vulnerabilidades de dependências aplicável ao módulo.
- Testes de autenticação/autorização relevantes passando.
- Revisão de exposição de dados sensíveis em logs/configs concluída.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para security, contrato e operação.
- When houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
