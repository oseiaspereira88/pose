package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Spec pose-release-archival-attested-by-the-ledger.

const fragmentPending = ".pose/changelogs/unreleased/alpha.md"
const fragmentArchived = ".pose/changelogs/v1.2.0/alpha.md"
const fragmentManifest = ".pose/releases/v1.2.0/manifest.json"

func archivedAlpha() ArchivedFragment {
	return ArchivedFragment{Version: "v1.2.0", Spec: "alpha", Pending: fragmentPending, Archived: fragmentArchived, Intact: true, Manifest: fragmentManifest}
}

func findingsWith(graph DeliveryIntegrityGraph, code string) []DeliveryIntegrityFinding {
	found := []DeliveryIntegrityFinding{}
	for _, f := range graph.Findings {
		if f.Code == code {
			found = append(found, f)
		}
	}
	return found
}

// The change set that created the fragment, as the spec's own commits recorded
// it before the release moved the file.
func alphaCreatedTheFragment() []ChangeSet {
	return []ChangeSet{{ID: "cs-alpha", Spec: "alpha", Paths: []ObservedPath{{Action: "created", Path: fragmentPending}}}}
}

func TestAnArchivedFragmentClaimResolvesThroughTheRelease(t *testing.T) {
	claims := []ArtifactClaim{{Spec: "alpha", Action: "created", Path: fragmentPending}}
	tracked := []string{fragmentArchived, fragmentManifest}

	without := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, ArtifactPolicy{})
	if len(findingsWith(without, "existence")) != 1 {
		t.Fatalf("without the ledger the moved fragment must fail existence, got %+v", without.Findings)
	}

	graph := BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, ArtifactPolicy{}, []ArchivedFragment{archivedAlpha()})
	if len(graph.Findings) != 0 {
		t.Fatalf("an archival the release attests must resolve the claim, got %+v", graph.Findings)
	}
	if len(graph.Archivals) != 1 || graph.Archivals[0] != archivedAlpha() {
		t.Fatalf("the archival that resolved the claim must be listed, got %+v", graph.Archivals)
	}
	want := map[string]bool{
		"artifact:" + fragmentPending + "|release:v1.2.0|archived-by": false,
		"release:v1.2.0|artifact:" + fragmentArchived + "|archives":   false,
	}
	for _, e := range graph.Edges {
		key := e.From + "|" + e.To + "|" + e.Type
		if _, ok := want[key]; ok {
			want[key] = true
		}
		if e.Type == "changes" && strings.Contains(e.To, fragmentArchived) {
			t.Fatal("the archival must never be recorded as a change the spec made")
		}
	}
	for key, seen := range want {
		if !seen {
			t.Errorf("missing edge %s", key)
		}
	}
	if got := graph.Reverse[fragmentArchived]; len(got) != 1 || got[0] != "alpha" {
		t.Errorf("the archived path must count as claimed by the spec, got %v", got)
	}
}

// A claim an earlier release rewrote into the rename is satisfied by the
// manifest of the version it names, with no spec edited.
func TestARenameAnEarlierReleaseWroteResolvesThroughItsManifest(t *testing.T) {
	claims := []ArtifactClaim{{Spec: "alpha", Action: "renamed", OldPath: fragmentPending, NewPath: fragmentArchived}}
	tracked := []string{fragmentArchived, fragmentManifest}

	without := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, ArtifactPolicy{})
	if len(findingsWith(without, "action-mismatch")) != 1 {
		t.Fatalf("without the ledger the rewritten claim must fail the action check, got %+v", without.Findings)
	}
	// Governing .pose/changelogs makes the spec's own creation of the fragment
	// visible to the undeclared check; it is the first half of the rename.
	governed := ArtifactPolicy{Enabled: true, GovernedRoots: []string{".pose/changelogs"}, Severities: map[string]string{"undeclared": "error"}}
	graph := BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, governed, []ArchivedFragment{archivedAlpha()})
	if len(graph.Findings) != 0 {
		t.Fatalf("the manifest of the named version must satisfy the rename, got %+v", graph.Findings)
	}
	if without := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, governed); len(findingsWith(without, "undeclared")) != 1 {
		t.Fatalf("without the ledger the spec's creation of the fragment reads as undeclared, got %+v", without.Findings)
	}
	other := archivedAlpha()
	other.Version, other.Archived = "v1.3.0", ".pose/changelogs/v1.3.0/alpha.md"
	graph = BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), tracked, ArtifactPolicy{}, []ArchivedFragment{other})
	if f := findingsWith(graph, "action-mismatch"); len(f) != 1 || !strings.Contains(f[0].Message, "no release manifest archives "+fragmentPending+" at "+fragmentArchived) {
		t.Fatalf("a manifest of another version must not satisfy the rename, got %+v", graph.Findings)
	}
}

// Everything the ledger does not attest stays a finding, and the message says
// what the manifests showed.
func TestOnlyAnArchivalTheReleaseAttestsResolvesAClaim(t *testing.T) {
	claims := []ArtifactClaim{{Spec: "alpha", Action: "created", Path: fragmentPending}}
	tracked := []string{fragmentArchived, fragmentManifest}
	cases := map[string]struct {
		archived []ArchivedFragment
		tracked  []string
		note     string
	}{
		"another spec's fragment": {[]ArchivedFragment{{Version: "v1.2.0", Spec: "beta", Pending: fragmentPending, Archived: fragmentArchived, Intact: true, Manifest: fragmentManifest}}, tracked, "release v1.2.0 archived it for spec beta"},
		"no manifest lists it":    {nil, tracked, "no release manifest archives " + fragmentPending},
		"archived content edited": {[]ArchivedFragment{{Version: "v1.2.0", Spec: "alpha", Pending: fragmentPending, Archived: fragmentArchived, Manifest: fragmentManifest}}, tracked, "no longer has the digest the manifest froze"},
		"archived file untracked": {[]ArchivedFragment{archivedAlpha()}, []string{"README.md", fragmentManifest}, "archived it at " + fragmentArchived + ", which is not tracked"},
		// Review of pose#106: a manifest outside the selected head attests nothing.
		"manifest untracked": {[]ArchivedFragment{archivedAlpha()}, []string{fragmentArchived}, "attests it in " + fragmentManifest + ", which is not tracked"},
		// Review of pose#106: a fragment belongs to exactly one release
		// (pose-release-lifecycle-closure R7); picking either would hide it.
		"two releases for one spec": {[]ArchivedFragment{archivedAlpha(), {Version: "v1.3.0", Spec: "alpha", Pending: fragmentPending, Archived: ".pose/changelogs/v1.3.0/alpha.md", Intact: true, Manifest: ".pose/releases/v1.3.0/manifest.json"}}, []string{fragmentArchived, fragmentManifest, ".pose/changelogs/v1.3.0/alpha.md", ".pose/releases/v1.3.0/manifest.json"}, "releases v1.2.0, v1.3.0 each archive it for spec alpha"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			graph := BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, claims, alphaCreatedTheFragment(), c.tracked, ArtifactPolicy{}, c.archived)
			f := findingsWith(graph, "existence")
			if len(f) != 1 || !strings.HasPrefix(f[0].Message, "declared current artifact is not tracked at the selected head; ") || !strings.Contains(f[0].Message, c.note) {
				t.Fatalf("want an existence finding naming %q, got %+v", c.note, graph.Findings)
			}
			if len(graph.Archivals) != 0 {
				t.Fatalf("nothing resolved, so nothing may be listed: %+v", graph.Archivals)
			}
		})
	}

	// A path outside the pending fragments is reported exactly as before.
	other := []ArtifactClaim{{Spec: "alpha", Action: "created", Path: "internal/gone.go"}}
	graph := BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, other, nil, tracked, ArtifactPolicy{}, []ArchivedFragment{archivedAlpha()})
	if f := findingsWith(graph, "existence"); len(f) != 1 || f[0].Message != "declared current artifact is not tracked at the selected head" {
		t.Fatalf("a non-fragment path must keep its finding unchanged, got %+v", graph.Findings)
	}
}

// A repository with no releases builds the same graph as before, digests
// included, so upgrading does not churn every instance's index.
func TestNoReleasesBuildsTheSameGraphAsBefore(t *testing.T) {
	claims := []ArtifactClaim{{Spec: "alpha", Action: "modified", Path: "internal/core.go"}}
	sets := []ChangeSet{{ID: "cs-1", Spec: "alpha", Paths: []ObservedPath{{Action: "modified", Path: "internal/core.go"}}}}
	before := BuildDeliveryIntegrity([]Spec{{Slug: "alpha"}}, claims, sets, []string{"internal/core.go"}, ArtifactPolicy{})
	for _, archived := range [][]ArchivedFragment{nil, {}} {
		after := BuildDeliveryIntegrityWithReleases([]Spec{{Slug: "alpha"}}, claims, sets, []string{"internal/core.go"}, ArtifactPolicy{}, archived)
		a, _ := json.Marshal(before)
		b, _ := json.Marshal(after)
		if string(a) != string(b) {
			t.Fatalf("no archivals must build the identical graph:\n%s\n%s", a, b)
		}
	}
}

func TestLoadArchivedFragmentsReadsWhatEachManifestAttests(t *testing.T) {
	root := t.TempDir()
	body := "---\nspec: alpha\ncategory: fixed\nbreaking: false\n---\n\nFixes <alpha> & more.\n"
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(fragmentArchived, body)
	manifest, _ := json.Marshal(ReleaseManifest{SchemaVersion: 1, Version: "v1.2.0", Specs: []string{"alpha"}, Fragments: []ReleaseFragment{
		{Spec: "alpha", Path: "alpha.md", Digest: ReleaseDigest(body)},
		{Spec: "escape", Path: "../../etc/passwd", Digest: "sha256:x"},
	}})
	write(".pose/releases/v1.2.0/manifest.json", string(manifest))
	write(".pose/releases/not-a-version/manifest.json", string(manifest))

	got := LoadArchivedFragments(root)
	if len(got) != 1 || !strings.HasPrefix(got[0].ManifestDigest, "sha256:") {
		t.Fatalf("want exactly the attested fragment with its manifest digest, got %+v", got)
	}
	got[0].ManifestDigest = ""
	if got[0] != archivedAlpha() {
		t.Fatalf("want exactly the attested fragment, intact, got %+v", got)
	}
	write(fragmentArchived, body+"edited after the cut\n")
	if got := LoadArchivedFragments(root); len(got) != 1 || got[0].Intact {
		t.Fatalf("an archived fragment edited after the cut is not the one attested, got %+v", got)
	}
	if got := LoadArchivedFragments(t.TempDir()); len(got) != 0 {
		t.Fatalf("a repository with no releases attests nothing, got %+v", got)
	}
}
