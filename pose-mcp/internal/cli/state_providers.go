package cli

// Derived section providers for `pose state` (R2): each one reuses the same
// subsystem API an equivalent command already exposes and returns counts +
// typed pointers only — never copied artifact content (spec Restrições).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// deriveSection computes one named derived section's body. status is
// non-empty only for sections that degrade openly when their data source
// is unavailable (e.g. "unavailable" for Arquitetura today).
func deriveSection(store pose.Store, name string) (body, status string) {
	switch name {
	case "Specs & Roadmaps":
		return provideSpecsRoadmaps(store), ""
	case "Follow-ups":
		return provideFollowups(store.Root), ""
	case "Capabilities":
		return provideCapabilities(store), ""
	case "Decisões & Conhecimento":
		return provideDecisionsKnowledge(store), ""
	case "Validação & Evidência":
		return provideValidationEvidence(store), ""
	case "Arquitetura":
		return provideArchitecture(), "unavailable"
	case "Docs":
		return provideDocs(store), ""
	default:
		return fmt.Sprintf("(no provider registered for section %q)", name), ""
	}
}

func provideSpecsRoadmaps(store pose.Store) string {
	specs, err := store.ListSpecs("")
	if err != nil {
		return fmt.Sprintf("- specs: erro ao listar (%v)", err)
	}
	counts := map[string]int{}
	var closeouts []pose.Spec
	for _, s := range specs {
		counts[s.Status]++
		if s.Status == "done" && s.CompletedAt != "" {
			closeouts = append(closeouts, s)
		}
	}
	roadmaps, err := store.ListRoadmaps()
	if err != nil {
		roadmaps = nil
	}
	rCounts := map[string]int{}
	for _, r := range roadmaps {
		rCounts[r.Status]++
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("- specs: total=%d draft=%d in-progress=%d blocked=%d done=%d superseded=%d abandoned=%d",
		len(specs), counts["draft"], counts["in-progress"], counts["blocked"], counts["done"], counts["superseded"], counts["abandoned"]))
	lines = append(lines, fmt.Sprintf("- roadmaps: total=%d active=%d done=%d", len(roadmaps), rCounts["active"], rCounts["done"]))

	sort.Slice(closeouts, func(i, j int) bool { return closeouts[i].CompletedAt > closeouts[j].CompletedAt })
	if len(closeouts) > 0 {
		lines = append(lines, "- últimos closeouts:")
		limit := closeouts
		if len(limit) > 5 {
			limit = limit[:5]
		}
		for _, c := range limit {
			lines = append(lines, fmt.Sprintf("  - spec:%s (%s)", c.Slug, c.CompletedAt))
		}
		if len(closeouts) > 5 {
			lines = append(lines, fmt.Sprintf("  - ... e mais %d (ver `pose_list_specs status:done`)", len(closeouts)-5))
		}
	}
	return strings.Join(lines, "\n")
}

func provideFollowups(root string) string {
	entries := collectFollowups(root)
	open := 0
	byCrit := map[string]int{}
	today := time.Now().UTC().Format("2006-01-02")
	var overdue []followup
	for _, f := range entries {
		if f.RawDisposition != "open" {
			continue
		}
		open++
		if f.Criticality != "" {
			byCrit[f.Criticality]++
		}
		if f.Review != "" && f.Review < today {
			overdue = append(overdue, f)
		}
	}
	unclassified := open - byCrit["high"] - byCrit["medium"] - byCrit["low"]

	var lines []string
	lines = append(lines, fmt.Sprintf("- abertos: %d", open))
	lines = append(lines, fmt.Sprintf("- por criticidade: high=%d medium=%d low=%d sem-classificação=%d",
		byCrit["high"], byCrit["medium"], byCrit["low"], unclassified))
	lines = append(lines, fmt.Sprintf("- vencidos (review < hoje): %d", len(overdue)))

	sort.Slice(overdue, func(i, j int) bool { return overdue[i].Review < overdue[j].Review })
	limit := overdue
	if len(limit) > 10 {
		limit = limit[:10]
	}
	for _, f := range limit {
		lines = append(lines, fmt.Sprintf("  - spec:%s (owner:%s review:%s)", f.Spec, f.Owner, f.Review))
	}
	if len(overdue) > 10 {
		lines = append(lines, fmt.Sprintf("  - ... e mais %d vencidos (ver `pose followups --open`)", len(overdue)-10))
	}
	return strings.Join(lines, "\n")
}

func provideCapabilities(store pose.Store) string {
	if !store.HasCapabilityAssessment() {
		return "- assessment: ausente (rode `pose assess init`)"
	}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		return fmt.Sprintf("- assessment: erro ao carregar (%v)", err)
	}
	total, sumScore, sumTarget, retired := 0, 0, 0, 0
	for _, m := range assessment.Mechanisms {
		total++
		sumScore += m.Score
		sumTarget += m.Target
		if m.Retired {
			retired++
		}
	}
	avgScore, avgTarget := 0, 0
	if total > 0 {
		avgScore, avgTarget = sumScore/total, sumTarget/total
	}
	daysSinceAssessed := "unknown"
	if assessedAt, err := time.Parse("2006-01-02", assessment.AssessedAt); err == nil {
		daysSinceAssessed = fmt.Sprintf("%d", int(time.Since(assessedAt).Hours()/24))
	}
	return strings.Join([]string{
		fmt.Sprintf("- assessment: presente, baseline_commit=commit:%s, assessed_at=%s (%s dias atrás)", assessment.BaselineCommit, assessment.AssessedAt, daysSinceAssessed),
		fmt.Sprintf("- mecanismos: %d, score médio=%d, target médio=%d, retirados=%d", total, avgScore, avgTarget, retired),
	}, "\n")
}

func provideDecisionsKnowledge(store pose.Store) string {
	adrPaths, _ := filepath.Glob(filepath.Join(store.Root, ".pose", "adr", "*.md"))
	sort.Strings(adrPaths)
	entries, err := store.ListKnowledge()
	if err != nil {
		entries = nil
	}
	active, expired := 0, 0
	today := time.Now().UTC().Format("2006-01-02")
	for _, e := range entries {
		if e.ExpiresAt == "" || e.ExpiresAt >= today {
			active++
		} else {
			expired++
		}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("- ADRs: total=%d", len(adrPaths)))
	recent := adrPaths
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	for i := len(recent) - 1; i >= 0; i-- {
		lines = append(lines, fmt.Sprintf("  - adr:%s", filepath.Base(recent[i])))
	}
	lines = append(lines, fmt.Sprintf("- knowledge: total=%d ativo=%d expirado=%d", len(entries), active, expired))
	return strings.Join(lines, "\n")
}

func provideValidationEvidence(store pose.Store) string {
	var lines []string

	reports, err := store.ListReports()
	switch {
	case err != nil:
		lines = append(lines, fmt.Sprintf("- execution history: erro ao carregar (%v)", err))
	case len(reports) == 0:
		lines = append(lines, "- execution history: nenhum registro em .pose/reports/history")
	default:
		latest := reports[0] // ListReports sorts descending by generated_at
		cutoff := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
		total, ok := 0, 0
		for _, r := range reports {
			if r.GeneratedAt < cutoff {
				continue
			}
			total++
			if strings.EqualFold(r.Outcome, "pass") || strings.EqualFold(r.Outcome, "success") {
				ok++
			}
		}
		lines = append(lines, fmt.Sprintf("- último registro: task=%s outcome=%s (%s)", latest.TaskSlug, latest.Outcome, latest.GeneratedAt))
		lines = append(lines, fmt.Sprintf("- últimos 30 dias: total=%d outcome_ok=%d outcome_outro=%d", total, ok, total-ok))
	}

	mdReports := newestReportsFirst(store.Root)
	lines = append(lines, fmt.Sprintf("- reports revisados (.md): total=%d", len(mdReports)))
	recent := mdReports
	if len(recent) > 5 {
		recent = recent[:5]
	}
	for _, name := range recent {
		lines = append(lines, fmt.Sprintf("  - report:%s", name))
	}
	return strings.Join(lines, "\n")
}

// newestReportsFirst lists .pose/reports/*.md by modification time,
// newest first — filename order is unreliable here because not every
// report predates the "<date>-<slug>.md" naming convention (e.g.
// README.md, workspace-experience-e2e.md sort after dated names
// alphabetically despite being unrelated to recency). Ties break on name
// for determinism.
func newestReportsFirst(root string) []string {
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "reports", "*.md"))
	type entry struct {
		name    string
		modTime time.Time
	}
	entries := make([]entry, 0, len(paths))
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		entries = append(entries, entry{name: filepath.Base(p), modTime: info.ModTime()})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].modTime.Equal(entries[j].modTime) {
			return entries[i].modTime.After(entries[j].modTime)
		}
		return entries[i].name < entries[j].name
	})
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.name
	}
	return names
}

// provideDocs projects `pose docs-check`'s live result (spec
// pose-docs-governance-contract R8) — opt-in by presence of the manifest,
// same degrade-by-absence contract as provideCapabilities above.
func provideDocs(store pose.Store) string {
	if !store.HasDocsManifest() {
		return "- manifest: ausente (rode `pose docs-init`)"
	}
	manifest, err := store.LoadDocsManifest()
	if err != nil {
		return fmt.Sprintf("- manifest: erro ao carregar (%v)", err)
	}
	result := store.CheckDocs(context.Background(), manifest)
	lines := []string{
		fmt.Sprintf("- manifest: presente, profile=%s, roots=%s", manifest.Profile, strings.Join(manifest.Roots, ",")),
		fmt.Sprintf("- docs: declaradas=%d não-declaradas=%d vencidas=%d erros=%d avisos=%d",
			result.Totals.Declared, result.Totals.Undeclared, result.Totals.Stale, result.Totals.Errors, result.Totals.Warnings),
	}
	return strings.Join(lines, "\n")
}

// provideArchitecture always degrades openly (status "unavailable"): no
// producer of a local GraphForge export file exists yet in this repo (spec
// Não-objetivos — the export is an optional, future data source, never a
// live service call from here).
func provideArchitecture() string {
	return "GraphForge export local ainda não é publicado por nenhum produtor neste repositório; seção indisponível nesta versão (spec pose-project-state-artifact, Não-objetivos e Compatibilidade)."
}
