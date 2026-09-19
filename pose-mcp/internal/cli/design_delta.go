package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/harne8/pose-mcp/internal/cli/cliout"
	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// assessDesign is deliberately read-only. It prepares the same canonical
// review subject used by bundle sealing, then delegates all structural parsing
// to the pose package. A missing/partial subject is a visible report state,
// not a fabricated pass and not a reason to write a cache.
func assessDesign(root string, args []string, stdout, stderr io.Writer, locale cliLocale) int {
	out := render(stdout, stderr)
	spec := ""
	jsonOutput := false
	options := posemodel.DesignDeltaOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--spec":
			if i+1 >= len(args) || strings.TrimSpace(args[i+1]) == "" {
				out.Failure(cliText(locale, "--spec requires a value.", "--spec exige um valor."))
				return 2
			}
			i++
			spec = args[i]
		case "--max-files", "--max-bytes":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				out.Failure(fmt.Sprintf(cliText(locale, "%s requires a value.", "%s exige um valor."), args[i]))
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				out.Failure(fmt.Sprintf(cliText(locale, "%s must be a positive integer.", "%s deve ser um inteiro positivo."), args[i-1]))
				return 2
			}
			if args[i-1] == "--max-files" {
				options.MaxFiles = value
			} else {
				options.MaxBytes = value
			}
		case "--json":
			jsonOutput = true
		default:
			out.Usage(cliText(locale,
				"Usage: pose assess design --spec <slug> [--json] [--max-files N] [--max-bytes N]",
				"Uso: pose assess design --spec <slug> [--json] [--max-files N] [--max-bytes N]"))
			return 2
		}
	}
	if spec == "" {
		out.Usage(cliText(locale,
			"Usage: pose assess design --spec <slug> [--json] [--max-files N] [--max-bytes N]",
			"Uso: pose assess design --spec <slug> [--json] [--max-files N] [--max-bytes N]"))
		return 2
	}
	store := posemodel.Store{Root: root}
	bundle, err := store.PrepareReviewBundle("spec:" + spec)
	if err != nil {
		out.Failure(fmt.Sprintf("pose assess design: %v", err))
		return 1
	}
	report, err := posemodel.AssessDesignDelta(root, bundle.Payload.Subject, "spec:"+spec, options)
	if err != nil {
		out.Failure(fmt.Sprintf("pose assess design: %v", err))
		return 1
	}
	if len(bundle.Blockers) > 0 {
		report.Warnings = append(report.Warnings, "review subject blockers: "+strings.Join(bundle.Blockers, "; "))
		sortStrings(report.Warnings)
	}
	if jsonOutput {
		if err := json.NewEncoder(out.Out()).Encode(report); err != nil {
			out.Failure(fmt.Sprintf("pose assess design: %v", err))
			return 1
		}
		return 0
	}
	writeDesignDeltaText(out, report, locale)
	return 0
}

func writeDesignDeltaText(out *cliout.Renderer, report posemodel.DesignDeltaReport, locale cliLocale) {
	out.Section(fmt.Sprintf("Design delta (%s)", report.Scope))
	out.Field("status", report.Status)
	out.Field("input_digest", report.InputDigest)
	out.Field("subject.entries", fmt.Sprintf("%d", report.Subject.Entries))
	out.Field("subject.base", shortDesignDeltaRevision(report.Subject.Base))
	out.Field("subject.head", shortDesignDeltaRevision(report.Subject.Head))
	out.Field("coverage.files", fmt.Sprintf("%d/%d", report.Coverage.FilesObserved, report.Coverage.FilesInSubject))
	out.Field("coverage.bytes", fmt.Sprintf("%d", report.Coverage.BytesRead))
	out.Field("coverage.detectors", fmt.Sprintf("%d", len(report.Coverage.Detectors)))
	for _, detector := range report.Coverage.Detectors {
		out.Field("detector."+detector.ID, fmt.Sprintf("%s observed=%d unknown=%d unsupported=%d", detector.State, detector.Observed, detector.Unknown, detector.Unsupported))
	}
	if len(report.Deltas) == 0 {
		out.Field("deltas", cliText(locale, "none observed", "nenhum observado"))
	} else {
		out.Field("deltas.count", fmt.Sprintf("%d", len(report.Deltas)))
		for _, delta := range report.Deltas {
			out.Field("delta."+delta.DisplayID, fmt.Sprintf("%s %s %s %s", delta.Kind, delta.Action, delta.State, delta.Subject))
		}
	}
	for _, warning := range report.Warnings {
		out.Finding(cliout.Finding{State: cliout.StateWarning, Code: "design-delta", Message: warning})
	}
}

func shortDesignDeltaRevision(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	if value == "" {
		return "unknown"
	}
	return value
}

func pluralSuffix(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
