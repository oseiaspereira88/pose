package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"sort"
	"strings"
)

// Validation evidence binds to the material it validated (spec
// pose-validation-evidence-material-binding, ADR
// validation-evidence-binds-to-validated-material). A commit whose every
// changed path is a derived administrative record does not change what the
// checks exercised, so it does not move the material digest.

// engineVersion is the running engine, registered by the binaries (internal/pose
// stays importable on its own). It is part of every material digest, so a new
// engine never inherits evidence the previous one produced.
var engineVersion = "unregistered"

// SetEngineVersion registers the running engine for material digests.
func SetEngineVersion(v string) { engineVersion = v }

// administrativeDirs is the closed list of derived-record directories. It is
// owned by the engine; adding a class is a contract change that needs an ADR.
var administrativeDirs = []string{".pose/indexes/", ".pose/results/", ".pose/state/", ".pose/assessments/"}

// materialIndexes are files inside an administrative directory that still
// change what the checks demand.
var materialIndexes = map[string]bool{".pose/indexes/validation-matrix.json": true}

// IsAdministrativePath reports whether p is a derived administrative record.
func IsAdministrativePath(p string) bool {
	if p == "" || p != path.Clean(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "../") {
		return false
	}
	if materialIndexes[p] {
		return false
	}
	for _, dir := range administrativeDirs {
		if strings.HasPrefix(p, dir) {
			return true
		}
	}
	if rest, ok := strings.CutPrefix(p, ".pose/"); ok {
		if i := strings.Index(rest, "/"); i > 0 && strings.HasPrefix(rest[:i], "review-") {
			return true
		}
	}
	return false
}

// TreeEntry is one entry of the validated head's tree.
type TreeEntry struct{ Mode, Blob string }

func regularMode(mode string) bool { return mode == "100644" || mode == "100755" }

// MaterialSetDigest digests the head state of the material a change set
// touched: every path it names (both sides of a rename) with its mode and blob
// at the validated head, minus the paths that are administrative records and
// still regular files there, plus the validation matrix. A symlink, submodule
// or directory in an administrative class is material; so is a path that left
// a class. tree maps each named path (and the matrix) to its entry; a path
// absent from the head tree is "absent".
func MaterialSetDigest(paths []ObservedPath, tree map[string]TreeEntry) string {
	names := map[string]bool{}
	for _, p := range paths {
		for _, name := range []string{p.Path, p.OldPath, p.NewPath} {
			if name != "" {
				names[name] = true
			}
		}
	}
	names[".pose/indexes/validation-matrix.json"] = true
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	h := sha256.New()
	for _, name := range sorted {
		entry, present := tree[name]
		if IsAdministrativePath(name) && (!present || regularMode(entry.Mode)) {
			continue
		}
		if !present {
			h.Write([]byte(name + "\x00absent\n"))
			continue
		}
		h.Write([]byte(name + "\x00" + entry.Mode + "\x00" + entry.Blob + "\n"))
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// MaterialDigestUnreadable marks a change set whose head tree could not be read
// or whose checkout is not complete. It never binds evidence and never lets
// evidence recorded without it pass as current.
const MaterialDigestUnreadable = "unreadable"

// MaterialBindingUnavailable is recorded for a scope whose material cannot be
// established, so the result is stale for it instead of falling back to the
// commit-bound rules.
const MaterialBindingUnavailable = "unavailable"

// MaterialBinding returns what a validation run should record for a spec: the
// scoped digest, the unavailable marker when a change set's material could not
// be read, or nothing when the spec has no material-bound change set.
func MaterialBinding(graph DeliveryIntegrityGraph, spec string) (string, bool) {
	for _, set := range graph.ChangeSets {
		if set.Spec == spec && set.MaterialDigest == MaterialDigestUnreadable {
			return MaterialBindingUnavailable, true
		}
	}
	if digest := ScopedMaterialDigest(graph, spec); digest != "" {
		return digest, true
	}
	return "", false
}

// ScopedMaterialDigest binds evidence to the claims and material of one spec's
// change sets and to the engine. It is empty when any change set has no
// material digest, so no binding can be recorded for it.
func ScopedMaterialDigest(graph DeliveryIntegrityGraph, spec string) string {
	claims := []ArtifactClaim{}
	digests := []string{}
	for _, claim := range graph.Claims {
		if claim.Spec == spec {
			claims = append(claims, claim)
		}
	}
	for _, set := range graph.ChangeSets {
		if set.Spec != spec {
			continue
		}
		if set.MaterialDigest == "" || set.MaterialDigest == MaterialDigestUnreadable {
			return ""
		}
		digests = append(digests, set.MaterialDigest)
	}
	if len(digests) == 0 {
		return ""
	}
	sort.Slice(claims, func(i, j int) bool { return claimKey(claims[i]) < claimKey(claims[j]) })
	sort.Strings(digests)
	raw, _ := json.Marshal(struct {
		Engine string          `json:"engine"`
		Claims []ArtifactClaim `json:"claims"`
		Digest []string        `json:"material"`
	}{engineVersion, claims, digests})
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// withoutMaterialDigest copies change sets without the material digest, so the
// commit-bound digests keep the representation results were recorded against.
func withoutMaterialDigest(sets []ChangeSet) []ChangeSet {
	out := make([]ChangeSet, len(sets))
	for i, set := range sets {
		set.MaterialDigest = ""
		out[i] = set
	}
	return out
}
