package cli

// assessment-staleness consumer (spec pose-capability-assessment-triggers):
// the second real consumer of the post-event hook registry built for
// pose-project-state-refresh-contract — proves RegisterHook/EmitHook is
// generic enough without touching that code. Registered on spec_closeout
// (release_cut stays inert, same gap as the sister spec: no producer
// inside pose-mcp today). Never mutates Score/Target — only marks a
// mechanism `stale` and lets `pose assess snapshot` clear it (R4).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func init() {
	RegisterHook("spec_closeout", assessmentStalenessConsumer)
}

// capabilityTriggerPolicy is read from .pose/policy/capabilities.json
// (R7): the anti-noise threshold. Defaults are conservative (direct hits
// only, one hit is enough) so the mechanism proves itself before a project
// tunes it.
type capabilityTriggerPolicy struct {
	MinHits int    `json:"min_hits"`
	Level   string `json:"level"` // "direct" (default) | "any"
	// DefaultOwner and ReviewDays (R3) size the demand's owner/SLA when the
	// project hasn't opted into per-capability ownership yet — same
	// "unowned" fallback convention as spec follow-ups.
	DefaultOwner string `json:"default_owner"`
	ReviewDays   int    `json:"review_days"`
}

func loadCapabilityTriggerPolicy(root string) capabilityTriggerPolicy {
	policy := capabilityTriggerPolicy{MinHits: 1, Level: "direct", DefaultOwner: "unowned", ReviewDays: 14}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "capabilities.json"))
	if err != nil {
		return policy
	}
	var parsed capabilityTriggerPolicy
	if json.Unmarshal(raw, &parsed) != nil {
		return policy
	}
	if parsed.MinHits > 0 {
		policy.MinHits = parsed.MinHits
	}
	if parsed.Level == "direct" || parsed.Level == "any" {
		policy.Level = parsed.Level
	}
	if parsed.DefaultOwner != "" {
		policy.DefaultOwner = parsed.DefaultOwner
	}
	if parsed.ReviewDays > 0 {
		policy.ReviewDays = parsed.ReviewDays
	}
	return policy
}

// assessmentStalenessConsumer is deliberately best-effort with no strict
// mode of its own (unlike state-refresh): it always returns nil so it can
// never fail the triggering command, regardless of .pose/policy/state.json
// strict_refresh (that setting belongs to the sister consumer). A resolution
// failure degrades to the capability_mapping_unavailable signal (R2
// Compatibilidade) instead of an error.
func assessmentStalenessConsumer(root string, ev HookEvent) error {
	store := pose.Store{Root: root}
	if !store.HasCapabilityAssessment() {
		return nil
	}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		return nil // best-effort: an unreadable assessment is not this event's problem
	}
	policy := loadCapabilityTriggerPolicy(root)

	hitsByMechanism, resolvedVia := resolveCapabilityHits(root, ev, assessment, policy)
	logCapabilityTriggerOutcome(root, ev, resolvedVia, hitsByMechanism)
	if len(hitsByMechanism) == 0 {
		return nil
	}

	raw, err := os.ReadFile(store.CapabilityAssessmentPath())
	if err != nil {
		return nil
	}
	content := string(raw)
	trigger := "spec:" + ev.Target
	since := ev.At.UTC().Format(time.RFC3339)
	changed := false
	for _, mechanismID := range sortedMechanismKeys(hitsByMechanism) {
		updated, err := addStaleMark(content, mechanismID, pose.StaleTrigger{
			Since: since, Trigger: trigger, Hits: hitsByMechanism[mechanismID],
		})
		if err != nil {
			continue // mechanism referenced by the map but absent locally — skip, don't fail
		}
		content, changed = updated, true
	}
	if !changed {
		return nil
	}
	_ = os.WriteFile(store.CapabilityAssessmentPath(), []byte(content), 0o644)
	return nil
}

func sortedMechanismKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// resolveCapabilityHits derives {mechanism_id: [hit identifiers]} for the
// event, preferring components_hit (spec_slug mode — the same resolver
// graphforge-components-hit-contract already exposes for exactly this
// "what did this spec touch" question) and falling back to matching
// touched files against each mechanism's declared `paths:` globs when
// GraphForge is unavailable. resolvedVia describes which path was taken,
// for the visible refresh-log signal (R2 Compatibilidade).
func resolveCapabilityHits(root string, ev HookEvent, assessment *pose.CapabilityAssessment, policy capabilityTriggerPolicy) (map[string][]string, string) {
	if caller := resolveComponentsHitCaller(root); caller != nil {
		if hits, ok := resolveViaComponentsHit(root, caller, ev, policy); ok {
			return hits, "components_hit"
		}
	}
	if hits, ok := resolveViaPaths(root, ev, assessment); ok {
		return hits, "paths"
	}
	return nil, "capability_mapping_unavailable"
}

func resolveViaComponentsHit(root string, caller componentsHitCaller, ev HookEvent, policy capabilityTriggerPolicy) (map[string][]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, ok, err := caller.ComponentsHitForSpec(ctx, ev.Target)
	if err != nil || !ok || result == nil {
		return nil, false
	}
	hits := map[string][]string{}
	for _, hit := range result.Hits {
		level, _ := hit["level"].(string)
		if policy.Level == "direct" && level != "direct" {
			continue
		}
		componentID, _ := hit["component_id"].(string)
		caps, _ := hit["capabilities"].([]any)
		for _, c := range caps {
			cap, _ := c.(map[string]any)
			mechanismID, _ := cap["mechanism_id"].(string)
			if mechanismID == "" {
				continue
			}
			hits[mechanismID] = appendUniqueString(hits[mechanismID], componentID)
		}
	}
	return filterByMinHits(hits, policy.MinHits), true
}

func resolveViaPaths(root string, ev HookEvent, assessment *pose.CapabilityAssessment) (map[string][]string, bool) {
	touched := touchedFilesForCommit(root, ev.Commit)
	if len(touched) == 0 {
		return nil, false
	}
	hits := map[string][]string{}
	for _, mechanism := range assessment.Mechanisms {
		if len(mechanism.Paths) == 0 {
			continue
		}
		for _, file := range touched {
			for _, glob := range mechanism.Paths {
				if matched, _ := filepath.Match(glob, file); matched {
					hits[mechanism.ID] = appendUniqueString(hits[mechanism.ID], file)
					break
				}
			}
		}
	}
	if len(hits) == 0 {
		return nil, false
	}
	return hits, true
}

// touchedFilesForCommit is a deliberately simple proxy for "files this
// event's change touched" when GraphForge cannot resolve the spec's full
// commit history: the event's own commit diffed against its parent. Less
// complete than components_hit's spec_slug resolution (which considers
// every commit associated with the spec), documented as a Decision.
func touchedFilesForCommit(root, commit string) []string {
	if commit == "" || commit == "0000000" {
		return nil
	}
	out, err := exec.Command("git", "-C", root, "show", "--pretty=format:", "--name-only", commit).Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	return files
}

func filterByMinHits(hits map[string][]string, minHits int) map[string][]string {
	if minHits <= 1 {
		return hits
	}
	filtered := map[string][]string{}
	for mechanismID, componentIDs := range hits {
		if len(componentIDs) >= minHits {
			filtered[mechanismID] = componentIDs
		}
	}
	return filtered
}

func appendUniqueString(values []string, next string) []string {
	if next == "" {
		return values
	}
	for _, v := range values {
		if v == next {
			return values
		}
	}
	return append(values, next)
}

// logCapabilityTriggerOutcome records this consumer's own visible signal
// in the shared refresh-log (R2 Compatibilidade: "capability_mapping_
// unavailable" must be visible, never silent) — disambiguated from the
// state-refresh consumer's entries via Consumer.
func logCapabilityTriggerOutcome(root string, ev HookEvent, resolvedVia string, hits map[string][]string) {
	entry := refreshLogEntry{
		At: time.Now().UTC().Format(time.RFC3339), Trigger: ev.Kind, Target: ev.Target,
		Consumer: "assessment-staleness", ChangedSections: sortedMechanismKeys(hits), Result: "ok",
	}
	if resolvedVia == "capability_mapping_unavailable" {
		entry.Result = "skipped"
		entry.Error = "capability_mapping_unavailable"
	}
	_ = appendRefreshLog(root, entry)
}

// collectCapabilityStaleFollowups projects every pending StaleTrigger in
// .pose/capabilities/assessment.md as a synthetic followup (R3) — reusing
// the exact same shape `pose followups` already aggregates, with an
// identifiable origin (Spec: "capability:<mechanism>") instead of writing
// into any spec's Final Report. The assessment.md StaleTriggers ARE the
// single source of truth (spec Restrições); this is a read-only view, not
// a second store — a follow-up here disappears the moment `pose assess
// snapshot` clears the mark, with no separate close step to keep in sync.
func collectCapabilityStaleFollowups(root string) []followup {
	store := pose.Store{Root: root}
	if !store.HasCapabilityAssessment() {
		return nil
	}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		return nil
	}
	policy := loadCapabilityTriggerPolicy(root)
	var out []followup
	for _, mechanism := range assessment.Mechanisms {
		for _, trigger := range mechanism.StaleTriggers {
			review := ""
			if since, err := time.Parse(time.RFC3339, trigger.Since); err == nil {
				review = since.AddDate(0, 0, policy.ReviewDays).Format("2006-01-02")
			}
			hitsText := "nenhum componente resolvido"
			if len(trigger.Hits) > 0 {
				hitsText = strings.Join(trigger.Hits, ", ")
			}
			out = append(out, followup{
				Spec: "capability:" + mechanism.ID, SpecStatus: "assessment-stale",
				RawDisposition: "open", Target: trigger.Trigger,
				Text: fmt.Sprintf("Reavaliar capability %s — atingida por %s (componentes: %s)",
					mechanism.ID, trigger.Trigger, hitsText),
				Owner: policy.DefaultOwner, Criticality: "medium", Review: review,
			})
		}
	}
	return out
}

// staleMechanismReport is the --json shape for `pose assess stale`: one
// entry per mechanism with at least one pending StaleTrigger.
type staleMechanismReport struct {
	Mechanism string              `json:"mechanism"`
	Triggers  []pose.StaleTrigger `json:"triggers"`
}

// assessStale lists every mechanism with pending StaleTriggers (R5) — the
// same data collectCapabilityStaleFollowups projects into `pose followups`,
// surfaced directly for a quick "what's stale right now" check without
// pulling in the rest of the follow-up backlog.
func assessStale(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	asJSON := false
	for _, arg := range args {
		switch arg {
		case "--json":
			asJSON = true
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), arg)
			return 2
		}
	}
	store := pose.Store{Root: root}
	if !store.HasCapabilityAssessment() {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: no capability assessment found (run `pose assess init`)",
			"Erro: nenhum capability assessment encontrado (rode `pose assess init`)"))
		return 1
	}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		fmt.Fprintf(stderr, "pose assess stale: %v\n", err)
		return 1
	}
	var report []staleMechanismReport
	for _, mechanism := range assessment.Mechanisms {
		if len(mechanism.StaleTriggers) == 0 {
			continue
		}
		report = append(report, staleMechanismReport{Mechanism: mechanism.ID, Triggers: mechanism.StaleTriggers})
	}
	if asJSON {
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}
	if len(report) == 0 {
		fmt.Fprintln(stdout, cliText(locale, "No stale capabilities.", "Nenhuma capability marcada como stale."))
		return 0
	}
	for _, entry := range report {
		fmt.Fprintf(stdout, cliText(locale, "- %s: %d pending trigger(s)\n", "- %s: %d gatilho(s) pendente(s)\n"),
			entry.Mechanism, len(entry.Triggers))
		for _, trigger := range entry.Triggers {
			hitsText := strings.Join(trigger.Hits, ", ")
			if hitsText == "" {
				hitsText = cliText(locale, "no resolved components", "nenhum componente resolvido")
			}
			fmt.Fprintf(stdout, "    %s  %s  (%s)\n", trigger.Since, trigger.Trigger, hitsText)
		}
	}
	return 0
}

// assessRequest lets a human or agent manually flag a mechanism as stale
// (R5) — the same addStaleMark path assessmentStalenessConsumer uses, just
// triggered on demand instead of by a spec_closeout event. Trigger is
// recorded as "manual" (or "manual:<reason>") so it is distinguishable from
// event-sourced marks in the same StaleTriggers list.
func assessRequest(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	var mechanismID, reason string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mechanism":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --mechanism requires an id", "Erro: --mechanism exige um id"))
				return 2
			}
			i++
			mechanismID = args[i]
		case "--reason":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --reason requires text", "Erro: --reason exige um texto"))
				return 2
			}
			i++
			reason = args[i]
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if mechanismID == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --mechanism is required", "Erro: --mechanism é obrigatório"))
		return 2
	}
	store := pose.Store{Root: root}
	if !store.HasCapabilityAssessment() {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: no capability assessment found (run `pose assess init`)",
			"Erro: nenhum capability assessment encontrado (rode `pose assess init`)"))
		return 1
	}
	raw, err := os.ReadFile(store.CapabilityAssessmentPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose assess request: %v\n", err)
		return 1
	}
	trigger := "manual"
	if reason != "" {
		trigger = "manual:" + reason
	}
	updated, err := addStaleMark(string(raw), mechanismID, pose.StaleTrigger{
		Since: time.Now().UTC().Format(time.RFC3339), Trigger: trigger,
	})
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: %v\n", "Erro: %v\n"), err)
		return 1
	}
	if err := os.WriteFile(store.CapabilityAssessmentPath(), []byte(updated), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose assess request: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Mechanism %s marked stale (run `pose assess snapshot` to reassess and clear it)\n",
		"Mecanismo %s marcado como stale (rode `pose assess snapshot` para reavaliar e limpar)\n"), mechanismID)
	return 0
}
