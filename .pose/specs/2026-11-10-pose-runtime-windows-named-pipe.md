---
slug: pose-runtime-windows-named-pipe
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-optional-worktree-runtime
remediates:
priority: 4
components: pose-mcp
task_type: feature
---

# Spec: Named pipe com ACL do dono para o runtime no Windows

## 1. Intent

### Goal

O runtime opcional funciona no Windows por um named pipe com a ACL do dono, com a mesma identidade, handshake e fallback standalone.

### Business value

O runtime reporta `unsupported` no Windows porque a biblioteca padrão não tem named pipe e o repositório não toma dependência para isso; a emenda do ADR de 2026-11-09 deixou o named pipe como follow-up.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um endpoint por named pipe com ACL só do dono e a credencial do par checada pelo sistema.
- R2: Mesmo handshake, ciclo de vida e fallback dos demais sistemas, com os testes do pacote rodando no Windows.
- R3: ADR: escolher a dependência (ou a chamada de sistema) antes do código.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
