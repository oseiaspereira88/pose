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
	// Names with an extension count too (.pose/telemetry.json), and every
	// package of the module is read, not a chosen few (found in review:
	// internal/usage writes .pose/usage/ and was not scanned).
	// Upper case and underscores count too (.pose/LICENSE, found in review).
	joined := regexp.MustCompile(`"\.pose",\s*"([A-Za-z_][A-Za-z0-9_.-]*)"`)
	literal := regexp.MustCompile(`"\.pose/([A-Za-z_][A-Za-z0-9_.-]*)(?:/|")`)
	// Names that follow ".pose" in an argument list without being under it:
	// setup.go passes ".pose", "AGENTS.md", "POSE.md" to git status.
	notUnderPose := map[string]bool{"AGENTS.md": true, "POSE.md": true}
	dirs := map[string]string{}
	err := filepath.WalkDir("../..", func(file string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && d.Name() == "testdata" {
			return filepath.SkipDir
		}
		if err != nil || d.IsDir() || !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			return err
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, re := range []*regexp.Regexp{joined, literal} {
			for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
				if re == joined && notUnderPose[m[1]] {
					continue
				}
				if _, seen := dirs[m[1]]; !seen {
					dirs[m[1]] = file
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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

// The usage verdict journal is reviewed in Git; it is governance.
func TestUsageVerdictJournalIsAGovernanceRecord(t *testing.T) {
	class, include := reviewBundlePathClass(".pose/usage/verdicts.jsonl", ScopeRef{Kind: "spec", Slug: "x"}, nil)
	if class != "governance" || !include {
		t.Fatalf("usage verdicts: class=%q include=%v", class, include)
	}
}
