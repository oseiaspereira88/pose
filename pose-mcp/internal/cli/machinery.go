package cli

// Machinery delivery (spec pose-machinery-distribution-contract).
//
// Rules, workflows, templates and skills are engine-owned *files*: unlike the
// managed manuals, they carry no headings, so there is no sub-file unit to
// negotiate. The ownership unit is the whole file, and the contract is the one
// copyFileWithBackup already implements — identical content is a no-op, and
// divergent content is backed up to <file>.pose-backup before the refresh.
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

func saveMachineryManifest(target string, paths []string) error {
	sort.Strings(paths)
	m := machineryManifest{SchemaVersion: 1, Paths: paths}
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
	var deliveredNow []string

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
			return copyFileWithBackup(src, machinerySource(src, path, locale), dst, filePerm(path), noBackup, stderr, target)
		})
		if err != nil {
			return fmt.Errorf("%s: %w", root, err)
		}
		if log != nil {
			log("machinery (merged): %s", "maquinário (mesclado): %s", root)
		}
	}

	return saveMachineryManifest(target, deliveredNow)
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
func machineryLocale(src fs.FS, root, localeFlag string) string {
	existing, err := os.ReadFile(filepath.Join(root, "POSE.md"))
	if err != nil {
		return localeFlag
	}
	return resolveDocLocale(src, "POSE.md", string(existing), localeFlag)
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
