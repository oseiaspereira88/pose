package cli

// `pose state` (spec pose-project-state-artifact): the native project-state
// artifact writer. Derived sections are computed here, in the CLI/Harness
// write layer (ADR-003) — never in internal/pose, which only reads and
// validates the persisted artifact. Every derived provider reuses the same
// subsystem APIs the equivalent commands already expose (ListSpecs,
// ListRoadmaps, ListKnowledge, LoadCapabilityAssessment, ListReports,
// collectFollowups) instead of re-parsing artifacts on its own.

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

	"github.com/harne8/pose-mcp/internal/pose"
)

// stateSectionOrder is the fixed v1 section list (R2/R3): curated first,
// then derived, in a stable, documented order.
var stateSectionOrder = []struct {
	name    string
	curated bool
}{
	{"Resumo executivo", true},
	{"Direção atual", true},
	{"Specs & Roadmaps", false},
	{"Follow-ups", false},
	{"Capabilities", false},
	{"Decisões & Conhecimento", false},
	{"Validação & Evidência", false},
	{"Arquitetura", false},
}

const (
	curatedExecSummaryPlaceholder = "<!-- Preencha um resumo executivo de 2-4 frases: o que este projeto é e onde está agora. -->"
	curatedDirectionPlaceholder   = "<!-- Preencha as prioridades vigentes: o que está em foco agora e o que vem a seguir. -->"
)

type stateHistorySection struct {
	Hash string `json:"hash"`
	Body string `json:"body"`
}

type stateHistoryEntry struct {
	GeneratedAt    string                         `json:"generated_at"`
	BaselineCommit string                         `json:"baseline_commit"`
	Sections       map[string]stateHistorySection `json:"sections"`
}

func cmdState(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return cmdStateValidate(root, stdout, stderr)
	}
	switch args[0] {
	case "init":
		return cmdStateInit(root, stdout, stderr)
	case "refresh":
		return cmdStateRefresh(root, args[1:], stdout, stderr)
	case "diff":
		return cmdStateDiff(root, stdout, stderr)
	default:
		return usageError(stderr, "Usage: pose state [init|refresh [--if-stale]|diff]")
	}
}

func cmdStateInit(root string, stdout, stderr io.Writer) int {
	store := pose.Store{Root: root}
	if store.HasProjectState() {
		fmt.Fprintf(stderr, "Error: project state already exists: %s\n", store.StatePath())
		return 1
	}
	if _, err := runRefresh(root, refreshOptions{Trigger: "manual"}, false); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Project state initialized: %s\n", store.StatePath())
	return 0
}

// cmdStateRefresh is the manual/CI entry point (R7): full refresh of every
// derived section. --if-stale skips the work entirely when the artifact is
// not yet stale — the cheap check a CI job wants to run every build without
// paying refresh cost on every single one.
func cmdStateRefresh(root string, args []string, stdout, stderr io.Writer) int {
	ifStale := false
	for _, a := range args {
		switch a {
		case "--if-stale":
			ifStale = true
		default:
			return usageError(stderr, "Usage: pose state refresh [--if-stale]")
		}
	}
	store := pose.Store{Root: root}
	if !store.HasProjectState() {
		fmt.Fprintf(stderr, "Error: project state not found: %s (run `pose state init` first)\n", store.StatePath())
		return 1
	}
	if ifStale {
		state, err := store.ProjectState(context.Background(), "")
		if err == nil && !state.Staleness.Stale {
			fmt.Fprintln(stdout, "Project state is not stale; --if-stale skipped the refresh. Result: SUCCESS")
			return 0
		}
	}
	if _, err := runRefresh(root, refreshOptions{Trigger: "manual"}, true); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Project state refreshed: %s\n", store.StatePath())
	return 0
}

func cmdStateValidate(root string, stdout, stderr io.Writer) int {
	store := pose.Store{Root: root}
	if !store.HasProjectState() {
		fmt.Fprintln(stdout, "Project state not initialized (run `pose state init`). Additive: this is a valid state.")
		return 0
	}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	brokenPointers := store.ValidatePointers(state)

	fmt.Fprintf(stdout, "Project state: %s\n", state.Path)
	fmt.Fprintf(stdout, "policy at generation: %s (current policy: %s)\n",
		pose.FormatStalenessPolicy(state.StalenessPolicyAtGeneration), pose.FormatStalenessPolicy(store.LoadStatePolicy()))
	fmt.Fprintf(stdout, "generated_at=%s baseline_commit=%s\n", state.GeneratedAt, state.BaselineCommit)
	fmt.Fprintf(stdout, "staleness: stale=%v age_days=%d/%d commits_since=%d/%d reason=%s\n",
		state.Staleness.Stale, state.Staleness.AgeDays, state.Staleness.MaxAgeDays,
		state.Staleness.CommitsSince, state.Staleness.MaxCommits, valueOrDash(state.Staleness.Reason))

	tampered := 0
	for _, sec := range state.Sections {
		if sec.Tampered {
			tampered++
			fmt.Fprintf(stdout, "[TAMPERED] section %q was hand-edited since the last refresh\n", sec.Name)
		}
	}
	for _, issue := range brokenPointers {
		fmt.Fprintf(stdout, "[BROKEN POINTER] %s\n", issue)
	}
	if state.RefreshPending != "" {
		fmt.Fprintf(stdout, "[REFRESH PENDING] a %q-triggered refresh failed and has not been retried yet — run `pose state refresh`\n", state.RefreshPending)
	}

	if tampered > 0 || len(brokenPointers) > 0 {
		fmt.Fprintln(stdout, "Result: FAILURE")
		return 1
	}
	if state.Staleness.Stale {
		fmt.Fprintln(stdout, "Result: STALE (not a failure — run `pose state refresh`)")
		return 0
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}

func cmdStateDiff(root string, stdout, stderr io.Writer) int {
	store := pose.Store{Root: root}
	entries, err := readStateHistory(store.StateHistoryPath())
	if err != nil {
		fmt.Fprintf(stderr, "Error: reading state history: %v\n", err)
		return 1
	}
	if len(entries) < 2 {
		fmt.Fprintln(stdout, "Not enough history yet (need at least 2 `pose state refresh` runs). Result: SUCCESS")
		return 0
	}
	from, to := entries[len(entries)-2], entries[len(entries)-1]
	fmt.Fprintf(stdout, "Project state diff: %s -> %s\n", from.GeneratedAt, to.GeneratedAt)
	names := make([]string, 0, len(to.Sections))
	for name := range to.Sections {
		names = append(names, name)
	}
	sort.Strings(names)
	changed := 0
	for _, name := range names {
		before, after := from.Sections[name], to.Sections[name]
		if before.Hash == after.Hash {
			continue
		}
		changed++
		fmt.Fprintf(stdout, "\n## %s\n", name)
		for _, line := range lineDiff(before.Body, after.Body) {
			fmt.Fprintln(stdout, line)
		}
	}
	if changed == 0 {
		fmt.Fprintln(stdout, "No section changed between the last two refreshes.")
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}

// lineDiff is a deliberately simple set-based line diff — the spec asks for
// "diff de contagens e apontadores por seção", not a general-purpose LCS
// diff algorithm.
func lineDiff(before, after string) []string {
	beforeLines := splitNonEmptyLines(before)
	afterLines := splitNonEmptyLines(after)
	beforeSet := map[string]bool{}
	for _, l := range beforeLines {
		beforeSet[l] = true
	}
	afterSet := map[string]bool{}
	for _, l := range afterLines {
		afterSet[l] = true
	}
	var out []string
	for _, l := range beforeLines {
		if !afterSet[l] {
			out = append(out, "- "+l)
		}
	}
	for _, l := range afterLines {
		if !beforeSet[l] {
			out = append(out, "+ "+l)
		}
	}
	return out
}

func splitNonEmptyLines(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

func appendStateHistory(path string, entry stateHistoryEntry) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func readStateHistory(path string) ([]stateHistoryEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []stateHistoryEntry
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry stateHistoryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // one unparseable line must not break the whole history
		}
		out = append(out, entry)
	}
	return out, nil
}

// gitHeadCommit resolves HEAD; "0000000" when git is unavailable, matching
// assessBaselineCommit's convention (internal/cli/assess.go) — a
// syntactically valid placeholder so the artifact still parses, never a
// real ref, so staleness-by-commit degrades to "unknown" at read time.
func gitHeadCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return "0000000"
	}
	commit := strings.TrimSpace(string(out))
	if commit == "" {
		return "0000000"
	}
	return commit
}

func valueOrDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}
