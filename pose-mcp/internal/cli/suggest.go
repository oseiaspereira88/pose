package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	posepkg "github.com/harne8/pose-mcp/internal/pose"
)

type taskMap struct {
	Tasks map[string]map[string]any `json:"tasks"`
}

func cmdSuggest(root string, args []string, stdout, stderr io.Writer) int {
	task, domain, pathHint := "", "", ""
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOut = true
		case "--domain", "--path":
			if i+1 >= len(args) {
				return usageError(stderr, "pose suggest: value required")
			}
			i++
			if args[i-1] == "--domain" {
				domain = args[i]
			} else {
				pathHint = args[i]
			}
		default:
			if strings.HasPrefix(args[i], "-") || task != "" {
				return usageError(stderr, "Usage: pose suggest [type] [--domain d] [--path p] [--json]")
			}
			task = args[i]
		}
	}
	if pathHint != "" && !confinedRelativePath(pathHint) {
		return usageError(stderr, "pose suggest: --path must remain inside project")
	}
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "task-map.json"))
	if err != nil {
		fmt.Fprintln(stderr, "pose suggest: task-map missing")
		return 2
	}
	var tm taskMap
	if json.Unmarshal(raw, &tm) != nil || tm.Tasks == nil {
		fmt.Fprintln(stderr, "pose suggest: invalid task-map")
		return 2
	}
	keys := make([]string, 0, len(tm.Tasks))
	for k := range tm.Tasks {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if task == "" {
		if jsonOut {
			_ = json.NewEncoder(stdout).Encode(keys)
		} else {
			fmt.Fprintln(stdout, "Available task types:")
			for _, k := range keys {
				fmt.Fprintf(stdout, "  - %s: %v\n", k, tm.Tasks[k]["description"])
			}
		}
		return 0
	}
	t, ok := tm.Tasks[task]
	if !ok {
		fmt.Fprintf(stderr, "pose suggest: unknown task type %q (available: %s)\n", task, strings.Join(keys, ", "))
		return 2
	}
	source := "explicit"
	if domain == "" && pathHint != "" {
		domain, source = inferSuggestDomain(root, pathHint)
	}
	domainInput := domain
	domain, aliased := normalizeSuggestDomain(domain)
	if aliased && source == "explicit" {
		source = "alias"
	}
	rules := stringSlice(t["rules"])
	byDomain, _ := t["rules_by_domain"].(map[string]any)
	domainKnown := true
	if domain != "" {
		var known bool
		if _, known = byDomain[domain]; known {
			rules = append(rules, stringSlice(byDomain[domain])...)
		}
		domainKnown = known
	}
	statuses := suggestRuleStatuses(root, rules)
	if jsonOut {
		payload := map[string]any{"name": task}
		for k, v := range t {
			payload[k] = v
		}
		if domain != "" {
			payload["domain_effective"] = domain
			payload["domain_source"] = source
			payload["rules_effective"] = rules
			if domainInput != domain {
				payload["domain_input"] = domainInput
			}
			if !domainKnown {
				payload["domain_status"] = "no rules declared for this domain in the task map"
			}
		}
		payload["rules_status"] = statuses
		if pathHint != "" {
			payload["path_input"] = pathHint
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
		return 0
	}
	fmt.Fprintf(stdout, "# POSE trail — %s\n\n", task)
	if d := fmt.Sprint(t["description"]); d != "<nil>" && d != "" {
		fmt.Fprintln(stdout, d)
	}
	if pathHint != "" {
		fmt.Fprintf(stdout, "- Path:      %s → domain: %s (%s)\n", pathHint, domain, source)
	}
	domainNote := ""
	if domain != "" && !domainKnown {
		domainNote = fmt.Sprintf("- Domain:    %s has no rules declared in the task map\n", domain)
	}
	fmt.Fprintf(stdout, "%s- Workflow:  %v\n- Skill:     %v\n", domainNote, t["workflow"], t["skill"])
	paths := []string{}
	for _, st := range statuses {
		switch st.Availability {
		case ruleInstalled:
			paths = append(paths, st.Path)
		case ruleExtensionNeeded:
			paths = append(paths, st.Rule+" (not installed: "+st.NextStep+")")
		default:
			paths = append(paths, st.Rule+" (unavailable: declared in the task map, no rule file and no extension known)")
		}
	}
	fmt.Fprintf(stdout, "- Rules:     %s\n- Spec:      %v\n- ADR:       %v\n- Knowledge: consume=%v, produce=%v\n- Validation: %v\n", strings.Join(paths, ", "), t["requires_spec"], t["requires_adr"], t["knowledge_consume"], t["knowledge_produce"], t["validation"])
	return 0
}

func stringSlice(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		if ss, ok := value.([]string); ok {
			return ss
		}
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
func inferSuggestDomain(root, pathHint string) (string, string) {
	norm := filepath.ToSlash(filepath.Clean(pathHint))
	parts := strings.Split(norm, "/")
	for _, p := range parts {
		if p == "k8s" || p == "charts" || p == "helm" {
			return "k8s", "hint-path"
		}
	}
	raw, _, e := posepkg.Store{Root: root}.ReadIndex("repo-map.json")
	if e != nil {
		return "", "undefined"
	}
	var data map[string]any
	if json.Unmarshal(raw, &data) != nil {
		return "", "undefined"
	}
	best := 0
	domain := ""
	for _, kind := range []string{"apps", "services", "packages"} {
		items, _ := data[kind].([]any)
		for _, rawItem := range items {
			item, _ := rawItem.(map[string]any)
			p, _ := item["path"].(string)
			if p != "" && (norm == p || strings.HasPrefix(norm, p+"/")) && len(p) > best {
				best = len(p)
				domain, _ = item["domain"].(string)
				if domain == "" || domain == "unknown" {
					lang, _ := item["language"].(string)
					domain = map[string]string{"go": "backend-go", "javascript": "frontend", "typescript": "frontend"}[lang]
				}
				// "backend" names no language: it is the Go domain only when the module is Go.
				if domain == "backend" {
					lang, _ := item["language"].(string)
					if _, goMod := os.Stat(filepath.Join(root, filepath.FromSlash(p), "go.mod")); lang == "go" || strings.HasSuffix(norm, ".go") || goMod == nil {
						domain = "backend-go"
					}
				}
			}
		}
	}
	if domain != "" {
		return domain, "repo-map"
	}
	return "", "undefined"
}

// suggestDomainAliases is the one taxonomy of domain names `suggest` accepts, from an
// explicit --domain or inferred from a path: the left side is what a person or a tool writes,
// the right side is the key the task map declares rules under.
var suggestDomainAliases = map[string]string{
	"go": "backend-go", "golang": "backend-go", "backend-go": "backend-go",
	"react": "frontend", "frontend-react": "frontend", "frontend": "frontend",
	"kubernetes": "k8s", "helm": "k8s", "k8s": "k8s",
}

// normalizeSuggestDomain returns the canonical domain and whether the input was an alias.
// A name the taxonomy does not know is returned as written: it is reported as having no
// rules, never silently turned into another domain.
func normalizeSuggestDomain(domain string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(domain))
	if canonical, ok := suggestDomainAliases[key]; ok {
		return canonical, canonical != domain
	}
	return domain, false
}

// Availability of a rule the task map names.
const (
	ruleInstalled       = "installed"
	ruleExtensionNeeded = "extension-needed"
	ruleUnavailable     = "unavailable"
)

// ruleStatus says whether a rule the trail names can be read here: declared is not
// installed, and installed is not applied.
type ruleStatus struct {
	Rule         string `json:"rule"`
	Availability string `json:"availability"`
	Path         string `json:"path,omitempty"`
	Extension    string `json:"extension,omitempty"`
	NextStep     string `json:"next_step,omitempty"`
}

func suggestRuleStatuses(root string, rules []string) []ruleStatus {
	byFile := map[string]string{}
	for id, file := range ruleExtensionFile {
		byFile[file] = id
	}
	out := make([]ruleStatus, 0, len(rules))
	for _, r := range rules {
		file := r + ".md"
		st := ruleStatus{Rule: r}
		switch {
		case ruleExtensionInstalled(root, file):
			st.Availability, st.Path = ruleInstalled, ".pose/rules/"+file
		case byFile[file] != "":
			st.Availability, st.Extension = ruleExtensionNeeded, byFile[file]
			st.NextStep = "pose extension install " + byFile[file]
		default:
			st.Availability = ruleUnavailable
		}
		out = append(out, st)
	}
	return out
}
