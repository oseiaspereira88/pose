// Workflow security contract (spec pose-ossf-security-baseline R2):
// every GitHub workflow declares explicit permissions, every third-party
// action is pinned to a full commit SHA, and first-party tag pinning is only
// allowed while its owned exception has not expired.
package version_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type securityExceptions struct {
	ExceptionsVersion int `json:"exceptions_version"`
	Exceptions        []struct {
		ID            string `json:"id"`
		Owner         string `json:"owner"`
		Justification string `json:"justification"`
		Expires       string `json:"expires"`
	} `json:"exceptions"`
}

func loadExceptions(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile("../../../.github/security-exceptions.json")
	if err != nil {
		t.Fatalf("reading security-exceptions.json: %v", err)
	}
	var doc securityExceptions
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing security-exceptions.json: %v", err)
	}
	valid := map[string]bool{}
	for _, e := range doc.Exceptions {
		if e.ID == "" || e.Owner == "" || strings.TrimSpace(e.Justification) == "" {
			t.Errorf("exception %q must declare id, owner and justification", e.ID)
			continue
		}
		expires, err := time.Parse("2006-01-02", e.Expires)
		if err != nil {
			t.Errorf("exception %q has invalid expires %q", e.ID, e.Expires)
			continue
		}
		if time.Now().After(expires) {
			t.Errorf("exception %q expired on %s — renew it with a fresh review or fix the underlying gap", e.ID, e.Expires)
			continue
		}
		valid[e.ID] = true
	}
	return valid
}

var (
	usesRe   = regexp.MustCompile(`(?m)^\s*(?:-\s+)?uses:\s*([^\s#]+)`)
	shaRefRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
	tagRefRe = regexp.MustCompile(`^v\d+[\w.-]*$`)
)

// firstPartyOwners are GitHub-platform orgs covered by the
// first-party-actions-tag-pinning exception while it remains valid.
var firstPartyOwners = map[string]bool{"actions": true, "github": true}

func TestWorkflowSecurityContract(t *testing.T) {
	valid := loadExceptions(t)
	workflows, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(workflows) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	for _, wf := range workflows {
		raw, err := os.ReadFile(wf)
		if err != nil {
			t.Fatal(err)
		}
		content := string(raw)
		name := filepath.Base(wf)
		if !regexp.MustCompile(`(?m)^permissions:`).MatchString(content) {
			t.Errorf("%s: missing top-level permissions block (least privilege)", name)
		}
		for _, m := range usesRe.FindAllStringSubmatch(content, -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "./") {
				continue // local action, reviewed in-repo
			}
			action, version, ok := strings.Cut(ref, "@")
			if !ok {
				t.Errorf("%s: action %q has no version pin at all", name, ref)
				continue
			}
			owner := strings.SplitN(action, "/", 2)[0]
			if firstPartyOwners[owner] {
				if !valid["first-party-actions-tag-pinning"] {
					t.Errorf("%s: %s is tag-pinned but the first-party-actions-tag-pinning exception is missing or expired", name, ref)
				} else if !tagRefRe.MatchString(version) && !shaRefRe.MatchString(version) {
					t.Errorf("%s: %s must be pinned to a version tag or commit SHA", name, ref)
				}
				continue
			}
			if !shaRefRe.MatchString(version) {
				t.Errorf("%s: third-party action %s must be pinned to a full commit SHA", name, ref)
			}
		}
	}
}
