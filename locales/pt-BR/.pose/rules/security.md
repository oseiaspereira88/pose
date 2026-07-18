# Rule: Security

## Quando consultar

Consulte este guia em tarefas com autenticação/autorização, dados sensíveis, integrações externas, dependências e superfícies de ataque.

## Padrões obrigatórios

- Princípio do menor privilégio para acesso a recursos e credenciais.
- Segredos devem ser armazenados apenas em mecanismos apropriados (nunca em código).
- Entrada externa deve ser validada/sanitizada conforme contexto.
- Logs e métricas não devem conter dados pessoais/sigilosos em texto claro.
- Dependências novas devem ser avaliadas por manutenção, licença e vulnerabilidades conhecidas.
- Controles de autenticação e autorização devem ter testes cobrindo casos positivos e negativos.

## Anti-padrões bloqueadores

- Commit de credenciais, tokens, chaves ou segredos em qualquer artefato versionado.
- Desativar TLS/verificações de certificado sem mitigação formal documentada.
- Confiar em input do cliente para decisões de autorização.
- Executar comandos dinâmicos sem validação/escape adequado.
- Ignorar alertas críticos de segurança sem registro de exceção aprovada.

## Checks mínimos

- Execução de scanner de segredos no diff/escopo alterado.
- Execução de scanner de vulnerabilidades de dependências aplicável ao módulo.
- Testes de autenticação/autorização relevantes passando.
- Revisão de exposição de dados sensíveis em logs/configs concluída.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para segurança, contrato e operação.
- Quando houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
