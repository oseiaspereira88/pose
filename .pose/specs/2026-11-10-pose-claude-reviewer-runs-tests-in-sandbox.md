---
slug: pose-claude-reviewer-runs-tests-in-sandbox
status: draft
created_at: 2026-11-10
completed_at:
supersedes:
depends_on: pose-claude-sandboxed-profile
remediates:
priority: 2
components: pose-mcp
task_type: feature
---

# Spec: O revisor Claude roda go test dentro do sandbox

## 1. Intent

### Goal

O perfil de revisão do Claude permite `go test` e `go vet` dentro do sandbox do sistema operacional, recusa-se a iniciar quando o sandbox não é imposto e preserva o que o perfil restrito faz hoje.

### Business value

O perfil de revisão do Claude teve de abrir mão de `go test`, `go vet`, `git log|diff|show` e `lint-spec` porque uma lista por prefixo não exclui flags que gravam ou executam; a spec `pose-claude-sandboxed-profile` deu ao Claude um perfil `sandboxed` imposto pelo sistema operacional, com checagem de dependências, mas a revisão ainda é read.

### Constraints

Rascunho escrito em 2026-11-10 a partir das pendências da sessão da B35; a spec é completada (artefatos, alvos de entrega, tarefas, validação) antes do start.

### Non-goals

O que não está nos requisitos abaixo.

## 2. Requirements

- R1: Um perfil de revisão com sandbox permite os comandos de teste e leitura e nega gravação fora do diretório do run, medido por uma tentativa real.
- R2: O perfil não inicia sem bubblewrap e socat, com a dica de instalação.
- R3: Uma revisão com o novo perfil reexecuta os testes do rastreio da spec.

## 3. Technical Plan

Os artefatos e os alvos de entrega são fixados antes do start.

## 6. Validation

A estratégia e o rastreio dos requisitos são escritos antes do start.
