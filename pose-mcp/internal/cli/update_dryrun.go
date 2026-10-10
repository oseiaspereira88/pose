package cli

// pose update --dry-run (spec pose-update-dry-run-reports-the-whole-update).
//
// The dry-run used to print only schema migrations and "no changes applied",
// while a real update merged manuals and machinery, stamped engine_version,
// seeded project identity and opened a configuration review. Four instances
// were declared current on that output while still stamped by an older
// engine. A second description of the update would drift from the update
// again, so the dry-run runs the update itself — the same cmdUpdate, without
// the binary self-update — on a disposable copy of what an update writes,
// and reports the difference.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harne8/pose-mcp/internal/version"
)

// dryRunCopied are the parts of an instance an update reads and writes; every
// top-level regular file is copied too (managed manuals, MCP config).
var dryRunCopied = []string{".pose", ".agents", ".claude", ".codex", ".github"}

func dryRunUpdate(root string, args []string, stdout, stderr io.Writer, text func(string, string) string) int {
	r := render(stdout, stderr)
	tmp, err := os.MkdirTemp("", "pose-update-dry-run-*")
	if err != nil {
		r.Failure("pose update: " + err.Error())
		return 1
	}
	defer removeDryRunCopy(tmp)
	shadow := filepath.Join(tmp, filepath.Base(root))
	if err := os.MkdirAll(shadow, 0o755); err != nil {
		r.Failure("pose update: " + err.Error())
		return 1
	}
	// A real repository, not an empty .git: --force runs install, which
	// refuses a target git does not recognise (found in review).
	if out, err := exec.Command("git", "init", "-q", shadow).CombinedOutput(); err != nil {
		r.Failure("pose update: preparing the dry-run copy: " + strings.TrimSpace(string(out)))
		return 1
	}
	before, err := snapshotForDryRun(root, shadow)
	if err != nil {
		r.Failure("pose update: preparing the dry-run copy: " + err.Error())
		return 1
	}
	engineBefore := recordedEngineVersion(root)

	real := []string{"--no-self"}
	for _, a := range args {
		if a != "--dry-run" && a != "--no-self" && a != "--self" {
			real = append(real, a)
		}
	}
	var out, errOut bytes.Buffer
	code := cmdUpdate(shadow, real, &out, &errOut)
	for _, line := range strings.Split(strings.TrimRight(out.String()+errOut.String(), "\n"), "\n") {
		if line = strings.TrimSpace(strings.ReplaceAll(line, shadow, root)); line != "" && !strings.HasPrefix(line, "Result:") && !strings.HasPrefix(line, "Resultado:") {
			r.ContractLine("[DRY-RUN] " + line)
		}
	}
	if code != 0 {
		r.ContractLine(text("Result: DRY-RUN — the update would fail; nothing was applied.", "Resultado: DRY-RUN — a atualização falharia; nada foi aplicado."))
		return code
	}
	after, err := hashTree(shadow, true)
	if err != nil {
		r.Failure("pose update: reading the dry-run copy: " + err.Error())
		return 1
	}
	changes := diffTrees(before, after)
	opened := 0
	for _, change := range changes {
		// A request's id hashes the second it is opened, so the copy's ids
		// are not the ones the update will write: count them, do not name
		// them (found in review).
		if change.verb == "create" && isActionRequestFile(change.path) {
			opened++
			continue
		}
		r.ContractLine("[DRY-RUN] " + text("would "+change.verb+": ", map[string]string{"create": "criaria", "modify": "modificaria", "remove": "removeria"}[change.verb]+": ") + change.path)
	}
	if opened > 0 {
		r.ContractLine(fmt.Sprintf("[DRY-RUN] "+text("would open %d action request(s) under .pose/actions/ (ids are assigned when they are opened)", "abriria %d pedido(s) de ação em .pose/actions/ (ids são atribuídos ao abrir)"), opened))
	}
	engineAfter := recordedEngineVersion(shadow)
	if engineAfter != engineBefore {
		r.ContractLine(fmt.Sprintf("[DRY-RUN] %s %s -> %s", text("would stamp engine_version:", "carimbaria engine_version:"), orDash(engineBefore), engineAfter))
	}
	if len(changes) == 0 {
		r.ContractLine(text("Result: DRY-RUN — the update would change nothing.", "Resultado: DRY-RUN — a atualização não mudaria nada."))
		return 0
	}
	r.ContractLine(fmt.Sprintf(text("Result: DRY-RUN — the update would change %d file(s); nothing was applied.", "Resultado: DRY-RUN — a atualização mudaria %d arquivo(s); nada foi aplicado."), len(changes)))
	return 0
}

func isActionRequestFile(path string) bool {
	return strings.HasPrefix(path, ".pose/actions/act-") && strings.HasSuffix(path, ".jsonl") && !strings.Contains(strings.TrimPrefix(path, ".pose/actions/"), "/")
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// recordedEngineVersion is the engine that last updated the instance.
func recordedEngineVersion(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "state", "machinery-manifest.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		EngineVersion string `json:"engine_version"`
	}
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	return doc.EngineVersion
}

// snapshotForDryRun copies what an update touches into shadow and returns the
// hashes of the copy as it was before the update.
func snapshotForDryRun(root, shadow string) (map[string]string, error) {
	c, err := newDryRunCopier(root, shadow)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		switch {
		case entry.Type().IsRegular():
			if err := copyPlainFile(path, filepath.Join(shadow, entry.Name())); err != nil {
				return nil, err
			}
		case entry.Type()&fs.ModeSymlink != 0:
			// A link to a file, or a broken one: the update writes through
			// both (found in review). A link to a directory outside the
			// copied set is not the update's to read.
			if info, err := os.Stat(path); err != nil || info.Mode().IsRegular() {
				if err := c.link(path, filepath.Join(shadow, entry.Name())); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, dir := range dryRunCopied {
		if err := c.copy(filepath.Join(root, dir), filepath.Join(shadow, dir)); err != nil {
			return nil, err
		}
	}
	if info, err := os.Stat(root); err == nil {
		c.dirModes = append(c.dirModes, dirMode{shadow, info.Mode().Perm()})
	}
	c.applyDirModes()
	return hashTree(shadow, true)
}

// dirMode is a copied directory and the permissions of its original. They are
// applied after the copy, deepest first, since a read-only directory cannot
// receive the files copied into it; a read-only .pose made the update fail
// while the dry-run, on 0755 copies, said it would change nothing (found in
// review).
type dirMode struct {
	path string
	mode fs.FileMode
}

func (c *dryRunCopier) applyDirModes() {
	for i := len(c.dirModes) - 1; i >= 0; i-- {
		_ = os.Chmod(c.dirModes[i].path, c.dirModes[i].mode)
	}
}

// removeDryRunCopy makes every directory of the copy writable again, so the
// read-only ones it reproduced can be removed.
func removeDryRunCopy(tmp string) {
	_ = filepath.WalkDir(tmp, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			_ = os.Chmod(path, 0o700)
		}
		return nil
	})
	_ = os.RemoveAll(tmp)
}

// dryRunCopier copies an instance into the shadow keeping its symlinks as
// symlinks, so the update meets the same links it would in the instance — it
// refuses some (a symlinked .pose/templates) and reads through others (found
// in review: turning links into plain directories hid those refusals). A link
// never points back into the instance: a target inside it is copied into the
// shadow and linked relatively, a target outside it is copied next to the
// shadow, so whatever the update writes through a link lands in a copy.
type dryRunCopier struct {
	root, rootReal, shadow, outside string
	copied                          map[string]string
	n                               int
	dirModes                        []dirMode
}

func newDryRunCopier(root, shadow string) (*dryRunCopier, error) {
	rootReal, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	return &dryRunCopier{root: root, rootReal: rootReal, shadow: shadow, outside: filepath.Join(filepath.Dir(shadow), ".linked"), copied: map[string]string{}}, nil
}

func (c *dryRunCopier) copy(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return nil
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return c.link(src, dst)
	}
	if !info.IsDir() {
		return nil
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// An unreadable directory is reproduced empty with its mode,
			// which the walk recorded before failing to list it.
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			if info, err := d.Info(); err == nil {
				c.dirModes = append(c.dirModes, dirMode{target, info.Mode().Perm()})
			}
			return os.MkdirAll(target, 0o755)
		}
		// A directory reached both through a link and by its own path is
		// copied once: the second pass would collide with the links and the
		// read-only files the first one wrote (found in review).
		if _, err := os.Lstat(target); err == nil {
			return nil
		}
		switch {
		case d.Type().IsRegular():
			return copyPlainFile(path, target)
		case d.Type()&fs.ModeSymlink != 0:
			return c.link(path, target)
		}
		return nil // special files are not copied
	})
}

// link recreates the symlink at src as dst, pointing at a copy of its target.
func (c *dryRunCopier) link(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		return nil // recreated already, by a pass that reached it through another link
	}
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		return c.brokenLink(src, dst)
	}
	copyAt, done := c.copied[resolved]
	if !done {
		if rel, err := filepath.Rel(c.rootReal, resolved); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			copyAt = filepath.Join(c.shadow, rel)
			// A link to a directory that contains it (`.pose/self -> .`,
			// `.agents/up -> ..`) points at a copy being made, or at the
			// shadow itself: link it, do not copy it into itself (found in
			// review).
			if copyAt == c.shadow || strings.HasPrefix(dst, copyAt+string(filepath.Separator)) {
				return os.Symlink(relativeTo(dst, copyAt), dst)
			}
		} else {
			c.n++
			copyAt = filepath.Join(c.outside, fmt.Sprint(c.n), filepath.Base(resolved))
		}
		c.copied[resolved] = copyAt
		// A directory is copied even when its path already exists in the
		// shadow: the path may exist only because a link to one file inside it
		// was copied first (found in review). copy skips what is already there.
		info, err := os.Stat(resolved)
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if err := c.copy(resolved, copyAt); err != nil {
				return err
			}
		} else if info.Mode().IsRegular() {
			if _, err := os.Lstat(copyAt); err != nil {
				if err := copyPlainFile(resolved, copyAt); err != nil {
					return err
				}
			}
		}
	}
	target := copyAt
	if strings.HasPrefix(copyAt, c.shadow+string(filepath.Separator)) {
		target = relativeTo(dst, copyAt)
	}
	return os.Symlink(target, dst)
}

// brokenLink recreates a link whose target does not exist as a link that is
// still broken, pointing at the same missing path inside the shadow or at a
// missing path next to it — never at the original target. Copying the target
// verbatim let the update create a file outside the copy through it (found in
// review: an absolute .pose/rules/security.md link made the dry-run write the
// external file and still say nothing was applied).
func (c *dryRunCopier) brokenLink(src, dst string) error {
	raw, err := os.Readlink(src)
	if err != nil {
		return nil
	}
	intended := raw
	if !filepath.IsAbs(intended) {
		intended = filepath.Join(filepath.Dir(src), raw)
	}
	intended = filepath.Clean(intended)
	// The stand-in keeps what the update can observe: the target is missing,
	// and its parent directory exists exactly when the original's does, so
	// writing through the link succeeds or fails as it would (found in review).
	parentExists := false
	if info, err := os.Stat(filepath.Dir(intended)); err == nil && info.IsDir() {
		parentExists = true
	}
	standIn := ""
	for _, base := range []string{c.root, c.rootReal} {
		if rel, err := filepath.Rel(base, intended); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			standIn = filepath.Join(c.shadow, rel)
			break
		}
	}
	if standIn == "" {
		c.n++
		standIn = filepath.Join(c.outside, fmt.Sprint(c.n), filepath.Base(intended))
		if !parentExists {
			standIn = filepath.Join(c.outside, fmt.Sprint(c.n), "missing", filepath.Base(intended))
		}
	}
	if parentExists {
		if err := os.MkdirAll(filepath.Dir(standIn), 0o755); err != nil {
			return err
		}
	}
	if strings.HasPrefix(standIn, c.shadow+string(filepath.Separator)) {
		return os.Symlink(relativeTo(dst, standIn), dst)
	}
	return os.Symlink(standIn, dst)
}

func relativeTo(link, target string) string {
	if rel, err := filepath.Rel(filepath.Dir(link), target); err == nil {
		return rel
	}
	return target
}

func copyPlainFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(src)
	if errors.Is(err, fs.ErrPermission) {
		// A file nobody may read is reproduced as one: same mode, no
		// content, so the update meets the same refusal — and a file the
		// update never reads no longer stops the dry-run (found in review).
		raw = nil
	} else if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, info.Mode().Perm())
}

// hashTree maps every regular file under root, except .git, to its content,
// reading through symlinks so content behind a link is compared too; a link
// back to a directory already being read is not followed again.
func hashTree(root string, skipGit bool) (map[string]string, error) {
	out := map[string]string{}
	err := hashTreeInto(out, root, "", skipGit, map[string]bool{})
	return out, err
}

func hashTreeInto(out map[string]string, dir, prefix string, skipGit bool, visiting map[string]bool) error {
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return nil
	}
	if visiting[real] {
		return nil
	}
	visiting[real] = true
	defer delete(visiting, real)
	return filepath.WalkDir(real, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(real, path)
		if prefix != "" {
			rel = filepath.Join(prefix, rel)
		}
		if skipGit && (rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator))) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			info, err := os.Stat(path)
			switch {
			case err != nil:
				return nil
			case info.IsDir():
				return hashTreeInto(out, path, rel, skipGit, visiting)
			case !info.Mode().IsRegular():
				return nil
			}
		} else if !d.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrPermission) {
			out[filepath.ToSlash(rel)] = "\x00unreadable"
			return nil
		}
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
}

type treeChange struct{ verb, path string }

func diffTrees(before, after map[string]string) []treeChange {
	changes := []treeChange{}
	for path, content := range after {
		if old, ok := before[path]; !ok {
			changes = append(changes, treeChange{"create", path})
		} else if old != content {
			changes = append(changes, treeChange{"modify", path})
		}
	}
	for path := range before {
		if _, ok := after[path]; !ok {
			changes = append(changes, treeChange{"remove", path})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].path < changes[j].path })
	return changes
}

// staleInstanceEngine returns the engine that last updated the instance when
// it is older than the running engine, or "". Pre-release suffixes are
// ignored: a 7.3.0-dev build is the 7.3.0 engine.
func staleInstanceEngine(root string) string {
	recorded := recordedEngineVersion(root)
	if recorded == "" {
		return ""
	}
	parse := func(v string) [3]int {
		var out [3]int
		v = strings.TrimPrefix(strings.SplitN(v, "-", 2)[0], "v")
		for i, part := range strings.SplitN(v, ".", 3) {
			fmt.Sscanf(part, "%d", &out[i])
		}
		return out
	}
	have, want := parse(recorded), parse(version.ReleaseBase())
	for i := 0; i < 3; i++ {
		if have[i] != want[i] {
			if have[i] < want[i] {
				return recorded
			}
			return ""
		}
	}
	return ""
}
