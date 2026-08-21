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
	locale := cliLocaleValue()
	statusFilter := ""
	componentsFilter := ""
	recentLimit := 0
	sinceFilter := ""
	jsonOutput := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--status":
			if i+1 < len(args) {
				statusFilter = args[i+1]
				i++
			}
		case "--components", "--component":
			if i+1 < len(args) {
				componentsFilter = args[i+1]
				i++
			}
		case "--recent":
			if i+1 < len(args) {
				if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
					recentLimit = n
				}
				i++
			} else {
				recentLimit = 10 // default recent limit
			}
		case "--since":
			if i+1 < len(args) {
				sinceFilter = args[i+1]
				i++
			}
		case "-h", "--help":
			dispatchCommandHelp("specs", nil, stdout, locale)
			return 0
		}
	}

	store := pose.Store{Root: root}
	specs, err := store.ListSpecs(statusFilter, componentsFilter)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "pose specs: %v\n", "pose specs: %v\n"), err)
		return 1
	}

	// Filter by --since if requested
	if sinceFilter != "" {
		sinceDate, parseErr := parseSinceDate(sinceFilter)
		if parseErr == nil {
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
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "d") {
		daysStr := strings.TrimSuffix(val, "d")
		days, err := strconv.Atoi(daysStr)
		if err == nil {
			return time.Now().UTC().AddDate(0, 0, -days), nil
		}
	}
	if t, err := time.Parse("2006-01-02", val); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid since format %q", val)
}
