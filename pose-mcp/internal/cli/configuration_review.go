package cli

// The configuration review (spec pose-update-configuration-review): an
// update that brings capabilities this project has not decided writes a
// review spec for the engine version and asks the maintainer, one decision
// request per capability. Nothing is adopted until an answer exists; the
// answer is applied with `pose adopt --request <act-id> --apply` or by
// `pose setup`.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

func engineRelease() string { return version.ReleaseBase() }

const (
	reviewRequestedBy   = "agent:pose-update"
	reviewContextKey    = "capability:"
	reviewRecipient     = "maintainer"
	reviewAnswerAdopt   = "adopt"
	reviewAnswerDecline = "decline"
	reviewAnswerDefer   = "defer"
)

func configurationReviewSlug(engine string) string {
	return "pose-configuration-review-" + strings.ReplaceAll(engine, ".", "-")
}

// reviewRequest is a configuration-review request and the capability it asks
// about.
type reviewRequest struct {
	View       posemodel.ActionRequestView
	Capability string
}

// configurationReviewRequests lists the engine's review requests.
func configurationReviewRequests(root string) ([]reviewRequest, error) {
	views, err := posemodel.Store{Root: root}.ListActionRequests()
	if err != nil {
		return nil, err
	}
	out := []reviewRequest{}
	for _, view := range views {
		if id := reviewCapability(view.Request); id != "" {
			out = append(out, reviewRequest{View: view, Capability: id})
		}
	}
	return out, nil
}

func reviewCapability(r posemodel.ActionRequest) string {
	if r.RequestedBy.Principal != reviewRequestedBy || !strings.HasPrefix(r.Context, reviewContextKey) {
		return ""
	}
	first := strings.SplitN(r.Context, "\n", 2)[0]
	return strings.TrimSpace(strings.TrimPrefix(first, reviewContextKey))
}

// reviewAnswerApplied reports whether the capability's state already reflects
// an answered request.
func reviewAnswerApplied(root string, req reviewRequest) bool {
	states, err := posemodel.CapabilityStates(root)
	if err != nil {
		return false
	}
	for _, state := range states {
		if state.ID != req.Capability {
			continue
		}
		switch req.View.Answer {
		case reviewAnswerAdopt:
			return state.State == posemodel.CapabilityOn
		case reviewAnswerDecline, reviewAnswerDefer:
			return state.Decision != nil && state.Decision.Request == req.View.Request.ID
		}
	}
	return false
}

const reviewSpecEN = `---
slug: {{SLUG}}
status: draft
created_at: {{DATE}}
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components:
task_type: feature
surface: minimal
delivers:
---

# Spec: Review the capabilities POSE {{ENGINE}} brings

## 1. Intent

### Goal

Decide, for each capability this project has not decided under POSE {{ENGINE}}, whether to adopt, decline or defer it. Nothing changes until the maintainer answers.

### Business value

An update brings capabilities; deciding each one explicitly keeps none in limbo and records why it is on or off.

### Constraints

Each decision is an action request to the ` + "`maintainer`" + ` role (` + "`pose action list`" + `, ` + "`pose setup`" + `). An answer is applied with ` + "`pose adopt --request <act-id> --apply`" + ` or by ` + "`pose setup`" + `; adopting dates the capability with the day it is applied, so earlier work is not re-judged.

### Non-goals

Changing anything the maintainer did not answer.

## 2. Requirements

### Functional

{{REQUIREMENTS}}

### Non-functional

- None.

### Security

- None.

### Compatibility

- None.

## 3. Technical Plan

### Affected areas

POSE policy only.

### Artifacts

- created: .pose/specs/{{DATE}}-{{SLUG}}.md
- modified: .pose/policy/adoption-decisions.json

### Technical risks

- None.

## 6. Validation

### Strategy

` + "`pose setup`" + ` lists no capability decision pending.

### Deterministic checks

#### Health
- Command: ` + "`pose doctor`" + `
- Expected: no ` + "`setup.capabilities`" + ` step

### Requirement trace

### Known gaps

## 7. Final Report

### Delivered scope

Not started.

### Residual risks

### Follow-ups
`

const reviewSpecPtBR = `---
slug: {{SLUG}}
status: draft
created_at: {{DATE}}
completed_at:
supersedes:
depends_on:
remediates:
priority: 1
components:
task_type: feature
surface: minimal
delivers:
---

# Spec: Revisar as capacidades que o POSE {{ENGINE}} traz

## 1. Intent

### Objetivo

Decidir, para cada capacidade que este projeto não decidiu sob o POSE {{ENGINE}}, se adota, recusa ou adia. Nada muda até o maintainer responder.

### Valor de negócio

Uma atualização traz capacidades; decidir cada uma explicitamente não deixa nenhuma no limbo e registra por que está ligada ou desligada.

### Restrições

Cada decisão é um action request ao papel ` + "`maintainer`" + ` (` + "`pose action list`" + `, ` + "`pose setup`" + `). Uma resposta é aplicada com ` + "`pose adopt --request <act-id> --apply`" + ` ou pelo ` + "`pose setup`" + `; adotar data a capacidade com o dia em que é aplicada, para que trabalho anterior não seja rejulgado.

### Não-objetivos

Mudar qualquer coisa que o maintainer não respondeu.

## 2. Requirements

### Funcionais

{{REQUIREMENTS}}

### Não-funcionais

- Nenhum.

### Segurança

- Nenhuma.

### Compatibilidade

- Nenhuma.

## 3. Technical Plan

### Áreas afetadas

Somente a policy do POSE.

### Artifacts

- created: .pose/specs/{{DATE}}-{{SLUG}}.md
- modified: .pose/policy/adoption-decisions.json

### Riscos técnicos

- Nenhum.

## 6. Validation

### Estratégia

` + "`pose setup`" + ` não lista decisão de capacidade pendente.

### Checks determinísticos

#### Saúde
- Comando: ` + "`pose doctor`" + `
- Esperado: nenhum passo ` + "`setup.capabilities`" + `

### Requirement trace

### Gaps conhecidos

## 7. Final Report

### Escopo entregue

Não iniciado.

### Riscos residuais

### Follow-ups
`

// scaffoldConfigurationReview writes the review spec and opens a request
// for each capability to review that has none. It returns the spec path and
// the requests it opened; nothing is written when nothing is new.
func scaffoldConfigurationReview(root, engine, locale string, now time.Time) (string, []string, error) {
	pending, err := posemodel.CapabilitiesToReview(root, engine)
	if err != nil {
		return "", nil, err
	}
	existing, err := configurationReviewRequests(root)
	if err != nil {
		return "", nil, err
	}
	asked := map[string]bool{}
	for _, req := range existing {
		switch req.View.State {
		case posemodel.ActionStateOpen:
			asked[req.Capability] = true
		case posemodel.ActionStateAnswered:
			// Answered and not applied yet: setup applies it; do not ask twice.
			if !reviewAnswerApplied(root, req) {
				asked[req.Capability] = true
			}
		}
	}
	toAsk := []posemodel.CapabilityState{}
	for _, state := range pending {
		if !asked[state.ID] {
			toAsk = append(toAsk, state)
		}
	}
	if len(toAsk) == 0 {
		return "", nil, nil
	}
	slug := configurationReviewSlug(engine)
	date := now.UTC().Format(time.DateOnly)
	rel := ""
	if matches, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*-"+slug+".md")); len(matches) > 0 {
		// One review spec per engine version: a capability that became
		// pending since is asked under the same spec, through a new request,
		// only when the spec already names it.
		r, _ := filepath.Rel(root, matches[0])
		rel = filepath.ToSlash(r)
		raw, err := os.ReadFile(matches[0])
		if err != nil {
			return "", nil, err
		}
		named := []posemodel.CapabilityState{}
		for _, state := range toAsk {
			if strings.Contains(string(raw), "`"+state.ID+"`") {
				named = append(named, state)
			}
		}
		toAsk = named
	} else {
		requirements := make([]string, 0, len(toAsk))
		for i, state := range toAsk {
			recommendation := "no recommendation"
			if state.DefaultForNew {
				recommendation = "recommended: adopt, as new instances do"
				if locale == "pt-BR" {
					recommendation = "recomendado: adotar, como instâncias novas fazem"
				}
			} else if locale == "pt-BR" {
				recommendation = "sem recomendação"
			}
			line := fmt.Sprintf("- R%d: The maintainer shall adopt, decline or defer `%s` — %s (since %s; %s).", i+1, state.ID, state.Effect, state.IntroducedIn, recommendation)
			if locale == "pt-BR" {
				line = fmt.Sprintf("- R%d: O maintainer deve adotar, recusar ou adiar `%s` — %s (desde %s; %s).", i+1, state.ID, state.Effect, state.IntroducedIn, recommendation)
			}
			requirements = append(requirements, line)
		}
		body := reviewSpecEN
		if locale == "pt-BR" {
			body = reviewSpecPtBR
		}
		body = strings.NewReplacer("{{SLUG}}", slug, "{{DATE}}", date, "{{ENGINE}}", engine, "{{REQUIREMENTS}}", strings.Join(requirements, "\n")).Replace(body)
		rel = filepath.ToSlash(filepath.Join(".pose", "specs", date+"-"+slug+".md"))
		path := filepath.Join(root, filepath.FromSlash(rel))
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			if errors.Is(err, os.ErrExist) {
				return rel, nil, nil
			}
			return "", nil, err
		}
		_, werr := file.WriteString(body)
		file.Close()
		if werr != nil {
			return "", nil, werr
		}
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", nil, err
	}
	store := posemodel.Store{Root: root}
	opened := []string{}
	for _, state := range toAsk {
		requirement := requirementNaming(string(raw), state.ID)
		if requirement == "" {
			continue
		}
		recommend := ""
		if state.DefaultForNew {
			recommend = reviewAnswerAdopt
		}
		request := posemodel.ActionRequest{
			Origin:      "spec:" + slug,
			Kind:        posemodel.ActionDecision,
			Question:    "Adopt " + state.ID + " in this project? " + state.Effect + ".",
			Context:     reviewContextKey + state.ID + "\nIntroduced in POSE " + state.IntroducedIn + ". Applied with `pose adopt --request <this id> --apply` or by `pose setup` once answered.",
			RequestedBy: posemodel.ActionPrincipal{Principal: reviewRequestedBy},
			Recipient:   posemodel.ObligationActor{Role: reviewRecipient},
			Options: []posemodel.ActionOption{
				{ID: reviewAnswerAdopt, Consequence: "turned on, dated the day it is applied; earlier work is not re-judged"},
				{ID: reviewAnswerDecline, Consequence: "stays off; the reason is recorded so it is not asked again"},
				{ID: reviewAnswerDefer, Consequence: "stays off; asked again under a newer POSE"},
			},
			Recommend: recommend,
			Targets:   []posemodel.NodeRef{{Artifact: "self", Kind: "requirement", ID: requirement}},
			Effects:   []posemodel.ObligationEffect{{Phase: posemodel.PhaseCloseout, Mode: posemodel.EffectBlock}},
		}
		view, err := store.OpenActionRequest(request, now)
		if err != nil {
			return rel, opened, err
		}
		opened = append(opened, view.Request.ID)
	}
	return rel, opened, nil
}

// requirementNaming returns the id of the requirement line that names the
// capability, e.g. R2.
func requirementNaming(spec, capability string) string {
	for _, line := range strings.Split(spec, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- R") || !strings.Contains(line, "`"+capability+"`") {
			continue
		}
		id, _, ok := strings.Cut(strings.TrimPrefix(line, "- "), ":")
		if ok {
			return id
		}
	}
	return ""
}

// applyReviewAnswer applies an answered configuration-review request.
func applyReviewAnswer(root string, req reviewRequest, date string) (string, error) {
	switch req.View.Answer {
	case reviewAnswerAdopt:
		docs, err := posemodel.LoadPolicyDocs(root)
		if err != nil {
			return "", err
		}
		entry, ok := posemodel.LookupCatalogEntry(req.Capability)
		if !ok {
			return "", fmt.Errorf("unknown capability %s", req.Capability)
		}
		if !entry.Adopted(docs) {
			if blocker := posemodel.CatalogAdoptBlocker(root, docs, entry); blocker != "" {
				return "", errors.New(blocker)
			}
		}
		entry.Adopt(docs, date)
		if _, _, _, _, err := docs.Rendered(posemodel.Store{Root: root}); err != nil {
			return "", fmt.Errorf("the resulting policy would be refused: %v", err)
		}
		if err := docs.Write(root, posemodel.Store{Root: root}); err != nil {
			return "", err
		}
		_ = posemodel.ClearAdoptionDecision(root, req.Capability)
		advanceReviewedVersion(root)
		return req.Capability + " adopted from " + date, nil
	case reviewAnswerDecline, reviewAnswerDefer:
		decision := posemodel.AdoptionDeclined
		if req.View.Answer == reviewAnswerDefer {
			decision = posemodel.AdoptionDeferred
		}
		reason := ""
		for i := len(req.View.Events) - 1; i >= 0; i-- {
			if req.View.Events[i].Type == posemodel.ActionEventAnswered {
				reason = strings.TrimSpace(req.View.Events[i].Reason)
				break
			}
		}
		if reason == "" {
			reason = "answered " + req.View.Answer + " in " + req.View.Request.ID
		}
		if err := posemodel.RecordAdoptionDecisionFor(root, req.Capability, posemodel.AdoptionDecision{Decision: decision, Reason: reason, Date: date,
			Request: req.View.Request.ID, Version: engineRelease()}); err != nil {
			return "", err
		}
		advanceReviewedVersion(root)
		return req.Capability + " " + decision + ": " + reason, nil
	}
	return "", fmt.Errorf("%s answered %q, which is not adopt, decline or defer", req.View.Request.ID, req.View.Answer)
}

// loadReviewRequest finds an answerable configuration-review request by id.
func loadReviewRequest(root, id string) (reviewRequest, error) {
	view, err := posemodel.Store{Root: root}.LoadActionRequest(id)
	if err != nil {
		return reviewRequest{}, err
	}
	capability := reviewCapability(view.Request)
	if capability == "" {
		return reviewRequest{}, fmt.Errorf("%s is not a configuration-review request", id)
	}
	if view.State != posemodel.ActionStateAnswered {
		return reviewRequest{}, fmt.Errorf("%s is %s: only an answered request is applied (answer it with `pose action resolve`)", id, view.State)
	}
	return reviewRequest{View: view, Capability: capability}, nil
}
