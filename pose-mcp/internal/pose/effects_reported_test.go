package pose

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHooksLiveWhereGitReadsThem(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	designDeltaGit(t, repo, "init", "-q")
	designDeltaGit(t, repo, "config", "user.name", "Fixture")
	designDeltaGit(t, repo, "config", "user.email", "fixture@example.invalid")
	designDeltaGit(t, repo, "commit", "-q", "--allow-empty", "-m", "base")
	if got, want := GitHooksDir(repo), filepath.Join(repo, ".git", "hooks"); got != want {
		t.Fatalf("a plain repository: %s, want %s", got, want)
	}
	linked := filepath.Join(base, "linked")
	designDeltaGit(t, repo, "worktree", "add", "-q", linked)
	if got, want := GitHooksDir(linked), filepath.Join(repo, ".git", "hooks"); got != want {
		t.Fatalf("a linked worktree: %s, want %s", got, want)
	}
	designDeltaGit(t, repo, "config", "core.hooksPath", "tools/hooks")
	if got, want := GitHooksDir(repo), filepath.Join(repo, "tools", "hooks"); got != want {
		t.Fatalf("core.hooksPath: %s, want %s", got, want)
	}
	absolute := filepath.Join(base, "shared-hooks")
	designDeltaGit(t, repo, "config", "core.hooksPath", absolute)
	if got := GitHooksDir(repo); got != absolute {
		t.Fatalf("an absolute core.hooksPath: %s", got)
	}
	bare := t.TempDir()
	if got, want := GitHooksDir(bare), filepath.Join(bare, ".git", "hooks"); got != want {
		t.Fatalf("no repository: %s, want %s", got, want)
	}
}

func TestAnUnsyncedPolicyStillWritesItsSecondDocument(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".pose", "policy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"review.json": `{"schema_version":1}`, "dor.json": `{"schema_version":1}`} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	docs, err := LoadPolicyDocsIn(NativeFiles(), root)
	if err != nil {
		t.Fatal(err)
	}
	docs.Review["marker"] = "one"
	docs.DoR["marker"] = "two"
	PublishSyncHook = func(kind, path string) error {
		if kind == "dir" && strings.HasSuffix(path, "review.json") {
			return errors.New("injected")
		}
		return nil
	}
	t.Cleanup(func() { PublishSyncHook = nil })
	if err := docs.Write(root, Store{Root: root}); !errors.Is(err, ErrPublishUnsynced) {
		t.Fatalf("the unsynced outcome was lost: %v", err)
	}
	if raw, _ := os.ReadFile(filepath.Join(dir, "dor.json")); !strings.Contains(string(raw), "two") {
		t.Fatalf("the second document was not written: %s", raw)
	}
}

func TestUnsyncedDelegationActivationStillChecksItsConfiguration(t *testing.T) {
	for _, syncFailure := range []int{1, 2} {
		t.Run(string(rune('0'+syncFailure)), func(t *testing.T) {
			s, docs, policy := adoptionFixture(t)
			entry, _ := LookupCatalogEntry("delegated-review")
			entry.Adopt(docs, "2026-11-02")
			priorFault, priorSync := PublishFault, PublishSyncHook
			t.Cleanup(func() { PublishFault, PublishSyncHook = priorFault, priorSync })
			changed, syncs := false, 0
			PublishFault = func(path string) error {
				if path == filepath.Join(s.Root, ".pose/policy/review.json") && !changed {
					changed = true
					policy.Routing = &DelegationRouting{Prefer: "any"}
					raw, err := CanonicalDelegationPolicy(policy)
					if err != nil {
						return err
					}
					return os.WriteFile(filepath.Join(s.Root, ".pose/policy/delegation.json"), raw, 0o600)
				}
				return nil
			}
			PublishSyncHook = func(kind, path string) error {
				if kind == "dir" && strings.HasSuffix(path, "review.json") {
					syncs++
					if syncs == syncFailure {
						return errors.New("injected sync failure")
					}
				}
				return nil
			}
			err := docs.Write(s.Root, s)
			if err == nil || errors.Is(err, ErrPublishUnsynced) || !strings.Contains(err.Error(), "left disabled") {
				t.Fatalf("configuration race read as a successful activation: %v", err)
			}
			current, err := s.GetReviewPolicy()
			if err != nil || current.DelegatedReview {
				t.Fatalf("racing activation remained enabled: %v", err)
			}
		})
	}
}
