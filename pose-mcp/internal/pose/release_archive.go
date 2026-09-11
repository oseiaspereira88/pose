package pose

// Release archival as a witness (spec pose-release-archival-attested-by-the-ledger).
//
// A spec declares the changelog fragment it adds under
// .pose/changelogs/unreleased/. `pose release prepare` then moves it under
// .pose/changelogs/<version>/, in a commit no change set of the spec contains.
// After the cut, no claim the spec can hold passes both existence and action:
// the pending path is gone, and a rename was never performed by the spec.
//
// Prepare used to rewrite the claim into that rename. It failed the action
// check on every released spec, and it changed the Technical Plan — the
// semantic section a sealed review bundle digests — of any spec closed before
// the cut. The release manifest already records which fragment it archived, for
// which spec, with which digest; it is read here as a witness of its own,
// distinct from declaration and from Git observation.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ArchivedFragment is what one release manifest attests about one fragment it
// consumed.
type ArchivedFragment struct {
	Version string `json:"version"`
	Spec    string `json:"spec"`
	// Pending is where the spec declared the fragment; Archived is where the
	// release moved it.
	Pending  string `json:"pending"`
	Archived string `json:"archived"`
	// Intact reports whether the archived file still has the digest the
	// manifest froze. A changed fragment is not the one the release attests.
	Intact bool `json:"intact"`
}

const pendingFragmentDir = ".pose/changelogs/unreleased/"

// LoadArchivedFragments reads every release manifest under root and returns
// what each attests, sorted by archived path. A manifest that cannot be read
// attests nothing; `pose release check` is where it is reported.
func LoadArchivedFragments(root string) []ArchivedFragment {
	entries, err := os.ReadDir(filepath.Join(root, ".pose", "releases"))
	if err != nil {
		return nil
	}
	archived := []ArchivedFragment{}
	for _, entry := range entries {
		if !entry.IsDir() || ValidateReleaseVersion(entry.Name()) != nil {
			continue
		}
		manifest, err := LoadReleaseManifest(root, entry.Name())
		if err != nil {
			continue
		}
		for _, fragment := range manifest.Fragments {
			if fragment.Spec == "" || fragment.Path == "" || strings.ContainsAny(fragment.Path, `/\`) {
				continue
			}
			if _, err := validateArtifactPathSyntax(fragment.Path); err != nil {
				continue
			}
			at := ".pose/changelogs/" + entry.Name() + "/" + fragment.Path
			raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(at)))
			archived = append(archived, ArchivedFragment{
				Version:  entry.Name(),
				Spec:     fragment.Spec,
				Pending:  pendingFragmentDir + fragment.Path,
				Archived: at,
				Intact:   err == nil && ReleaseDigest(string(raw)) == fragment.Digest,
			})
		}
	}
	sort.Slice(archived, func(i, j int) bool { return archived[i].Archived < archived[j].Archived })
	return archived
}

// archivalLedger answers, for one claim, whether a release attests the
// archival that claim depends on — and, when it does not, what the manifests
// showed instead.
type archivalLedger struct {
	byPending map[string][]ArchivedFragment
	tracked   func(string) bool
}

func newArchivalLedger(archived []ArchivedFragment, tracked func(string) bool) archivalLedger {
	ledger := archivalLedger{byPending: map[string][]ArchivedFragment{}, tracked: tracked}
	for _, a := range archived {
		ledger.byPending[a.Pending] = append(ledger.byPending[a.Pending], a)
	}
	return ledger
}

// resolve returns the archival attesting that spec's fragment at pending went
// to archivedAt (any version when archivedAt is ""), or a note on why none
// does. ok is false with an empty note when pending is not a fragment path at
// all, so the caller reports exactly what it reported before.
func (l archivalLedger) resolve(spec, pending, archivedAt string) (ArchivedFragment, string, bool) {
	if !strings.HasPrefix(pending, pendingFragmentDir) {
		return ArchivedFragment{}, "", false
	}
	candidates := l.byPending[pending]
	if len(candidates) == 0 {
		return ArchivedFragment{}, "no release manifest archives " + pending, false
	}
	notes := []string{}
	for _, a := range candidates {
		switch {
		case archivedAt != "" && a.Archived != archivedAt:
			continue
		case a.Spec != spec:
			notes = append(notes, "release "+a.Version+" archived it for spec "+a.Spec)
		case !a.Intact:
			notes = append(notes, "release "+a.Version+" archived it at "+a.Archived+", which no longer has the digest the manifest froze")
		case !l.tracked(a.Archived):
			notes = append(notes, "release "+a.Version+" archived it at "+a.Archived+", which is not tracked")
		default:
			return a, "", true
		}
	}
	if len(notes) == 0 {
		return ArchivedFragment{}, "no release manifest archives " + pending + " at " + archivedAt, false
	}
	return ArchivedFragment{}, strings.Join(notes, "; "), false
}

// explainsCreation reports whether an observed creation of a pending fragment
// is the first half of a rename an earlier release wrote into the spec and
// attested: the spec created the fragment, the release moved it. Without it, a
// repository governing .pose/changelogs sees the spec's own creation as
// undeclared, because its claim now names the rename instead.
func (l archivalLedger) explainsCreation(spec string, declared []ArtifactClaim, observed ObservedPath) bool {
	if observed.Action != "created" && observed.Action != "modified" {
		return false
	}
	for _, claim := range declared {
		if claim.Action == "renamed" && claim.OldPath == observed.Path {
			if _, _, ok := l.resolve(spec, claim.OldPath, claim.NewPath); ok {
				return true
			}
		}
	}
	return false
}
