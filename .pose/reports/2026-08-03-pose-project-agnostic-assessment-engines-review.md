---
type: review
spec: pose-project-agnostic-assessment-engines
reviewed_at: 2026-08-03
reviewer: agent:codex-review-20260803
decision: approved
---

# Review: Project-agnostic assessment engines

## Review summary

- Decision: approved.
- Change type: bugfix sistêmico em backend Go, contratos CLI/MCP e artifacts
  editoriais gerados.
- Scope: `spec:pose-project-agnostic-assessment-engines`.
- No critical, high, medium or low finding remains open.

## Rules applied during review

- `.pose/workflows/bugfix.md`: reprodução, causa raiz, regressão e correção
  coesa sem adapter específico de projeto.
- `.pose/workflows/review.md`: segunda passagem separada, evidência de gates,
  classificação de findings e decisão digest-bound.
- `.pose/rules/backend-go.md`: erros propagados, scans limitados e testes de
  contratos/negativos.
- `.pose/rules/security.md`: raiz confinada, traversal/symlink rejeitados,
  arquivos limitados e ausência de segredos.
- `.pose/rules/documentation-style.md`: spec, changelog, reports e assessment
  templates sem identidades fabricadas.
- `.pose/rules/knowledge-governance.md`: decision-log institucional com owner,
  TTL, fontes e `knowledge-check --strict`.
- Não aplicáveis: frontend React e Kubernetes; nenhum arquivo desses domínios
  mudou.

## Functional and contract review

- Discovery deriva label, slug, linguagem, métricas, metadata e topologia do
  root selecionado e rejeita paths absolutos, `..` e symlink externo.
- Integration varre a raiz uma vez e exige contexto de Protobuf, REST/OpenAPI,
  chamada de mensagem ou MCP; contratos e gaps são ordenados antes dos IDs.
- Technical debt separa comentário, código e string, suporta construções reais
  como Rust `todo!()`, exclui testes/fixtures/gerados e exige referência exata
  de arquivo ou componente em backlog ativo.
- `coverage_ref` é aditivo; schema version, nomes CLI/MCP e paths públicos foram
  preservados. Não há breaking change.
- Harness contém apenas o mirror produzido por `scripts/sync-pose-sources.sh`.

## Findings remediated during review

- F1 high, resolved: templates e inventários continham identidades e conclusões
  do produtor. Removidos e cobertos pela fixture neutra.
- F2 medium, resolved: nomes JSON comuns podiam parecer tools MCP. Parser do
  catálogo exige `tools[].inputSchema`; regressão cobre nome de package/env.
- F3 medium, resolved: palavras de dívida dentro de strings eram contadas.
  Scanner lexical distingue comentários, código, strings e lifetime Rust.
- F4 medium, resolved: metadata podia reduzir o scan a módulos declarados e um
  build oculto podia introduzir ruído. A raiz é varrida uma vez; metadata só
  atribui componente; diretórios ocultos/build são excluídos.
- F5 medium, resolved: referência a um arquivo cobria todo seu componente.
  Cobertura de arquivo agora é exata e cobertura ampla exige referência
  explícita `component:`/`components:`/`module:` ou slug delimitado.

## Validation evidence

- `go test ./...` em `pose-mcp`: success, incluindo scaffold parity.
- `go vet ./...` em `pose-mcp`: success.
- `go test ./...` e `go vet ./...` em `harness`: success.
- `/home/go/go/bin/govulncheck ./...`: zero vulnerabilidade alcançável.
- `pose knowledge-check --strict`: success, 2 artifacts válidos e zero vencido.
- `pose check --strict`: success.
- `pose validate`: success em todos os módulos/checks required.
- Dogfooding pré-closeout: 2 componentes descobertos; debt com 1 `panic` real
  coberto pela spec ativa e zero TODO/FIXME/stub. O assessment obrigatório
  pós-closeout marcou esse `panic` preexistente como descoberto, provando que uma
  spec `done` não permanece artificialmente como cobertura ativa.

## Recurrence and prevention

Não há evidência de `task_slug` recorrente acima do threshold. A causa é
sistêmica porque o defeito foi distribuído em várias releases; a prevenção está
no decision-log, fixture neutra, scan de identidades e gates de scaffold/Harness.

## Residual risk

Static assessment prova declarações observadas no repositório, não disponibilidade
em runtime. O relatório preserva essa limitação e não fabrica runtime health.

## Safe release condition

Publicar somente depois de `review-check` e closeout da spec passarem, preparar
release patch, repetir gates e verificar os assets publicados de forma
independente.
