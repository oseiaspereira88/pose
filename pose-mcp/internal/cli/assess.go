package cli

// pose assess (spec pose-capability-mechanism): capability assessment as a
// POSE-native artifact. Bare `assess` validates structure, evidence and
// staleness; `init` scaffolds the 16 default mechanisms; `snapshot` appends
// to the append-only history; `diff` compares two snapshots. Scores are
// human judgment — nothing here computes one.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

type capabilityPolicy struct {
	StaleAfterDays    int `json:"stale_after_days"`
	StaleAfterCommits int `json:"stale_after_commits"`
}

func defaultCapabilityPolicy() capabilityPolicy {
	return capabilityPolicy{StaleAfterDays: 30, StaleAfterCommits: 200}
}

func loadCapabilityPolicy(root string) capabilityPolicy {
	policy := defaultCapabilityPolicy()
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "capabilities.json"))
	if err != nil {
		return policy
	}
	var loaded capabilityPolicy
	if json.Unmarshal(raw, &loaded) == nil {
		if loaded.StaleAfterDays > 0 {
			policy.StaleAfterDays = loaded.StaleAfterDays
		}
		if loaded.StaleAfterCommits > 0 {
			policy.StaleAfterCommits = loaded.StaleAfterCommits
		}
	}
	return policy
}

func cmdAssess(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	if len(args) > 0 {
		switch args[0] {
		case "init":
			return assessInit(root, stdout, stderr, locale)
		case "snapshot":
			return assessSnapshot(root, stdout, stderr, locale)
		case "diff":
			return assessDiff(root, args[1:], stdout, stderr, locale)
		case "--json":
			return assessValidate(root, true, stdout, stderr, locale)
		default:
			fmt.Fprintln(stderr, cliText(locale,
				"Usage: pose assess [init|snapshot|diff [--from <ts>] [--to <ts>] [--json]|--json]",
				"Uso: pose assess [init|snapshot|diff [--from <ts>] [--to <ts>] [--json]|--json]"))
			return 2
		}
	}
	return assessValidate(root, false, stdout, stderr, locale)
}

func assessInit(root string, stdout, stderr io.Writer, locale cliLocale) int {
	store := pose.Store{Root: root}
	path := store.CapabilityAssessmentPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(stderr, cliText(locale,
			"Error: %s already exists; edit it instead of re-initializing\n",
			"Erro: %s já existe; edite-o em vez de re-inicializar\n"), path)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "pose assess init: %v\n", err)
		return 1
	}
	content := capabilityTemplate(assessBaselineCommit(root))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose assess init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Capability assessment created: %s (16 default mechanisms; edit scores, targets, evidence and prose)\n",
		"Capability assessment criado: %s (16 mecanismos default; edite scores, targets, evidência e prosa)\n"), path)
	return 0
}

// assessBaselineCommit resolves HEAD for the scaffold; "0000000" when git is
// unavailable so the artifact still parses and the user edits it in.
func assessBaselineCommit(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "0000000"
	}
	commit := strings.TrimSpace(string(out))
	if commit == "" {
		return "0000000"
	}
	return commit
}

type assessReport struct {
	Path        string   `json:"path"`
	Mechanisms  int      `json:"mechanisms"`
	Errors      []string `json:"errors,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	AssessedAt  string   `json:"assessed_at"`
	AgeDays     int      `json:"age_days"`
	CommitsLag  string   `json:"commits_since_baseline"` // number or "unknown"
	StaleByDays bool     `json:"stale_by_days"`
	StaleByLag  bool     `json:"stale_by_commits"`
}

// runAssessValidation is the shared R6 validation: structure, evidence,
// stable-id contract, staleness. It never writes.
func runAssessValidation(root string) (*assessReport, error) {
	store := pose.Store{Root: root}
	assessment, err := store.LoadCapabilityAssessment()
	if err != nil {
		return nil, err
	}
	report := &assessReport{Path: assessment.Path, Mechanisms: len(assessment.Mechanisms), AssessedAt: assessment.AssessedAt}
	report.Errors = append(report.Errors, store.ValidateCapabilityEvidence(assessment)...)

	history, err := pose.LoadCapabilityHistory(store.CapabilityHistoryPath())
	if err != nil {
		return nil, err
	}
	if effective := pose.EffectiveSnapshots(history); len(effective) > 0 {
		latest := effective[len(effective)-1]
		for _, id := range pose.RenumberedMechanisms(latest, assessment) {
			report.Errors = append(report.Errors,
				fmt.Sprintf("mechanism %q was published in the latest snapshot but is gone from the assessment; retire it (`- retired: true`) instead of removing it", id))
		}
	}

	policy := loadCapabilityPolicy(root)
	if assessed, err := time.Parse("2006-01-02", assessment.AssessedAt); err == nil {
		report.AgeDays = int(time.Since(assessed).Hours() / 24)
		if report.AgeDays > policy.StaleAfterDays {
			report.StaleByDays = true
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("assessment is %d days old (policy: %d); consider reassessing", report.AgeDays, policy.StaleAfterDays))
		}
	}
	report.CommitsLag = "unknown"
	if out, err := exec.Command("git", "-C", root, "rev-list", "--count", assessment.BaselineCommit+"..HEAD").Output(); err == nil {
		lag := strings.TrimSpace(string(out))
		report.CommitsLag = lag
		if n, err := strconv.Atoi(lag); err == nil && n > policy.StaleAfterCommits {
			report.StaleByLag = true
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("baseline_commit is %d commits behind HEAD (policy: %d); consider reassessing", n, policy.StaleAfterCommits))
		}
	}
	return report, nil
}

func assessValidate(root string, asJSON bool, stdout, stderr io.Writer, locale cliLocale) int {
	report, err := runAssessValidation(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose assess: %v\n", err)
		return 1
	}
	if asJSON {
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		if len(report.Errors) > 0 {
			return 1
		}
		return 0
	}
	for _, issue := range report.Errors {
		fmt.Fprintf(stdout, "[ERRO] %s\n", issue)
	}
	for _, warning := range report.Warnings {
		fmt.Fprintf(stdout, "[AVISO] %s\n", warning)
	}
	if len(report.Errors) > 0 {
		fmt.Fprintf(stdout, cliText(locale,
			"Result: FAILURE — %d error(s) in %s\n",
			"Resultado: FALHA — %d erro(s) em %s\n"), len(report.Errors), report.Path)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Result: SUCCESS — %d mechanisms, assessed %s (%d days ago, %s commits since baseline)\n",
		"Resultado: SUCESSO — %d mecanismos, avaliado em %s (%d dias atrás, %s commits desde a baseline)\n"),
		report.Mechanisms, report.AssessedAt, report.AgeDays, report.CommitsLag)
	return 0
}

func assessSnapshot(root string, stdout, stderr io.Writer, locale cliLocale) int {
	report, err := runAssessValidation(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	if len(report.Errors) > 0 {
		for _, issue := range report.Errors {
			fmt.Fprintf(stdout, "[ERRO] %s\n", issue)
		}
		fmt.Fprintln(stderr, cliText(locale,
			"Error: fix validation errors before snapshotting",
			"Erro: corrija os erros de validação antes do snapshot"))
		return 1
	}
	store := pose.Store{Root: root}
	raw, err := os.ReadFile(store.CapabilityAssessmentPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	assessment, err := pose.ParseCapabilityAssessment(string(raw))
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	event := pose.CapabilitySnapshot{
		Schema:         pose.CapabilitySnapshotSchema,
		At:             time.Now().UTC().Format(time.RFC3339),
		BaselineCommit: assessment.BaselineCommit,
		ContentHash:    pose.CapabilityContentHash(string(raw)),
		Scores:         assessment.ScoresOf(),
	}
	history, err := pose.LoadCapabilityHistory(store.CapabilityHistoryPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	if len(history) > 0 && history[len(history)-1].ContentHash == event.ContentHash {
		fmt.Fprintf(stdout, cliText(locale,
			"No change since the last snapshot (content hash %s); nothing appended\n",
			"Sem mudança desde o último snapshot (content hash %s); nada acrescentado\n"), event.ContentHash)
		return 0
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	f, err := os.OpenFile(store.CapabilityHistoryPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	defer f.Close()
	if _, err := f.Write(append(encoded, '\n')); err != nil {
		fmt.Fprintf(stderr, "pose assess snapshot: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Snapshot appended: %s (%d mechanisms, content hash %s)\n",
		"Snapshot acrescentado: %s (%d mecanismos, content hash %s)\n"), event.At, len(event.Scores), event.ContentHash)
	return 0
}

func assessDiff(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	var fromTS, toTS, against string
	asJSON := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--against":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --against requires a project id", "Erro: --against exige um project id"))
				return 2
			}
			i++
			against = args[i]
		case "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --from requires a timestamp", "Erro: --from exige um timestamp"))
				return 2
			}
			i++
			fromTS = args[i]
		case "--to":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --to requires a timestamp", "Erro: --to exige um timestamp"))
				return 2
			}
			i++
			toTS = args[i]
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if against != "" {
		return assessAgainst(root, against, asJSON, stdout, stderr, locale)
	}
	store := pose.Store{Root: root}
	history, err := pose.LoadCapabilityHistory(store.CapabilityHistoryPath())
	if err != nil {
		fmt.Fprintf(stderr, "pose assess diff: %v\n", err)
		return 1
	}
	effective := pose.EffectiveSnapshots(history)
	if len(effective) < 2 {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: need at least two snapshots to diff (run `pose assess snapshot`)",
			"Erro: são necessários ao menos dois snapshots para diff (rode `pose assess snapshot`)"))
		return 1
	}
	from := effective[len(effective)-2]
	to := effective[len(effective)-1]
	if fromTS != "" {
		if picked := pickSnapshot(effective, fromTS); picked != nil {
			from = *picked
		} else {
			fmt.Fprintf(stderr, cliText(locale, "Error: no snapshot matches --from %s\n", "Erro: nenhum snapshot corresponde a --from %s\n"), fromTS)
			return 1
		}
	}
	if toTS != "" {
		if picked := pickSnapshot(effective, toTS); picked != nil {
			to = *picked
		} else {
			fmt.Fprintf(stderr, cliText(locale, "Error: no snapshot matches --to %s\n", "Erro: nenhum snapshot corresponde a --to %s\n"), toTS)
			return 1
		}
	}
	diff := pose.DiffCapabilitySnapshots(from, to)
	if asJSON {
		encoded, _ := json.MarshalIndent(diff, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}
	fmt.Fprintf(stdout, "%s -> %s\n", diff.FromAt, diff.ToAt)
	for _, entry := range diff.Raised {
		fmt.Fprintf(stdout, "  ^ %s: %d -> %d\n", entry.ID, entry.From, entry.To)
	}
	for _, entry := range diff.Lowered {
		fmt.Fprintf(stdout, "  v %s: %d -> %d\n", entry.ID, entry.From, entry.To)
	}
	for _, id := range diff.Added {
		fmt.Fprintf(stdout, "  + %s\n", id)
	}
	for _, id := range diff.Removed {
		fmt.Fprintf(stdout, "  - %s\n", id)
	}
	for _, id := range diff.Retired {
		fmt.Fprintf(stdout, "  r %s (retired)\n", id)
	}
	fmt.Fprintf(stdout, cliText(locale, "  = %d stable\n", "  = %d estáveis\n"), len(diff.Stable))
	return 0
}

func pickSnapshot(events []pose.CapabilitySnapshot, prefix string) *pose.CapabilitySnapshot {
	for i := len(events) - 1; i >= 0; i-- {
		if strings.HasPrefix(events[i].At, prefix) {
			event := events[i]
			return &event
		}
	}
	return nil
}

// capabilityTemplate scaffolds the method's 16 default mechanisms. Scores
// start at 0 with a modest default target — the point of the artifact is
// that a human assesses; the scaffold only provides stable ids.
func capabilityTemplate(baselineCommit string) string {
	type mech struct{ id, title string }
	defaults := []mech{
		{"install-upgrade-runtime", "Install, upgrade and local-first runtime"},
		{"spec-lifecycle-closeout", "Spec lifecycle and closeout"},
		{"task-routing-workflows-skills", "Task routing, workflows, rules and skills"},
		{"dependencies-readiness-roadmaps", "Dependencies, readiness and roadmaps"},
		{"validation-structural-integrity", "Validation matrix and structural checks"},
		{"evidence-history-insights", "Evidence, history and insights"},
		{"followups-recurrence", "Follow-ups and recurrence"},
		{"operational-knowledge", "Knowledge governance"},
		{"mcp-agent-interop", "MCP and agent interoperability"},
		{"policy-identity-audit", "Policy, identity and audit"},
		{"ci-release-supply-chain", "CI, release and supply-chain trust"},
		{"import-adoption-interop", "Import and adoption interoperability"},
		{"metrics-observability", "Metrics and observability"},
		{"docs-localization-diagnostics", "Documentation, localization and diagnostics"},
		{"extensibility-ecosystem", "Extensibility and ecosystem"},
		{"multi-repo-enterprise", "Multi-repository and enterprise operation"},
	}
	var b strings.Builder
	fmt.Fprintf(&b, "---\nschema_version: %d\nassessed_at: %s\nbaseline_commit: %s\nmethod: describe how this assessment was measured\n---\n\n",
		pose.CapabilityAssessmentSchema, time.Now().UTC().Format("2006-01-02"), baselineCommit)
	b.WriteString("# Capability assessment\n\n")
	b.WriteString("Scores are human judgment on a 0-5 scale; the target is not always 5.\n")
	b.WriteString("Evidence uses typed references (spec:/report:/adr:/knowledge:/doc:/commit:/check:/url:).\n")
	for _, m := range defaults {
		fmt.Fprintf(&b, "\n## Mechanism: %s\n- title: %s\n- score: 0\n- target: 3\n- evidence:\n- gaps:\n\nDescribe the current state and why the score holds.\n", m.id, m.title)
	}
	return b.String()
}

// assessAgainst compares the local score vector with another project's,
// resolved through the same authorization allowlist as the cross-repository
// portfolio projection (self + scanned projects dir + POSE_PROJECT_ROOTS).
// Scores and targets only — prose never crosses roots.
func assessAgainst(root, projectID string, asJSON bool, stdout, stderr io.Writer, locale cliLocale) int {
	known, err := discoverAuthorizedProjects(root, "")
	if err != nil {
		fmt.Fprintf(stderr, "pose assess diff --against: %v\n", err)
		return 1
	}
	otherRoot, authorized := known[projectID]
	if !authorized {
		fmt.Fprintf(stderr, cliText(locale,
			"Error: project %q is not an authorized root (self, HARNE8_PROJECTS_DIR scan, or POSE_PROJECT_ROOTS)\n",
			"Erro: projeto %q não é um root autorizado (self, scan de HARNE8_PROJECTS_DIR, ou POSE_PROJECT_ROOTS)\n"), projectID)
		return 1
	}
	local, err := pose.Store{Root: root}.LoadCapabilityAssessment()
	if err != nil {
		fmt.Fprintf(stderr, "pose assess diff --against: local: %v\n", err)
		return 1
	}
	other, err := pose.Store{Root: otherRoot}.LoadCapabilityAssessment()
	if err != nil {
		fmt.Fprintf(stderr, "pose assess diff --against: %s: %v\n", projectID, err)
		return 1
	}
	type cell struct {
		Score  int  `json:"score"`
		Target int  `json:"target"`
		Has    bool `json:"present"`
	}
	matrix := map[string]map[string]cell{}
	add := func(project string, a *pose.CapabilityAssessment) {
		for _, m := range a.Mechanisms {
			if m.Retired {
				continue
			}
			if matrix[m.ID] == nil {
				matrix[m.ID] = map[string]cell{}
			}
			matrix[m.ID][project] = cell{Score: m.Score, Target: m.Target, Has: true}
		}
	}
	add("local", local)
	add(projectID, other)
	if asJSON {
		encoded, _ := json.MarshalIndent(map[string]any{"against": projectID, "matrix": matrix}, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
		return 0
	}
	var ids []string
	for id := range matrix {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	fmt.Fprintf(stdout, "%-36s %-12s %s\n", "mechanism", "local", projectID)
	for _, id := range ids {
		row := matrix[id]
		fmt.Fprintf(stdout, "%-36s %-12s %s\n", id, formatCell(row["local"].Score, row["local"].Target, row["local"].Has), formatCell(row[projectID].Score, row[projectID].Target, row[projectID].Has))
	}
	return 0
}

func formatCell(score, target int, present bool) string {
	if !present {
		return "-"
	}
	return fmt.Sprintf("%d/%d", score, target)
}
