package cli

// The onboarding spec: adopting POSE in a project is its first governed piece
// of work (spec pose-onboarding-spec). A new instance gets it at install; it
// passes lint, the strict check and the Definition of Ready as written, so the
// person learns start, evidence and close on work whose subject is POSE.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const onboardingSlug = "pose-onboarding"

const onboardingSpecEN = `---
slug: pose-onboarding
status: draft
created_at: {{DATE}}
completed_at:
supersedes:
depends_on:
remediates:
priority: 0
components:
task_type: feature
surface: minimal
delivers:
---

# Spec: Adopt POSE in {{PROJECT_NAME}}

## 1. Intent

### Goal

Adopt POSE in {{PROJECT_NAME}}: a declared project identity, a maintainer who can prove their answers, the commit gate, a reviewed set of capabilities, and the installed instance committed, so every checkout reads the same governance.

### Business value

This is the project's first governed piece of work, and the record of how it adopted POSE. Working through it teaches the lifecycle every later spec follows: draft, ` + "`pose start`" + `, evidence, ` + "`pose close`" + `.

### Constraints

Every step is confirmed by the maintainer; ` + "`pose setup`" + ` shows each one and changes nothing without an explicit yes.

### Non-goals

Configuring the project's build and test commands; that is ` + "`pose init --wizard`" + ` or ` + "`.pose/indexes/validation-matrix.json`" + `, when the project has code to check.

## 2. Requirements

### Functional

- R1: The project's identity shall be declared in ` + "`.pose/project.json`" + `, so every checkout and the MCP server resolve the same project.
- R2: At least one person shall hold the ` + "`maintainer`" + ` role in ` + "`.pose/policy/actions.json`" + ` with a registered key (` + "`pose identity add`" + `), so an answer to an action request can be proven.
- R3: The pre-commit gate shall be installed (` + "`pose hooks install`" + `).
- R4: The capability defaults shall be reviewed: ` + "`pose setup`" + ` lists no capability decision pending.
- R5: The installed instance shall be committed, so ` + "`pose doctor`" + ` reports no warning and ` + "`pose state --attention`" + ` reads complete coverage.

### Non-functional

- None.

### Security

- A security key (` + "`ssh-keygen -t ed25519-sk`" + `) is preferred for a person: it proves presence at every signature.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

POSE configuration only: identity, roles and keys, the adoption record.

### Artifacts

- created: .pose/specs/{{DATE}}-pose-onboarding.md
- created: .pose/starts/pose-onboarding.json
- modified: .pose/policy/actions.json
- modified: .pose/policy/adoption-decisions.json

### Technical risks

- None.

## 4. Tasks

### Planning
- [ ] ` + "`pose start spec:pose-onboarding`" + ` records the baseline this spec is reconciled against

### Implementation
- [ ] ` + "`pose setup`" + `: register yourself with a key and the maintainer role (R2)
- [ ] ` + "`pose setup`" + `: install the commit gate (R3)
- [ ] ` + "`pose setup`" + `: adopt, decline or defer each capability it lists (R4)
- [ ] Commit the instance with ` + "`POSE-Spec: pose-onboarding`" + ` in the message (R5)

### Validation
- [ ] Run the checks below, fill the requirement trace, then ` + "`pose close spec:pose-onboarding`" + `

## 5. Decisions

### Decision D1
- Basis: R4
- Minimal option: turn on nothing until each capability is understood
- Selected option: keep the defaults ` + "`pose install`" + ` adopted, each dated with the install day
- Rationale: the defaults are what a new POSE project is expected to run with, and work created before a capability's date is never judged by it
- Consequences: ` + "`pose adopt <capability> --off --apply`" + ` turns one off; ` + "`pose adopt --list`" + ` shows what each changes
- Falsifier: a default blocks routine work without catching a real problem

## 6. Validation

### Strategy

Run ` + "`pose setup`" + ` until it reports nothing to set up, then the checks below.

### Deterministic checks

#### Health
- Command: ` + "`pose doctor`" + `
- Expected: no warning and no error

#### Structure
- Command: ` + "`pose check --strict`" + `
- Expected: pass

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

Not started. Next: ` + "`pose start spec:pose-onboarding`" + `, then ` + "`pose setup`" + `.

### Residual risks

### Follow-ups
`

const onboardingSpecPtBR = `---
slug: pose-onboarding
status: draft
created_at: {{DATE}}
completed_at:
supersedes:
depends_on:
remediates:
priority: 0
components:
task_type: feature
surface: minimal
delivers:
---

# Spec: Adotar o POSE em {{PROJECT_NAME}}

## 1. Intent

### Objetivo

Adotar o POSE em {{PROJECT_NAME}}: identidade do projeto declarada, um maintainer que consegue provar suas respostas, o gate de commit, um conjunto de capacidades revisado e a instância instalada commitada, para que todo checkout leia a mesma governança.

### Valor de negócio

Este é o primeiro trabalho governado do projeto e o registro de como ele adotou o POSE. Percorrê-lo ensina o ciclo de vida que toda spec seguinte segue: rascunho, ` + "`pose start`" + `, evidência, ` + "`pose close`" + `.

### Restrições

Todo passo é confirmado pelo maintainer; ` + "`pose setup`" + ` mostra cada um e não muda nada sem um sim explícito.

### Não-objetivos

Configurar os comandos de build e teste do projeto; isso é ` + "`pose init --wizard`" + ` ou ` + "`.pose/indexes/validation-matrix.json`" + `, quando o projeto tiver código a verificar.

## 2. Requirements

### Funcionais

- R1: A identidade do projeto deve estar declarada em ` + "`.pose/project.json`" + `, para que todo checkout e o servidor MCP resolvam o mesmo projeto.
- R2: Ao menos uma pessoa deve ter o papel ` + "`maintainer`" + ` em ` + "`.pose/policy/actions.json`" + ` com uma chave registrada (` + "`pose identity add`" + `), para que uma resposta a um action request possa ser provada.
- R3: O gate de pre-commit deve estar instalado (` + "`pose hooks install`" + `).
- R4: Os padrões de capacidades devem estar revisados: ` + "`pose setup`" + ` não lista decisão de capacidade pendente.
- R5: A instância instalada deve estar commitada, para que ` + "`pose doctor`" + ` não relate aviso e ` + "`pose state --attention`" + ` leia cobertura completa.

### Não-funcionais

- Nenhum.

### Segurança

- Uma chave de segurança (` + "`ssh-keygen -t ed25519-sk`" + `) é preferível para uma pessoa: prova presença a cada assinatura.

### Compatibilidade

- Nenhuma.

## 3. Technical Plan

### Áreas afetadas

Somente configuração do POSE: identidade, papéis e chaves, o registro de adoção.

### Artifacts

- created: .pose/specs/{{DATE}}-pose-onboarding.md
- created: .pose/starts/pose-onboarding.json
- modified: .pose/policy/actions.json
- modified: .pose/policy/adoption-decisions.json

### Riscos técnicos

- Nenhum.

## 4. Tasks

### Planejamento
- [ ] ` + "`pose start spec:pose-onboarding`" + ` registra a baseline contra a qual esta spec é reconciliada

### Implementação
- [ ] ` + "`pose setup`" + `: registre-se com uma chave e o papel maintainer (R2)
- [ ] ` + "`pose setup`" + `: instale o gate de commit (R3)
- [ ] ` + "`pose setup`" + `: adote, recuse ou adie cada capacidade listada (R4)
- [ ] Commite a instância com ` + "`POSE-Spec: pose-onboarding`" + ` na mensagem (R5)

### Validação
- [ ] Rode os checks abaixo, preencha o requirement trace, depois ` + "`pose close spec:pose-onboarding`" + `

## 5. Decisions

### Decision D1
- Basis: R4
- Minimal option: não ligar nada até entender cada capacidade
- Selected option: manter os padrões que o ` + "`pose install`" + ` adotou, cada um datado com o dia da instalação
- Rationale: os padrões são o que um projeto POSE novo deve usar, e trabalho criado antes da data de uma capacidade nunca é julgado por ela
- Consequences: ` + "`pose adopt <capability> --off --apply`" + ` desliga uma; ` + "`pose adopt --list`" + ` mostra o que cada uma muda
- Falsifier: um padrão bloqueia trabalho rotineiro sem pegar um problema real

## 6. Validation

### Estratégia

Rode ` + "`pose setup`" + ` até não haver nada a configurar, depois os checks abaixo.

### Checks determinísticos

#### Saúde
- Comando: ` + "`pose doctor`" + `
- Esperado: nenhum aviso e nenhum erro

#### Estrutura
- Comando: ` + "`pose check --strict`" + `
- Esperado: passa

### Requirement trace

### Gaps conhecidos

## 7. Final Report

### Escopo entregue

Não iniciado. Próximo: ` + "`pose start spec:pose-onboarding`" + `, depois ` + "`pose setup`" + `.

### Riscos residuais

### Follow-ups
`

// scaffoldOnboardingSpec writes the onboarding spec into a new instance. It
// never overwrites one, under any date.
func scaffoldOnboardingSpec(target, locale, projectName, date string) (string, bool, error) {
	if existing := findOnboardingSpec(target); existing != "" {
		return existing, false, nil
	}
	body := onboardingSpecEN
	if locale == "pt-BR" {
		body = onboardingSpecPtBR
	}
	if strings.TrimSpace(projectName) == "" {
		projectName = filepath.Base(target)
	}
	body = strings.NewReplacer("{{DATE}}", date, "{{PROJECT_NAME}}", projectName).Replace(body)
	rel := filepath.ToSlash(filepath.Join(".pose", "specs", date+"-"+onboardingSlug+".md"))
	path := filepath.Join(target, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return rel, false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer file.Close()
	if _, err := file.WriteString(body); err != nil {
		return "", false, err
	}
	return rel, true, nil
}

// findOnboardingSpec returns the onboarding spec's project-relative path, or "".
func findOnboardingSpec(root string) string {
	matches, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*-"+onboardingSlug+".md"))
	if len(matches) == 0 {
		return ""
	}
	rel, err := filepath.Rel(root, matches[0])
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}
