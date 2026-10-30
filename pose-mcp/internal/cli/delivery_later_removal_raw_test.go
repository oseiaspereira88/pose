package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// gammaKept reports whether gamma's existence finding for internal/x.go
// remains in the delivery graph of the later-removal fixture.
func gammaKept(t *testing.T, root string) bool {
	t.Helper()
	graph, err := buildCurrentDeliveryGraph(root)
	if err != nil {
		t.Fatal(err)
	}
	_, kept := existenceFindings(graph)["gamma internal/x.go"]
	return kept
}

func TestLaterRemovalReviewRoundThree(t *testing.T) {
	// LR-R3-01: a replace ref that drops gamma's trailer, or a graft that
	// cuts gamma out of the history, changes no change set.
	t.Run("replace ref drops a trailer", func(t *testing.T) {
		root := laterRemovalFixture(t)
		gamma := artifactGit(t, root, "log", "--format=%H", "-n", "1", "--grep", "^POSE-Spec: gamma$")
		tree := artifactGit(t, root, "rev-parse", gamma+"^{tree}")
		parent := artifactGit(t, root, "rev-parse", gamma+"^")
		fake := artifactGit(t, root, "commit-tree", tree, "-p", parent, "-m", "gamma, without its trailer")
		artifactGit(t, root, "replace", gamma, fake)
		if !gammaKept(t, root) {
			t.Fatal("a replace ref hid gamma's change after the removal")
		}
	})
	// LR-R4-01: a graft that cuts a's later change out of the history, while
	// a's earlier change stays, changes no change set; the repository's
	// graft file and one the environment names alike.
	t.Run("graft cuts a later change out", func(t *testing.T) {
		for _, where := range []string{"repository", "environment"} {
			root, b, u := graftFixture(t)
			graft := filepath.Join(root, ".git", "info", "grafts")
			if where == "environment" {
				graft = filepath.Join(t.TempDir(), "grafts")
				t.Setenv("GIT_GRAFT_FILE", graft)
			}
			if err := os.MkdirAll(filepath.Dir(graft), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(graft, []byte(u+" "+b+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			graph, err := buildCurrentDeliveryGraph(root)
			if err != nil {
				t.Fatal(err)
			}
			if _, kept := existenceFindings(graph)["alpha internal/x.go"]; !kept {
				t.Fatalf("a graft in the %s hid alpha's change after the removal", where)
			}
		}
	})
	// LR-R3-02: a shallow repository refuses every removal.
	t.Run("shallow repository", func(t *testing.T) {
		root := laterRemovalFixture(t)
		first := artifactGit(t, root, "rev-list", "--max-parents=0", "HEAD")
		if err := os.WriteFile(filepath.Join(root, ".git", "shallow"), []byte(first+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, claims, sets, tracked, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
		if err != nil || !strings.Contains(gaps[""], "shallow") {
			t.Fatalf("a shallow repository: %v %v", gaps, err)
		}
		for _, removal := range artifactRemovals(root, claims, sets, tracked, gaps) {
			if len(removal.Supersedes) > 0 {
				t.Fatalf("a shallow repository satisfied a claim: %+v", removal)
			}
		}
	})
	// LR-R3-03: a change set names commits, not tags of them.
	t.Run("annotated tag", func(t *testing.T) {
		r := newRemovalRepo(t)
		r.write(removalPath, "1\n")
		a := r.commit("a creates")
		r.rm(removalPath)
		r.commit("b removes")
		r.git("tag", "-a", "-m", "b", "b-tag")
		tag := r.git("rev-parse", "b-tag")
		ok, why := superseded(artifactRemovals(r.root, removalClaims(removalPath), removalSets([]string{a}, []string{tag}), nil, nil))
		if ok || !strings.Contains(why, "not a commit") {
			t.Fatalf("a tag as a deleting commit: %v %q", ok, why)
		}
	})
	// LR-R3-04: the rename stream is read whole and strictly.
	t.Run("rename stream", func(t *testing.T) {
		for _, bad := range []string{"Dgarbage\x00other\x00", "D\x00other", "D\x00a\x00D", "\x00R100\x00a\x00b\x00", "D\x00\x00", "R100\x00a\x00"} {
			if _, err := renamedAwayPaths([]byte(bad)); err == nil {
				t.Errorf("malformed stream %q was accepted", bad)
			}
		}
		set, err := renamedAwayPaths([]byte("D\x00Rx\x00R100\x00a\x00b\x00C075\x00c\x00d\x00"))
		if err != nil || !set["a"] || set["c"] || set["Rx"] || len(set) != 1 {
			t.Fatalf("a valid stream: %v %v", set, err)
		}
		if set, err := renamedAwayPaths(nil); err != nil || len(set) != 0 {
			t.Fatalf("an empty diff: %v %v", set, err)
		}
	})
}

// graftFixture is alpha creating internal/x.go (A1), beta removing it (B),
// alpha adding it back (A2) and an unattributed commit deleting it (U); it
// returns the root, B and U.
func graftFixture(t *testing.T) (string, string, string) {
	t.Helper()
	r := newRemovalRepo(t)
	spec := func(slug, artifacts string) {
		r.write(".pose/specs/"+slug+"/spec.md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-10-30\n---\n\n# Spec: "+slug+"\n\n## 3. Technical Plan\n\n### Artifacts\n"+artifacts+"\n## 4. Tasks\nwork\n")
	}
	r.write(".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error","existence":"error"}}`)
	r.write("README.md", "fixture\n")
	r.commit("baseline")
	spec("alpha", "- created: internal/x.go\n")
	r.write(removalPath, "package internal\n")
	r.commit("alpha creates\n\nPOSE-Spec: alpha")
	spec("beta", "- removed: internal/x.go\n")
	r.rm(removalPath)
	b := r.commit("beta removes\n\nPOSE-Spec: beta")
	r.write(removalPath, "package internal\n// back\n")
	r.commit("alpha adds it back\n\nPOSE-Spec: alpha")
	r.rm(removalPath)
	u := r.commit("an unattributed deletion")
	graph, err := buildCurrentDeliveryGraph(r.root)
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := existenceFindings(graph)["alpha internal/x.go"]; !kept {
		t.Fatal("without a graft the fixture does not keep alpha's finding")
	}
	return r.root, b, u
}

// recordedFixture is alpha creating internal/x.go (A1, trailer), beta
// removing it (B, trailer), an untagged commit adding it back (A2) that only
// a recorded report attributes to alpha, and an unattributed deletion; it
// returns the root and A2.
func recordedFixture(t *testing.T) (string, string) {
	t.Helper()
	r := newRemovalRepo(t)
	spec := func(slug, artifacts string) {
		r.write(".pose/specs/"+slug+"/spec.md", "---\nslug: "+slug+"\nstatus: done\ncreated_at: 2026-10-30\n---\n\n# Spec: "+slug+"\n\n## 3. Technical Plan\n\n### Artifacts\n"+artifacts+"\n## 4. Tasks\nwork\n")
	}
	r.write(".pose/policy/artifacts.json", `{"schema_version":1,"enabled":true,"adopted_at":"2026-08-03","governed_roots":["internal"],"severities":{"action-mismatch":"error","undeclared":"error","existence":"error"}}`)
	r.write("README.md", "fixture\n")
	r.commit("baseline")
	spec("alpha", "- created: internal/x.go\n")
	r.write(removalPath, "package internal\n")
	r.commit("alpha creates\n\nPOSE-Spec: alpha")
	spec("beta", "- removed: internal/x.go\n")
	r.rm(removalPath)
	r.commit("beta removes\n\nPOSE-Spec: beta")
	r.write(removalPath, "package internal\n// back\n")
	a2 := r.commit("alpha adds it back, untagged")
	r.rm(removalPath)
	r.commit("an unattributed deletion")
	return r.root, a2
}

// writeRecorded writes the report history of alpha: the given lines, then
// a record of the change set of A2 whose paths are padded to size bytes.
func writeRecorded(t *testing.T, root, a2 string, size int, before ...string) {
	t.Helper()
	set := posemodel.ChangeSet{ID: "cs-recorded-a2", Spec: "alpha", Selector: "range", Commits: []string{a2}, Paths: []posemodel.ObservedPath{{Action: "created", Path: removalPath}}}
	for n := 0; len(set.Paths)*48 < size; n++ {
		set.Paths = append(set.Paths, posemodel.ObservedPath{Action: "created", Path: fmt.Sprintf("other/a-long-generated-file-name-%06d.txt", n)})
	}
	record, err := json.Marshal(map[string]any{"task": "alpha", "change_set": set})
	if err != nil {
		t.Fatal(err)
	}
	body := strings.Join(append(before, string(record)), "\n") + "\n"
	writeArtifactTestFile(t, root, ".pose/reports/history/report-alpha.jsonl", body)
}

func TestLaterRemovalReviewRoundFive(t *testing.T) {
	alphaKept := func(t *testing.T, root string) bool {
		t.Helper()
		graph, err := buildCurrentDeliveryGraph(root)
		if err != nil {
			t.Fatal(err)
		}
		_, kept := existenceFindings(graph)["alpha internal/x.go"]
		return kept
	}
	// LR-R5-01: a recorded change set wider than a scanner's default token
	// is read, so alpha's later change keeps its finding.
	t.Run("wide recorded change set", func(t *testing.T) {
		root, a2 := recordedFixture(t)
		writeRecorded(t, root, a2, 160*1024)
		_, _, sets, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
		recorded := false
		for _, set := range sets {
			recorded = recorded || set.ID == "cs-recorded-a2"
		}
		if err != nil || gaps[""] != "" || !recorded {
			t.Fatalf("a wide recorded change set was not read: %v %v %v", recorded, gaps, err)
		}
		if !alphaKept(t, root) {
			t.Fatal("a wide recorded change set was dropped and the removal accepted")
		}
	})
	// What the loader cannot read refuses every removal, with its cause.
	t.Run("unreadable records", func(t *testing.T) {
		limit := recordedChangeSetLineLimit
		t.Cleanup(func() { recordedChangeSetLineLimit = limit })
		for name, setup := range map[string]func(root, a2 string){
			"over the limit": func(root, a2 string) {
				recordedChangeSetLineLimit = 64 * 1024
				writeRecorded(t, root, a2, 160*1024)
			},
			"malformed line": func(root, a2 string) {
				writeRecorded(t, root, a2, 0, `{"task":"alpha"}`, `{"change_set": {"id": `)
			},
			"unreadable file": func(root, a2 string) {
				if err := os.MkdirAll(filepath.Join(root, ".pose", "reports", "history", "dir.jsonl"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		} {
			recordedChangeSetLineLimit = limit
			root, a2 := recordedFixture(t)
			setup(root, a2)
			_, claims, sets, tracked, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
			if err != nil || !strings.Contains(gaps[""], "recorded change sets could not be read") {
				t.Fatalf("%s: %v %v", name, gaps, err)
			}
			for _, removal := range artifactRemovals(root, claims, sets, tracked, gaps) {
				if len(removal.Supersedes) > 0 || !strings.Contains(removal.Refusals["alpha"], "recorded change sets") {
					t.Fatalf("%s: a removal was judged on part of the records: %+v", name, removal)
				}
			}
			if !alphaKept(t, root) {
				t.Fatalf("%s: alpha's finding was dropped", name)
			}
		}
	})
	// A record without a change set is an ordinary report record.
	t.Run("records without a change set", func(t *testing.T) {
		root, a2 := recordedFixture(t)
		writeRecorded(t, root, a2, 0, `{"task":"alpha","sequence":1}`, "")
		_, _, _, _, _, gaps, err := collectArtifactGraphInputsWithGapsIn(posemodel.NativeFiles(), root)
		if err != nil || gaps[""] != "" {
			t.Fatalf("ordinary records made a gap: %v %v", gaps, err)
		}
		if !alphaKept(t, root) {
			t.Fatal("the recorded later change was not judged")
		}
	})
}
