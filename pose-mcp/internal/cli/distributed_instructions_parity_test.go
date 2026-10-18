package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Spec pose-distributed-instructions-parity: the instructions POSE distributes describe the engine
// as it is, and `suggest` and `migrate` say what is actually available.

// instructionRoots are the trees inventoried for distributed instructions: the source files and
// every mirror that install or update hands to an instance (the embedded scaffold and its
// locales). The inventory is the walk, not a list of file names, so a mirror added later is
// covered without editing this test.
var instructionRoots = []string{
	"AGENTS.md", "POSE.md", ".agents", ".pose/workflows", ".pose/rules", "locales",
	"pose-mcp/internal/scaffold/dist",
}

func distributedInstructionFiles(t *testing.T) (root string, files []string) {
	t.Helper()
	root, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Skip("the distributed instruction sources are not part of this tree")
	}
	for _, entry := range instructionRoots {
		base := filepath.Join(root, filepath.FromSlash(entry))
		_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") {
				files = append(files, path)
			}
			return nil
		})
	}
	if len(files) < 20 {
		t.Fatalf("the inventory found %d instruction files: the roots moved", len(files))
	}
	return root, files
}

var discoverCommandRE = regexp.MustCompile("`?pose assess discover[^`\\n]*")

// R1 and R2: a workflow, skill, rule or manual that tells the reader to run discovery says
// `--if-stale`, in every file that is distributed, in every locale.
func TestDistributedInstructionsAskForDiscoveryOnlyWhenStale(t *testing.T) {
	root, files := distributedInstructionFiles(t)
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			for _, m := range discoverCommandRE.FindAllString(line, -1) {
				if !strings.Contains(m, "--if-stale") {
					rel, _ := filepath.Rel(root, file)
					t.Errorf("%s:%d asks for an unconditional discovery: %s", rel, i+1, strings.TrimSpace(m))
				}
			}
		}
	}
}

// R2: the mirrors of one source say the same thing.
func TestTheEmbeddedMirrorsOfTheWorkflowsAndRulesMatchTheSources(t *testing.T) {
	root, _ := distributedInstructionFiles(t)
	for _, rel := range []string{
		".pose/workflows/feature.md", ".pose/rules/delivery-evidence.md",
		"locales/pt-BR/.pose/workflows/feature.md", "locales/pt-BR/.pose/rules/delivery-evidence.md",
	} {
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		mirror, err := os.ReadFile(filepath.Join(root, "pose-mcp", "internal", "scaffold", "dist", filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("%s has no embedded mirror: %v", rel, err)
		}
		if string(source) != string(mirror) {
			t.Errorf("%s and its embedded mirror differ", rel)
		}
	}
}

func suggestJSON(t *testing.T, root string, args ...string) map[string]any {
	t.Helper()
	code, out := runPose(t, root, append([]string{"suggest", "feature", "--json"}, args...)...)
	if code != 0 {
		t.Fatalf("suggest %v: %d %s", args, code, out)
	}
	start := strings.Index(out, "{")
	var doc map[string]any
	if err := json.Unmarshal([]byte(out[start:]), &doc); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	return doc
}

func suggestFixture(t *testing.T, withBackendRule bool) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/indexes/task-map.json", `{"tasks":{"feature":{"description":"d","workflow":"w","skill":"s","rules":["security"],"rules_by_domain":{"frontend":["frontend-react"],"backend-go":["backend-go"],"k8s":["kubernetes"]}}}}`)
	writeCloseoutCLIFile(t, root, ".pose/rules/security.md", "# security\n")
	if withBackendRule {
		writeCloseoutCLIFile(t, root, ".pose/rules/backend-go.md", "# backend-go\n")
	}
	return root
}

// R3: every spelling of a domain the taxonomy knows reaches the same rules; one it does not know
// is reported, not turned into another domain.
func TestSuggestResolvesDomainAliasesThroughOneTaxonomy(t *testing.T) {
	root := suggestFixture(t, true)
	for _, alias := range []string{"go", "golang", "backend-go", "Go"} {
		doc := suggestJSON(t, root, "--domain", alias)
		if doc["domain_effective"] != "backend-go" {
			t.Errorf("%s: domain_effective = %v", alias, doc["domain_effective"])
		}
		rules, _ := doc["rules_effective"].([]any)
		if len(rules) != 2 || rules[1] != "backend-go" {
			t.Errorf("%s: rules = %v", alias, rules)
		}
	}
	if doc := suggestJSON(t, root, "--domain", "go"); doc["domain_input"] != "go" || doc["domain_source"] != "alias" {
		t.Errorf("the alias is not disclosed: %v %v", doc["domain_input"], doc["domain_source"])
	}
	doc := suggestJSON(t, root, "--domain", "cobol")
	if doc["domain_effective"] != "cobol" || doc["domain_status"] == nil {
		t.Errorf("an unknown domain must be reported as having no rules: %v", doc)
	}
	if rules, _ := doc["rules_effective"].([]any); len(rules) != 1 {
		t.Errorf("an unknown domain must not borrow another domain's rules: %v", rules)
	}
}

// R3: a path under a backend module of another language is not the Go domain.
func TestSuggestDoesNotTurnEveryBackendIntoGo(t *testing.T) {
	root := suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc-go","domain":"backend","language":"go"},{"path":"svc-py","domain":"backend","language":"python"}]}`)
	if doc := suggestJSON(t, root, "--path", "svc-go/internal"); doc["domain_effective"] != "backend-go" || doc["domain_source"] != "repo-map" {
		t.Errorf("a Go backend: %v %v", doc["domain_effective"], doc["domain_source"])
	}
	if doc := suggestJSON(t, root, "--path", "svc-py/app"); doc["domain_effective"] == "backend-go" {
		t.Errorf("a Python backend was routed to the Go rules: %v", doc)
	}
}

// R4 and R6: declared, installed and installable are different facts, and the human output
// and the JSON say the same.
func TestSuggestSeparatesDeclaredFromInstalledRules(t *testing.T) {
	root := suggestFixture(t, false)
	doc := suggestJSON(t, root, "--domain", "go")
	statuses, _ := doc["rules_status"].([]any)
	byRule := map[string]map[string]any{}
	for _, raw := range statuses {
		st, _ := raw.(map[string]any)
		byRule[st["rule"].(string)] = st
	}
	if byRule["security"]["availability"] != "installed" || byRule["security"]["path"] != ".pose/rules/security.md" {
		t.Errorf("security = %v", byRule["security"])
	}
	if st := byRule["backend-go"]; st["availability"] != "extension-needed" || st["extension"] != "pose-rule-backend-go" || st["next_step"] != "pose extension install pose-rule-backend-go" || st["path"] != nil {
		t.Errorf("backend-go = %v", st)
	}
	code, out := runPose(t, root, "suggest", "feature", "--domain", "go")
	if code != 0 || strings.Contains(out, ".pose/rules/backend-go.md") || !strings.Contains(out, "pose extension install pose-rule-backend-go") {
		t.Errorf("the human output announces a rule file that is not there: %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, ".pose", "rules", "backend-go.md")); err == nil {
		t.Error("suggest installed a rule")
	}
	installed := suggestFixture(t, true)
	if code, out := runPose(t, installed, "suggest", "feature", "--domain", "go"); code != 0 || !strings.Contains(out, ".pose/rules/backend-go.md") {
		t.Errorf("an installed rule is named by its path: %d\n%s", code, out)
	}
}

// R5: migrate names what exists and promises nothing.
func TestMigrateSaysThereIsNoApplyWithoutPromisingOne(t *testing.T) {
	root := t.TempDir()
	code, out := runPose(t, root, "migrate", "v7", "--apply")
	if code == 0 {
		t.Fatalf("an apply that does not exist succeeded: %s", out)
	}
	for _, want := range []string{"no apply is available", "pose migrate v7 --dry-run", "writes nothing"} {
		if !strings.Contains(out, want) {
			t.Errorf("the refusal lacks %q: %s", want, out)
		}
	}
	for _, promise := range []string{"7.0 apply", "will carry", "6.x"} {
		if strings.Contains(out, promise) {
			t.Errorf("the refusal promises a future apply (%q): %s", promise, out)
		}
	}
}

// R2: what install actually writes into a new instance is checked, not only the sources: no
// unconditional discovery in any instruction the instance receives, in either locale.
func TestAFreshInstallReceivesInstructionsThatAskForDiscoveryOnlyWhenStale(t *testing.T) {
	instance := freshInstall(t)
	checked := 0
	_ = filepath.WalkDir(instance, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".md") || strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		checked++
		for _, m := range discoverCommandRE.FindAllString(string(raw), -1) {
			if !strings.Contains(m, "--if-stale") {
				rel, _ := filepath.Rel(instance, path)
				t.Errorf("%s: the installed instruction asks for an unconditional discovery: %s", rel, strings.TrimSpace(m))
			}
		}
		return nil
	})
	if checked < 10 {
		t.Fatalf("only %d installed instruction files were inspected", checked)
	}
}
