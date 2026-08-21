# Regra: Backend Rust

## Quando consultar

Consulte este guia para handlers HTTP, microsserviços, serviços gRPC, persistência, concorrência e runtimes assíncronos (Tokio, Axum, Actix-web, Tonic) em Rust.

## Padrões obrigatórios

- Retorne tipos `Result<T, E>` explícitos e use enums de erro tipados (ex: `thiserror`) para bibliotecas e camadas de domínio, e `anyhow` em entrypoints de aplicação.
- Respeite invariantes de runtime assíncrono: delegue chamadas síncronas bloqueantes ou de computação pesada para `tokio::task::spawn_blocking`.
- Aplique regras rígidas de ownership e borrowing; prefira referências (`&str`, `&[T]`) em vez de clones desnecessários (`.clone()`) em caminhos críticos.
- Garanta que todos os recursos (locks, file descriptors, conexões) tenham escopo correto para liberação automática via Drop.
- Use `tracing` ou logs estruturados com campos contextuais devidamente sanitizados.

## Anti-padrões bloqueantes

- Usar `.unwrap()` ou `.expect()` no fluxo de execução de requisições sem justificativa explícita de impossibilidade matemática comprovada.
- Usar blocos `unsafe` sem documentação das invariantes de segurança e aprovação formal.
- Bloquear a threadpool assíncrona do Tokio com loops síncronos longos ou operações de I/O bloqueantes.
- Provocar deadlocks por aquisição fora de ordem de múltiplos guards `Mutex` ou `RwLock`.
- Ignorar silenciosamente erros com `let _ = ...` em operações críticas.

## Checagens mínimas

- Rodar `cargo test` nos crates afetados.
- Rodar `cargo clippy --all-targets -- -D warnings` sem avisos bloqueantes.
- Rodar `cargo fmt --check`.
- Rodar `cargo audit` ou checagem de dependências ao atualizar bibliotecas externas.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
