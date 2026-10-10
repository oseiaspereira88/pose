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
	defer os.RemoveAll(tmp)
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
	for _, change := range changes {
		r.ContractLine("[DRY-RUN] " + text("would "+change.verb+": ", map[string]string{"create": "criaria", "modify": "modificaria", "remove": "removeria"}[change.verb]+": ") + change.path)
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
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		// Stat follows a symlink: the update reads through it.
		if info, err := os.Stat(filepath.Join(root, entry.Name())); err == nil && info.Mode().IsRegular() {
			if err := copyPlainFile(filepath.Join(root, entry.Name()), filepath.Join(shadow, entry.Name())); err != nil {
				return nil, err
			}
		}
	}
	for _, dir := range dryRunCopied {
		src := filepath.Join(root, dir)
		info, err := os.Stat(src)
		if err != nil || !info.IsDir() {
			continue
		}
		if err := copyDryRunTree(src, filepath.Join(shadow, dir), map[string]bool{}); err != nil {
			return nil, err
		}
	}
	return hashTree(shadow, true)
}

// copyDryRunTree copies src into dst, following symlinks the way the update
// reads through them: a symlinked .pose/policy is policy the update sees, and
// skipping it made the dry-run predict files and requests the real update
// does not create (found in review). visited breaks symlink cycles; broken
// links and special files are skipped.
func copyDryRunTree(src, dst string, visited map[string]bool) error {
	real, err := filepath.EvalSymlinks(src)
	if err != nil {
		return nil
	}
	if visited[real] {
		return nil
	}
	visited[real] = true
	defer delete(visited, real)
	return filepath.WalkDir(real, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(real, path)
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			return copyPlainFile(path, target)
		case d.Type()&fs.ModeSymlink != 0:
			info, err := os.Stat(path)
			switch {
			case err != nil:
				return nil
			case info.IsDir():
				return copyDryRunTree(path, target, visited)
			case info.Mode().IsRegular():
				return copyPlainFile(path, target)
			}
		}
		return nil
	})
}

func copyPlainFile(src, dst string) error {
	raw, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, raw, info.Mode().Perm())
}

// hashTree maps every regular file under root, except .git, to its content.
func hashTree(root string, skipGit bool) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if skipGit && (rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator))) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(raw)
		return nil
	})
	return out, err
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
