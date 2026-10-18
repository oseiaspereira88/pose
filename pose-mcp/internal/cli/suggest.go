package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
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
	var document map[string]json.RawMessage
	if json.Unmarshal(raw, &document) != nil || document == nil {
		return usageError(stderr, "pose suggest: invalid task-map")
	}
	// The decoder of a struct matches a field in any case (and by Unicode folding), so two keys
	// that differ only in case would be one field and the later would win: the exact key is
	// selected here and any other that folds to it is refused.
	for key := range document {
		if key != "tasks" && strings.EqualFold(key, "tasks") {
			return usageError(stderr, fmt.Sprintf("pose suggest: invalid task-map: the key %q is ambiguous with \"tasks\"", key))
		}
	}
	if dupErr := posepkg.RejectDuplicateJSONKeys(raw); dupErr != nil {
		return usageError(stderr, fmt.Sprintf("pose suggest: invalid task-map: %v", dupErr))
	}
	if tasks, present := document["tasks"]; !present || json.Unmarshal(tasks, &tm.Tasks) != nil || tm.Tasks == nil {
		return usageError(stderr, "pose suggest: invalid task-map")
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
	if ok && t == nil {
		return usageError(stderr, fmt.Sprintf("pose suggest: invalid task-map entry %q: the entry is null", task))
	}
	if !ok {
		fmt.Fprintf(stderr, "pose suggest: unknown task type %q (available: %s)\n", task, strings.Join(keys, ", "))
		return 2
	}
	rules, byDomain, cfgErr := suggestTaskRules(t)
	if cfgErr != nil {
		return usageError(stderr, fmt.Sprintf("pose suggest: invalid task-map entry %q: %v", task, cfgErr))
	}
	references, refErr := suggestTaskReferences(root, t)
	if refErr != nil {
		return usageError(stderr, fmt.Sprintf("pose suggest: invalid task-map entry %q: %v", task, refErr))
	}
	source, domainProblem := "explicit", ""
	if domain == "" && pathHint != "" {
		domain, source, domainProblem = inferSuggestDomain(root, pathHint)
		if domainProblem != "" {
			_ = usageError(stderr, "pose suggest: "+domainProblem+"; the trail below has no domain rules")
		}
	}
	domainInput := domain
	domain, aliased := normalizeSuggestDomain(domain)
	if aliased && source == "explicit" {
		source = "alias"
	}
	domainKnown := true
	if domain != "" {
		extra, known := byDomain[domain]
		if known {
			rules = append(rules, extra...)
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
		payload["references_status"] = references
		if pathHint != "" {
			payload["path_input"] = pathHint
			if domain == "" {
				// No domain is a finding of its own: the index could not say, or it said none.
				payload["domain_source"] = source
				payload["domain_status"] = "no domain inferred"
				if domainProblem != "" {
					payload["domain_status"] = domainProblem
				}
			}
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
	if domain == "" && domainProblem != "" {
		domainNote = fmt.Sprintf("- Domain:    unknown (%s)\n", domainProblem)
	}
	if domain != "" && !domainKnown {
		domainNote = fmt.Sprintf("- Domain:    %s has no rules declared in the task map\n", domain)
	}
	fmt.Fprintf(stdout, "%s- Workflow:  %v%s\n- Skill:     %v%s\n", domainNote, t["workflow"], referenceNote(references, "workflow"), t["skill"], referenceNote(references, "skill"))
	paths := []string{}
	for _, st := range statuses {
		switch st.Availability {
		case ruleInstalled:
			paths = append(paths, st.Path)
		case ruleExtensionNeeded:
			paths = append(paths, st.Rule+" (not installed: "+st.NextStep+")")
		case ruleUnreadable:
			paths = append(paths, st.Rule+" (unreadable: "+st.Detail+")")
		default:
			paths = append(paths, st.Rule+" (unavailable: declared in the task map, no rule file and no extension known)")
		}
	}
	fmt.Fprintf(stdout, "- Rules:     %s\n- Spec:      %v\n- ADR:       %v\n- Knowledge: consume=%v, produce=%v\n- Validation: %v\n", strings.Join(paths, ", "), t["requires_spec"], t["requires_adr"], t["knowledge_consume"], t["knowledge_produce"], t["validation"])
	return 0
}

func stringSlice(value any) []string {
	out, _ := strictStringSlice(value)
	return out
}

// strictStringSlice reads a list of strings and refuses anything else: an element that is
// not a string, or a value that is not a list, is a malformed configuration and not an
// empty set of rules.
func strictStringSlice(value any) ([]string, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case []string:
		return v, nil
	case []any:
		out := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("element %d is %T, not a string", i, item)
			}
			out = append(out, s)
		}
		return out, nil
	}
	return nil, fmt.Errorf("expected a list of strings, got %T", value)
}

// suggestTaskRules reads the rules of one task-map entry: the base rules and the rules per
// domain. Every identifier must be a rule name the rule reader accepts, so a path is never
// announced as a rule.
func suggestTaskRules(t map[string]any) ([]string, map[string][]string, error) {
	// Absent is not null: a field the entry does not declare adds nothing, one declared as
	// null is a configuration that was lost.
	if raw, present := t["rules"]; present && raw == nil {
		return nil, nil, fmt.Errorf("rules: is null")
	}
	rules, err := strictStringSlice(t["rules"])
	if err != nil {
		return nil, nil, fmt.Errorf("rules: %w", err)
	}
	byDomain := map[string][]string{}
	if raw, present := t["rules_by_domain"]; present {
		if raw == nil {
			return nil, nil, fmt.Errorf("rules_by_domain: is null")
		}
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("rules_by_domain: expected an object, got %T", raw)
		}
		for domain, value := range m {
			if value == nil {
				return nil, nil, fmt.Errorf("rules_by_domain.%s: is null", domain)
			}
			list, lerr := strictStringSlice(value)
			if lerr != nil {
				return nil, nil, fmt.Errorf("rules_by_domain.%s: %w", domain, lerr)
			}
			byDomain[domain] = list
		}
	}
	check := func(where string, list []string) error {
		for _, rule := range list {
			if verr := posepkg.ValidateName(rule); verr != nil {
				return fmt.Errorf("%s: %w", where, verr)
			}
		}
		return nil
	}
	if err := check("rules", rules); err != nil {
		return nil, nil, err
	}
	for domain, list := range byDomain {
		if err := check("rules_by_domain."+domain, list); err != nil {
			return nil, nil, err
		}
	}
	return rules, byDomain, nil
}

// inferSuggestDomain says which domain a path belongs to, where that was read from and, when
// the answer is "none", whether that is because nothing matched or because the index could not
// be read: the third result is empty only when the index was read and simply had no match.
func inferSuggestDomain(root, pathHint string) (domain, source, problem string) {
	norm := normalizeModulePath(pathHint)
	for _, p := range strings.Split(norm, "/") {
		if p == "k8s" || p == "charts" || p == "helm" {
			return "k8s", "hint-path", ""
		}
	}
	raw, _, e := posepkg.Store{Root: root}.ReadIndex("repo-map.json")
	if e != nil {
		return "", "repo-map-unavailable", "the repo-map could not be read (" + e.Error() + ")"
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", "repo-map-unavailable", "the repo-map is not valid JSON (" + err.Error() + ")"
	}
	if err := posepkg.RejectDuplicateJSONKeys(raw); err != nil {
		return "", "repo-map-invalid", "the repo-map repeats a key or holds a control character (" + err.Error() + ")"
	}
	if data == nil {
		return "", "repo-map-invalid", "the repo-map is null, not an object"
	}
	// The shape of the document is checked before anything is selected from it: a collection
	// that is not a list, an item that is not an object or a path that is not a relative
	// string is a repo-map that cannot be read, not a repo-map without a match.
	type module struct{ path, domain, language string }
	var modules []module
	for _, kind := range []string{"apps", "services", "packages"} {
		value, present := data[kind]
		if !present {
			continue
		}
		items, ok := value.([]any)
		if !ok {
			return "", "repo-map-invalid", fmt.Sprintf("the repo-map's %s is %T, not a list", kind, value)
		}
		for i, rawItem := range items {
			item, ok := rawItem.(map[string]any)
			if !ok {
				return "", "repo-map-invalid", fmt.Sprintf("the repo-map's %s[%d] is %T, not an object", kind, i, rawItem)
			}
			var m module
			for field, dst := range map[string]*string{"path": &m.path, "domain": &m.domain, "language": &m.language} {
				v, present := item[field]
				if !present {
					continue
				}
				s, ok := v.(string)
				if !ok {
					return "", "repo-map-invalid", fmt.Sprintf("the repo-map's %s[%d].%s is %T, not a string", kind, i, field, v)
				}
				*dst = s
			}
			if _, hasPath := item["path"]; !hasPath {
				return "", "repo-map-invalid", fmt.Sprintf("the repo-map's %s[%d] has no path", kind, i)
			}
			if strings.HasPrefix(filepath.ToSlash(m.path), "/") || filepath.IsAbs(m.path) || m.path == ".." || strings.HasPrefix(path.Clean(filepath.ToSlash(m.path)), "../") {
				return "", "repo-map-invalid", fmt.Sprintf("the repo-map's %s[%d].path %q is not a path inside the project", kind, i, m.path)
			}
			modules = append(modules, m)
		}
	}
	// Two entries for one canonical path must say the same thing: the order of the collections
	// is not a way to choose between a Go module and one that is not.
	seenModule := map[string]module{}
	for _, m := range modules {
		canonical := normalizeModulePath(m.path)
		if prior, dup := seenModule[canonical]; dup && (prior.domain != m.domain || prior.language != m.language) {
			return "", "repo-map-invalid", fmt.Sprintf("the repo-map declares the module %q twice with different domain or language", canonical)
		}
		seenModule[canonical] = m
	}
	best := -1 // the root module (an empty path) matches everything, with the least specificity
	for _, m := range modules {
		p := normalizeModulePath(m.path)
		if !(p == "" || norm == p || strings.HasPrefix(norm, p+"/")) || len(p) <= best {
			continue
		}
		best = len(p)
		domain = m.domain
		if domain == "" || domain == "unknown" {
			domain = map[string]string{"go": "backend-go", "javascript": "frontend", "typescript": "frontend"}[m.language]
		}
		// "backend" names no language: it is the Go domain only when the module is Go.
		if domain == "backend" {
			goMod := false
			if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(p), "go.mod")); err == nil && info.Mode().IsRegular() && insideProject(root, filepath.Join(root, filepath.FromSlash(p))) && insideProject(root, filepath.Join(root, filepath.FromSlash(p), "go.mod")) {
				goMod = true
			}
			if m.language == "go" || strings.HasSuffix(norm, ".go") || goMod {
				domain = "backend-go"
			}
		}
	}
	if domain != "" {
		return domain, "repo-map", ""
	}
	return "", "repo-map-no-match", ""
}

// normalizeModulePath makes a module path and a path hint comparable: slash-separated,
// cleaned, without a leading "./", and "" for the root.
func normalizeModulePath(p string) string {
	// The path is compared as it is written: a name with a space in it is a different name, and
	// the same representation validates and compares.
	p = path.Clean(filepath.ToSlash(p))
	if p == "." || p == "/" {
		return ""
	}
	return strings.TrimPrefix(p, "./")
}

// insideProject reports that dir, with its links followed, is inside the project root.
func insideProject(root, dir string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
	// ruleUnreadable: something is at the rule's place, or the place cannot be looked into,
	// and the rule cannot be read. Installing an extension is not the remedy.
	ruleUnreadable = "unreadable"
)

// ruleStatus says whether a rule the trail names can be read here: declared is not
// installed, and installed is not applied.
type ruleStatus struct {
	Rule         string `json:"rule"`
	Availability string `json:"availability"`
	Path         string `json:"path,omitempty"`
	Extension    string `json:"extension,omitempty"`
	NextStep     string `json:"next_step,omitempty"`
	Detail       string `json:"detail,omitempty"`
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
		state, stateErr := ruleFileState(root, file)
		switch {
		case state == ruleFileReadable:
			st.Availability, st.Path = ruleInstalled, ".pose/rules/"+file
		case state == ruleFileUnreadable:
			st.Availability, st.Detail = ruleUnreadable, stateErr.Error()
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

// Reserved: the fields suggest derives. A task cannot declare them, or its configuration would
// overwrite what the command found.
var suggestDerivedFields = []string{"name", "domain_effective", "domain_source", "domain_input", "domain_status", "rules_effective", "rules_status", "references_status", "path_input"}

// referenceStatus says whether a workflow or a skill the task names can be read.
type referenceStatus struct {
	Field        string `json:"field"`
	Ref          string `json:"ref"`
	Availability string `json:"availability"`
	Detail       string `json:"detail,omitempty"`
}

// suggestTaskReferences validates the workflow and the skill of a task: a string, a name that is
// not a path, and a file that is there and can be read. An absent reference is a finding of its
// own, not a reference to follow; a field of another type, or one that is derived, is a
// configuration error.
func suggestTaskReferences(root string, t map[string]any) ([]referenceStatus, error) {
	for _, reserved := range suggestDerivedFields {
		if _, present := t[reserved]; present {
			return nil, fmt.Errorf("%s is derived by the command and cannot be declared by the task", reserved)
		}
	}
	// The operational fields the trail prints, with the types the distributed task-map uses: a
	// field that is declared and has another type (or is null) is a configuration that was lost.
	for field, allowed := range map[string]string{
		"description": "s", "validation": "s", "requires_spec": "bs", "requires_adr": "bs",
		"knowledge_consume": "bs", "knowledge_produce": "bs",
	} {
		value, present := t[field]
		if !present {
			continue
		}
		switch value.(type) {
		case string:
			// Every operational field admits text.
		case bool:
			if !strings.Contains(allowed, "b") {
				return nil, fmt.Errorf("%s: expected %s, got a boolean", field, suggestFieldWant(allowed))
			}
		default:
			return nil, fmt.Errorf("%s: expected %s, got %T", field, suggestFieldWant(allowed), value)
		}
	}
	out := []referenceStatus{}
	for _, field := range []string{"workflow", "skill"} {
		raw, present := t[field]
		if !present {
			continue
		}
		ref, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("%s: expected a string, got %T", field, raw)
		}
		if ref == "" {
			continue
		}
		var rel string
		if field == "workflow" {
			if !confinedRelativePath(ref) || filepath.IsAbs(ref) {
				return nil, fmt.Errorf("workflow: %q is not a path inside the project", ref)
			}
			rel = ref
		} else {
			if verr := posepkg.ValidateName(ref); verr != nil {
				return nil, fmt.Errorf("skill: %w", verr)
			}
			rel = filepath.ToSlash(filepath.Join(".agents", "skills", ref, "SKILL.md"))
		}
		status := referenceStatus{Field: field, Ref: ref, Availability: "available"}
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		switch {
		case errors.Is(err, os.ErrNotExist):
			status.Availability, status.Detail = "unavailable", rel+" does not exist"
		case err != nil:
			status.Availability, status.Detail = "unreadable", err.Error()
		case !info.Mode().IsRegular():
			status.Availability, status.Detail = "unreadable", rel+" is not a regular file"
		default:
			// Present and regular is not readable: the file is opened, as the reader of it will.
			if file, oerr := os.Open(filepath.Join(root, filepath.FromSlash(rel))); oerr != nil {
				status.Availability, status.Detail = "unreadable", oerr.Error()
			} else {
				file.Close()
			}
		}
		out = append(out, status)
	}
	return out, nil
}

func referenceNote(references []referenceStatus, field string) string {
	for _, r := range references {
		if r.Field == field && r.Availability != "available" {
			return " (" + r.Availability + ": " + r.Detail + ")"
		}
	}
	return ""
}

func suggestFieldWant(allowed string) string {
	if allowed == "s" {
		return "a string"
	}
	return "a boolean or a string"
}
