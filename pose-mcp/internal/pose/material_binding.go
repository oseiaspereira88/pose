package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Validation evidence binds to the material it validated (spec
// pose-validation-evidence-material-binding, ADR
// validation-evidence-binds-to-validated-material). A commit whose every
// changed path is a derived administrative record does not change what the
// checks exercised, so it does not move the material digest.

// administrativeDirs is the closed list of derived-record directories. It is
// owned by the engine; adding a class is a contract change that needs an ADR.
// engineVersion is the running engine, registered by the binaries (internal/pose
// stays importable on its own). It is part of every material digest, so a new
// engine never inherits evidence the previous one produced.
var engineVersion = "unregistered"

// SetEngineVersion registers the running engine for material digests.
func SetEngineVersion(v string) { engineVersion = v }

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

func isSpecFile(p string) bool {
	rest, ok := strings.CutPrefix(p, ".pose/specs/")
	return ok && !strings.Contains(rest, "/") && strings.HasSuffix(rest, ".md")
}

var specLifecycleLine = regexp.MustCompile(`^[+-](status|completed_at):`)

// CommitIsAdministrative reports whether a commit changed only administrative
// records: every path is in a class (a rename needs both sides in one), or is a
// spec file whose only changed lines are the status and completed_at frontmatter
// lines. A symlink or mode change is material. Anything unreadable is material.
func CommitIsAdministrative(paths []ObservedPath, diff []byte) bool {
	if len(paths) == 0 {
		return false
	}
	specs := map[string]bool{}
	for _, p := range paths {
		names := []string{p.Path, p.OldPath, p.NewPath}
		for _, name := range names {
			if name == "" {
				continue
			}
			switch {
			case IsAdministrativePath(name):
			case isSpecFile(name) && p.OldPath == "" && p.NewPath == "" && p.Action == "M":
				specs[name] = true
			default:
				return false
			}
		}
	}
	file := ""
	for _, line := range strings.Split(string(diff), "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			file = ""
			if i := strings.LastIndex(line, " b/"); i >= 0 {
				file = line[i+3:]
			}
		case strings.HasPrefix(line, "new mode "), strings.HasPrefix(line, "old mode "),
			strings.HasPrefix(line, "new file mode 120000"), strings.HasPrefix(line, "deleted file mode 120000"),
			strings.HasPrefix(line, "rename from "), strings.HasPrefix(line, "rename to "):
			return false
		case specs[file] && (strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-")) &&
			!strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "---"):
			if !specLifecycleLine.MatchString(line) {
				return false
			}
		}
	}
	return true
}

// MaterialDiffDigest digests the diffs of the material commits (those that are
// not administrative). The validation matrix is not administrative, so a commit
// that changes it is material. Empty when the inputs disagree.
func MaterialDiffDigest(diffs [][]byte, material []bool) string {
	if len(diffs) != len(material) {
		return ""
	}
	h := sha256.New()
	for i, d := range diffs {
		if !material[i] {
			continue
		}
		sum := sha256.Sum256(d)
		h.Write(sum[:])
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ScopedMaterialDigest binds evidence to the claims and material of one spec's
// change sets and to the engine. It is empty when any change set has no
// material digest, which keeps such evidence on the commit-bound rule.
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
		if set.MaterialDigest == "" {
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
