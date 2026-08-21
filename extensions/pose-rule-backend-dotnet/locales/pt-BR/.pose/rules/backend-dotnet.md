# Regra: Backend .NET

## Quando consultar

Consulte este guia para ASP.NET Core Web APIs, minimal APIs, background workers, repositórios Entity Framework Core e serviços gRPC em C# / .NET.

## Padrões obrigatórios

- Use async/await de ponta a ponta; propague `CancellationToken` através dos endpoints HTTP e consultas de repositório.
- Registre dependências com os tempos de vida corretos (Transient, Scoped, Singleton) e evite dependências cativas (ex: Scoped injetado em Singleton).
- Use `.AsNoTracking()` para consultas somente-leitura no Entity Framework Core a fim de reduzir overhead de memória.
- Valide modelos de requisição nas fronteiras de controller ou pipeline (FluentValidation, DataAnnotations).
- Use logs estruturados via `ILogger<T>` com message templates em vez de concatenação de strings.

## Anti-padrões bloqueantes

- Chamar `.Result` ou `.Wait()` em `Task` / `ValueTask` (sync-over-async causando threadpool starvation e deadlocks).
- Capturar instâncias de `DbContext` dentro de serviços Singleton ou loops longos em background.
- Executar SQL puro com interpolação de strings em vez de `FromSqlInterpolated` parametrizado.
- Expor entidades de domínio ou persistência diretamente nos contratos públicos de API sem DTOs.
- Ignorar ou descartar exceções sem o devido registro em log diagnóstico.

## Checagens mínimas

- Rodar `dotnet test` nos projetos de teste da solution afetada.
- Rodar `dotnet format --verify-no-changes` ou analisadores de código sem warnings fatais.
- Rodar `dotnet build` sem erros de compilação.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
