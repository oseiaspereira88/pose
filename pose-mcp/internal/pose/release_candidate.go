package pose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Spec pose-release-manifest-tag-integrity: a candidate (a cut that has no tag yet) is judged by
// one function, shared by `pose release check`, `pose release status` and the structural checker,
// so none of them can call sound what another refuses.

// EngineReleaseBase is the running engine's release base, set by the binary from internal/version
// (this package does not import it). It is the version evidence inside the engine repository.
var EngineReleaseBase = ""

// ReleaseVersionEvidence selects the authoritative version a cut is compared with: the file the
// policy declares, or, only inside the engine repository, the compiled engine version.
func ReleaseVersionEvidence(root string, policy ReleasePolicy) (map[string]string, error) {
	if policy.VersionSource != nil {
		value, err := ReadReleaseVersionSource(root, *policy.VersionSource)
		if err != nil {
			return nil, err
		}
		return map[string]string{"source": filepath.ToSlash(filepath.Clean(policy.VersionSource.Path)), "kind": policy.VersionSource.Kind, "value": value}, nil
	}
	if !ProjectHasEngineVersionFile(root) {
		return nil, fmt.Errorf("release policy declares no version_source: the engine version is authoritative only inside the engine repository; set version_source in .pose/policy/release.json to the file that holds this project's version")
	}
	return map[string]string{"source": EngineVersionFile, "value": "v" + EngineReleaseBase}, nil
}

// LoadChangelogPolicyStrict reads the changelog policy for a judgment: no file is the defaults, a
// file that cannot be read or decoded, or of a schema this engine does not read, is an error.
func LoadChangelogPolicyStrict(root string) (ChangelogPolicy, error) {
	path := filepath.Join(root, ".pose", "policy", "changelog.json")
	absent, absentErr := ProvenAbsent(path)
	if absentErr != nil {
		return ChangelogPolicy{}, absentErr
	}
	if absent {
		return LoadChangelogPolicy(root), nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return ChangelogPolicy{}, err
	}
	return ParseChangelogPolicy(raw)
}

// ParseChangelogPolicy decodes a changelog policy and holds it to its contract.
func ParseChangelogPolicy(raw []byte) (ChangelogPolicy, error) {
	policy := ChangelogPolicy{SchemaVersion: 1}
	if err := json.Unmarshal(raw, &policy); err != nil {
		return policy, fmt.Errorf("invalid changelog policy: %w", err)
	}
	if policy.SchemaVersion != 1 {
		return policy, fmt.Errorf("the changelog policy schema_version %d is not supported by this engine", policy.SchemaVersion)
	}
	if len(policy.Categories) == 0 {
		policy.Categories = DefaultChangelogCategories()
	}
	return policy, nil
}

// notesCompatibilitySection drops the "## Compatibility" section of release notes. That section is
// derived from the contract registry of the engine that renders it, so an engine newer than the
// one that cut a release renders it differently: it is not part of what the fragments freeze.
func notesWithoutCompatibility(notes string) string {
	const heading = "## Compatibility\n\n"
	i := strings.Index(notes, heading)
	if i < 0 {
		return notes
	}
	rest := notes[i+len(heading):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		return notes[:i] + rest[j+1:]
	}
	return notes[:i]
}

// NotesAreTheRendering reports whether notes are the rendering of the fragments under the changelog
// policy, and, when they differ only in the contract-registry section, that the difference is a
// limit and not a contradiction.
func NotesAreTheRendering(notes, version string, fragments []ReleaseFragment, policy ChangelogPolicy) (ok bool, limit string) {
	rendered := RenderReleaseNotes(version, fragments, policy)
	if notes == rendered {
		return true, ""
	}
	if notesWithoutCompatibility(notes) == notesWithoutCompatibility(rendered) {
		return true, "the compatibility section of the notes was not re-derived: it depends on the contract registry of the engine that cut the release"
	}
	return false, ""
}

// ReleaseCandidateGaps judges a cut that has no tag against the working tree: the policy it was cut
// under, the fragments, the notes as their rendering, the input digest, the changelog policy and
// the version file, and the files it froze. Nothing here reads a tag.
func ReleaseCandidateGaps(root, target string, manifest *ReleaseManifest) (gaps, limits []string) {
	gaps = []string{}
	policy, perr := LoadReleasePolicy(root)
	if perr != nil {
		gaps = append(gaps, perr.Error())
	} else if manifest.PolicyDigest != ReleaseDigest(policy) {
		gaps = append(gaps, "policy digest differs from prepared manifest")
	}
	changelog, clErr := LoadChangelogPolicyStrict(root)
	if clErr != nil {
		gaps = append(gaps, "the changelog policy cannot be used: "+clErr.Error())
	}
	archive, aerr := LoadReleaseFragments(filepath.Join(root, ".pose", "changelogs", target), changelog)
	if aerr != nil {
		gaps = append(gaps, aerr.Error())
	}
	notes, nerr := os.ReadFile(filepath.Join(root, ".pose", "changelogs", target+".md"))
	switch {
	case nerr != nil:
		gaps = append(gaps, "canonical notes missing")
	case ReleaseDigest(string(notes)) != manifest.NotesDigest:
		gaps = append(gaps, "canonical notes digest mismatch")
	case clErr == nil:
		if ok, limit := NotesAreTheRendering(string(notes), manifest.Version, manifest.Fragments, changelog); !ok {
			gaps = append(gaps, "the canonical notes are not the rendering of the fragments the manifest froze")
		} else if limit != "" {
			limits = append(limits, limit)
		}
	}
	if len(archive) != len(manifest.Fragments) {
		gaps = append(gaps, "archived fragment count differs from manifest")
	}
	bySpec := map[string]string{}
	for _, f := range archive {
		bySpec[f.Spec] = f.Digest
	}
	for _, f := range manifest.Fragments {
		if bySpec[f.Spec] != f.Digest {
			gaps = append(gaps, "fragment digest mismatch: "+f.Spec)
		}
	}
	if aerr == nil && perr == nil && clErr == nil {
		rebuilt := NewReleaseManifest(manifest.Version, manifest.PreviousVersion, manifest.PreparedAt, archive, policy, changelog, manifest.VersionEvidence)
		if rebuilt.ReleaseInputDigest != manifest.ReleaseInputDigest {
			gaps = append(gaps, "release input digest mismatch")
		}
	}
	if perr == nil {
		if evidence, eerr := ReleaseVersionEvidence(root, policy); eerr != nil {
			gaps = append(gaps, "version evidence could not be read: "+eerr.Error())
		} else {
			if evidence["value"] != manifest.Version || evidence["value"] != manifest.VersionEvidence["value"] {
				gaps = append(gaps, "the project's version file says "+evidence["value"]+", the manifest recorded "+manifest.VersionEvidence["value"])
			}
			if evidence["source"] != manifest.VersionEvidence["source"] || evidence["kind"] != manifest.VersionEvidence["kind"] {
				gaps = append(gaps, "the version evidence of the manifest names "+manifest.VersionEvidence["source"]+" ("+manifest.VersionEvidence["kind"]+"), the release policy names "+evidence["source"]+" ("+evidence["kind"]+")")
			}
		}
	}
	gaps = append(gaps, ReleaseFilesGaps(root, target, manifest)...)
	return gaps, limits
}
