package pose

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeStateFixture(t *testing.T, root, generatedAt, baseline string, sections map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("---\nschema_version: 1\ngenerated_at: " + generatedAt + "\nbaseline_commit: " + baseline +
		"\nstaleness_policy: max_age_days=7,max_commits=20\n---\n\n# Project State\n")
	order := []struct{ name, kind string }{
		{"Resumo executivo", "curated"},
		{"Direção atual", "curated"},
		{"Specs & Roadmaps", "derived"},
		{"Follow-ups", "derived"},
	}
	for _, sec := range order {
		body := sections[sec.name]
		b.WriteString("\n## " + sec.name + "\n")
		if sec.kind == "curated" {
			b.WriteString("<!-- state:curated -->\n\n" + body + "\n")
		} else {
			b.WriteString("<!-- state:derived hash:" + ContentHash12(body) + " -->\n\n" + body + "\n")
		}
	}
	path := filepath.Join(root, ".pose", "state", "project-state.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseProjectState_CuratedAndDerivedSections(t *testing.T) {
	root := t.TempDir()
	writeStateFixture(t, root, "2026-07-23T10:00:00Z", "abc1234", map[string]string{
		"Resumo executivo": "Projeto X faz Y.",
		"Direção atual":    "Foco em Z este mês.",
		"Specs & Roadmaps": "- specs: total=3 draft=1 done=2",
		"Follow-ups":       "- abertos: 2",
	})
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatalf("ProjectState: %v", err)
	}
	if len(state.Sections) != 4 {
		t.Fatalf("sections = %d, want 4: %+v", len(state.Sections), state.Sections)
	}
	if state.Tampered {
		t.Fatal("untampered fixture reported as tampered")
	}
	if state.StalenessPolicyAtGeneration.MaxAgeDays != 7 || state.StalenessPolicyAtGeneration.MaxCommits != 20 {
		t.Errorf("staleness_policy_at_generation = %+v, want {7 20}", state.StalenessPolicyAtGeneration)
	}
	var found bool
	for _, sec := range state.Sections {
		if sec.Name == "Specs & Roadmaps" {
			found = true
			if sec.Kind != "derived" || sec.Body != "- specs: total=3 draft=1 done=2" {
				t.Errorf("Specs & Roadmaps section = %+v", sec)
			}
		}
		if sec.Name == "Resumo executivo" && sec.Kind != "curated" {
			t.Errorf("Resumo executivo kind = %q, want curated", sec.Kind)
		}
	}
	if !found {
		t.Fatal("Specs & Roadmaps section missing")
	}
}

func TestStalenessPolicy_FormatParseRoundTrip(t *testing.T) {
	policy := StatePolicy{MaxAgeDays: 14, MaxCommits: 50}
	formatted := FormatStalenessPolicy(policy)
	parsed := parseStalenessPolicyField(formatted)
	if parsed != policy {
		t.Fatalf("round trip = %+v, want %+v (formatted: %q)", parsed, policy, formatted)
	}
}

func TestStalenessPolicy_ParseToleratesAbsentOrMalformed(t *testing.T) {
	for _, value := range []string{"", "garbage", "max_age_days", "max_age_days=abc"} {
		if got := parseStalenessPolicyField(value); got != (StatePolicy{}) {
			t.Errorf("parseStalenessPolicyField(%q) = %+v, want zero value", value, got)
		}
	}
}

func TestParseProjectState_DetectsTamperedSection(t *testing.T) {
	root := t.TempDir()
	writeStateFixture(t, root, "2026-07-23T10:00:00Z", "abc1234", map[string]string{
		"Resumo executivo": "resumo",
		"Direção atual":    "direção",
		"Specs & Roadmaps": "- specs: total=3",
		"Follow-ups":       "- abertos: 2",
	})
	path := filepath.Join(root, ".pose", "state", "project-state.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Hand-edit the derived body without touching its stamped hash.
	tampered := strings.Replace(string(raw), "- specs: total=3", "- specs: total=999 (tampered by hand)", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o644); err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatalf("ProjectState: %v", err)
	}
	if !state.Tampered {
		t.Fatal("hand-edited derived section was not flagged as tampered")
	}
	for _, sec := range state.Sections {
		if sec.Name == "Specs & Roadmaps" && !sec.Tampered {
			t.Errorf("section %q not individually flagged tampered", sec.Name)
		}
		if sec.Name == "Follow-ups" && sec.Tampered {
			t.Errorf("untouched section %q was wrongly flagged tampered", sec.Name)
		}
	}
}

func TestParseProjectState_RejectsUnsupportedSchema(t *testing.T) {
	fm := map[string]string{"schema_version": "2", "generated_at": "2026-07-23T10:00:00Z", "baseline_commit": "abc1234"}
	if _, err := ParseProjectState(fm, "## Resumo executivo\n<!-- state:curated -->\n\nx\n"); err == nil {
		t.Fatal("unsupported schema_version was accepted")
	}
}

func TestParseProjectState_RejectsMissingFrontmatter(t *testing.T) {
	fm := map[string]string{"schema_version": "1"}
	if _, err := ParseProjectState(fm, "## Resumo executivo\n<!-- state:curated -->\n\nx\n"); err == nil {
		t.Fatal("missing generated_at/baseline_commit was accepted")
	}
}

func TestProjectState_NotFoundIsNominal(t *testing.T) {
	store := Store{Root: t.TempDir()}
	if _, err := store.ProjectState(context.Background(), ""); err == nil {
		t.Fatal("missing artifact must error, not silently succeed")
	}
	if store.HasProjectState() {
		t.Fatal("HasProjectState true for a root without the artifact")
	}
}

func TestProjectState_SectionFilter(t *testing.T) {
	root := t.TempDir()
	writeStateFixture(t, root, "2026-07-23T10:00:00Z", "abc1234", map[string]string{
		"Resumo executivo": "r", "Direção atual": "d",
		"Specs & Roadmaps": "- specs: total=1", "Follow-ups": "- abertos: 0",
	})
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "follow-ups") // case-insensitive
	if err != nil {
		t.Fatalf("ProjectState(section): %v", err)
	}
	if len(state.Sections) != 1 || state.Sections[0].Name != "Follow-ups" {
		t.Fatalf("section filter = %+v", state.Sections)
	}
	if _, err := store.ProjectState(context.Background(), "Does Not Exist"); err == nil {
		t.Fatal("unknown section must error")
	}
}

func TestProjectState_StalenessByAge(t *testing.T) {
	root := t.TempDir()
	old := time.Now().UTC().AddDate(0, 0, -30).Format(time.RFC3339)
	writeStateFixture(t, root, old, "abc1234", map[string]string{
		"Resumo executivo": "r", "Direção atual": "d",
		"Specs & Roadmaps": "- specs: total=1", "Follow-ups": "- abertos: 0",
	})
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !state.Staleness.Stale || state.Staleness.Reason != "age_exceeded" {
		t.Fatalf("staleness = %+v, want stale by age", state.Staleness)
	}
}

func runStateTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(out)
}

func TestProjectState_StalenessByCommitCount(t *testing.T) {
	root := t.TempDir()
	runStateTestGit(t, root, "init")
	runStateTestGit(t, root, "config", "user.email", "test@example.com")
	runStateTestGit(t, root, "config", "user.name", "State Test")
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	runStateTestGit(t, root, "add", "a.txt")
	runStateTestGit(t, root, "commit", "-m", "first")
	baseline := strings.TrimSpace(runStateTestGit(t, root, "rev-parse", "HEAD"))
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte{byte('2' + i)}, 0o644); err != nil {
			t.Fatal(err)
		}
		runStateTestGit(t, root, "commit", "-am", "change")
	}

	writeStateFixture(t, root, time.Now().UTC().Format(time.RFC3339), baseline, map[string]string{
		"Resumo executivo": "r", "Direção atual": "d",
		"Specs & Roadmaps": "- specs: total=1", "Follow-ups": "- abertos: 0",
	})
	if err := os.MkdirAll(filepath.Join(root, ".pose", "policy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "policy", "state.json"), []byte(`{"max_commits":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if state.Staleness.CommitsSince != 3 {
		t.Fatalf("commits_since = %d, want 3", state.Staleness.CommitsSince)
	}
	if !state.Staleness.Stale || state.Staleness.Reason != "commits_exceeded" {
		t.Fatalf("staleness = %+v, want stale by commit count", state.Staleness)
	}
}

func TestExtractPointersAndValidate(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".pose", "specs", "demo-spec"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".pose", "specs", "demo-spec", "spec.md"),
		[]byte("---\nslug: demo-spec\nstatus: done\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeStateFixture(t, root, "2026-07-23T10:00:00Z", "abc1234", map[string]string{
		"Resumo executivo": "resumo com spec:demo-spec citado em prosa (nunca escaneado)",
		"Direção atual":    "d",
		"Specs & Roadmaps": "- último closeout: spec:demo-spec\n- outro: spec:ghost-spec",
		"Follow-ups":       "- abertos: 0",
	})
	store := Store{Root: root}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	pointers := state.ExtractPointers()
	if len(pointers) != 2 {
		t.Fatalf("ExtractPointers = %v, want 2 (curated prose must not be scanned)", pointers)
	}
	issues := store.ValidatePointers(state)
	if len(issues) != 1 || !strings.Contains(issues[0], "ghost-spec") {
		t.Fatalf("ValidatePointers = %v, want exactly one issue mentioning ghost-spec", issues)
	}
}
