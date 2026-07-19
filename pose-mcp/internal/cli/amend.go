package cli

// pose amend — append-only spec amendment history (spec
// pose-spec-amendment-history). Records material requirement changes with
// affected IDs, rationale, author/reviewer aliases and timestamp; the
// closeout gate in lint-spec rejects unacknowledged changes.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

var amendAliasRE = regexp.MustCompile(`^@[a-z0-9][a-z0-9._-]*$`)
var amendIDRE = regexp.MustCompile(`^R\d+$`)

func cmdAmend(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	usage := func() int {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose amend <slug> --list | --baseline --author @alias | --ids R1[,R2] --change added|withdrawn|semantic|editorial --rationale <text> --author @alias [--reviewer @alias]",
			"Uso: pose amend <slug> --list | --baseline --author @alias | --ids R1[,R2] --change added|withdrawn|semantic|editorial --rationale <texto> --author @alias [--reviewer @alias]"))
		return 2
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return usage()
	}
	slug := args[0]
	args = args[1:]
	list, baseline := false, false
	var ids []string
	change, rationale, author, reviewer := "", "", "", ""
	for i := 0; i < len(args); i++ {
		next := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch args[i] {
		case "--list":
			list = true
		case "--baseline":
			baseline = true
		case "--ids":
			v, ok := next()
			if !ok {
				return usage()
			}
			for _, id := range strings.Split(v, ",") {
				if id = strings.TrimSpace(id); id != "" {
					ids = append(ids, id)
				}
			}
		case "--change":
			if change, _ = next(); change == "" {
				return usage()
			}
		case "--rationale":
			if rationale, _ = next(); rationale == "" {
				return usage()
			}
		case "--author":
			if author, _ = next(); author == "" {
				return usage()
			}
		case "--reviewer":
			if reviewer, _ = next(); reviewer == "" {
				return usage()
			}
		default:
			return usage()
		}
	}

	root, err := projectRoot()
	if err != nil {
		fmt.Fprintf(stderr, "pose amend: %v\n", err)
		return 2
	}
	specPath := filepath.Join(root, ".pose", "specs", slug, "spec.md")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "pose amend: spec not found: %s\n", "pose amend: spec não encontrada: %s\n"), specPath)
		return 2
	}
	logPath := posepkg.AmendmentsPath(specPath)
	events, err := posepkg.LoadAmendments(logPath)
	if err != nil {
		fmt.Fprintf(stderr, "pose amend: %s: %v\n", logPath, err)
		return 1
	}

	if list {
		if len(events) == 0 {
			fmt.Fprintln(stdout, cliText(locale, "(no amendments recorded)", "(nenhum amendment registrado)"))
			return 0
		}
		for _, e := range events {
			who := e.Author
			if e.Reviewer != "" {
				who += " / " + e.Reviewer
			}
			fmt.Fprintf(stdout, "- %s [%s] %s (%s)", e.At, e.Change, strings.Join(e.IDs, ","), who)
			if e.Rationale != "" {
				fmt.Fprintf(stdout, ": %s", e.Rationale)
			}
			fmt.Fprintln(stdout)
		}
		findings := posepkg.UnacknowledgedChanges(string(raw), events)
		fmt.Fprintf(stdout, "amend.events=%d\namend.unacknowledged=%d\n", len(events), len(findings))
		for _, f := range findings {
			fmt.Fprintf(stdout, "- PENDING: %s\n", f)
		}
		return 0
	}

	if !amendAliasRE.MatchString(author) {
		fmt.Fprintln(stderr, cliText(locale, "pose amend: --author must be a pseudonymous @alias", "pose amend: --author deve ser um @alias pseudônimo"))
		return 2
	}
	if reviewer != "" && !amendAliasRE.MatchString(reviewer) {
		fmt.Fprintln(stderr, cliText(locale, "pose amend: --reviewer must be a pseudonymous @alias", "pose amend: --reviewer deve ser um @alias pseudônimo"))
		return 2
	}

	current := posepkg.CurrentRequirementHashes(string(raw))
	event := posepkg.Amendment{
		Schema: posepkg.AmendmentSchema,
		At:     time.Now().UTC().Format(time.RFC3339),
		Author: author, Reviewer: reviewer, Rationale: rationale,
		Hashes: map[string]string{},
	}
	if baseline {
		event.Change = "baseline"
		for id, h := range current {
			event.IDs = append(event.IDs, id)
			event.Hashes[id] = h
		}
		sort.Strings(event.IDs)
	} else {
		if len(ids) == 0 || change == "" || rationale == "" {
			return usage()
		}
		if change == "baseline" || !posepkg.ValidAmendmentChanges[change] {
			fmt.Fprintf(stderr, cliText(locale, "pose amend: invalid --change %q (use added|withdrawn|semantic|editorial)\n", "pose amend: --change inválido %q (use added|withdrawn|semantic|editorial)\n"), change)
			return 2
		}
		event.Change = change
		for _, id := range ids {
			if !amendIDRE.MatchString(id) {
				fmt.Fprintf(stderr, cliText(locale, "pose amend: invalid requirement ID %q\n", "pose amend: ID de requisito inválido %q\n"), id)
				return 2
			}
			hash, declared := current[id]
			if change == "withdrawn" {
				if declared {
					fmt.Fprintf(stderr, cliText(locale, "pose amend: %s is still declared in Requirements — remove or mark it before acknowledging withdrawal\n", "pose amend: %s ainda está declarado em Requirements — remova/marque antes de reconhecer a retirada\n"), id)
					return 1
				}
				event.Hashes[id] = "" // stays addressable, acknowledged as gone
			} else {
				if !declared {
					fmt.Fprintf(stderr, cliText(locale, "pose amend: %s is not declared in Requirements (change %q needs the current text)\n", "pose amend: %s não está declarado em Requirements (change %q precisa do texto atual)\n"), id, change)
					return 1
				}
				event.Hashes[id] = hash
			}
			event.IDs = append(event.IDs, id)
		}
	}

	line, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Amendment recorded: [%s] %s → %s\n", "Amendment registrado: [%s] %s → %s\n"), event.Change, strings.Join(event.IDs, ","), logPath)
	return 0
}
