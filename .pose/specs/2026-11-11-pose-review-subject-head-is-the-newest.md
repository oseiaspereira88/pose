---
slug: pose-review-subject-head-is-the-newest
status: done
created_at: 2026-11-11
completed_at: 2026-11-11
supersedes:
depends_on:
remediates:
priority: 1
components: pose-mcp
task_type: bugfix
delivers: governance:review-subject-head-is-the-newest
---

# Spec: A cabeça do sujeito de uma revisão é a mais nova das cabeças dos seus change sets

## 1. Intent

### Goal

O `Subject.Head` de um bundle de revisão, que é o commit em que a run delegada cria o worktree selado, é a mais nova das cabeças dos change sets do escopo, e não a cabeça do que o ordenamento por id deixa por último.

### Business value

Achado ao fechar o roadmap `pose-shared-runtime`, 2026-11-10: `reviewBundleSubject` percorre os change sets ordenados por id e atribui `subject.Head = set.ResolvedHead` a cada volta, então a cabeça do sujeito era a do último set em ordem de id, arbitrária em relação à história. Num roadmap de seis specs a run delegada recebeu um worktree em `aa575b36`, anterior a commits de outros membros (e do que eles tocaram), e falhou com "sealed checkout inputs differ" e "child … has no sealed review bundle"; só congelar à mão o estado e repetir revisões fez passar. Com a cabeça sendo a mais nova entre as dos sets, o worktree selado contém tudo o que os membros fizeram.

### Constraints

- Para um escopo de um só change set (todo escopo de spec) nada muda.
- Os bundles já selados guardam a cabeça que tinham; só os novos mudam.
- Quando o Git não consegue ordenar as cabeças, vale a cabeça do último set, como antes.

### Non-goals

- Fazer o worktree selado ser o HEAD do despacho: a cabeça do sujeito continua sendo um commit dos change sets; mudanças posteriores de arquivos governados (a matriz de validação) ainda exigem que o estado seja congelado entre o selamento e a revisão.

## 2. Requirements

- R1: De várias cabeças distintas de change sets, `newestResolvedHead` devolve aquela de que nenhuma outra descende (a primeira da ordem topológica das histórias), qualquer que seja a ordem dos sets; com menos de duas cabeças distintas, ou se o Git não as ordena, devolve vazio e `reviewBundleSubject` mantém a cabeça do último set.
- R2: `reviewBundleSubject` usa essa cabeça para `Subject.Head`.

## 3. Technical Plan

### Affected areas

`pose-mcp/internal/pose/review_bundle.go`.

### Artifacts

- created: .pose/specs/2026-11-11-pose-review-subject-head-is-the-newest.md
- created: .pose/starts/pose-review-subject-head-is-the-newest.json
- created: .pose/changelogs/unreleased/pose-review-subject-head-is-the-newest.md
- modified: pose-mcp/internal/pose/review_bundle.go
- created: pose-mcp/internal/pose/review_subject_head_newest_test.go
- modified: .pose/indexes/validation-matrix.json

### Delivery targets

- governance:review-subject-head-is-the-newest module:pose-mcp profile:backend-go entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes

Nenhuma no formato: bundles novos de escopos com vários change sets passam a ter outra cabeça.

## 4. Tasks

- [x] Achar por que o worktree da revisão do roadmap era anterior a commits dos membros (a atribuição na volta)
- [x] A cabeça mais nova pela ordem topológica, e um teste dela em ordens diferentes
- [x] Rodar os testes de bundle, sujeito, milestone, roadmap e atribuição
- [ ] Revisão independente e fechamento

## 5. Decisions

### Decision D1
- Basis: R1, R2
- Minimal option: ordenar os change sets por data do commit
- Selected option: ordem topológica das histórias das cabeças (`git rev-list --topo-order -n 1`)
- Rationale: commits do mesmo segundo (comuns em lotes e testes) empatam por data; a ordem topológica lista um descendente antes dos seus ancestrais e não depende de relógio
- Consequences: se as cabeças divergem (nenhuma descende de todas) vale a primeira da ordem topológica, uma cabeça entre as mais novas
- Falsifier: um escopo de vários change sets cujo sujeito tem uma cabeça de que outra cabeça descende

## 6. Validation

### Strategy

Um teste com três commits e as três ordens de sets (e os casos de uma cabeça só e de uma cabeça desconhecida); os testes de bundle, sujeito, milestone, roadmap e atribuição passam (142 s). A prova de ponta a ponta é a revisão delegada de um roadmap sem congelamento manual da ordem dos commits dos membros.

### Deterministic checks

`go test` do teste do rastreio e a matriz estrita em clone limpo.

### Requirement trace
- R1 [satisfied] test:TestTheSubjectHeadIsTheNewestHeadOfItsChangeSets check:review-subject-head-integration
- R2 [satisfied] test:TestTheSubjectHeadIsTheNewestHeadOfItsChangeSets

### Known gaps
- R2 não tem teste de `reviewBundleSubject` com vários change sets (o fixture de um milestone com membros e trailers é grande); o teste cobre a função que escolhe a cabeça, e a falha sem a mudança é a de compilação do método.
- Uma mudança em arquivo governado depois da cabeça (a matriz) ainda faz o checkout selado diferir.

---

## 7. Final Report

### Delivered scope
O sujeito de um escopo com vários change sets tem como cabeça a mais nova das cabeças dos sets (R1, R2).

### Files and modules changed
- pose-mcp/internal/pose/review_bundle.go e o teste; a matriz ganha `review-subject-head-integration`.

### Validation executed
- Command: `pose validate --strict --jobs 4` em clone limpo do head revisado
- Result: registrado pela evidência de validação selada; revisão delegada independente concluída com `pose review conclude`.

### Residual risks
- Cabeças que divergem escolhem a primeira da ordem topológica.

### Follow-ups

- [open] Fazer o worktree selado conter os arquivos governados como estavam ao selar (ou verificá-los contra o bundle sem exigir o mesmo commit), para dispensar o congelamento da matriz entre selar e revisar. (owner:@pose-maintainers crit:low review:2027-02-08)
