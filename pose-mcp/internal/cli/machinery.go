package cli

// Machinery delivery (spec pose-machinery-distribution-contract).
//
// Rules, workflows, templates and skills are engine-owned *files*: unlike the
// managed manuals, they carry no headings, so there is no sub-file unit to
// negotiate. The ownership unit is the whole file: identical content is a
// no-op, and content the instance edited is backed up to <file>.pose-backup
// before the refresh. Which content the instance edited is read from the
// digest the manifest recorded when POSE delivered it (see
// deliverMachineryFile).
//
// What did not exist before is delivery on a plain `pose upgrade`: machinery
// only ever reached an instance through `pose install --force`, so instances
// drifted from the engine indefinitely.
//
// Delivering on every upgrade raises a question the manuals never faced: a file
// the instance *deleted* would silently come back. The delivery manifest below
// is what tells the two cases apart — a path recorded as delivered and now
// absent was removed on purpose and stays removed, while a path the instance
// has never seen is new engine content and is delivered.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// machineryRoots are the trees the engine owns and delivers.
var machineryRoots = []string{".pose/workflows", ".pose/rules", ".pose/templates", ".agents/skills"}

// machineryManifestPath records which machinery paths this engine has already
// delivered to the instance.
func machineryManifestPath(target string) string {
	return filepath.Join(target, ".pose", "state", "machinery-manifest.json")
}

type machineryManifest struct {
	SchemaVersion int      `json:"schema_version"`
	Paths         []string `json:"paths"`
	// EngineVersion is the delivering binary's version.Version (spec
	// pose-instance-engine-version-tracking) — omitted, not guessed, for a
	// manifest written before this field existed.
	EngineVersion string `json:"engine_version,omitempty"`
	// Digests records the SHA-256 of the content delivered to each path, so a
	// later delivery can tell a file the instance edited from one it merely
	// received from an older release (spec
	// pose-machinery-backs-up-only-local-edits). Without it, every file a
	// release changed was backed up and reported as "customized".
	Digests map[string]string `json:"digests,omitempty"`
	// Manuals records, for each managed manual, the digest of every section
	// POSE last wrote (the preamble under ""). A merge uses it to tell a section
	// the instance edited from one an older release wrote, the same question
	// Digests answers for machinery (spec pose-manual-merge-backs-up-only-local-edits).
	Manuals map[string]map[string]string `json:"manuals,omitempty"`
}

// readMachineryManifest returns the manifest as stored, or an empty one.
func readMachineryManifest(target string) machineryManifest {
	var m machineryManifest
	raw, err := os.ReadFile(machineryManifestPath(target))
	if err != nil || json.Unmarshal(raw, &m) != nil {
		return machineryManifest{}
	}
	return m
}

func storeMachineryManifest(target string, m machineryManifest) error {
	m.SchemaVersion = 1
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := machineryManifestPath(target)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return writeAtomic(path, append(raw, '\n'), 0o644)
}

// instanceEngineVersion reads the engine version this instance's machinery
// was last delivered by, or "" when the manifest predates this field.
func instanceEngineVersion(target string) string {
	raw, err := os.ReadFile(machineryManifestPath(target))
	if err != nil {
		return ""
	}
	var m machineryManifest
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	return m.EngineVersion
}

func loadMachineryManifest(target string) map[string]bool {
	delivered := map[string]bool{}
	raw, err := os.ReadFile(machineryManifestPath(target))
	if err != nil {
		return delivered
	}
	var m machineryManifest
	if json.Unmarshal(raw, &m) != nil {
		return delivered
	}
	for _, p := range m.Paths {
		delivered[p] = true
	}
	return delivered
}

// loadMachineryDigests returns the digest recorded for each delivered path, or
// an empty map for a manifest written before digests were recorded.
func loadMachineryDigests(target string) map[string]string {
	raw, err := os.ReadFile(machineryManifestPath(target))
	if err != nil {
		return map[string]string{}
	}
	var m machineryManifest
	if json.Unmarshal(raw, &m) != nil || m.Digests == nil {
		return map[string]string{}
	}
	return m.Digests
}

func saveMachineryManifest(target string, paths []string, digests map[string]string) error {
	sort.Strings(paths)
	// The manual records are written by the manual merge, which runs before
	// machinery delivery on the same update; keep them.
	m := readMachineryManifest(target)
	m.Paths, m.EngineVersion, m.Digests = paths, Version, digests
	return storeMachineryManifest(target, m)
}

// deliverMachinery refreshes every engine-owned machinery tree into target.
//
// locale is the instance's locale, not the shell's: an instance installed with
// --locale pt-BR must not be silently rewritten in English by a plain upgrade,
// exactly as refreshManagedDocs already guarantees for the manuals. The source
// for each file is resolved once — the locale overlay when it exists, English
// otherwise — so a translated file is never written twice and never backs
// itself up.
//
// force ignores the deletion record, restoring the wholesale-reset meaning
// `pose install --force` has always had. noBackup suppresses the .pose-backup
// copy, matching the flag of the same name.
func deliverMachinery(src fs.FS, target, locale string, force, noBackup bool, stderr io.Writer, log func(english, portuguese string, a ...any)) error {
	deliveredBefore := loadMachineryManifest(target)
	recordedDigest := loadMachineryDigests(target)
	var deliveredNow []string
	digestsNow := map[string]string{}

	for _, root := range machineryRoots {
		if _, err := fs.Stat(src, root); err != nil {
			continue
		}
		err := fs.WalkDir(src, root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			deliveredNow = append(deliveredNow, path)
			dst := filepath.Join(target, filepath.FromSlash(path))

			// A path this engine already delivered and the instance has since
			// removed stays removed: re-creating it would undo a deliberate
			// decision. --force is the escape hatch, unchanged.
			if !force && deliveredBefore[path] {
				if _, statErr := os.Stat(dst); os.IsNotExist(statErr) {
					return nil
				}
			}
			digest, err := deliverMachineryFile(src, machinerySource(src, path, locale), dst, filePerm(path), recordedDigest[path], noBackup, stderr, target)
			if err != nil {
				return err
			}
			digestsNow[path] = digest
			return nil
		})
		if err != nil {
			return fmt.Errorf("%s: %w", root, err)
		}
		if log != nil {
			log("machinery (merged): %s", "maquinário (mesclado): %s", root)
		}
	}

	return saveMachineryManifest(target, deliveredNow, digestsNow)
}

// deliverMachineryFile writes one machinery file and returns the digest of the
// content it delivered.
//
// Identical content is a no-op. A file that still matches what POSE delivered
// last time is refreshed without a backup: nobody edited it, so there is
// nothing to keep. A file that differs from its recorded digest was edited by
// the instance and is backed up as customized. With no recorded digest — a
// manifest older than digests — a local edit cannot be ruled out, so it is
// backed up, and the report says why rather than calling it customized.
func deliverMachineryFile(src fs.FS, from, dst string, perm os.FileMode, recorded string, noBackup bool, stderr io.Writer, target string) (string, error) {
	content, err := fs.ReadFile(src, from)
	if err != nil {
		return "", err
	}
	digest := contentDigest(content)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if existing, err := os.ReadFile(dst); err == nil {
		if bytes.Equal(existing, content) {
			return digest, nil
		}
		unedited := recorded != "" && contentDigest(existing) == recorded
		if !unedited && !noBackup {
			if err := os.WriteFile(dst+".pose-backup", existing, perm); err == nil {
				rel, _ := filepath.Rel(target, dst)
				if recorded != "" {
					fmt.Fprintf(stderr, "[pose-install] backed up customized: %s → %s.pose-backup (changed since POSE delivered it)\n", rel, rel)
				} else {
					fmt.Fprintf(stderr, "[pose-install] backed up: %s → %s.pose-backup (no record of what POSE delivered, so a local edit cannot be ruled out)\n", rel, rel)
				}
			}
		}
	}
	if err := os.WriteFile(dst, content, perm); err != nil {
		return "", err
	}
	return digest, nil
}

func contentDigest(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// machinerySource returns the distribution path to deliver for a canonical
// machinery path: the locale overlay when the distribution carries one, the
// canonical English file otherwise.
func machinerySource(src fs.FS, path, locale string) string {
	if locale == "" || locale == "en" {
		return path
	}
	candidate := "locales/" + locale + "/" + path
	if _, err := fs.Stat(src, candidate); err == nil {
		return candidate
	}
	return path
}

// machineryLocale resolves the locale an instance is actually installed in,
// reusing the manual-based detection so machinery and manuals never disagree.
// explicit must be true only when localeFlag came from an operator-typed
// --locale, never when it is merely the caller's zero-value default — see
// resolveDocLocale.
func machineryLocale(src fs.FS, root, localeFlag string, explicit bool) string {
	existing, err := os.ReadFile(filepath.Join(root, "POSE.md"))
	if err != nil {
		return localeFlag
	}
	return resolveDocLocale(src, "POSE.md", string(existing), localeFlag, explicit)
}

// localeSkillMirrors lists the locales/<tag>/.agents/skills trees present in an
// instance, so skills-check can hold them to the same contract as the installed
// tree instead of letting a translated skill drift unchecked.
func localeSkillMirrors(root string) []string {
	entries, err := os.ReadDir(filepath.Join(root, "locales"))
	if err != nil {
		return nil
	}
	var mirrors []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		candidate := filepath.Join(root, "locales", e.Name(), ".agents", "skills")
		if isDir(candidate) {
			mirrors = append(mirrors, candidate)
		}
	}
	sort.Strings(mirrors)
	return mirrors
}
