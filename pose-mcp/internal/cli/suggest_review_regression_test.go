package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression tests for the independent review of pose-distributed-instructions-parity.

func suggestWithTaskMap(t *testing.T, taskMap string) string {
	t.Helper()
	root := t.TempDir()
	writeCloseoutCLIFile(t, root, ".pose/indexes/task-map.json", taskMap)
	writeCloseoutCLIFile(t, root, ".pose/rules/security.md", "# security\n")
	return root
}

func TestSuggestRootModuleIsTheFallbackOfLeastSpecificity(t *testing.T) {
	root := suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"./","language":"go"},{"path":"web","language":"typescript"}]}`)
	for _, hint := range []string{"main.go", "./main.go", "cmd/tool/main.go"} {
		if doc := suggestJSON(t, root, "--path", hint); doc["domain_effective"] != "backend-go" {
			t.Errorf("%s: the root module did not apply: %v", hint, doc["domain_effective"])
		}
	}
	if doc := suggestJSON(t, root, "--path", "web/app.ts"); doc["domain_effective"] != "frontend" {
		t.Errorf("a nested module must win over the root: %v", doc["domain_effective"])
	}
}

func TestSuggestKubernetesRuleIsAnnouncedWithItsExtension(t *testing.T) {
	root := suggestFixture(t, true)
	doc := suggestJSON(t, root, "--domain", "kubernetes")
	statuses, _ := doc["rules_status"].([]any)
	for _, raw := range statuses {
		st, _ := raw.(map[string]any)
		if st["rule"] == "kubernetes" {
			if st["availability"] != "extension-needed" || st["extension"] != "pose-rule-kubernetes" {
				t.Errorf("kubernetes = %v", st)
			}
			return
		}
	}
	t.Fatalf("no status for the kubernetes rule: %v", doc)
}

func TestSuggestAnUnreadableRuleIsNotInstalledOrInstallable(t *testing.T) {
	root := suggestFixture(t, false)
	// A directory where the rule file belongs, and a file that cannot be opened.
	os.MkdirAll(filepath.Join(root, ".pose", "rules", "backend-go.md"), 0o755)
	doc := suggestJSON(t, root, "--domain", "go")
	for _, raw := range doc["rules_status"].([]any) {
		st, _ := raw.(map[string]any)
		if st["rule"] == "backend-go" && (st["availability"] != "unreadable" || st["next_step"] != nil || st["detail"] == nil) {
			t.Errorf("a directory is not an installed rule, and installing is not its remedy: %v", st)
		}
	}
	// A directory named go.mod does not make a module Go.
	root = suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","domain":"backend","language":"python"}]}`)
	os.MkdirAll(filepath.Join(root, "svc", "go.mod"), 0o755)
	if doc := suggestJSON(t, root, "--path", "svc/app"); doc["domain_effective"] == "backend-go" {
		t.Errorf("a go.mod directory routed a Python backend to Go: %v", doc["domain_effective"])
	}
}

func TestSuggestRefusesAMalformedRulesConfiguration(t *testing.T) {
	for name, taskMap := range map[string]string{
		"string":     `{"tasks":{"feature":{"rules":["security"],"rules_by_domain":{"backend-go":"backend-go"}}}}`,
		"number":     `{"tasks":{"feature":{"rules":["security"],"rules_by_domain":{"backend-go":[17]}}}}`,
		"base":       `{"tasks":{"feature":{"rules":["security", 3]}}}`,
		"domain-map": `{"tasks":{"feature":{"rules":["security"],"rules_by_domain":["backend-go"]}}}`,
	} {
		root := suggestWithTaskMap(t, taskMap)
		if code, out := runPose(t, root, "suggest", "feature", "--domain", "go"); code == 0 || !strings.Contains(out, "invalid task-map entry") {
			t.Errorf("%s: a malformed configuration was accepted: %d %s", name, code, out)
		}
	}
}

func TestSuggestRefusesARuleIdentifierThatIsAPath(t *testing.T) {
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["../../outside"]}}}`)
	writeCloseoutCLIFile(t, root, "outside.md", "# outside\n")
	if code, out := runPose(t, root, "suggest", "feature"); code == 0 || strings.Contains(out, "outside.md") && strings.Contains(out, "installed") {
		t.Errorf("a path was announced as an installed rule: %d %s", code, out)
	}
}

func TestSuggestSaysWhyItHasNoDomainForAPath(t *testing.T) {
	root := suggestFixture(t, true)
	for name, repoMap := range map[string]string{"malformed": "{", "absent": ""} {
		if repoMap != "" {
			writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", repoMap)
		}
		doc := suggestJSON(t, root, "--path", "svc/app")
		if doc["domain_source"] != "repo-map-unavailable" || doc["domain_status"] == nil {
			t.Errorf("%s: an unreadable map looks like no match: %v", name, doc)
		}
	}
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"other","language":"go"}]}`)
	if doc := suggestJSON(t, root, "--path", "svc/app"); doc["domain_source"] != "repo-map-no-match" {
		t.Errorf("a map without a match is a different answer: %v", doc)
	}
}

func TestSuggestRefusesNullWhereAListOrAnEntryWasConfigured(t *testing.T) {
	for name, taskMap := range map[string]string{
		"task":        `{"tasks":{"feature":null}}`,
		"rules":       `{"tasks":{"feature":{"rules":null}}}`,
		"by-domain":   `{"tasks":{"feature":{"rules":["security"],"rules_by_domain":null}}}`,
		"domain-list": `{"tasks":{"feature":{"rules":["security"],"rules_by_domain":{"backend-go":null}}}}`,
	} {
		root := suggestWithTaskMap(t, taskMap)
		if code, out := runPose(t, root, "suggest", "feature", "--domain", "go"); code == 0 || !strings.Contains(out, "invalid task-map entry") {
			t.Errorf("%s: a configuration that was lost was accepted: %d %s", name, code, out)
		}
	}
	// An empty list and an absent field are legitimate.
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":[],"rules_by_domain":{"backend-go":[]}}}}`)
	if code, out := runPose(t, root, "suggest", "feature", "--domain", "go"); code != 0 {
		t.Errorf("an empty list is a configuration: %d %s", code, out)
	}
}

func TestSuggestRefusesARepoMapWhoseShapeIsWrong(t *testing.T) {
	for name, repoMap := range map[string]string{
		"collection":  `{"services":"lost"}`,
		"item":        `{"services":[3]}`,
		"path type":   `{"services":[{"path":7,"domain":"backend-go"}]}`,
		"domain type": `{"services":[{"path":"svc","domain":5}]}`,
		"language":    `{"services":[{"path":"svc","language":true}]}`,
		"no path":     `{"services":[{"domain":"backend-go"}]}`,
		"absolute":    `{"services":[{"path":"/","domain":"backend-go"}]}`,
		"outside":     `{"services":[{"path":"../x","domain":"backend-go"}]}`,
	} {
		root := suggestFixture(t, true)
		writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", repoMap)
		doc := suggestJSON(t, root, "--path", "svc/main.py")
		if doc["domain_source"] != "repo-map-invalid" || doc["domain_effective"] != nil {
			t.Errorf("%s: a repo-map of the wrong shape was read as a match or as no match: %v", name, doc["domain_source"])
		}
	}
}

func TestSuggestSaysWhenARuleOrTheRulesDirectoryIsABrokenLink(t *testing.T) {
	root := suggestFixture(t, false)
	if err := os.Symlink(filepath.Join(root, "gone.md"), filepath.Join(root, ".pose", "rules", "backend-go.md")); err != nil {
		t.Skip("symlinks are not available")
	}
	doc := suggestJSON(t, root, "--domain", "go")
	for _, raw := range doc["rules_status"].([]any) {
		st, _ := raw.(map[string]any)
		if st["rule"] == "backend-go" && (st["availability"] != "unreadable" || st["next_step"] != nil || !strings.Contains(fmt.Sprint(st["detail"]), "link")) {
			t.Errorf("a broken link to a rule is not an absent rule, and is said to be a link: %v", st)
		}
	}
	lost := suggestFixture(t, false)
	os.RemoveAll(filepath.Join(lost, ".pose", "rules"))
	os.Symlink(filepath.Join(lost, "gone"), filepath.Join(lost, ".pose", "rules"))
	doc = suggestJSON(t, lost, "--domain", "go")
	for _, raw := range doc["rules_status"].([]any) {
		st, _ := raw.(map[string]any)
		if st["availability"] != "unreadable" {
			t.Errorf("a rules directory that is a broken link is not a project without rules: %v", st)
		}
	}
}

// Third review: the root of the repo-map, repeated keys, one representation of a path, and the
// fields of the task that the command derives.
func TestSuggestRefusesARepoMapThatIsNullOrRepeatsAKey(t *testing.T) {
	for name, repoMap := range map[string]string{
		"null root":      `null`,
		"repeated field": `{"services":[{"path":"svc","domain":"backend-go","domain":"unknown"}]}`,
		"repeated list":  `{"services":[],"services":[{"path":"svc","domain":"backend-go"}]}`,
		"conflict":       `{"services":[{"path":"svc","language":"go"},{"path":"./svc","domain":"backend","language":"python"}]}`,
	} {
		root := suggestFixture(t, true)
		writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", repoMap)
		doc := suggestJSON(t, root, "--path", "svc/main.py")
		if doc["domain_source"] != "repo-map-invalid" || doc["domain_effective"] != nil {
			t.Errorf("%s: read as %v", name, doc["domain_source"])
		}
	}
	// The same module twice, said the same way, is one module.
	root := suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","language":"go"}],"packages":[{"path":"./svc","language":"go"}]}`)
	if doc := suggestJSON(t, root, "--path", "svc/main.go"); doc["domain_effective"] != "backend-go" {
		t.Errorf("an equivalent repetition was refused: %v", doc["domain_source"])
	}
}

func TestSuggestRefusesATaskMapThatRepeatsAKey(t *testing.T) {
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],"rules":[]}}}`)
	if code, out := runPose(t, root, "suggest", "feature"); code == 0 || !strings.Contains(out, "invalid task-map") {
		t.Fatalf("a task-map that repeats a key was read (%d): %s", code, out)
	}
}

func TestSuggestComparesPathsAsWrittenAndDoesNotFollowALinkOutOfTheProject(t *testing.T) {
	root := suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","language":"go"},{"path":" svc","domain":"backend","language":"python"}]}`)
	if doc := suggestJSON(t, root, "--path", " svc/main.py"); doc["domain_effective"] != "backend" {
		t.Errorf("a name with a leading space must be the module written with it, not the one without: %v (%v)", doc["domain_effective"], doc["domain_source"])
	}
	outside := t.TempDir()
	writeCloseoutCLIFile(t, outside, "go.mod", "module outside\n")
	linked := suggestFixture(t, true)
	writeCloseoutCLIFile(t, linked, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","domain":"backend","language":"python"}]}`)
	if err := os.Symlink(outside, filepath.Join(linked, "svc")); err != nil {
		t.Skip("symlinks are not available")
	}
	if doc := suggestJSON(t, linked, "--path", "svc/app.py"); doc["domain_effective"] == "backend-go" {
		t.Errorf("a go.mod outside the project made a backend Go: %v", doc["domain_effective"])
	}
}

func TestSuggestDoesNotLetATaskOverwriteWhatItDerivesOrNameWhatIsNotThere(t *testing.T) {
	for name, task := range map[string]string{
		"derived name":   `"name":"other"`,
		"derived domain": `"domain_effective":"backend-go"`,
		"derived rules":  `"rules_effective":[]`,
		"number":         `"workflow":17`,
		"boolean":        `"skill":false`,
	} {
		root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],`+task+`}}}`)
		if code, out := runPose(t, root, "suggest", "feature"); code == 0 {
			t.Errorf("%s: accepted: %s", name, out)
		}
	}
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],"workflow":".pose/workflows/feature.md","skill":"pose-feature"}}}`)
	doc := suggestJSON(t, root, "--domain", "go")
	statuses, _ := doc["references_status"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("references_status = %v", doc["references_status"])
	}
	for _, raw := range statuses {
		if st, _ := raw.(map[string]any); st["availability"] != "unavailable" {
			t.Errorf("a reference that is not there is announced as available: %v", st)
		}
	}
	code, out := runPose(t, root, "suggest", "feature", "--domain", "go")
	if code != 0 || !strings.Contains(out, "unavailable") {
		t.Errorf("the human output does not say the references are not there (%d): %s", code, out)
	}
	writeCloseoutCLIFile(t, root, ".pose/workflows/feature.md", "# feature\n")
	doc = suggestJSON(t, root, "--domain", "go")
	for _, raw := range doc["references_status"].([]any) {
		if st, _ := raw.(map[string]any); st["field"] == "workflow" && st["availability"] != "available" {
			t.Errorf("a workflow that is there: %v", st)
		}
	}
}

// Fourth review: unreadable references, keys that fold to the same field, the file go.mod points
// at, and the operational fields of a task.
func TestSuggestSaysAReferenceThatCannotBeReadIsNotAvailable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],"workflow":".pose/workflows/feature.md","skill":"pose-feature"}}}`)
	writeCloseoutCLIFile(t, root, ".pose/workflows/feature.md", "# feature\n")
	writeCloseoutCLIFile(t, root, ".agents/skills/pose-feature/SKILL.md", "# skill\n")
	doc := suggestJSON(t, root, "--domain", "go")
	for _, raw := range doc["references_status"].([]any) {
		if st, _ := raw.(map[string]any); st["availability"] != "available" {
			t.Fatalf("the control reference is not available: %v", st)
		}
	}
	for _, rel := range []string{".pose/workflows/feature.md", ".agents/skills/pose-feature/SKILL.md"} {
		os.Chmod(filepath.Join(root, rel), 0)
		defer os.Chmod(filepath.Join(root, rel), 0o644)
	}
	doc = suggestJSON(t, root, "--domain", "go")
	for _, raw := range doc["references_status"].([]any) {
		if st, _ := raw.(map[string]any); st["availability"] != "unreadable" {
			t.Errorf("a reference that cannot be opened is announced as %v: %v", st["availability"], st["field"])
		}
	}
}

func TestSuggestRefusesKeysThatFoldToTheTasksField(t *testing.T) {
	for _, key := range []string{"Tasks", "TASKS", "TaSkS", "taſks"} {
		for _, exactFirst := range []bool{true, false} {
			exact := `"tasks":{"feature":{"rules":["security"]}}`
			folded := `"` + key + `":{"feature":{"rules":[]}}`
			doc := "{" + folded + "," + exact + "}"
			if exactFirst {
				doc = "{" + exact + "," + folded + "}"
			}
			root := suggestWithTaskMap(t, doc)
			if code, out := runPose(t, root, "suggest", "feature"); code == 0 || !strings.Contains(out, "ambiguous") {
				t.Errorf("%s (exact first %v): the order of the document decided the rules (%d): %s", key, exactFirst, code, out)
			}
		}
	}
}

func TestSuggestDoesNotFollowGoModOutOfTheProject(t *testing.T) {
	outside := t.TempDir()
	writeCloseoutCLIFile(t, outside, "go.mod", "module outside\n")
	root := suggestFixture(t, true)
	writeCloseoutCLIFile(t, root, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","domain":"backend","language":"python"}]}`)
	os.MkdirAll(filepath.Join(root, "svc"), 0o755)
	if err := os.Symlink(filepath.Join(outside, "go.mod"), filepath.Join(root, "svc", "go.mod")); err != nil {
		t.Skip("symlinks are not available")
	}
	if doc := suggestJSON(t, root, "--path", "svc/app.py"); doc["domain_effective"] == "backend-go" {
		t.Errorf("a go.mod that is a link out of the project made a backend Go: %v", doc["domain_effective"])
	}
	// An internal go.mod still counts.
	inner := suggestFixture(t, true)
	writeCloseoutCLIFile(t, inner, ".pose/indexes/repo-map.json", `{"services":[{"path":"svc","domain":"backend","language":"python"}]}`)
	writeCloseoutCLIFile(t, inner, "svc/go.mod", "module inner\n")
	if doc := suggestJSON(t, inner, "--path", "svc/app.py"); doc["domain_effective"] != "backend-go" {
		t.Errorf("a go.mod inside the project must still count: %v", doc["domain_effective"])
	}
}

func TestSuggestRefusesOperationalFieldsOfAnImpossibleType(t *testing.T) {
	for name, field := range map[string]string{
		"requires_spec list":  `"requires_spec":[]`,
		"requires_adr object": `"requires_adr":{}`,
		"validation boolean":  `"validation":false`,
		"consume number":      `"knowledge_consume":17`,
		"description null":    `"description":null`,
		"produce list":        `"knowledge_produce":["a"]`,
		"validation number":   `"validation":3`,
	} {
		root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],`+field+`}}}`)
		if code, out := runPose(t, root, "suggest", "feature"); code == 0 {
			t.Errorf("%s: accepted: %s", name, out)
		}
	}
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],"description":"d","validation":"run the checks","requires_spec":true,"requires_adr":"when structural","knowledge_consume":false,"knowledge_produce":"yes"}}}`)
	if code, out := runPose(t, root, "suggest", "feature"); code != 0 {
		t.Errorf("the supported types were refused (%d): %s", code, out)
	}
}

// R6-B1: the checker and suggest read the same field names; a spelling only the checker knows
// is refused by both instead of becoming a field that was left out.
func TestSuggestAndTheCheckerRefuseANonCanonicalSpellingOfAnOperationalField(t *testing.T) {
	for _, field := range []string{`"Rules":["security"]`, `"Workflow":".pose/workflows/feature.md"`, `"Skill":"pose-feature"`, `"Validation":"x"`, `"RULES_BY_DOMAIN":{}`, `"Domain_Effective":"x"`} {
		taskMap := `{"tasks":{"feature":{"rules":["security"],` + field + `}}}`
		root := suggestWithTaskMap(t, taskMap)
		if code, out := runPose(t, root, "suggest", "feature"); code == 0 {
			t.Errorf("suggest accepted %s: %s", field, out)
		}
		var out bytes.Buffer
		checker := &nativeChecker{root: root, mode: "strict", locale: localeEN, stdout: &out}
		checker.checkTaskMap()
		if !strings.Contains(out.String(), "field names are exact") {
			t.Errorf("the checker accepted %s: %q", field, out.String())
		}
	}
	root := suggestWithTaskMap(t, `{"tasks":{"feature":{"rules":["security"],"description":"d"}}}`)
	if code, out := runPose(t, root, "suggest", "feature"); code != 0 {
		t.Errorf("the canonical spelling was refused (%d): %s", code, out)
	}
}
