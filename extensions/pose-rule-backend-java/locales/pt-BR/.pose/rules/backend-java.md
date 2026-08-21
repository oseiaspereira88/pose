# Regra: Backend Java

## Quando consultar

Consulte este guia para serviços backend Spring Boot, Quarkus, Micronaut, Jakarta EE ou Kotlin que tratam requisições HTTP, mensageria, regras de negócio e persistência.

## Padrões obrigatórios

- Aplique camadas arquiteturais claras: Controllers/Endpoints -> Application Services -> Domain Services -> Repositories.
- Use injeção de dependência via construtor; evite injeção de campo com `@Autowired`.
- Garanta que todos os recursos fecháveis (streams, conexões, transações) utilizem `try-with-resources`.
- Declare explicitamente os limites de transação (`@Transactional`) com níveis adequados de isolamento e propagação.
- Anote contratos de nulidade (`@NonNull`, `@Nullable`) ou aproveite a null-safety nativa do Kotlin.
- Use logs estruturados via SLF4J sem expor credenciais ou tokens sensíveis.

## Anti-padrões bloqueantes

- Capturar `Throwable`, `Error` ou `Exception` de forma genérica e silenciar o erro sem re-lançar ou registrar em log.
- Usar `System.out.println` ou `e.printStackTrace()` no lugar de loggers estruturados.
- Executar queries de banco de dados sem paginação ou limites de registros.
- Manter transações de banco de dados abertas durante chamadas remotas longas a APIs de terceiros.
- Compartilhar estado mutável entre múltiplas threads sem sincronização adequada ou coleções concorrentes.

## Checagens mínimas

- Rodar `./gradlew test` ou `mvn test` nos módulos afetados.
- Rodar Checkstyle, SpotBugs ou Spotless conforme configuração do módulo.
- Garantir que a compilação passe sem erros ou warnings fatais.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
