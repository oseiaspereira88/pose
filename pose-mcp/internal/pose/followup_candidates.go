package pose

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// FollowupCandidate is an open follow-up the engine has a reason to bring to
// a person's attention for reconciliation (spec
// pose-followup-reconciliation-candidates). A candidate is never a
// disposition: a done target or an existing test suggests the item may be
// settled, it does not show the follow-up's intent was met.
type FollowupCandidate struct {
	Spec     string   `json:"spec"`
	Ordinal  int      `json:"ordinal"`
	Text     string   `json:"text"`
	Kinds    []string `json:"kinds"`
	Evidence []string `json:"evidence"`
	Limits   string   `json:"limits"`
}

// Candidate kinds.
const (
	CandidateTargetTerminal  = "target-terminal"
	CandidateEvidencePresent = "evidence-present"
	CandidateOverdue         = "overdue"
)

var (
	followupSpecRef  = regexp.MustCompile("(?:spec:|`)([a-z0-9][a-z0-9-]{3,})`?")
	followupTestName = regexp.MustCompile(`\bTest[A-Z][A-Za-z0-9_]+`)
)

const followupCandidateLimits = "the engine matched names and dates only; whether the follow-up's intent is met needs a person's or an agent's judgment, and the disposition is written in the spec"

// FollowupCandidates lists open follow-ups with a reconciliation reason.
func (s Store) FollowupCandidates(today string) ([]FollowupCandidate, error) {
	if today == "" {
		today = time.Now().UTC().Format(time.DateOnly)
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	status := map[string]string{}
	for _, sp := range specs {
		status[sp.Slug] = sp.Status
	}
	tests := s.testNames()
	var out []FollowupCandidate
	for _, item := range ParseSpecFollowups(s.Root) {
		if item.RawDisposition != "open" {
			continue
		}
		candidate := FollowupCandidate{Spec: item.Spec, Ordinal: item.Ordinal, Text: item.Text, Limits: followupCandidateLimits}
		seen := map[string]bool{}
		for _, m := range followupSpecRef.FindAllStringSubmatch(item.Text, -1) {
			slug := m[1]
			if slug == item.Spec || seen[slug] {
				continue
			}
			seen[slug] = true
			if st, ok := status[slug]; ok && (st == "done" || st == "superseded" || st == "abandoned") {
				candidate.Kinds = appendCandidateKind(candidate.Kinds, CandidateTargetTerminal)
				candidate.Evidence = append(candidate.Evidence, "spec:"+slug+" is "+st)
			}
		}
		for _, name := range followupTestName.FindAllString(item.Text, -1) {
			if tests[name] {
				candidate.Kinds = appendCandidateKind(candidate.Kinds, CandidateEvidencePresent)
				candidate.Evidence = append(candidate.Evidence, "test:"+name+" exists")
			}
		}
		if item.Review != "" && item.Review < today {
			candidate.Kinds = appendCandidateKind(candidate.Kinds, CandidateOverdue)
			candidate.Evidence = append(candidate.Evidence, "review date "+item.Review+" passed")
		}
		if len(candidate.Kinds) > 0 {
			sort.Strings(candidate.Kinds)
			out = append(out, candidate)
		}
	}
	return out, nil
}

// testNames collects Go test function names in the project, bounded to
// *_test.go files outside vendored and hidden trees.
func (s Store) testNames() map[string]bool {
	names := map[string]bool{}
	declaration := regexp.MustCompile(`(?m)^func (Test[A-Z][A-Za-z0-9_]+)\(`)
	_ = filepath.WalkDir(s.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == "vendor" || (strings.HasPrefix(base, ".") && path != s.Root) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, m := range declaration.FindAllStringSubmatch(string(raw), -1) {
			names[m[1]] = true
		}
		return nil
	})
	return names
}

func appendCandidateKind(values []string, value string) []string {
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}
