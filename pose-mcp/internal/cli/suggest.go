package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	rules := stringSlice(t["rules"])
	byDomain, _ := t["rules_by_domain"].(map[string]any)
	if domain != "" {
		rules = append(rules, stringSlice(byDomain[domain])...)
	}
	if jsonOut {
		payload := map[string]any{"name": task}
		for k, v := range t {
			payload[k] = v
		}
		if domain != "" {
			payload["domain_effective"] = domain
			payload["domain_source"] = source
			payload["rules_effective"] = rules
		}
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
	fmt.Fprintf(stdout, "- Workflow:  %v\n- Skill:     %v\n", t["workflow"], t["skill"])
	paths := []string{}
	for _, r := range rules {
		paths = append(paths, ".pose/rules/"+r+".md")
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
	raw, e := os.ReadFile(filepath.Join(root, ".pose", "indexes", "repo-map.json"))
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
				if domain == "backend" {
					domain = "backend-go"
				}
			}
		}
	}
	if domain != "" {
		return domain, "repo-map"
	}
	return "", "undefined"
}
