package pose

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Spec pose-release-manifest-tag-integrity: a release manifest is valid only if its shape and
// the projections it repeats agree with the fragments it freezes, and a tag is a release only
// if the tree it points at holds that manifest, its notes and its fragments.

// ReleaseManifestSchemaVersion is the manifest schema this engine reads.
const ReleaseManifestSchemaVersion = 1

// ReleaseManifestIssue is one reason a manifest is not valid, localized to its field.
type ReleaseManifestIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (i ReleaseManifestIssue) String() string { return i.Field + ": " + i.Message }

// releaseManifestFields are the exact names of the manifest schema.
var releaseManifestFields = map[string]bool{
	"schema_version": true, "version": true, "previous_version": true, "prepared_at": true, "specs": true,
	"categories": true, "fragments": true, "breaking": true, "notes_digest": true, "release_input_digest": true,
	"policy_digest": true, "version_evidence": true,
}

// releaseFragmentFields are the exact names of a frozen fragment.
var releaseFragmentFields = map[string]bool{
	"spec": true, "category": true, "breaking": true, "refs": true, "body": true, "path": true, "digest": true,
}

var releaseDigestRE = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidateReleaseManifestBytes decodes raw as the manifest of version and checks what can be
// checked from the manifest and nothing else: the schema and its shapes, unknown fields,
// the version, the paths, and the projections (specs, categories, breaking) and the input digest it
// repeats, recomposed from the fragments it freezes. A parseable JSON object is not a manifest.
func ValidateReleaseManifestBytes(raw []byte, version string) (*ReleaseManifest, []ReleaseManifestIssue) {
	var issues []ReleaseManifestIssue
	add := func(field, msg string) { issues = append(issues, ReleaseManifestIssue{Field: field, Message: msg}) }
	// One JSON document, no repeated key, the exact names of the schema, and every field the
	// schema requires present: a struct decoder alone accepts a second object after the first,
	// the last of two repeated keys, `VERSION` for `version` and an absent `breaking`.
	if err := rejectDuplicateJSONKeys(raw, false, true); err != nil {
		add("manifest", "is not an unambiguous JSON document: "+err.Error())
		return nil, issues
	}
	var fields map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&fields); err != nil || fields == nil {
		add("manifest", "does not decode as a JSON object")
		return nil, issues
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		add("manifest", "has content after the manifest object")
		return nil, issues
	}
	for name, value := range fields {
		if !releaseManifestFields[name] {
			add(name, "is not a field of the release manifest schema (field names are exact)")
		} else if string(bytes.TrimSpace(value)) == "null" {
			add(name, "is null")
		}
	}
	for name := range releaseManifestFields {
		if _, ok := fields[name]; !ok && name != "previous_version" {
			add(name, "is required and absent")
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}
	// The objects inside are held to the same contract as the root: exact names, every
	// required field, no null (refs is the one optional field).
	var rawFragments []map[string]json.RawMessage
	if err := json.Unmarshal(fields["fragments"], &rawFragments); err != nil {
		add("fragments", "is not a list of objects")
		return nil, issues
	}
	for i, obj := range rawFragments {
		if obj == nil {
			add(fmt.Sprintf("fragments[%d]", i), "is null")
			continue
		}
		// Free text (body, refs) may span lines; the names a fragment is addressed by may not.
		for _, name := range []string{"spec", "category", "path", "digest"} {
			var value string
			if json.Unmarshal(obj[name], &value) == nil && strings.ContainsFunc(value, unicode.IsControl) {
				add(fmt.Sprintf("fragments[%d].%s", i, name), "holds a control character")
			}
		}
		for name, value := range obj {
			switch {
			case !releaseFragmentFields[name]:
				add(fmt.Sprintf("fragments[%d].%s", i, name), "is not a field of a frozen fragment (field names are exact)")
			case string(bytes.TrimSpace(value)) == "null":
				add(fmt.Sprintf("fragments[%d].%s", i, name), "is null")
			}
		}
		for name := range releaseFragmentFields {
			if _, ok := obj[name]; !ok && name != "refs" {
				add(fmt.Sprintf("fragments[%d].%s", i, name), "is required and absent")
			}
		}
	}
	if len(issues) > 0 {
		return nil, issues
	}
	var manifest ReleaseManifest
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&manifest); err != nil {
		add("manifest", "does not decode as a release manifest: "+err.Error())
		return nil, issues
	}
	if manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		add("schema_version", fmt.Sprintf("%d is not a schema this engine reads (supported: %d)", manifest.SchemaVersion, ReleaseManifestSchemaVersion))
		return &manifest, issues
	}
	if err := ValidateReleaseVersion(manifest.Version); err != nil {
		add("version", err.Error())
	} else if version != "" && manifest.Version != version {
		add("version", fmt.Sprintf("%q is not the release of this directory (%q)", manifest.Version, version))
	}
	if manifest.PreviousVersion != "" {
		if err := ValidateReleaseVersion(manifest.PreviousVersion); err != nil {
			add("previous_version", err.Error())
		} else if manifest.PreviousVersion == manifest.Version {
			add("previous_version", "equals the release itself")
		} else if ValidateReleaseVersion(manifest.Version) == nil && CompareReleasePrecedence(manifest.PreviousVersion, manifest.Version) >= 0 {
			add("previous_version", "is not older than the release by version precedence")
		}
	}
	if _, err := time.Parse(time.RFC3339, manifest.PreparedAt); err != nil {
		add("prepared_at", "is not RFC3339")
	}
	for _, field := range []struct{ name, value string }{{"notes_digest", manifest.NotesDigest}, {"release_input_digest", manifest.ReleaseInputDigest}, {"policy_digest", manifest.PolicyDigest}} {
		if !releaseDigestRE.MatchString(field.value) {
			add(field.name, "is not a sha256 digest")
		}
	}
	for key := range manifest.VersionEvidence {
		if key != "source" && key != "kind" && key != "value" {
			add("version_evidence."+key, "is not part of the version evidence contract (source, kind, value)")
		}
	}
	evidence := manifest.VersionEvidence
	switch {
	case evidence["value"] == "":
		add("version_evidence.value", "is absent: the manifest records no version the cut was compared with")
	case manifest.Version != "" && evidence["value"] != manifest.Version:
		add("version_evidence.value", fmt.Sprintf("%q is not the version of the release (%q)", evidence["value"], manifest.Version))
	}
	if source := evidence["source"]; source == "" {
		add("version_evidence.source", "is absent: the manifest names no file the version came from")
	} else if filepath.IsAbs(source) || strings.Contains(source, "\\") || path.Clean(source) != source || strings.HasPrefix(source, "../") {
		add("version_evidence.source", "is not a clean project-relative path")
	}
	if kind := evidence["kind"]; kind != "" && kind != ReleaseVersionSourceText && kind != ReleaseVersionSourceJSON {
		add("version_evidence.kind", fmt.Sprintf("%q is not a version source kind (%s, %s, or none for the engine version file)", kind, ReleaseVersionSourceText, ReleaseVersionSourceJSON))
	}
	specs, categories, breaking := []string{}, map[string]int{}, false
	seen := map[string]bool{}
	for i, f := range manifest.Fragments {
		field := fmt.Sprintf("fragments[%d]", i)
		switch {
		case f.Spec == "":
			add(field+".spec", "is empty")
		case seen[f.Spec]:
			add(field+".spec", fmt.Sprintf("%q is frozen more than once", f.Spec))
		}
		seen[f.Spec] = true
		if f.Path == "" || strings.ContainsAny(f.Path, `/\`) || f.Path == "." || f.Path == ".." {
			add(field+".path", "is not a file name inside the release directory")
		}
		if !releaseDigestRE.MatchString(f.Digest) {
			add(field+".digest", "is not a sha256 digest")
		}
		if f.Category == "" {
			add(field+".category", "is empty")
		}
		specs = append(specs, f.Spec)
		categories[f.Category]++
		breaking = breaking || f.Breaking
	}
	if !equalStrings(manifest.Specs, specs) {
		add("specs", "differs from the specs of the frozen fragments")
	}
	if !equalCounts(manifest.Categories, categories) {
		add("categories", "differs from the categories of the frozen fragments")
	}
	if manifest.Breaking != breaking {
		add("breaking", "differs from the breaking flag of the frozen fragments")
	}
	// The input digest is composed of the manifest's own fields: it can be recomposed without
	// the policy or the notes, and a manifest whose digest does not follow from its fields was
	// edited after it was prepared.
	input := struct {
		Version   string            `json:"version"`
		Previous  string            `json:"previous"`
		Fragments []ReleaseFragment `json:"fragments"`
		Policy    string            `json:"policy"`
		Evidence  map[string]string `json:"evidence"`
	}{manifest.Version, manifest.PreviousVersion, manifest.Fragments, manifest.PolicyDigest, manifest.VersionEvidence}
	if ReleaseDigest(input) != manifest.ReleaseInputDigest {
		add("release_input_digest", "does not follow from the version, the fragments, the policy digest and the version evidence")
	}
	return &manifest, issues
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// Why a tag could not be read as a release.
var (
	ErrReleaseTagMissing      = errors.New("pose: the release tag does not exist")
	ErrReleaseSnapshotMissing = errors.New("pose: the tag's tree does not hold the release snapshot")
)

// ReleaseSnapshot is what the tree a tag points at says about its release.
type ReleaseSnapshot struct {
	Tag string `json:"tag"`
	// ManifestDigest is the digest of the manifest as the tagged tree holds it, which the
	// working tree's manifest is bound to.
	ManifestDigest string                 `json:"manifest_digest,omitempty"`
	Commit         string                 `json:"commit"`
	Annotated      bool                   `json:"annotated"`
	Manifest       *ReleaseManifest       `json:"manifest,omitempty"`
	Issues         []ReleaseManifestIssue `json:"issues,omitempty"`
	Gaps           []string               `json:"gaps,omitempty"`
	// Limits say what could not be re-verified from the snapshot, which is not a failure.
	Limits []string `json:"limits,omitempty"`
}

// Intact reports a snapshot with a valid manifest and nothing contradicting it.
func (s ReleaseSnapshot) Intact() bool {
	return s.Manifest != nil && len(s.Issues) == 0 && len(s.Gaps) == 0
}

// ErrReleaseTagNotCommit means a tag exists and does not lead to a commit.
var ErrReleaseTagNotCommit = errors.New("pose: the release tag does not point at a commit")

// ResolveReleaseTag peels a tag to the commit it names: a lightweight tag is that commit, an
// annotated one is the object it points at. A name that is not a tag is not resolved. Three
// answers are kept apart: the tag is not there (ErrReleaseTagMissing), it is there and is not a
// commit (ErrReleaseTagNotCommit), and Git could not answer (any other error, which is neither).
func ResolveReleaseTag(root, tag string) (commit string, annotated bool, err error) {
	if ValidateReleaseVersion(tag) != nil {
		return "", false, fmt.Errorf("pose: %q is not a release tag", tag)
	}
	if _, verr := gitOutput(root, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag); verr != nil {
		if gitNotARepository(root) {
			// No repository: there is no tag, which is a different thing from a tag Git could not read.
			return "", false, fmt.Errorf("%w: %s (not a git repository)", ErrReleaseTagMissing, tag)
		}
		var commandErr *gitCommandError
		if errors.As(verr, &commandErr) && commandErr.Stderr != "" {
			return "", false, fmt.Errorf("pose: the tag %s could not be resolved: %w", tag, verr)
		}
		var exit *exec.ExitError
		if errors.As(verr, &exit) && exit.ExitCode() == 1 {
			return "", false, fmt.Errorf("%w: %s", ErrReleaseTagMissing, tag)
		}
		return "", false, fmt.Errorf("pose: the tag %s could not be resolved: %w", tag, verr)
	}
	kind, err := gitOutput(root, "cat-file", "-t", "refs/tags/"+tag)
	if err != nil {
		return "", false, fmt.Errorf("pose: the tag %s could not be inspected: %w", tag, err)
	}
	out, err := gitOutput(root, "rev-parse", "--verify", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		return "", false, fmt.Errorf("%w: %s", ErrReleaseTagNotCommit, tag)
	}
	return strings.TrimSpace(out), strings.TrimSpace(kind) == "tag", nil
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", &gitCommandError{Op: args[0], Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	if strings.TrimSpace(stderr.String()) != "" {
		return "", &gitCommandError{Op: args[0], Stderr: strings.TrimSpace(stderr.String()), Err: errors.New("Git reported an incomplete or uncertain result")}
	}
	return string(out), nil
}

// ListReleaseTags preserves warnings about broken refs as incomplete inventory.
// Git may omit a broken loose tag while returning success; absence cannot be inferred.
func ListReleaseTags(root string) (string, error) {
	return gitOutput(root, "tag", "--list", "v*")
}

// gitTreeEntry looks one path up in a commit's tree. found=false with a nil error is the tree
// proving the path is not there; any failure to read the tree is an error, never absence.
func gitTreeEntry(root, commit, rel string) (mode, kind, object string, found bool, err error) {
	out, err := gitOutput(root, "ls-tree", "-z", commit, "--", rel)
	if err != nil {
		return "", "", "", false, err
	}
	meta, name, ok := strings.Cut(strings.TrimSuffix(out, "\x00"), "\t")
	fields := strings.Fields(meta)
	if !ok || len(fields) != 3 || name != rel {
		if strings.TrimSpace(out) != "" {
			return "", "", "", false, fmt.Errorf("pose: the tree entry of %s could not be read", rel)
		}
		// Nothing at the path. That is absence only when every directory above it is a tree: a link,
		// a file or a submodule in the place of an ancestor hides what is below it, and Git then
		// answers with nothing for the whole path.
		for dir := path.Dir(rel); dir != "." && dir != "/" && dir != ""; dir = path.Dir(dir) {
			parentOut, perr := gitOutput(root, "ls-tree", "-z", commit, "--", dir)
			if perr != nil {
				return "", "", "", false, perr
			}
			parentMeta, parentName, parentOK := strings.Cut(strings.TrimSuffix(parentOut, "\x00"), "\t")
			parentFields := strings.Fields(parentMeta)
			if strings.TrimSpace(parentOut) == "" {
				continue // this level is not there either: keep climbing to the first that is
			}
			if !parentOK || len(parentFields) != 3 || parentName != dir {
				return "", "", "", false, fmt.Errorf("pose: the tree entry of %s could not be read", dir)
			}
			if parentFields[1] != "tree" {
				return "", "", "", false, fmt.Errorf("pose: %s of the tagged tree is not a directory (mode %s), so %s cannot be proven absent", dir, parentFields[0], rel)
			}
			break
		}
		return "", "", "", false, nil
	}
	return fields[0], fields[1], fields[2], true, nil
}

// gitBlobAt reads one regular file of a commit's tree; a path the tree proves is not there is
// os.ErrNotExist, and a link, a submodule or a directory in its place is an error: the bytes of a
// link are not the content of a file.
func gitBlobAt(root, commit, rel string) ([]byte, error) {
	mode, kind, object, found, err := gitTreeEntry(root, commit, rel)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, os.ErrNotExist
	}
	if kind != "blob" || (mode != "100644" && mode != "100755") {
		return nil, fmt.Errorf("pose: %s of the tagged tree is not a regular file (mode %s)", rel, mode)
	}
	out, err := gitOutput(root, "cat-file", "blob", object)
	if err != nil {
		return nil, fmt.Errorf("pose: git cat-file %s: %w", rel, err)
	}
	return []byte(out), nil
}

// gitTreeNames lists the entries of one directory of a commit's tree. A directory the tree proves is
// not there is exists=false, which is not an error: a release with no fragments leaves no
// directory, and Git cannot version an empty one. A tree that cannot be read, and an entry that is
// not a directory, are errors.
func gitTreeNames(root, commit, dir string) (names []string, exists bool, err error) {
	_, kind, _, found, err := gitTreeEntry(root, commit, dir)
	if err != nil {
		return nil, false, err
	}
	if !found {
		return nil, false, nil
	}
	if kind != "tree" {
		return nil, true, fmt.Errorf("pose: %s of the tagged tree is a %s, not a directory", dir, kind)
	}
	out, err := gitOutput(root, "ls-tree", "--name-only", "-z", commit+":"+dir)
	if err != nil {
		return nil, true, err
	}
	for _, name := range strings.Split(out, "\x00") {
		if name != "" {
			names = append(names, name)
		}
	}
	return names, true, nil
}

// releasePolicyAt reads the release policy the tagged tree held: the policy the cut was made
// under, not the one the project has today.
func releasePolicyAt(root, commit string) (ReleasePolicy, string, error) {
	for _, rel := range []string{".pose/policy/release.json", ".pose/release-policy.json"} {
		raw, err := gitBlobAt(root, commit, rel)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return ReleasePolicy{}, rel, err
		}
		var policy ReleasePolicy
		if err := json.Unmarshal(raw, &policy); err != nil {
			return ReleasePolicy{}, rel, fmt.Errorf("invalid release policy: %w", err)
		}
		if policy.SchemaVersion != 1 {
			return ReleasePolicy{}, rel, fmt.Errorf("invalid release policy: schema_version must be 1")
		}
		return policy, rel, nil
	}
	return ReleasePolicy{}, "", os.ErrNotExist
}

// TagHoldsAdoptedReleasePolicy reports that the tree of tag holds a release policy that adopted
// governed releases. A tree that holds none is false; one whose policy cannot be read is an error.
func TagHoldsAdoptedReleasePolicy(root, tag string) (bool, error) {
	commit, _, err := ResolveReleaseTag(root, tag)
	if err != nil {
		return false, err
	}
	policy, _, err := releasePolicyAt(root, commit)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return policy.AdoptedAt != "", nil
}

// ReadReleaseSnapshot reads the release from the tree its tag points at, never from the
// working tree: manifest, notes and frozen fragments are blobs of the tagged commit. A tag whose
// tree holds no manifest is refused whatever the working tree holds. The projections the
// manifest repeats are recomposed from the fragment blobs and the policy blob of that tree,
// not trusted because they agree with each other. It does not compare the tagged commit with
// any other: a later commit, provider evidence or the next cycle cannot make a correct cut
// stale.
func ReadReleaseSnapshot(root, tag string) (ReleaseSnapshot, error) {
	commit, annotated, err := ResolveReleaseTag(root, tag)
	if err != nil {
		return ReleaseSnapshot{Tag: tag}, err
	}
	snap := ReleaseSnapshot{Tag: tag, Commit: commit, Annotated: annotated}
	manifestPath := ".pose/releases/" + tag + "/manifest.json"
	raw, err := gitBlobAt(root, commit, manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return snap, fmt.Errorf("%w: %s has no %s at %s", ErrReleaseSnapshotMissing, tag, manifestPath, commit[:12])
	}
	if err != nil {
		return snap, err
	}
	manifest, issues := ValidateReleaseManifestBytes(raw, tag)
	snap.Manifest, snap.Issues = manifest, issues
	snap.ManifestDigest = ReleaseDigest(string(raw))
	if manifest == nil || manifest.SchemaVersion != ReleaseManifestSchemaVersion {
		return snap, nil
	}
	gap := func(format string, args ...any) { snap.Gaps = append(snap.Gaps, fmt.Sprintf(format, args...)) }
	// The policies the cut was made under: the release policy and the changelog policy of the tagged
	// tree, held to their own contracts. A policy of today is never applied to a cut of yesterday.
	policy, policyPath, perr := releasePolicyAt(root, commit)
	policyKnown := false
	switch {
	case errors.Is(perr, os.ErrNotExist):
		gap("the tagged tree holds no release policy, so the manifest's policy digest cannot be tied to the policy of the cut")
	case perr != nil:
		gap("the release policy of the tagged tree (%s) could not be read: %v", policyPath, perr)
	default:
		if verr := ValidateReleasePolicy(policy); verr != nil {
			gap("the release policy of the tagged tree (%s) does not satisfy its contract: %v", policyPath, verr)
		} else if ReleaseDigest(policy) != manifest.PolicyDigest {
			gap("the manifest's policy digest is not the digest of the release policy of the tagged tree (%s)", policyPath)
		} else {
			policyKnown = true
		}
	}
	changelog, clErr := changelogPolicyAt(root, commit)
	if clErr != nil {
		gap("the changelog policy of the tagged tree could not be read: %v", clErr)
	}
	// Frozen fragments: each is a blob of the tagged tree, with the identity, category,
	// breaking flag, references and body the manifest repeats, at a path nothing else uses.
	frozen := map[string]string{}
	specs := map[string]bool{}
	for _, f := range manifest.Fragments {
		if other, dup := frozen[f.Path]; dup {
			gap("fragments %s and %s share the file %s", other, f.Spec, f.Path)
		}
		frozen[f.Path] = f.Spec
		if specs[f.Spec] {
			gap("fragment spec %s is frozen more than once", f.Spec)
		}
		specs[f.Spec] = true
		if clErr == nil && !changelog.ValidCategories()[f.Category] {
			gap("fragment %s has category %q, which the changelog policy of the tagged tree does not accept", f.Spec, f.Category)
		}
		blob, ferr := gitBlobAt(root, commit, ".pose/changelogs/"+tag+"/"+f.Path)
		switch {
		case errors.Is(ferr, os.ErrNotExist):
			gap("fragment %s is missing from the tagged tree", f.Spec)
			continue
		case ferr != nil:
			gap("fragment %s could not be read from the tagged tree: %v", f.Spec, ferr)
			continue
		case ReleaseDigest(string(blob)) != f.Digest:
			gap("fragment %s differs from the digest the manifest froze", f.Spec)
			continue
		}
		fm, body := SplitFrontmatter(string(blob))
		body = strings.TrimSpace(stripHTMLComments(body))
		switch {
		case strings.TrimSpace(fm["spec"]) != f.Spec:
			gap("fragment %s names spec %q in the tagged tree", f.Spec, strings.TrimSpace(fm["spec"]))
		case strings.TrimSpace(fm["category"]) == "" || strings.TrimSpace(fm["category"]) != f.Category:
			gap("fragment %s has category %q in the tagged tree, the manifest says %q", f.Spec, strings.TrimSpace(fm["category"]), f.Category)
		case (fm["breaking"] == "true") != f.Breaking:
			gap("fragment %s differs from the manifest in its breaking flag", f.Spec)
		case fm["refs"] != f.Refs:
			gap("fragment %s differs from the manifest in its references", f.Spec)
		case body == "":
			gap("fragment %s has no body in the tagged tree", f.Spec)
		case body != f.Body:
			gap("fragment %s differs from the manifest in its body", f.Spec)
		}
	}
	names, exists, lerr := gitTreeNames(root, commit, ".pose/changelogs/"+tag)
	switch {
	case lerr != nil:
		gap("the fragments of the tagged tree could not be listed: %v", lerr)
	case !exists && len(manifest.Fragments) > 0:
		gap("the tagged tree holds no fragment directory, the manifest freezes %d", len(manifest.Fragments))
	default:
		sort.Strings(names)
		for _, name := range names {
			if _, ok := frozen[name]; !ok {
				gap("the tagged tree holds fragment %s that the manifest does not freeze", name)
			}
		}
	}
	// Notes: the blob is the rendering of the frozen fragments under the changelog policy of the
	// cut, not whatever text its digest was recomputed for.
	notes, err := gitBlobAt(root, commit, ".pose/changelogs/"+tag+".md")
	switch {
	case errors.Is(err, os.ErrNotExist):
		gap("canonical notes are missing from the tagged tree")
	case err != nil:
		gap("canonical notes could not be read from the tagged tree: %v", err)
	case ReleaseDigest(string(notes)) != manifest.NotesDigest:
		gap("canonical notes digest differs from the manifest in the tagged tree")
	case clErr == nil:
		if ok, limit := NotesAreTheRendering(string(notes), manifest.Version, manifest.Fragments, changelog); !ok {
			gap("the canonical notes of the tagged tree are not the rendering of the fragments the manifest froze")
		} else if limit != "" {
			snap.Limits = append(snap.Limits, limit)
		}
	}
	// Version evidence: the file of the tagged tree that holds the version, read the way the
	// policy of the cut says to read it.
	if g, limit := snapshotVersionEvidence(root, commit, manifest, policy, policyKnown); g != "" {
		snap.Gaps = append(snap.Gaps, g)
	} else if limit != "" {
		snap.Limits = append(snap.Limits, limit)
	}
	return snap, nil
}

func changelogPolicyAt(root, commit string) (ChangelogPolicy, error) {
	raw, err := gitBlobAt(root, commit, ".pose/policy/changelog.json")
	switch {
	case errors.Is(err, os.ErrNotExist):
		return ChangelogPolicy{SchemaVersion: 1, Categories: DefaultChangelogCategories()}, nil
	case err != nil:
		return ChangelogPolicy{}, err
	}
	return ParseChangelogPolicy(raw)
}

// snapshotVersionEvidence re-verifies the version a manifest recorded against the file of the
// tagged tree that holds it, read the way the release policy of that tree says to read it. The
// evidence must name the source and the kind the policy names (or the engine's own version file
// when the policy names none); the file must be there; a text or JSON source is read and must say
// the version of the release. What cannot be recomputed is a limit, and only that.
func snapshotVersionEvidence(root, commit string, manifest *ReleaseManifest, policy ReleasePolicy, policyKnown bool) (gap, limit string) {
	source, kind, value := manifest.VersionEvidence["source"], manifest.VersionEvidence["kind"], manifest.VersionEvidence["value"]
	if source == "" {
		return "the manifest records no version source", ""
	}
	if !policyKnown {
		return "", "the version evidence was not tied to the release policy of the cut, which could not be verified"
	}
	if policy.VersionSource != nil {
		if filepath.ToSlash(filepath.Clean(policy.VersionSource.Path)) != source || policy.VersionSource.Kind != kind {
			return "the version evidence names a source or kind other than the version_source of the release policy of the tagged tree", ""
		}
	} else if source != EngineVersionFile || kind != "" {
		return "the release policy of the tagged tree declares no version_source, and the version evidence names a source other than the engine's own version file", ""
	}
	raw, err := gitBlobAt(root, commit, source)
	if errors.Is(err, os.ErrNotExist) {
		return "version source " + source + " is missing from the tagged tree", ""
	}
	if err != nil {
		return "version source " + source + " could not be read from the tagged tree: " + err.Error(), ""
	}
	if policy.VersionSource == nil {
		// The engine's own version file is a Go source that holds the number in code.
		return "", "the version evidence is the engine's own version file: it exists in the tagged tree and its value was not re-read"
	}
	got, err := parseReleaseVersionSource(raw, *policy.VersionSource)
	if err != nil {
		return "version source " + source + " in the tagged tree is not readable: " + err.Error(), ""
	}
	if got != value || got != manifest.Version {
		return fmt.Sprintf("the tagged tree's version source says %s, the manifest recorded %s for %s", got, value, manifest.Version), ""
	}
	return "", ""
}

// gitCommandError keeps the exit status of a failed git command, so a caller can tell "there is
// no such thing" (an exit status Git documents) from "Git could not answer".
type gitCommandError struct {
	Op     string
	Stderr string
	Err    error
}

func (e *gitCommandError) Error() string {
	return "pose: git " + e.Op + ": " + strings.TrimSpace(e.Stderr+" "+e.Err.Error())
}
func (e *gitCommandError) Unwrap() error { return e.Err }
