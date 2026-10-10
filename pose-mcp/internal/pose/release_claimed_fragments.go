package pose

// Release prepare guards fragments other specs claim (spec
// pose-release-prepare-guards-claimed-fragments).
//
// The v7.1.0 freeze moved three fragments from .pose/changelogs/unreleased/
// into the release archive while another spec still claimed them at their
// unreleased path. The fragment's own spec is attributed for the move; the
// other spec's claim then names a path no longer tracked, and CI's structural
// gate failed on main. Local checks passed because they read a stale index.

import (
	"fmt"
	"path"
	"sort"
)

// ForeignFragmentClaims lists every spec, other than a fragment's own, whose
// Artifacts claim that fragment at its unreleased path.
func (s Store) ForeignFragmentClaims(fragments []ReleaseFragment) ([]string, error) {
	if len(fragments) == 0 {
		return nil, nil
	}
	owners := map[string]string{}
	for _, f := range fragments {
		owners[path.Join(".pose/changelogs/unreleased", path.Base(f.Path))] = f.Spec
	}
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return nil, err
	}
	policy, err := LoadArtifactPolicy(s.Root)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, listed := range specs {
		// ListSpecs carries metadata only; the claims live in the body.
		full, err := s.GetSpec(listed.Slug)
		if err != nil || full == nil {
			continue
		}
		spec := *full
		claims, _, err := ParseArtifactClaims(spec, policy)
		if err != nil {
			continue
		}
		for _, claim := range claims {
			for _, p := range []string{claim.Path, claim.OldPath, claim.NewPath} {
				if owner, ok := owners[p]; ok && owner != spec.Slug {
					out = append(out, fmt.Sprintf("spec %s claims %s, the fragment of %s; the release moves it, so describe it in prose or claim it from the fragment's own spec", spec.Slug, p, owner))
				}
			}
		}
	}
	sort.Strings(out)
	return out, nil
}
