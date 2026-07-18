package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func cmdUpgrade(root string, args []string, stdout, stderr io.Writer) int {
	dry := false
	if len(args) > 1 {
		return usageError(stderr, "Usage: pose upgrade [--dry-run]")
	}
	if len(args) == 1 {
		if args[0] != "--dry-run" {
			return usageError(stderr, "Usage: pose upgrade [--dry-run]")
		}
		dry = true
	}
	if _, e := os.Stat(filepath.Join(root, ".git")); e != nil {
		fmt.Fprintln(stderr, "pose upgrade: a git repository is required")
		return 1
	}
	poseDir := filepath.Join(root, ".pose")
	if _, e := os.Stat(poseDir); e != nil {
		fmt.Fprintln(stderr, "pose upgrade: .pose not found")
		return 1
	}
	current := 0
	if b, e := os.ReadFile(filepath.Join(poseDir, "schema-version")); e == nil {
		current, e = strconv.Atoi(strings.TrimSpace(string(b)))
		if e != nil {
			fmt.Fprintln(stderr, "pose upgrade: invalid schema-version")
			return 1
		}
	}
	if current > nativeSchemaVersion {
		fmt.Fprintf(stderr, "pose upgrade: instance v%d is newer than engine v%d; downgrade is unsupported\n", current, nativeSchemaVersion)
		return 1
	}
	if current == nativeSchemaVersion {
		fmt.Fprintf(stdout, "[INFO] instance already at schema v%d. Nothing to do.\n", current)
		return 0
	}
	fmt.Fprintf(stdout, "[INFO] schema upgrade: v%d -> v%d\n", current, nativeSchemaVersion)
	if dry {
		fmt.Fprintln(stdout, "[DRY-RUN] would apply: 001-baseline")
		fmt.Fprintln(stdout, "Result: DRY-RUN — no changes applied.")
		return 0
	}
	for _, rel := range []string{".pose/roadmaps", ".pose/changelogs/unreleased", ".pose/reports/history"} {
		if e := os.MkdirAll(filepath.Join(root, filepath.FromSlash(rel)), 0o755); e != nil {
			fmt.Fprintln(stderr, e)
			return 1
		}
	}
	if e := writeAtomic(filepath.Join(poseDir, "schema-version"), []byte(fmt.Sprintf("%d\n", nativeSchemaVersion)), 0o644); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	fmt.Fprintf(stdout, "Result: SUCCESS — schema v%d.\n", nativeSchemaVersion)
	return 0
}

func cmdKnowledgeHousekeeping(root string, args []string, stdout, stderr io.Writer) int {
	return knowledgeHousekeepingAt(root, args, stdout, stderr, time.Now().UTC())
}
func knowledgeHousekeepingAt(root string, args []string, stdout, stderr io.Writer, now time.Time) int {
	mode := "list-expired"
	apply := false
	if len(args) > 0 {
		mode = args[0]
		args = args[1:]
	}
	for _, a := range args {
		if a == "--apply" {
			apply = true
		} else if a == "--dry-run" {
			apply = false
		} else {
			return usageError(stderr, "Usage: pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]")
		}
	}
	if mode != "list-expired" && mode != "archive-expired" && mode != "purge-archived" {
		return usageError(stderr, "Usage: pose knowledge-housekeeping <list-expired|archive-expired|purge-archived> [--dry-run|--apply]")
	}
	base := filepath.Join(root, ".pose", "knowledge")
	dir := base
	if mode == "purge-archived" {
		dir = filepath.Join(base, "archive")
	}
	_ = os.MkdirAll(filepath.Join(base, "archive"), 0o755)
	entries, _ := os.ReadDir(dir)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		fm, err := readFlatFrontmatter(path)
		if err != nil {
			continue
		}
		expires, err := time.Parse("2006-01-02", fm["expires_at"])
		if err != nil || !expires.Before(now) {
			continue
		}
		if mode == "list-expired" {
			fmt.Fprintf(stdout, "%s|%s\n", path, fm["expires_at"])
			continue
		}
		if mode == "purge-archived" && expires.AddDate(0, 0, 180).After(now) {
			continue
		}
		if mode == "archive-expired" {
			target := filepath.Join(base, "archive", e.Name())
			if apply {
				err = os.Rename(path, target)
				fmt.Fprintf(stdout, "ARCHIVED|%s|%s|%s\n", path, target, fm["expires_at"])
			} else {
				fmt.Fprintf(stdout, "DRY-RUN ARCHIVE|%s|%s|%s\n", path, target, fm["expires_at"])
			}
		} else if apply {
			err = os.Remove(path)
			fmt.Fprintf(stdout, "PURGED|%s|%s\n", path, fm["expires_at"])
		} else {
			fmt.Fprintf(stdout, "DRY-RUN PURGE|%s|%s\n", path, fm["expires_at"])
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	return 0
}

func cmdReportsHousekeeping(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose reports-housekeeping <list-stale|archive-stale|purge-archived> [--older-than N] [--dry-run|--apply]")
	}
	mode := args[0]
	args = args[1:]
	days := 0
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--older-than":
			if i+1 >= len(args) {
				return 2
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return 2
			}
			days = n
		case "--apply":
			apply = true
		case "--dry-run":
			apply = false
		default:
			return usageError(stderr, "pose reports-housekeeping: invalid argument")
		}
	}
	if mode != "list-stale" && mode != "archive-stale" && mode != "purge-archived" {
		return usageError(stderr, "pose reports-housekeeping: invalid command")
	}
	if days == 0 {
		if mode == "purge-archived" {
			days = 365
		} else {
			days = 120
		}
	}
	base := filepath.Join(root, ".pose", "reports")
	dir := base
	if mode == "purge-archived" {
		dir = filepath.Join(base, "archive")
	}
	archive := filepath.Join(base, "archive")
	_ = os.MkdirAll(archive, 0o755)
	entries, _ := os.ReadDir(dir)
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" || len(e.Name()) < 11 {
			continue
		}
		date, err := time.Parse("2006-01-02", e.Name()[:10])
		if err != nil || !strings.HasPrefix(e.Name()[10:], "-") || !date.Before(cutoff) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if mode == "list-stale" {
			fmt.Fprintf(stdout, "%s|%s\n", path, e.Name()[:10])
			continue
		}
		if mode == "archive-stale" {
			target := filepath.Join(archive, e.Name())
			if apply {
				err = os.Rename(path, target)
				fmt.Fprintf(stdout, "ARCHIVED|%s|%s|%s\n", path, target, e.Name()[:10])
			} else {
				fmt.Fprintf(stdout, "DRY-RUN ARCHIVE|%s|%s|%s\n", path, target, e.Name()[:10])
			}
		} else if apply {
			err = os.Remove(path)
			fmt.Fprintf(stdout, "PURGED|%s|%s\n", path, e.Name()[:10])
		} else {
			fmt.Fprintf(stdout, "DRY-RUN PURGE|%s|%s\n", path, e.Name()[:10])
		}
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	return 0
}

func cmdKnowledgeCheck(root string, args []string, stdout, stderr io.Writer) int {
	mode := "strict"
	max := -1
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--max-overdue":
			if i+1 >= len(args) {
				return 2
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 0 {
				return 2
			}
			max = n
		default:
			return usageError(stderr, "Usage: pose knowledge-check [--strict|--tolerant] [--max-overdue N]")
		}
	}
	if max < 0 {
		if mode == "tolerant" {
			max = 2
		} else {
			max = 0
		}
	}
	dir := filepath.Join(root, ".pose", "knowledge")
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	errors, warnings, checked, overdue := 0, 0, 0, 0
	now := time.Now().UTC()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		checked++
		fm, readErr := readFlatFrontmatter(filepath.Join(dir, e.Name()))
		if readErr != nil {
			fmt.Fprintf(stderr, "[ERROR] %s: %v\n", e.Name(), readErr)
			errors++
			continue
		}
		required := []string{"type", "slug", "owner", "sensitivity", "created_at", "last_reviewed_at", "expires_at"}
		for _, k := range required {
			v := fm[k]
			if v == "" || strings.HasPrefix(v, "<") {
				fmt.Fprintf(stderr, "[ERROR] %s: missing or placeholder field: %s\n", e.Name(), k)
				errors++
			}
		}
		if !oneOf(fm["type"], "handoff", "note", "decision-log") {
			errors++
			fmt.Fprintf(stderr, "[ERROR] %s: invalid type\n", e.Name())
		}
		if !oneOf(fm["sensitivity"], "public-internal", "restricted") {
			errors++
			fmt.Fprintf(stderr, "[ERROR] %s: invalid sensitivity\n", e.Name())
		}
		created, ce := time.Parse("2006-01-02", fm["created_at"])
		reviewed, re := time.Parse("2006-01-02", fm["last_reviewed_at"])
		expires, ee := time.Parse("2006-01-02", fm["expires_at"])
		if ce != nil || re != nil || ee != nil {
			errors++
			fmt.Fprintf(stderr, "[ERROR] %s: invalid ISO date\n", e.Name())
		} else {
			if expires.Before(created) || expires.Sub(created) > 90*24*time.Hour {
				errors++
				fmt.Fprintf(stderr, "[ERROR] %s: invalid TTL\n", e.Name())
			}
			if reviewed.Before(created) {
				warnings++
			}
			if expires.Before(now) {
				overdue++
			}
		}
	}
	fmt.Fprintf(stdout, "knowledge.schema.errors=%d\nknowledge.schema.warnings=%d\nknowledge.schema.checked=%d\nknowledge.overdue_count=%d\nknowledge.max_overdue=%d\n", errors, warnings, checked, overdue, max)
	if errors > 0 || overdue > max {
		fmt.Fprintln(stdout, "Result: FAILURE")
		if mode == "strict" {
			return 1
		}
		fmt.Fprintln(stdout, "Result: TOLERATED_FAILURE")
	}
	if errors == 0 && overdue <= max {
		fmt.Fprintln(stdout, "Result: SUCCESS")
	}
	return 0
}

func readFlatFrontmatter(path string) (map[string]string, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("frontmatter missing")
	}
	out := map[string]string{}
	closed := false
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			closed = true
			break
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if ok {
			out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(strings.SplitN(v, "#", 2)[0]), "\"'")
		}
	}
	if !closed {
		return nil, fmt.Errorf("frontmatter not closed")
	}
	return out, nil
}
func oneOf(v string, values ...string) bool {
	for _, x := range values {
		if v == x {
			return true
		}
	}
	return false
}

func cmdHooks(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose hooks <install|uninstall|status> [--force]")
	}
	mode := args[0]
	force := false
	for _, a := range args[1:] {
		if a == "--force" {
			force = true
		} else {
			return usageError(stderr, "pose hooks: invalid argument")
		}
	}
	gitHooks := filepath.Join(root, ".git", "hooks")
	if _, e := os.Stat(gitHooks); e != nil {
		fmt.Fprintln(stderr, "pose hooks: .git/hooks not found")
		return 2
	}
	exe, e := os.Executable()
	if e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	for _, hook := range []string{"pre-commit", "post-merge"} {
		dst := filepath.Join(gitHooks, hook)
		managed := false
		if target, e := os.Readlink(dst); e == nil {
			abs := target
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(gitHooks, abs)
			}
			managed = sameFilePath(abs, exe)
		}
		switch mode {
		case "status":
			if managed {
				fmt.Fprintf(stdout, "[INSTALLED] %s -> %s\n", hook, exe)
			} else if _, e := os.Lstat(dst); e == nil {
				fmt.Fprintf(stdout, "[CONFLICT] %s\n", hook)
			} else {
				fmt.Fprintf(stdout, "[ABSENT] %s\n", hook)
			}
		case "install":
			if _, e := os.Lstat(dst); e == nil && !managed {
				if !force {
					fmt.Fprintf(stderr, "[WARN] %s exists; use --force\n", hook)
					continue
				}
				backup := fmt.Sprintf("%s.backup.%d", dst, time.Now().Unix())
				if e = os.Rename(dst, backup); e != nil {
					fmt.Fprintln(stderr, e)
					return 1
				}
			}
			if managed {
				_ = os.Remove(dst)
			}
			if e = os.Symlink(exe, dst); e != nil {
				fmt.Fprintln(stderr, e)
				return 1
			}
			fmt.Fprintf(stdout, "[OK] %s installed\n", hook)
		case "uninstall":
			if managed {
				_ = os.Remove(dst)
				backups, _ := filepath.Glob(dst + ".backup.*")
				sort.Strings(backups)
				if len(backups) > 0 {
					_ = os.Rename(backups[len(backups)-1], dst)
				}
				fmt.Fprintf(stdout, "[OK] %s removed\n", hook)
			} else {
				fmt.Fprintf(stdout, "[INFO] %s preserved\n", hook)
			}
		default:
			return usageError(stderr, "pose hooks: invalid command")
		}
	}
	return 0
}
func sameFilePath(a, b string) bool {
	aa, _ := filepath.Abs(a)
	bb, _ := filepath.Abs(b)
	return filepath.Clean(aa) == filepath.Clean(bb)
}

func HookMain(name string, stdout, stderr io.Writer) int {
	root, e := projectRoot()
	if e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	switch name {
	case "pre-commit":
		return cmdCheck(root, []string{"--tolerant"}, stdout, stderr)
	case "post-merge":
		return cmdIndex(root, nil, stdout, stderr)
	}
	return 2
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	if e := os.MkdirAll(filepath.Dir(path), 0o755); e != nil {
		return e
	}
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if e := os.WriteFile(tmp, data, perm); e != nil {
		return e
	}
	return os.Rename(tmp, path)
}

func cmdReleaseNotes(root string, args []string, stdout, stderr io.Writer) int {
	version, filter, dir := "", "", filepath.Join(root, ".pose", "changelogs", "unreleased")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version", "--filter", "--dir":
			if i+1 >= len(args) {
				return usageError(stderr, "pose release-notes: value required")
			}
			i++
			switch args[i-1] {
			case "--version":
				version = args[i]
			case "--filter":
				filter = args[i]
			case "--dir":
				dir = args[i]
			}
		default:
			return usageError(stderr, "Usage: pose release-notes [--version v] [--filter prefix] [--dir path]")
		}
	}
	if !confinedOutput(root, dir) {
		return usageError(stderr, "pose release-notes: --dir must remain inside project")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	categories := []string{"security", "added", "changed", "fixed", "deprecated", "removed"}
	titles := map[string]string{"security": "Security", "added": "Added", "changed": "Changed", "fixed": "Fixed", "deprecated": "Deprecated", "removed": "Removed"}
	buckets := map[string][]string{}
	breaking := []string{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, er := os.ReadFile(path)
		if er != nil {
			continue
		}
		fm, er := readFlatFrontmatter(path)
		if er != nil {
			continue
		}
		parts := strings.SplitN(string(raw), "---", 3)
		if len(parts) < 3 {
			continue
		}
		body := strings.TrimSpace(parts[2])
		for {
			start := strings.Index(body, "<!--")
			if start < 0 {
				break
			}
			end := strings.Index(body[start+4:], "-->")
			if end < 0 {
				body = body[:start]
				break
			}
			body = body[:start] + body[start+4+end+3:]
		}
		body = strings.Join(strings.Fields(body), " ")
		if body == "" {
			continue
		}
		spec := fm["spec"]
		if spec == "" {
			spec = strings.TrimSuffix(e.Name(), ".md")
		}
		if filter != "" && !strings.HasPrefix(spec, filter) {
			continue
		}
		cat := fm["category"]
		if titles[cat] == "" {
			cat = "changed"
		}
		line := fmt.Sprintf("- %s (`%s`)", body, spec)
		if strings.EqualFold(fm["breaking"], "true") {
			breaking = append(breaking, line)
		}
		buckets[cat] = append(buckets[cat], line)
	}
	if version == "" {
		fmt.Fprintln(stdout, "## Unreleased")
	} else {
		fmt.Fprintf(stdout, "## POSE %s\n", version)
	}
	fmt.Fprintln(stdout)
	if len(breaking) > 0 {
		fmt.Fprintln(stdout, "### ⚠️ Breaking changes")
		fmt.Fprintln(stdout, strings.Join(breaking, "\n"))
		fmt.Fprintln(stdout)
	}
	empty := true
	for _, cat := range categories {
		if len(buckets[cat]) == 0 {
			continue
		}
		empty = false
		fmt.Fprintf(stdout, "### %s\n%s\n\n", titles[cat], strings.Join(buckets[cat], "\n"))
	}
	if empty {
		fmt.Fprintln(stdout, "_No user-facing changes recorded._")
	}
	return 0
}
