# Rule: Backend Rust

## When to consult

Consult this guide for HTTP handlers, microservices, gRPC services, persistence, concurrency, and async runtimes (Tokio, Axum, Actix-web, Tonic) in Rust.

## Required patterns

- Return explicit `Result<T, E>` types and use typed error enums (e.g. `thiserror`) for libraries and domain layers, and `anyhow` for application entrypoints.
- Honor async runtime invariants: offload compute-heavy or blocking synchronous calls to `tokio::task::spawn_blocking`.
- Enforce strict ownership and borrow mechanics; prefer borrowing (`&str`, `&[T]`) over unnecessary cloning (`.clone()`) on hot paths.
- Ensure all resources (locks, file descriptors, connections) are scoped properly to release on drop.
- Use `tracing` or structured logging with sanitized contextual fields.

## Blocking anti-patterns

- Using `.unwrap()` or `.expect()` in request execution flows or production paths without documented mathematical impossibility proofs.
- Using `unsafe` blocks without documented safety invariant rationale and safety review.
- Blocking the Tokio asynchronous threadpool with long-running synchronous loops or blocking I/O.
- Creating deadlocks via un-ordered acquisition of multiple `Mutex` or `RwLock` guards.
- Silently ignoring errors with `let _ = ...` on critical operations.

## Minimum checks

- Run `cargo test` across affected crates.
- Run `cargo clippy --all-targets -- -D warnings` without blocking diagnostics.
- Run `cargo fmt --check`.
- Run `cargo audit` or dependency check when updating dependencies.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
