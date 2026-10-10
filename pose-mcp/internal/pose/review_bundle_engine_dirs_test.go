package pose

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Spec pose-review-subject-classifies-engine-records: every directory the
// engine writes under .pose/ has a review-subject class. `pose adopt` wrote
// .pose/review-ledgers/ and the bundle refused to seal a scope that carried
// it as an unclassified path (found in Harne8's independent review), so the
// set is read from the engine's own source rather than kept by hand.
func TestEveryEngineDirectoryUnderPoseIsClassified(t *testing.T) {
	joined := regexp.MustCompile(`"\.pose",\s*"([a-z][a-z0-9-]*)"`)
	literal := regexp.MustCompile(`"\.pose/([a-z][a-z0-9-]*)(?:/|")`)
	dirs := map[string]string{}
	for _, pkg := range []string{".", "../cli", "../mcpserver"} {
		files, _ := filepath.Glob(filepath.Join(pkg, "*.go"))
		for _, file := range files {
			if strings.HasSuffix(file, "_test.go") {
				continue
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			for _, re := range []*regexp.Regexp{joined, literal} {
				for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
					if _, seen := dirs[m[1]]; !seen {
						dirs[m[1]] = file
					}
				}
			}
		}
	}
	if len(dirs) < 20 {
		t.Fatalf("the scan found only %d directories; the patterns no longer match the source", len(dirs))
	}
	var unclassified []string
	for dir, file := range dirs {
		scope := ScopeRef{Kind: "spec", Slug: "probe"}
		// A name may be a file (.pose/schema-version) or a directory.
		asFile, _ := reviewBundlePathClass(".pose/"+dir, scope, nil)
		asDir, _ := reviewBundlePathClass(".pose/"+dir+"/probe.json", scope, nil)
		if asFile == "" && asDir == "" {
			unclassified = append(unclassified, dir+" (written in "+file+")")
		}
	}
	sort.Strings(unclassified)
	if len(unclassified) > 0 {
		t.Fatalf("directories the engine writes under .pose/ have no review-subject class:\n%s", strings.Join(unclassified, "\n"))
	}
}

// The legacy ledger is an authority record: it is reviewed with the scope
// that writes it, not skipped as derived output.
func TestLegacyLedgerIsAGovernanceRecord(t *testing.T) {
	class, include := reviewBundlePathClass(".pose/review-ledgers/legacy-harne8-agents-20261010T032511Z.json", ScopeRef{Kind: "spec", Slug: "x"}, nil)
	if class != "governance" || !include {
		t.Fatalf("legacy ledger: class=%q include=%v", class, include)
	}
}
