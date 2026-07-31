package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/version"
)

func cmdUpgrade(root string, args []string, stdout, stderr io.Writer) int {
	commandLocale := cliLocaleValue()
	text := func(english, portuguese string) string { return cliText(commandLocale, english, portuguese) }
	localeFlag := ""
	dry := false
	force := false
	skipSelf := false

	i := 0
	for i < len(args) {
		a := args[i]
		switch a {
		case "--dry-run":
			dry = true
			i++
		case "--force", "-f":
			force = true
			i++
		case "--no-self":
			skipSelf = true
			i++
		case "--self":
			// Accepted for backward compatibility
			i++
		case "--locale":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, text("pose upgrade: %s requires a value\n", "pose upgrade: %s exige um valor\n"), a)
				return 2
			}
			localeFlag = args[i+1]
			if localeFlag == "pt-BR" {
				commandLocale = localePtBR
			} else {
				commandLocale = localeEN
			}
			i += 2
		default:
			return usageError(stderr, text("Usage: pose upgrade [--dry-run] [--force] [--no-self] [--locale tag]", "Uso: pose upgrade [--dry-run] [--force] [--no-self] [--locale tag]"))
		}
	}

	if !dry && !skipSelf {
		if err := performSelfUpdate(stdout, stderr); err != nil {
			// Don't treat missing binary or offline dev environment as fatal
			if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "no such file") {
				fmt.Fprintf(stderr, text("[WARN] self-update check: %v\n", "[WARN] checagem de auto-atualização: %v\n"), err)
			}
		}
	}

	if _, e := os.Stat(filepath.Join(root, ".git")); e != nil {
		fmt.Fprintln(stderr, text("pose upgrade: a git repository is required", "pose upgrade: um repositório git é obrigatório"))
		return 1
	}
	poseDir := filepath.Join(root, ".pose")
	if fi, e := os.Lstat(poseDir); e != nil {
		fmt.Fprintln(stderr, text("pose upgrade: .pose not found", "pose upgrade: .pose não encontrado"))
		return 1
	} else if fi.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintln(stderr, text("pose upgrade: refusing to follow symlink at .pose", "pose upgrade: recusando seguir symlink em .pose"))
		return 1
	}

	current := 0
	if b, e := os.ReadFile(filepath.Join(poseDir, "schema-version")); e == nil {
		current, e = strconv.Atoi(strings.TrimSpace(string(b)))
		if e != nil {
			fmt.Fprintln(stderr, text("pose upgrade: invalid schema-version", "pose upgrade: schema-version inválido"))
			return 1
		}
	}
	if current > nativeSchemaVersion {
		fmt.Fprintf(stderr, text("pose upgrade: instance v%d is newer than engine v%d; downgrade is unsupported\n", "pose upgrade: instância v%d é mais recente que engine v%d; downgrade não é suportado\n"), current, nativeSchemaVersion)
		return 1
	}

	if dry {
		if current < nativeSchemaVersion {
			fmt.Fprintf(stdout, text("[INFO] schema upgrade: v%d -> v%d\n", "[INFO] atualização de schema: v%d -> v%d\n"), current, nativeSchemaVersion)
			fmt.Fprintln(stdout, text("[DRY-RUN] would apply: 001-baseline", "[DRY-RUN] aplicaria: 001-baseline"))
		} else {
			fmt.Fprintf(stdout, text("[INFO] instance already at schema v%d. Nothing to do.\n", "[INFO] instância já está no schema v%d. Nada a fazer.\n"), current)
		}
		if force {
			fmt.Fprintln(stdout, text("[DRY-RUN] would refresh scaffolds and rules (--force)", "[DRY-RUN] atualizaria scaffolds e regras (--force)"))
		}
		fmt.Fprintln(stdout, text("Result: DRY-RUN — no changes applied.", "Resultado: DRY-RUN — nenhuma alteração aplicada."))
		return 0
	}

	for _, rel := range []string{".pose/roadmaps", ".pose/changelogs/unreleased", ".pose/reports/history", ".pose/feedback"} {
		if e := ensureManagedDirSafe(root, rel); e != nil {
			fmt.Fprintf(stderr, "pose upgrade: %v\n", e)
			return 1
		}
	}
	if e := writeAtomic(filepath.Join(poseDir, "schema-version"), []byte(fmt.Sprintf("%d\n", nativeSchemaVersion)), 0o644); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}

	if force {
		fmt.Fprintln(stdout, text("[INFO] refreshing scaffolds, rules, workflows and MCP config...", "[INFO] atualizando scaffolds, regras, workflows e configuração MCP..."))
		installArgs := []string{root, "--force"}
		if localeFlag != "" {
			installArgs = append(installArgs, "--locale", localeFlag)
		} else if commandLocale == localePtBR {
			installArgs = append(installArgs, "--locale", "pt-BR")
		}
		if code := cmdInstall(installArgs, stdout, stderr); code != 0 {
			fmt.Fprintln(stderr, text("pose upgrade: scaffold refresh failed", "pose upgrade: falha na atualização de scaffolds"))
			return code
		}
	}

	if current == nativeSchemaVersion && !force {
		fmt.Fprintf(stdout, text("[INFO] instance already at schema v%d. Nothing to do.\n", "[INFO] instância já está no schema v%d. Nada a fazer.\n"), current)
		return 0
	}

	fmt.Fprintf(stdout, text("Result: SUCCESS — POSE upgraded to engine v%s (schema v%d).\n", "Resultado: SUCESSO — POSE atualizado para engine v%s (schema v%d).\n"), version.Version, nativeSchemaVersion)
	return 0
}

func performSelfUpdate(stdout, stderr io.Writer) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving binary symlink: %w", err)
	}

	fmt.Fprintf(stdout, "[INFO] checking latest release from github.com/%s...\n", releaseRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", releaseRepo), nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pose-cli/"+version.ReleaseBase())

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(stdout, "[INFO] offline or network unreachable; skipping binary self-update: %v\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stdout, "[INFO] GitHub release check returned status %d; skipping binary self-update\n", resp.StatusCode)
		return nil
	}

	var relData struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&relData); err != nil {
		return fmt.Errorf("parsing release JSON: %w", err)
	}

	latestVer := strings.TrimPrefix(relData.TagName, "v")
	currentVer := version.ReleaseBase()

	if latestVer == currentVer {
		fmt.Fprintf(stdout, "[INFO] pose binary is already at latest release (v%s)\n", latestVer)
		return nil
	}

	fmt.Fprintf(stdout, "[INFO] updating pose binary: v%s -> v%s...\n", currentVer, latestVer)
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	assetName := fmt.Sprintf("pose_%s_%s_%s.tar.gz", latestVer, goos, goarch)
	if goos == "windows" {
		assetName = fmt.Sprintf("pose_%s_%s_%s.zip", latestVer, goos, goarch)
	}

	assetURL := fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", releaseRepo, latestVer, assetName)
	fmt.Fprintf(stdout, "[INFO] downloading %s...\n", assetURL)

	assetResp, err := client.Get(assetURL)
	if err != nil || assetResp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading release asset failed")
	}
	defer assetResp.Body.Close()

	tmpFile, err := os.CreateTemp("", "pose-update-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, assetResp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing release archive: %w", err)
	}
	_ = tmpFile.Close()

	extractedBin, err := extractPoseBinary(tmpFile.Name(), goos)
	if err != nil {
		return fmt.Errorf("extracting pose binary: %w", err)
	}
	defer os.Remove(extractedBin)

	backupPath := execPath + ".old"
	_ = os.Remove(backupPath)
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	if err := copyDiskFile(extractedBin, execPath, 0o755); err != nil {
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("replacing binary: %w", err)
	}
	_ = os.Remove(backupPath)

	fmt.Fprintf(stdout, "[INFO] pose binary updated successfully to v%s at %s\n", latestVer, execPath)
	return nil
}

func extractPoseBinary(archivePath, goos string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") || goos == "windows" {
		r, err := zip.OpenReader(archivePath)
		if err != nil {
			return "", err
		}
		defer r.Close()
		for _, f := range r.File {
			if f.Name == "pose" || f.Name == "pose.exe" {
				rc, err := f.Open()
				if err != nil {
					return "", err
				}
				tmp, err := os.CreateTemp("", "pose-bin-*")
				if err != nil {
					rc.Close()
					return "", err
				}
				_, err = io.Copy(tmp, rc)
				rc.Close()
				tmp.Close()
				return tmp.Name(), err
			}
		}
		return "", fmt.Errorf("pose binary not found in zip archive")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if filepath.Base(header.Name) == "pose" || filepath.Base(header.Name) == "pose.exe" {
			tmp, err := os.CreateTemp("", "pose-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmp, tr); err != nil {
				tmp.Close()
				_ = os.Remove(tmp.Name())
				return "", err
			}
			tmp.Close()
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("pose binary not found in tar.gz archive")
}

func copyDiskFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ensureManagedDirSafe creates root/rel like os.MkdirAll, but first Lstat's
// the target and every already-existing ancestor under root: if any of them
// is a symlink, the upgrade refuses rather than silently writing through an
// instance-controlled escape (spec pose-upgrade-compatibility-lab).
func ensureManagedDirSafe(root, rel string) error {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if fi, err := os.Lstat(full); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to follow symlink at %s", rel)
		}
		return nil
	}
	for dir := filepath.Dir(full); len(dir) > len(root) && strings.HasPrefix(dir, root); dir = filepath.Dir(dir) {
		if fi, err := os.Lstat(dir); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			relDir, _ := filepath.Rel(root, dir)
			return fmt.Errorf("refusing to follow symlink at %s", filepath.ToSlash(relDir))
		}
	}
	return os.MkdirAll(full, 0o755)
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
	// Consumption refs (spec pose-knowledge-consumption-traceability R1):
	// knowledge:<slug> citations in specs must resolve to governed artifacts.
	refFailures := validateKnowledgeRefs(root, stderr)
	errors += refFailures
	fmt.Fprintf(stdout, "knowledge.schema.errors=%d\nknowledge.schema.warnings=%d\nknowledge.schema.checked=%d\nknowledge.overdue_count=%d\nknowledge.max_overdue=%d\nknowledge.ref_failures=%d\n", errors, warnings, checked, overdue, max, refFailures)
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
