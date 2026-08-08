// Action runtime currency (spec pose-action-runtime-currency-gate).
//
// Nothing detected the Node.js 20 deprecation. It was found because a human
// read the annotation block under a passing run, weeks after the runner had
// silently started substituting Node 24 — until then every gate was green and
// every workflow was one GitHub-controlled withdrawal away from failing at
// once.
//
// The signal was machine-readable the whole time: each action declares its
// runtime in its own action.yml. This check reads the recorded runtime for
// every pinned action and fails when it is one GitHub has deprecated, or when
// a referenced action has no record at all.
//
// The record is a second source of truth, and its only failure mode is drifting
// from the real action.yml — which is why the CI step that re-resolves each
// action at its pinned ref is the requirement that carries this design, not
// this offline half.
package version_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type actionRuntimes struct {
	SchemaVersion      int `json:"schema_version"`
	DeprecatedRuntimes []struct {
		Runtime       string `json:"runtime"`
		Owner         string `json:"owner"`
		Announced     string `json:"announced"`
		Justification string `json:"justification"`
	} `json:"deprecated_runtimes"`
	Runtimes map[string]struct {
		Ref   string `json:"ref"`
		Using string `json:"using"`
	} `json:"runtimes"`
}

var actionUsesRe = regexp.MustCompile(`(?m)^\s*(?:-\s+)?uses:\s*([^\s#]+)`)

// referencedActions returns every `owner/repo[/path]` referenced by a workflow,
// with its pinned ref.
func referencedActions(t *testing.T) map[string]string {
	t.Helper()
	workflows, err := filepath.Glob("../../../.github/workflows/*.yml")
	if err != nil || len(workflows) == 0 {
		t.Fatalf("no workflows found: %v", err)
	}
	out := map[string]string{}
	for _, wf := range workflows {
		raw, err := os.ReadFile(wf)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range actionUsesRe.FindAllStringSubmatch(string(raw), -1) {
			ref := m[1]
			if strings.HasPrefix(ref, "./") {
				continue // local action, reviewed in-repo
			}
			action, pinned, ok := strings.Cut(ref, "@")
			if !ok {
				continue // the pinning contract already fails this
			}
			out[action] = pinned
		}
	}
	return out
}

func loadActionRuntimes(t *testing.T) actionRuntimes {
	t.Helper()
	raw, err := os.ReadFile("../../../.github/action-runtimes.json")
	if err != nil {
		t.Fatalf("reading action-runtimes.json: %v", err)
	}
	var doc actionRuntimes
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing action-runtimes.json: %v", err)
	}
	if doc.SchemaVersion != 1 {
		t.Fatalf("unsupported action-runtimes schema %d", doc.SchemaVersion)
	}
	return doc
}

func TestActionRuntimeCurrency(t *testing.T) {
	doc := loadActionRuntimes(t)
	deprecated := map[string]string{}
	for _, d := range doc.DeprecatedRuntimes {
		if d.Runtime == "" || d.Owner == "" || d.Announced == "" || strings.TrimSpace(d.Justification) == "" {
			t.Errorf("deprecated runtime %q must declare runtime, owner, announced and justification", d.Runtime)
			continue
		}
		deprecated[d.Runtime] = d.Announced
	}

	referenced := referencedActions(t)
	if len(referenced) == 0 {
		t.Fatal("no action references were found — the extraction is broken, not the workflows")
	}

	var unrecorded []string
	for action, pinned := range referenced {
		rec, ok := doc.Runtimes[action]
		if !ok {
			unrecorded = append(unrecorded, action)
			continue
		}
		if rec.Ref != pinned {
			t.Errorf("%s: pinned at %s but recorded at %s — the record was not refreshed with the bump, so its runtime claim describes a different version",
				action, short(pinned), short(rec.Ref))
		}
		if announced, bad := deprecated[rec.Using]; bad {
			t.Errorf("%s runs on %s, deprecated since %s — the runner substitutes a newer runtime today, and every workflow using it fails when that substitution is withdrawn",
				action, rec.Using, announced)
		}
	}
	if len(unrecorded) > 0 {
		sort.Strings(unrecorded)
		t.Errorf("%d referenced action(s) have no runtime record, so their runtime is unchecked: %s — read `runs.using` at the pinned ref and add it to .github/action-runtimes.json",
			len(unrecorded), strings.Join(unrecorded, ", "))
	}

	// A record for an action no longer referenced is stale bookkeeping that
	// makes the manifest look more complete than it is.
	for action := range doc.Runtimes {
		if _, ok := referenced[action]; !ok {
			t.Errorf("%s has a runtime record but is not referenced by any workflow — remove the record", action)
		}
	}
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}
