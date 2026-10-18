package cli

import (
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
