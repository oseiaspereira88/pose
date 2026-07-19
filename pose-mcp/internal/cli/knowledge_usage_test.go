package cli

// Knowledge consumption behavior (spec pose-knowledge-consumption-traceability):
// stable citation validation, usage projection without TTL mutation, and
// sensitivity-filtered advisory suggestions.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func knowledgeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".pose/knowledge/2026-07-01-handoff-cache-design.md", `---
type: handoff
slug: cache-design
owner: @core
sensitivity: public-internal
created_at: 2026-07-01
last_reviewed_at: 2026-07-01
expires_at: 2026-09-01
---

# Cache design

Sharded cache invalidation strategy and eviction budget notes.
`)
	write(".pose/knowledge/2026-07-01-note-secret-topology.md", `---
type: note
slug: secret-topology
owner: @core
sensitivity: restricted
created_at: 2026-07-01
last_reviewed_at: 2026-07-01
expires_at: 2026-09-01
---

# Secret topology

Restricted deployment topology with cache invalidation details.
`)
	write(".pose/specs/consumer/spec.md", `---
slug: consumer
status: in-progress
---

## 3. Technical Plan

Reuses the eviction budget from knowledge:cache-design.
`)
	return root
}

func TestKnowledgeUsageProjection(t *testing.T) {
	root := knowledgeFixture(t)
	var out, errB bytes.Buffer
	if code := cmdKnowledgeUsage(root, &out, &errB); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
	s := out.String()
	if !strings.Contains(s, "cache-design owner:@core expires:2026-09-01 citations:1 cited_by:consumer") {
		t.Errorf("missing citation projection: %s", s)
	}
	if !strings.Contains(s, "secret-topology") || !strings.Contains(s, "citations:0") {
		t.Errorf("uncited artifact should appear with zero citations: %s", s)
	}
	if !strings.Contains(s, "TTL is never extended automatically") {
		t.Errorf("usage output must state the TTL invariant: %s", s)
	}
}

func TestKnowledgeRefValidation(t *testing.T) {
	root := knowledgeFixture(t)
	var errB bytes.Buffer
	if n := validateKnowledgeRefs(root, &errB); n != 0 {
		t.Fatalf("valid refs should pass, got %d: %s", n, errB.String())
	}
	spec := filepath.Join(root, ".pose", "specs", "consumer", "spec.md")
	raw, _ := os.ReadFile(spec)
	if err := os.WriteFile(spec, append(raw, []byte("\nAlso cites knowledge:ghost-artifact.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	errB.Reset()
	if n := validateKnowledgeRefs(root, &errB); n != 1 {
		t.Fatalf("dangling ref should fail once, got %d", n)
	}
	if !strings.Contains(errB.String(), "knowledge:ghost-artifact") || !strings.Contains(errB.String(), "consumer") {
		t.Errorf("diagnostic should name the ref and the citing spec: %s", errB.String())
	}
}

func TestKnowledgeSuggestFiltersRestrictedAndExplains(t *testing.T) {
	root := knowledgeFixture(t)
	var out, errB bytes.Buffer
	if code := cmdKnowledgeSuggest(root, []string{"cache", "invalidation", "eviction"}, &out, &errB); code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, errB.String())
	}
	s := out.String()
	if !strings.Contains(s, "cache-design") {
		t.Errorf("expected cache-design suggestion: %s", s)
	}
	if strings.Contains(s, "secret-topology") {
		t.Errorf("restricted artifact must never be suggested: %s", s)
	}
	if !strings.Contains(s, "restricted_filtered=1") {
		t.Errorf("filter count must be visible: %s", s)
	}
	if !strings.Contains(s, "rationale:shared-terms[") || !strings.Contains(s, "Confirm relevance before citing") {
		t.Errorf("suggestions must expose rationale and require confirmation: %s", s)
	}
}
