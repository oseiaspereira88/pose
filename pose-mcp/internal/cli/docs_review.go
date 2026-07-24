package cli

// Docs review-pending triggers (spec pose-docs-assessment-followups): the
// third real consumer of the post-event hook registry built for
// pose-project-state-refresh-contract — reused, unmodified, alongside the
// assessment-staleness consumer (spec pose-capability-assessment-triggers).
// Marks live in .pose/docs-review.jsonl, never inside a doc file (spec
// Restrições). Never mutates a doc; resolving (`pose docs-review resolve`)
// is the human/agent act.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

func init() {
	RegisterHook("spec_closeout", docsReviewConsumer)
}

// docsReviewPolicy is read from .pose/policy/docs.json (R3): same shape
// and same conservative defaults as capabilityTriggerPolicy, kept as a
// separate file/type because docs and capabilities are independently
// tunable knobs, not one shared policy.
type docsReviewPolicy struct {
	MinHits      int    `json:"min_hits"`
	Level        string `json:"level"` // "direct" (default) | "any"
	DefaultOwner string `json:"default_owner"`
	ReviewDays   int    `json:"review_days"`
}

func loadDocsReviewPolicy(root string) docsReviewPolicy {
	policy := docsReviewPolicy{MinHits: 1, Level: "direct", DefaultOwner: "unowned", ReviewDays: 14}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "docs.json"))
	if err != nil {
		return policy
	}
	var parsed docsReviewPolicy
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

// docsReviewConsumer is best-effort with no strict mode of its own — same
// posture as assessmentStalenessConsumer: it always returns nil.
func docsReviewConsumer(root string, ev HookEvent) error {
	store := pose.Store{Root: root}
	if !store.HasDocsManifest() {
		return nil
	}
	manifest, err := store.LoadDocsManifest()
	if err != nil {
		return nil
	}
	policy := loadDocsReviewPolicy(root)

	hits, resolvedVia := resolveDocsReviewHits(root, ev, manifest, policy)
	logDocsReviewOutcome(root, ev, resolvedVia, hits)
	if len(hits) == 0 {
		return nil
	}

	existing, _ := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	pending := pose.PendingDocsReviews(existing)

	trigger := "spec:" + ev.Target
	since := ev.At.UTC().Format(time.RFC3339)
	for _, doc := range sortedMechanismKeys(hits) {
		if alreadyMarked(pending[doc], trigger, hits[doc]) {
			continue
		}
		_ = appendDocsReviewEvent(root, pose.DocsReviewEvent{
			At: since, Doc: doc, Kind: "marked", Trigger: trigger, Hits: hits[doc],
		})
	}
	return nil
}

// alreadyMarked reports whether an identical (same trigger, same hits)
// mark is already open on the doc — avoids growing the JSONL with a
// no-op re-mark when a closeout gate reruns without anything changing.
func alreadyMarked(triggers []pose.DocsReviewTrigger, trigger string, hits []string) bool {
	for _, t := range triggers {
		if t.Trigger == trigger && reflect.DeepEqual(t.Hits, hits) {
			return true
		}
	}
	return false
}

// resolveDocsReviewHits derives {doc_path: [hit identifiers]}, preferring
// components_hit (component ids matched against `owns: ["component:<id>"]`
// entries — inert until a local GraphForge export producer exists, same
// degrade-by-absence contract as capability triggers) and falling back to
// touched files matched against each doc's `owns:` paths/globs — the
// spec's own compatibility note: this fallback is the doc-review
// mechanism's full-strength path, not a degraded one, because `owns:` is
// expressed as paths by default.
func resolveDocsReviewHits(root string, ev HookEvent, manifest *pose.DocsManifest, policy docsReviewPolicy) (map[string][]string, string) {
	if caller := resolveComponentsHitCaller(root); caller != nil {
		if hits, ok := resolveDocsViaComponentsHit(caller, ev, manifest, policy); ok {
			return hits, "components_hit"
		}
	}
	if hits, ok := resolveDocsViaPaths(root, ev, manifest); ok {
		return hits, "paths"
	}
	return nil, "docs_mapping_unavailable"
}

func resolveDocsViaComponentsHit(caller componentsHitCaller, ev HookEvent, manifest *pose.DocsManifest, policy docsReviewPolicy) (map[string][]string, bool) {
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
		if componentID == "" {
			continue
		}
		ref := "component:" + componentID
		for _, entry := range manifest.Entries {
			if pose.MatchesOwns(ref, entry.Owns) {
				hits[entry.Path] = appendUniqueString(hits[entry.Path], componentID)
			}
		}
	}
	return filterByMinHits(hits, policy.MinHits), len(hits) > 0
}

func resolveDocsViaPaths(root string, ev HookEvent, manifest *pose.DocsManifest) (map[string][]string, bool) {
	touched := touchedFilesForCommit(root, ev.Commit)
	if len(touched) == 0 {
		return nil, false
	}
	hits := map[string][]string{}
	for _, entry := range manifest.Entries {
		if len(entry.Owns) == 0 {
			continue
		}
		for _, file := range touched {
			if pose.MatchesOwns(file, entry.Owns) {
				hits[entry.Path] = appendUniqueString(hits[entry.Path], file)
			}
		}
	}
	if len(hits) == 0 {
		return nil, false
	}
	return hits, true
}

func appendDocsReviewEvent(root string, ev pose.DocsReviewEvent) error {
	path := pose.Store{Root: root}.DocsReviewPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	encoded, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	_, err = f.Write(append(encoded, '\n'))
	return err
}

// logDocsReviewOutcome mirrors logCapabilityTriggerOutcome — visible signal
// in the shared refresh-log, disambiguated via Consumer.
func logDocsReviewOutcome(root string, ev HookEvent, resolvedVia string, hits map[string][]string) {
	entry := refreshLogEntry{
		At: time.Now().UTC().Format(time.RFC3339), Trigger: ev.Kind, Target: ev.Target,
		Consumer: "docs-review", ChangedSections: sortedMechanismKeys(hits), Result: "ok",
	}
	if resolvedVia == "docs_mapping_unavailable" {
		entry.Result = "skipped"
		entry.Error = "docs_mapping_unavailable"
	}
	_ = appendRefreshLog(root, entry)
}

// collectDocsReviewFollowups projects every pending DocsReviewTrigger as a
// synthetic followup (R2) — same read-only-projection shape as
// collectCapabilityStaleFollowups, origin "docs:<doc>".
func collectDocsReviewFollowups(root string) []followup {
	store := pose.Store{Root: root}
	if !store.HasDocsManifest() {
		return nil
	}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		return nil
	}
	pending := pose.PendingDocsReviews(events)
	if len(pending) == 0 {
		return nil
	}
	manifest, _ := store.LoadDocsManifest()
	ownerFor := map[string]string{}
	if manifest != nil {
		for _, e := range manifest.Entries {
			ownerFor[e.Path] = e.Owner
		}
	}
	policy := loadDocsReviewPolicy(root)
	var out []followup
	for _, doc := range sortedDocsReviewKeys(pending) {
		for _, trigger := range pending[doc] {
			owner := ownerFor[doc]
			if owner == "" {
				owner = policy.DefaultOwner
			}
			review := ""
			if since, err := time.Parse(time.RFC3339, trigger.Since); err == nil {
				review = since.AddDate(0, 0, policy.ReviewDays).Format("2006-01-02")
			}
			hitsText := "nenhum componente resolvido"
			if len(trigger.Hits) > 0 {
				hitsText = strings.Join(trigger.Hits, ", ")
			}
			out = append(out, followup{
				Spec: "docs:" + doc, SpecStatus: "review-pending",
				RawDisposition: "open", Target: trigger.Trigger,
				Text: fmt.Sprintf("Revisar doc %s — atingida por %s (componentes: %s)",
					doc, trigger.Trigger, hitsText),
				Owner: owner, Criticality: "medium", Review: review,
			})
		}
	}
	return out
}

func sortedDocsReviewKeys(m map[string][]pose.DocsReviewTrigger) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func cmdDocsReview(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) == 0 {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose docs-review resolve <doc> [--no-change --reason <text>] [--commit <sha>]|request <doc>|--all-stale [--reason <text>]",
			"Uso: pose docs-review resolve <doc> [--no-change --reason <texto>] [--commit <sha>]|request <doc>|--all-stale [--reason <texto>]"))
		return 2
	}
	switch args[0] {
	case "resolve":
		return docsReviewResolve(root, args[1:], stdout, stderr, locale)
	case "request":
		return docsReviewRequest(root, args[1:], stdout, stderr, locale)
	default:
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[0])
		return 2
	}
}

func docsReviewResolve(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	var doc, reason, commit string
	noChange := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--no-change":
			noChange = true
		case "--reason":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --reason requires text", "Erro: --reason exige um texto"))
				return 2
			}
			i++
			reason = args[i]
		case "--commit":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --commit requires a hash", "Erro: --commit exige um hash"))
				return 2
			}
			i++
			commit = args[i]
		default:
			if doc != "" || strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
				return 2
			}
			doc = args[i]
		}
	}
	if doc == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: doc path is required", "Erro: o caminho da doc é obrigatório"))
		return 2
	}
	if noChange && reason == "" {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: --no-change requires --reason", "Erro: --no-change exige --reason"))
		return 2
	}
	store := pose.Store{Root: root}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-review resolve: %v\n", err)
		return 1
	}
	pending := pose.PendingDocsReviews(events)
	if len(pending[doc]) == 0 {
		fmt.Fprintf(stderr, cliText(locale, "Error: no pending review for %q\n", "Erro: nenhuma revisão pendente para %q\n"), doc)
		return 1
	}
	outcome := "updated"
	if noChange {
		outcome = "no_change_needed"
	} else if commit == "" {
		commit = gitHeadCommit(root)
	}
	ev := pose.DocsReviewEvent{
		At: time.Now().UTC().Format(time.RFC3339), Doc: doc, Kind: "resolved",
		Outcome: outcome, Commit: commit, Reason: reason,
	}
	if err := appendDocsReviewEvent(root, ev); err != nil {
		fmt.Fprintf(stderr, "pose docs-review resolve: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Doc %s resolved: %s\n", "Doc %s resolvida: %s\n"), doc, outcome)
	return 0
}

func docsReviewRequest(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	allStale := false
	var doc, reason string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--all-stale":
			allStale = true
		case "--reason":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --reason requires text", "Erro: --reason exige um texto"))
				return 2
			}
			i++
			reason = args[i]
		default:
			if doc != "" || strings.HasPrefix(args[i], "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
				return 2
			}
			doc = args[i]
		}
	}
	if !allStale && doc == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: doc path or --all-stale is required", "Erro: caminho da doc ou --all-stale é obrigatório"))
		return 2
	}
	store := pose.Store{Root: root}
	if !store.HasDocsManifest() {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: no docs manifest found (run `pose docs-init`)",
			"Erro: nenhum manifesto de docs encontrado (rode `pose docs-init`)"))
		return 1
	}
	manifest, err := store.LoadDocsManifest()
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-review request: %v\n", err)
		return 1
	}
	events, err := pose.LoadDocsReviewEvents(store.DocsReviewPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-review request: %v\n", err)
		return 1
	}
	pending := pose.PendingDocsReviews(events)
	since := time.Now().UTC().Format(time.RFC3339)

	if allStale {
		result := store.CheckDocs(context.Background(), manifest)
		marked := 0
		for _, issue := range result.Issues {
			if issue.Rule != "stale" {
				continue
			}
			if alreadyMarked(pending[issue.Path], "stale-bridge", nil) {
				continue
			}
			if err := appendDocsReviewEvent(root, pose.DocsReviewEvent{
				At: since, Doc: issue.Path, Kind: "marked", Trigger: "stale-bridge",
			}); err != nil {
				fmt.Fprintf(stderr, "pose docs-review request: %v\n", err)
				return 1
			}
			marked++
		}
		fmt.Fprintf(stdout, cliText(locale, "Marked %d stale doc(s) for review\n", "%d doc(s) vencidas marcadas para revisão\n"), marked)
		return 0
	}

	trigger := "manual"
	if reason != "" {
		trigger = "manual:" + reason
	}
	if err := appendDocsReviewEvent(root, pose.DocsReviewEvent{
		At: since, Doc: doc, Kind: "marked", Trigger: trigger,
	}); err != nil {
		fmt.Fprintf(stderr, "pose docs-review request: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Doc %s marked for review (run `pose docs-review resolve %s` when done)\n",
		"Doc %s marcada para revisão (rode `pose docs-review resolve %s` quando terminar)\n"), doc, doc)
	return 0
}
