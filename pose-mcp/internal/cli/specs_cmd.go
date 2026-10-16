package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// cmdSpecs handles the `pose specs` discovery and listing CLI command.
func cmdSpecs(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "facts" {
		return cmdSpecFacts(root, args[1:], stdout, stderr)
	}
	locale := cliLocaleValue()
	statusFilter := ""
	componentsFilter := ""
	recentLimit := 0
	sinceFilter := ""
	jsonOutput := false

	// Every argument is either understood or refused before the listing: an
	// ignored flag or value would return every spec as if it matched (spec
	// pose-cli-mcp-operation-results R3).
	const specsUsage = "Usage: pose specs [--status <s,...>] [--components|--component <c,...>] [--since <YYYY-MM-DD|<days>d>] [--recent [n]] [--json]"
	refuse := func(reason string) int {
		render(io.Discard, stderr).Failure("pose specs: " + reason)
		return usageError(stderr, specsUsage)
	}
	value := func(i int) (string, bool) {
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
			return "", false
		}
		return args[i+1], true
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--status", "--components", "--component", "--since":
			v, ok := value(i)
			if !ok {
				return refuse(args[i] + " needs a value")
			}
			switch args[i] {
			case "--status":
				statusFilter = v
			case "--since":
				if _, err := parseSinceDate(v); err != nil {
					return refuse(err.Error())
				}
				sinceFilter = v
			default:
				componentsFilter = v
			}
			i++
		case "--recent":
			// A bare --recent keeps its documented default of 10.
			v, ok := value(i)
			if !ok {
				recentLimit = 10
				break
			}
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return refuse("--recent needs a positive number, got " + strconv.Quote(v))
			}
			recentLimit = n
			i++
		case "-h", "--help":
			dispatchCommandHelp("specs", nil, stdout, locale)
			return 0
		default:
			return refuse("unknown argument " + strconv.Quote(args[i]))
		}
	}

	store := pose.Store{Root: root}
	specs, err := store.ListSpecs(statusFilter, componentsFilter)
	if err != nil && jsonOutput {
		return writeOperationFailure(stdout, stderr, "specs.list", "", "listing-failed", "pose specs: "+err.Error())
	}
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "pose specs: %v\n", "pose specs: %v\n"), err)
		return 1
	}

	// Filter by --since if requested
	if sinceFilter != "" {
		if sinceDate, parseErr := parseSinceDate(sinceFilter); parseErr == nil {
			filtered := []pose.Spec{}
			for _, sp := range specs {
				dateStr := sp.CreatedAt
				if dateStr == "" {
					dateStr = sp.CompletedAt
				}
				if dateStr != "" {
					if t, err := time.Parse("2006-01-02", dateStr); err == nil {
						if !t.Before(sinceDate) {
							filtered = append(filtered, sp)
						}
					}
				}
			}
			specs = filtered
		}
	}

	// Sort: by CreatedAt descending (newest first), fallback to Slug
	sort.Slice(specs, func(i, j int) bool {
		if specs[i].CreatedAt != specs[j].CreatedAt {
			return specs[i].CreatedAt > specs[j].CreatedAt
		}
		return specs[i].Slug < specs[j].Slug
	})

	totalCount := len(specs)
	if recentLimit > 0 && len(specs) > recentLimit {
		specs = specs[:recentLimit]
	}

	if jsonOutput {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(specs); err != nil {
			fmt.Fprintf(stderr, "pose specs: encoding JSON: %v\n", err)
			return 1
		}
		return 0
	}

	text := func(en, pt string) string { return cliText(locale, en, pt) }
	fmt.Fprintf(stdout, text("POSE Specs (%d total, showing %d):\n\n", "Specs POSE (%d total, exibindo %d):\n\n"), totalCount, len(specs))

	if len(specs) == 0 {
		fmt.Fprintln(stdout, text("  No matching specifications found.", "  Nenhuma especificação correspondente encontrada."))
		return 0
	}

	fmt.Fprintf(stdout, "  %-12s  %-12s  %-35s  %s\n",
		text("DATE", "DATA"),
		text("STATUS", "STATUS"),
		text("SLUG", "SLUG"),
		text("TITLE", "TÍTULO"))
	fmt.Fprintf(stdout, "  %-12s  %-12s  %-35s  %s\n",
		"──────────", "────────────", "───────────────────────────────────", "────────────────────────────────────────────")

	for _, sp := range specs {
		date := sp.CreatedAt
		if date == "" {
			date = "—"
		}
		title := sp.Title
		if title == "" {
			title = sp.Slug
		}
		if len(title) > 44 {
			title = title[:41] + "..."
		}
		slug := sp.Slug
		if len(slug) > 35 {
			slug = slug[:32] + "..."
		}
		fmt.Fprintf(stdout, "  %-12s  %-12s  %-35s  %s\n", date, sp.Status, slug, title)
	}
	fmt.Fprintln(stdout)
	return 0
}

func parseSinceDate(val string) (time.Time, error) {
	return pose.ParseSinceDate(val)
}

// cmdSpecFacts prints the factual part of a spec's report, derived from the
// delivery index (spec pose-progressive-spec-surface).
func cmdSpecFacts(root string, args []string, stdout, stderr io.Writer) int {
	// One slug and an optional --json in any position; anything else is
	// refused before the facts are derived (spec pose-cli-mcp-operation-results R3).
	const factsUsage = "Usage: pose specs facts <slug> [--json]"
	slug, jsonOutput := "", false
	for _, arg := range args {
		switch {
		case arg == "--json":
			jsonOutput = true
		case strings.HasPrefix(arg, "-") || slug != "":
			render(io.Discard, stderr).Failure("pose specs facts: unknown argument " + strconv.Quote(arg))
			return usageError(stderr, factsUsage)
		default:
			slug = arg
		}
	}
	if slug == "" {
		return usageError(stderr, factsUsage)
	}
	facts, err := pose.Store{Root: root}.DeriveSpecFacts(slug)
	if err != nil {
		if jsonOutput {
			return writeOperationFailure(stdout, stderr, "specs.facts", slug, "facts-unavailable", "pose specs facts: "+err.Error())
		}
		render(stdout, stderr).Failure("pose specs facts: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, facts)
	}
	out := render(stdout, stderr)
	out.Field("facts.source", facts.Source)
	out.Field("facts.commits", fmt.Sprint(facts.Commits))
	for _, p := range facts.Paths {
		out.Field("facts.path", p.Action+": "+p.Path)
	}
	for _, c := range facts.Checks {
		out.Field("facts.check", c.Module+" "+c.Check+" ("+c.EvidenceClass+") "+c.Outcome)
	}
	for _, l := range facts.Limitations {
		out.Field("facts.limitation", l)
	}
	return 0
}
