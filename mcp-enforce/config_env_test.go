package mcpenforce

// ConfigEnvSuffixes exists so the configuration surface can be enumerated by a
// consumer (spec pose-composition-contract). A hand-maintained list would drift
// from what ConfigFromEnv actually reads, which is the defect that spec is
// closing one layer up — so two properties are pinned here.
//
// 1. Every declared suffix is live: setting it changes the resulting config.
//    Catches a name left behind after the read was removed.
// 2. The declaration matches the reads in the source: every `prefix + "X"` in
//    extract.go appears in the list, and nothing in the list is absent from it.
//    Catches a read added without declaring it.
//
// Property 2 parses this package's own source, which is legitimate here in a
// way it would not be across a repository boundary: the file is right there and
// the alternative is trusting a comment.

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestConfigEnvSuffixesAreLive(t *testing.T) {
	const prefix = "TESTCFG_"
	for _, suffix := range ConfigEnvSuffixes {
		t.Run(suffix, func(t *testing.T) {
			// PolicyConfig carries a func field and is not comparable, so the
			// settable fields are compared explicitly.
			fingerprint := func(c PolicyConfig) string {
				return fmt.Sprintf("%s|%s|%s|%t|%t|%t", c.OPAURL, c.OPAPath, c.Timeout,
					c.RequirePrincipal, c.RequireIdentity, c.RequireProjectScope)
			}
			base := fingerprint(ConfigFromEnv(prefix, "default/path"))
			value := "1"
			if strings.HasPrefix(suffix, "OPA_") {
				value = "changed"
			}
			if suffix == "OPA_TIMEOUT" {
				value = "7"
			}
			t.Setenv(prefix+suffix, value)
			if fingerprint(ConfigFromEnv(prefix, "default/path")) == base {
				t.Errorf("%s%s is declared but setting it changes nothing — either the read was removed or the name is wrong", prefix, suffix)
			}
		})
	}
}

func TestConfigEnvSuffixesMatchTheSource(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(".", "extract.go"))
	if err != nil {
		t.Fatalf("reading extract.go: %v", err)
	}
	readRe := regexp.MustCompile(`os\.Getenv\(prefix \+ "([A-Z_]+)"\)`)
	found := map[string]bool{}
	for _, m := range readRe.FindAllStringSubmatch(string(raw), -1) {
		found[m[1]] = true
	}
	if len(found) == 0 {
		t.Fatal("no prefixed environment reads found in extract.go — the extraction is broken, not the source")
	}

	declared := map[string]bool{}
	for _, s := range ConfigEnvSuffixes {
		declared[s] = true
	}

	var undeclared, unread []string
	for name := range found {
		if !declared[name] {
			undeclared = append(undeclared, name)
		}
	}
	for name := range declared {
		if !found[name] {
			unread = append(unread, name)
		}
	}
	sort.Strings(undeclared)
	sort.Strings(unread)

	if len(undeclared) > 0 {
		t.Errorf("ConfigFromEnv reads %v without declaring them in ConfigEnvSuffixes — a consumer composing this service cannot discover them", undeclared)
	}
	if len(unread) > 0 {
		t.Errorf("ConfigEnvSuffixes declares %v that ConfigFromEnv does not read — the published surface would claim options that do nothing", unread)
	}
}
