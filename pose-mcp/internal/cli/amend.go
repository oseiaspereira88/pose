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
var amendNodeIDRE = regexp.MustCompile(`^[RAD]\d+$`)

func cmdAmend(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	usage := func() int {
		fmt.Fprintln(stderr, cliText(locale,
			"Usage: pose amend <slug> --list | --nodes [--json] | --baseline --author @alias | --ids R1[,A2,D3] --change added|withdrawn|semantic|editorial|transition --rationale <text> --author @alias [--reviewer @alias]",
			"Uso: pose amend <slug> --list | --nodes [--json] | --baseline --author @alias | --ids R1[,A2,D3] --change added|withdrawn|semantic|editorial|transition --rationale <texto> --author @alias [--reviewer @alias]"))
		return 2
	}
	if len(args) == 0 || strings.HasPrefix(args[0], "--") {
		return usage()
	}
	slug := args[0]
	args = args[1:]
	list, baseline, nodes, jsonOutput := false, false, false, false
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
		case "--nodes":
			nodes = true
		case "--json":
			jsonOutput = true
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
	// Resolve through the store, which knows flat dated, folder and legacy
	// layouts alike; building `<slug>/spec.md` here left every flat spec
	// without amendments (spec pose-flat-spec-amendments).
	specPath := filepath.Join(root, ".pose", "specs", slug, "spec.md")
	if resolved, resolveErr := (posepkg.Store{Root: root}).GetSpec(slug); resolveErr == nil {
		specPath = resolved.Path
		if info, statErr := os.Stat(specPath); statErr == nil && info.IsDir() {
			specPath = filepath.Join(specPath, "spec.md")
		}
	}
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
	adopted, err := posepkg.Store{Root: root}.ContractNodesAdopted()
	if err != nil {
		render(stdout, stderr).Failure("pose amend: " + err.Error())
		return 1
	}
	projection := posepkg.ProjectContractNodes(slug, string(raw))

	if nodes {
		if jsonOutput {
			return writeJSON(stdout, map[string]any{"contract_nodes": projection, "capability_adopted": adopted})
		}
		r := render(stdout, stderr)
		for _, node := range projection.Nodes {
			r.Field("node."+node.ID, strings.TrimSpace(node.Kind+" "+node.State+" "+node.Hash+" "+strings.Join(node.Relations, ",")))
		}
		for _, diagnostic := range projection.Diagnostics {
			r.Field("node.diagnostic", diagnostic.ID+" "+diagnostic.Code+": "+diagnostic.Message)
		}
		r.Field("contract_nodes.digest", projection.Digest)
		r.Field("contract_nodes.capability_adopted", fmt.Sprint(adopted))
		return 0
	}

	if list {
		if len(events) == 0 {
			fmt.Fprintln(stdout, cliText(locale, "(no amendments recorded)", "(nenhum amendment registrado)"))
			return 0
		}
		r := render(stdout, stderr)
		for _, e := range events {
			if e.Schema == posepkg.AmendmentSchemaV2 {
				// Before/after per node and the origin's assurance; nothing
				// is inferred for a node the event did not record.
				for _, id := range e.IDs {
					before, after := e.Before[id], e.After[id]
					r.Field("amend."+e.At+"."+id, fmt.Sprintf("[%s] %s %s → %s %s (%s, assurance %s)", e.Change, before.State, before.Hash, after.State, after.Hash, strings.TrimSpace(e.Author+" "+e.Reviewer), e.Assurance))
				}
				continue
			}
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
		findings := posepkg.UnacknowledgedNodeChanges(slug, string(raw), events, adopted)
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

	if adopted {
		return appendNodeAmendment(root, slug, logPath, projection, events, baseline, ids, change, rationale, author, reviewer, stdout, stderr)
	}
	if change == "transition" {
		render(stdout, stderr).Failure("pose amend: --change transition needs the contract_nodes_version capability")
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
				if amendNodeIDRE.MatchString(id) {
					render(stdout, stderr).Failure("pose amend: " + id + " is an assumption or decision; recording it needs the contract_nodes_version capability")
					return 2
				}
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
	if hookErr := EmitHook(root, HookEvent{Kind: "spec_amend", Target: slug, Commit: gitHeadCommit(root), At: time.Now().UTC()}); hookErr != nil {
		fmt.Fprintf(stderr, "pose amend: %v\n", hookErr)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale, "Amendment recorded: [%s] %s → %s\n", "Amendment registrado: [%s] %s → %s\n"), event.Change, strings.Join(event.IDs, ","), logPath)
	return 0
}

// appendNodeAmendment records a schema-2 event under the adopted
// contract_nodes_version capability: every node of a baseline, or the named
// R/A/D nodes with their state before and after.
func appendNodeAmendment(root, slug, logPath string, projection posepkg.ContractNodesProjection, events []posepkg.Amendment, baseline bool, ids []string, change, rationale, author, reviewer string, stdout, stderr io.Writer) int {
	r := render(stdout, stderr)
	current := posepkg.CurrentNodeStates(projection)
	acknowledged := posepkg.AcknowledgedNodeStates(events)
	event := posepkg.Amendment{
		Schema: posepkg.AmendmentSchemaV2,
		At:     time.Now().UTC().Format(time.RFC3339),
		Author: author, Reviewer: reviewer, Rationale: rationale, Assurance: "declared",
		Hashes: map[string]string{}, Before: map[string]posepkg.NodeState{}, After: map[string]posepkg.NodeState{},
	}
	state := func(node posepkg.ContractNode) posepkg.NodeState {
		return posepkg.NodeState{Hash: node.Hash, State: node.State, Relations: node.Relations}
	}
	if baseline {
		event.Change = "baseline"
		for _, node := range projection.Nodes {
			event.IDs = append(event.IDs, node.ID)
			event.After[node.ID] = state(node)
			event.Hashes[node.ID] = node.Hash
		}
	} else {
		if len(ids) == 0 || change == "" || rationale == "" {
			r.Failure("pose amend: --ids, --change and --rationale are required")
			return 2
		}
		if change == "baseline" || !(posepkg.ValidAmendmentChanges[change] || change == "transition") {
			r.Failure(fmt.Sprintf("pose amend: invalid --change %q (use added|withdrawn|semantic|editorial|transition)", change))
			return 2
		}
		event.Change = change
		for _, id := range ids {
			if !amendNodeIDRE.MatchString(id) {
				r.Failure(fmt.Sprintf("pose amend: invalid node ID %q", id))
				return 2
			}
			node, declared := current[id]
			before, known := acknowledged[id]
			if known {
				event.Before[id] = before
			}
			switch change {
			case "withdrawn":
				if strings.HasPrefix(id, "R") {
					if declared {
						r.Failure("pose amend: " + id + " is still declared in Requirements — remove it before acknowledging withdrawal")
						return 1
					}
					event.After[id] = posepkg.NodeState{State: "withdrawn"}
					event.Hashes[id] = ""
				} else {
					if !declared || node.State != "withdrawn" {
						r.Failure("pose amend: " + id + " must declare Status: withdrawn before the withdrawal is acknowledged")
						return 1
					}
					event.After[id] = state(node)
					event.Hashes[id] = node.Hash
				}
			default:
				if !declared {
					r.Failure(fmt.Sprintf("pose amend: %s is not declared (change %q needs the current node)", id, change))
					return 1
				}
				switch change {
				case "transition":
					if !known || !posepkg.ContractNodeTransitionAllowed(node.Kind, before.State, node.State) {
						r.Failure(fmt.Sprintf("pose amend: %s cannot move from %q to %q", id, before.State, node.State))
						return 1
					}
				case "editorial":
					if known && !posepkg.EditorialAllowed(node, before) {
						r.Failure("pose amend: " + id + " changed its state or relations; that is semantic and cannot be acknowledged as editorial")
						return 1
					}
				}
				event.After[id] = state(node)
				event.Hashes[id] = node.Hash
			}
			event.IDs = append(event.IDs, id)
		}
	}
	sort.Strings(event.IDs)
	line, err := json.Marshal(event)
	if err != nil {
		r.Failure(err.Error())
		return 1
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		r.Failure(err.Error())
		return 1
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		r.Failure(err.Error())
		return 1
	}
	if hookErr := EmitHook(root, HookEvent{Kind: "spec_amend", Target: slug, Commit: gitHeadCommit(root), At: time.Now().UTC()}); hookErr != nil {
		r.Failure("pose amend: " + hookErr.Error())
		return 1
	}
	r.Field("amend.recorded", fmt.Sprintf("[%s] %s → %s", event.Change, strings.Join(event.IDs, ","), logPath))
	return 0
}
