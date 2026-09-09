// changelog.json and dor.json are read through types of their own
// (spec pose-changelog-and-dor-policy-types).
//
// `categories` shipped in changelog.json from the start and nothing read it:
// the valid set was written out three times instead — the fragment loader,
// `pose check`, and implicitly the notes renderer. A file that promises a
// setting it does not have is worse than one that promises nothing, and three
// copies of a list is how they come to disagree.
//
// dor.json is the mirror image: it shipped none of the keys `readiness.go`
// looked for, so the gate read as unadopted on a fresh install and nothing said
// whether that was the intent.

package pose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePolicyFixture(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, ".pose", "policy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The setting has to reach the fragment loader, which is the gate an author
// meets first.
func TestChangelogCategoriesGovernWhatAFragmentMayDeclare(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "changelog.json",
		`{"schema_version":1,"adopted_at":"2026-08-03","categories":["added","fixed"]}`)
	policy := LoadChangelogPolicy(root)

	dir := filepath.Join(root, ".pose", "changelogs", "unreleased")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "alpha.md"),
		[]byte("---\nspec: alpha\ncategory: security\nbreaking: false\n---\n\nBody.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadReleaseFragments(dir, policy)
	if err == nil {
		t.Fatal("a category the policy does not list was accepted")
	}
	// The message has to name what is accepted, from the policy — not from a
	// list written beside it.
	if !strings.Contains(err.Error(), "added|fixed") {
		t.Errorf("the refusal does not quote the configured categories: %v", err)
	}
	if strings.Contains(err.Error(), "security") && strings.Contains(err.Error(), "want added|changed") {
		t.Errorf("the refusal quotes a hard-coded list: %v", err)
	}
}

// An absent or empty policy keeps the six defaults, so an instance that never
// touches the file behaves exactly as it did.
func TestChangelogCategoriesFallBackToTheDefaults(t *testing.T) {
	root := t.TempDir()
	if got := LoadChangelogPolicy(root).ValidCategories(); len(got) != 6 || !got["security"] {
		t.Errorf("an absent policy did not fall back to the six defaults: %v", got)
	}
	writePolicyFixture(t, root, "changelog.json", `{"schema_version":1,"categories":[]}`)
	if got := LoadChangelogPolicy(root).ValidCategories(); len(got) != 6 {
		t.Errorf("an empty list did not fall back: %v", got)
	}
}

// The notes keep their presentation order, which is not the order anyone writes
// the list in. A category the policy adds is rendered after the known ones
// rather than dropped, under a heading of its own.
func TestRenderOrderIsPresentationAndStillCarriesAnAddedCategory(t *testing.T) {
	policy := ChangelogPolicy{Categories: append(DefaultChangelogCategories(), "performance")}
	order := policy.RenderOrder()
	if len(order) != 7 || order[0] != "security" || order[len(order)-1] != "performance" {
		t.Fatalf("render order = %v", order)
	}
	notes := RenderReleaseNotes("v9.9.9", []ReleaseFragment{
		{Spec: "alpha", Category: "performance", Body: "Faster."},
		{Spec: "beta", Category: "fixed", Body: "Fixed."},
	}, policy)
	if !strings.Contains(notes, "## Performance") {
		t.Errorf("a configured category got no heading:\n%s", notes)
	}
	if strings.Index(notes, "## Fixed") > strings.Index(notes, "## Performance") {
		t.Errorf("the added category was rendered before the known ones:\n%s", notes)
	}
}

// The Definition of Ready is opt-in, and the shipped policy now says so with an
// explicit empty date rather than by omitting the key.
func TestDoRIsUnadoptedUntilADateIsSet(t *testing.T) {
	root := t.TempDir()
	if LoadDoRPolicy(root).AppliesTo("2026-09-09") {
		t.Error("an absent policy adopted the gate")
	}
	writePolicyFixture(t, root, "dor.json", `{"schemaVersion":1,"adopted_at":"","defaultTaskType":"feature"}`)
	if LoadDoRPolicy(root).AppliesTo("2026-09-09") {
		t.Error("an explicitly empty date adopted the gate")
	}
	writePolicyFixture(t, root, "dor.json", `{"schemaVersion":1,"adopted_at":"2026-09-01","defaultTaskType":"feature"}`)
	policy := LoadDoRPolicy(root)
	if !policy.AppliesTo("2026-09-09") {
		t.Error("a spec created after adoption escaped the gate")
	}
	if policy.AppliesTo("2026-08-31") {
		t.Error("a spec created before adoption was held to the gate")
	}
	if policy.AppliesTo("") {
		t.Error("a spec with no creation date was held to the gate")
	}
}

// Both halves of dor.json come through one type. Reading it with two anonymous
// structs is what let `adopted_at` be looked for in a file that never had it.
func TestDoRPolicyCarriesBothHalvesOfTheFile(t *testing.T) {
	root := t.TempDir()
	writePolicyFixture(t, root, "dor.json",
		`{"schemaVersion":1,"adopted_at":"2026-09-01","defaultTaskType":"bugfix","taskTypes":{"bugfix":["Intent"]}}`)
	policy := LoadDoRPolicy(root)
	if policy.DefaultTaskType != "bugfix" || len(policy.TaskTypes["bugfix"]) != 1 || policy.AdoptedAt != "2026-09-01" {
		t.Errorf("the type does not describe the whole file: %+v", policy)
	}
}
