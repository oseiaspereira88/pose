package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	usagepkg "github.com/harne8/pose-mcp/internal/usage"
)

type commandUsageResult struct {
	SemanticOutcome    string
	Findings           []usagepkg.Finding
	FindingCount       int
	FindingsBySeverity map[string]int
	FindingSetComplete bool
}

type usageFinding struct {
	ID       string
	Severity string
}

type usageOutput struct {
	io.Writer
	result *commandUsageResult
}

func (w *usageOutput) note(result commandUsageResult) {
	copy := result
	w.result = &copy
}

func noteCommandUsage(stdout io.Writer, result commandUsageResult) {
	if output, ok := stdout.(*usageOutput); ok {
		output.note(result)
	}
}

func mainWithUsage(args []string, stdout, stderr io.Writer) int {
	started := time.Now()
	root, rootErr := projectRoot()
	original := append([]string(nil), args...)
	tool := "help"
	if len(original) > 0 {
		tool = original[0]
	}
	wrapped := &usageOutput{Writer: stdout}
	code := mainCommand(args, wrapped, stderr)
	if rootErr != nil || !shouldRecordCLIUsage(tool, code) {
		return code
	}
	result := defaultCommandUsage(tool, code)
	if wrapped.result != nil {
		result = *wrapped.result
	}
	execution := "completed"
	if code == 2 {
		execution = "invalid"
	} else if code != 0 && !isGateCommand(tool) {
		execution = "failed"
	}
	scopeArgs := original[1:]
	observation := usagepkg.Observation{
		At: time.Now().UTC(), Tool: tool, Surface: "cli",
		DurationMS:       float64(time.Since(started).Microseconds()) / 1000,
		ExecutionOutcome: execution, SemanticOutcome: result.SemanticOutcome,
		Findings: result.Findings, FindingCount: result.FindingCount,
		FindingsBySeverity: result.FindingsBySeverity,
		FindingSetComplete: result.FindingSetComplete,
		Scope:              strings.Join(scopeArgs, "\x00"), Version: Version,
	}
	_ = usagepkg.Record(root, observation)
	return code
}

func shouldRecordCLIUsage(tool string, code int) bool {
	if os.Getenv("POSE_USAGE_SUPPRESS_CLI") == "1" || strings.EqualFold(os.Getenv("POSE_USAGE_SUPPRESS_CLI"), "true") {
		return false
	}
	switch tool {
	case "usage", "help", "-h", "--help", "version", "--version", "-v", "telemetry", "serve-mcp":
		return false
	}
	// A free-form first token is not part of the bounded command catalog. A
	// recognized command with invalid arguments is still useful operational
	// evidence and is recorded with execution_outcome=invalid.
	return code != 2 || isKnownCLICommand(tool)
}

func isKnownCLICommand(tool string) bool {
	switch tool {
	case "init", "new-spec", "new-roadmap", "new-adr", "new-knowledge", "followups", "amend", "assess", "state",
		"docs-init", "docs-check", "docs-review", "docs-sync", "report-limitation", "feedback", "report", "validate", "check",
		"review", "review-check", "closeout-check", "close", "continuous-closeout", "artifact-check", "artifact-backfill", "surface-check", "roadmap-check",
		"upgrade", "index", "knowledge-check", "knowledge-housekeeping", "knowledge-usage", "knowledge-suggest", "reports-housekeeping",
		"recurrence-check", "recurrence-effect", "hooks", "suggest", "stats", "stacks", "skills-check", "record-deployment",
		"record-incident", "dora-metrics", "adoption-metrics", "events-housekeeping", "semantic-suggest", "suggest-feedback",
		"portfolio-projection", "reconcile-evidence", "release-notes", "release", "release-package-manifests", "install", "doctor",
		"import", "extension", "lint-spec", "history-check":
		return true
	default:
		return false
	}
}

func defaultCommandUsage(tool string, code int) commandUsageResult {
	result := commandUsageResult{SemanticOutcome: "unknown"}
	if code == 2 {
		return result
	}
	if !isGateCommand(tool) {
		return result
	}
	if code == 0 {
		result.SemanticOutcome = "pass"
		return result
	}
	result.SemanticOutcome = "fail"
	result.FindingCount = 1
	result.FindingsBySeverity = map[string]int{"error": 1}
	return result
}

func isGateCommand(tool string) bool {
	switch tool {
	case "check", "validate", "knowledge-check", "recurrence-check", "lint-spec", "history-check", "skills-check", "review-check", "closeout-check", "artifact-check", "surface-check", "roadmap-check":
		return true
	default:
		return false
	}
}

func countedUsageResult(semantic string, errors, warnings int, complete bool) commandUsageResult {
	bySeverity := map[string]int{}
	if errors > 0 {
		bySeverity["error"] = errors
	}
	if warnings > 0 {
		bySeverity["warning"] = warnings
	}
	result := commandUsageResult{SemanticOutcome: semantic, FindingCount: errors + warnings, FindingsBySeverity: bySeverity, FindingSetComplete: complete}
	return result
}

func noteUsageFindings(stdout io.Writer, semantic string, findings []usageFinding, complete bool) {
	converted := make([]usagepkg.Finding, 0, len(findings))
	for _, finding := range findings {
		converted = append(converted, usagepkg.Finding{ID: finding.ID, Severity: finding.Severity})
	}
	noteCommandUsage(stdout, commandUsageResult{SemanticOutcome: semantic, Findings: converted, FindingCount: len(converted), FindingSetComplete: complete})
}

func cmdUsage(root string, args []string, stdout, stderr io.Writer) int {
	query := usagepkg.Query{SinceDays: 30}
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--since-days":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose usage [--since-days N] [--tool NAME] [--surface cli|mcp] [--json]")
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 {
				return usageError(stderr, "pose usage: --since-days must be an integer >= 0")
			}
			query.SinceDays = value
		case "--tool":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return usageError(stderr, "pose usage: --tool requires a value")
			}
			i++
			query.Tool = args[i]
		case "--surface":
			if i+1 >= len(args) {
				return usageError(stderr, "pose usage: --surface requires cli|mcp")
			}
			i++
			query.Surface = args[i]
		default:
			return usageError(stderr, "Usage: pose usage [--since-days N] [--tool NAME] [--surface cli|mcp] [--json]")
		}
	}
	report, err := usagepkg.Aggregate(root, query)
	if err != nil {
		fmt.Fprintf(stderr, "pose usage: %v\n", err)
		return 2
	}
	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(report)
		return 0
	}
	window := fmt.Sprintf("last %d days", report.SinceDays)
	if report.SinceDays == 0 {
		window = "all history"
	}
	fmt.Fprintf(stdout, "# POSE usage (%s)\n\n", window)
	if !report.Available {
		fmt.Fprintf(stdout, "unavailable: %s\n", report.Reason)
		fmt.Fprintf(stdout, "records.scanned=%d\nrecords.invalid=%d\n", report.RecordsScanned, report.InvalidRecords)
		return 0
	}
	tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TOOL\tSURFACE\tCALLS\tPASS\tFAIL\tEXEC ERR\tFINDINGS\tERROR F\tWARN F\tUNIQUE\tNEW\tRESOLVED\tREOPENED\tP95 MS")
	for _, row := range report.Rows {
		executionErrors := row.Failed + row.Invalid + row.Denied + row.Errors
		fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%.2f\n",
			row.Tool, row.Surface, row.Calls, row.Pass, row.Fail, executionErrors,
			row.FindingsObserved, row.FindingsBySeverity["error"], row.FindingsBySeverity["warning"],
			row.UniqueFindings, row.NewFindings,
			row.ResolvedFindings, row.ReopenedFindings, row.P95DurationMS)
	}
	_ = tw.Flush()
	fmt.Fprintf(stdout, "\nrecords.scanned=%d\nrecords.matched=%d\nrecords.invalid=%d\n", report.RecordsScanned, report.RecordsMatched, report.InvalidRecords)
	return 0
}
