package cli

// pose install — clone-free native installer (spec pose-cli-embed-standalone):
// installs the embedded POSE distribution into a target repository and seeds
// MCP configuration that invokes this same binary through `pose serve-mcp`.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

func cmdInstall(args []string, stdout, stderr io.Writer) int {
	commandLocale := cliLocaleValue()
	text := func(english, portuguese string) string { return cliText(commandLocale, english, portuguese) }
	var target, projectName, projectID, locale string
	locale = "en"
	localeExplicit := false
	force, skipMCP, allowNonGit, noBackup := false, false, false, false

	i := 0
	for i < len(args) {
		a := args[i]
		switch a {
		case "--project-name", "--project-id", "--locale":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, text("pose install: %s requires a value\n", "pose install: %s exige um valor\n"), a)
				return 2
			}
			v := args[i+1]
			switch a {
			case "--project-name":
				projectName = v
			case "--project-id":
				projectID = v
			case "--locale":
				locale = v
				localeExplicit = true
				// The operator's explicit request also drives this
				// command's own diagnostic output (log lines, the
				// post-install gate report), not just the scaffold
				// content — matching --locale's meaning on `pose update`
				// (spec pose-post-install-gate-locale).
				if v == "pt-BR" {
					commandLocale = localePtBR
				} else {
					commandLocale = localeEN
				}
			}
			i += 2
		case "--force":
			force = true
			i++
		case "--skip-mcp":
			skipMCP = true
			i++
		case "--allow-non-git":
			allowNonGit = true
			i++
		case "--no-backup":
			noBackup = true
			i++
		case "-h", "--help":
			fmt.Fprintln(stdout, text("Usage: pose install <target-dir> [--project-name n] [--project-id id] [--locale tag] [--force] [--skip-mcp] [--allow-non-git] [--no-backup]", "Uso: pose install <target-dir> [--project-name n] [--project-id id] [--locale tag] [--force] [--skip-mcp] [--allow-non-git] [--no-backup]"))
			return 0
		default:
			if strings.HasPrefix(a, "-") {
				fmt.Fprintf(stderr, text("pose install: unknown option: %s\n", "pose install: opção desconhecida: %s\n"), a)
				return 2
			}
			if target != "" {
				fmt.Fprintf(stderr, text("pose install: unexpected argument: %s\n", "pose install: argumento inesperado: %s\n"), a)
				return 2
			}
			target = a
			i++
		}
	}
	if target == "" {
		fmt.Fprintln(stderr, text("pose install: target directory is required", "pose install: diretório alvo é obrigatório"))
		return 2
	}
	abs, err := filepath.Abs(target)
	if err != nil || !isDir(abs) {
		fmt.Fprintf(stderr, text("pose install: target directory does not exist: %s\n", "pose install: diretório alvo não existe: %s\n"), target)
		return 1
	}
	target = abs
	if !allowNonGit {
		if err := exec.Command("git", "-C", target, "rev-parse", "--git-dir").Run(); err != nil {
			fmt.Fprintf(stderr, text("pose install: target is not a git repository: %s (use --allow-non-git)\n", "pose install: alvo não é um repositório git: %s (use --allow-non-git)\n"), target)
			return 1
		}
	}
	if projectName == "" || projectID == "" {
		// An already-installed target's declared identity is recovered from
		// its own AGENTS.md/.mcp.json before ever falling back to the
		// directory's current basename — the same recovery
		// `refreshManagedDocs` already does for a plain `pose update`.
		// Without this, any `--force` update or install rerun (this
		// function is also what `pose update --force` delegates to)
		// silently renamed the project's declared identity the moment the
		// repository was checked out or cloned under a different directory
		// name than the one it was first installed with.
		detectedName, detectedID := "", ""
		if existing, err := os.ReadFile(filepath.Join(target, "AGENTS.md")); err == nil {
			detectedName, detectedID = detectProjectIdentity(string(existing), target)
		}
		if projectName == "" {
			if detectedName != "" {
				projectName = detectedName
			} else {
				projectName = filepath.Base(target)
			}
		}
		if projectID == "" {
			if detectedID != "" {
				projectID = detectedID
			} else {
				projectID = "proj." + projectName
			}
		}
	}
	log := func(english, portuguese string, a ...any) {
		fmt.Fprintf(stdout, "[pose-install] "+text(english, portuguese)+"\n", a...)
	}
	log("target:       %s", "alvo:         %s", target)
	log("project name: %s", "nome do projeto: %s", projectName)
	log("project id:   %s", "id do projeto:   %s", projectID)

	// A target already carrying a `.pose` instance may already fail
	// `check --strict` on its own governance debt, unrelated to anything
	// this run is about to do — reindexing/reseeding module-metadata
	// alone can flip an already-broken review-closeout or provenance
	// state (spec pose-install-gate-failure-recovery-notice's own
	// follow-up: "decide whether the index step should ... run before
	// delivery so the failure is honest about what did not happen").
	// Checking here, before any write, lets the final gate failure (if
	// any) say so plainly instead of reading as something this install
	// broke.
	preExistingFailure := false
	if _, err := os.Stat(filepath.Join(target, ".pose")); err == nil {
		if cmdCheck(target, []string{"--strict"}, io.Discard, io.Discard) != 0 {
			preExistingFailure = true
			log("warning: target already fails `pose check --strict` before this run touches anything — pre-existing governance debt, not something this command is about to cause", "aviso: o alvo já falha em `pose check --strict` antes desta execução tocar em qualquer coisa — dívida de governança pré-existente, não algo que este comando está prestes a causar")
		}
	}

	dist := scaffold.Dist()

	// An already-installed target's real locale is detected from its
	// existing POSE.md the same way `pose update` (without --force) already
	// does — but only when the caller did not pass --locale explicitly.
	// resolveDocLocale's own short-circuit only recognizes an explicit
	// non-en value; without localeExplicit here, an explicit `--locale en`
	// on an already pt-BR-localized target would fall through to detection
	// and get silently overridden back to pt-BR. machineryLocale itself
	// returns `locale` unchanged when POSE.md is absent (fresh install:
	// keeps defaulting to en). Without any of this, `pose install <target>`
	// rerun on an instance, or `pose update --force` (which delegates
	// entirely to cmdInstall), would silently revert an already-localized
	// project's machinery to English — github.com/oseiaspereira88/pose#18.
	if !localeExplicit {
		locale = machineryLocale(dist, target, locale, false)
	}

	// 1. Native machinery: this binary is the runtime. Delivery is shared with
	// `pose upgrade` (spec pose-machinery-distribution-contract), which is why
	// it lives in deliverMachinery rather than inline here.
	if err := deliverMachinery(dist, target, locale, force, noBackup, stderr, log); err != nil {
		fmt.Fprintf(stderr, text("pose install: merging machinery: %v\n", "pose install: mesclando maquinário: %v\n"), err)
		return 1
	}

	// .claude/skills symlinks (embed cannot carry them).
	if linked, err := recreateClaudeSkillSymlinks(target); err == nil {
		if linked {
			log("machinery: .claude/skills (symlinks)", "maquinário: .claude/skills (symlinks)")
		} else {
			log("warning: could not create .claude/skills symlinks (unsupported filesystem?) — Claude Code discovery degraded", "aviso: não foi possível criar symlinks em .claude/skills (filesystem sem suporte?) — descoberta do Claude Code degradada")
		}
	}

	// 2. Config indexes, module-metadata discovery, policy and
	// review-profiles: seed only what is absent. Shared with `pose update`
	// (see seedAbsentInstanceConfig) so a plain update without --force seeds
	// the same subsystems a fresh install does instead of leaving an old
	// instance with docs/machinery that reference subsystems (assessments,
	// spec-graph.json, policy/) it never actually seeded — that produced a
	// "Result: SUCCESS" update whose very next `pose check --strict` failed
	// with broken references, undetected by `pose doctor`
	// (spec pose-update-instance-config-completeness).
	seedAbsentInstanceConfig(dist, target, log)

	// 3. Legal texts vendored under .pose/.
	_ = copyFile(dist, "LICENSE", filepath.Join(target, ".pose", "LICENSE"), 0o644)
	_ = copyFile(dist, "NOTICE", filepath.Join(target, ".pose", "NOTICE"), 0o644)
	log("vendored: .pose/LICENSE, .pose/NOTICE", "incorporados: .pose/LICENSE, .pose/NOTICE")

	// 4. Root docs with locale + placeholders. "" is auto-detection's own
	// spelling of English (resolveDocLocale/machineryLocale's return value
	// for the canonical, un-prefixed manual) — treat it exactly like the
	// literal "en" the operator would type, not as an unresolved locale
	// request; logging "locale '' not available" for it would misreport an
	// auto-detected default as a rejected explicit ask.
	docsPrefix := ""
	if locale != "" && locale != "en" {
		if _, err := fs.Stat(dist, "locales/"+locale); err == nil {
			docsPrefix = "locales/" + locale + "/"
			log("locale: %s (docs/templates localized)", "locale: %s (docs/templates localizados)", locale)
			// Machinery already resolved this locale per file in step 1; every
			// overlay directory mirrors its English path exactly under
			// locales/<locale>/ (spec pose-localization-docs-contract).
		} else {
			log("locale '%s' not available — falling back to en", "locale '%s' indisponível — fallback para en", locale)
		}
	}
	replacer := strings.NewReplacer("{{PROJECT_NAME}}", projectName, "{{PROJECT_ID}}", projectID)
	for _, doc := range []string{"AGENTS.md", "POSE.md"} {
		dst := filepath.Join(target, doc)
		b, err := fs.ReadFile(dist, docsPrefix+doc)
		if err != nil {
			b, err = fs.ReadFile(dist, doc)
		}
		if err != nil {
			fmt.Fprintf(stderr, "pose install: %s: %v\n", doc, err)
			return 1
		}
		raw := string(b)
		if doc == "AGENTS.md" {
			// Excerpt the target's own README.md/CLAUDE.md into the
			// "Project context" placeholder before the {{PROJECT_NAME}}
			// token is substituted, so the injected summary still receives
			// it (spec pose-onboarding-context-extraction). A target with
			// neither file leaves raw unchanged.
			raw = injectExtractedProjectContext(raw, target)
		}
		content := replacer.Replace(raw)
		// An existing manual is merged, never skipped: engine-owned sections
		// refresh while the instance keeps what it wrote under the sections the
		// canonical manual tags as instance-owned. Skipping (the pre-existing
		// behavior) meant canonical manual content never reached an installed
		// repository at all; --force still resets the file wholesale.
		if existing, readErr := os.ReadFile(dst); readErr == nil && !force {
			// A rerun that switches locale (`pose install <target> --locale
			// pt-BR` over an English instance) hands the merge headings in a
			// different language than what `existing` already has: matching
			// them literally would treat every local section as unknown to
			// the engine and append it as a duplicate instead of recognizing
			// it as the same section (spec
			// pose-locale-switch-section-identity).
			var merged string
			var preserved, dropsContent bool
			existingResolved := resolveDocLocale(dist, doc, string(existing), "", false)
			targetResolved := strings.TrimSuffix(strings.TrimPrefix(docsPrefix, "locales/"), "/")
			if targetResolved != existingResolved {
				existingCanonicalPrefix := ""
				if existingResolved != "" {
					existingCanonicalPrefix = "locales/" + existingResolved + "/"
				}
				existingCanonical, ecErr := fs.ReadFile(dist, existingCanonicalPrefix+doc)
				var translation map[string]string
				if ecErr == nil {
					translation = buildHeadingTranslation(string(existingCanonical), content)
				}
				merged, preserved = MergeManagedDocAcrossLocale(content, string(existing), translation)
				dropsContent = MergeAcrossLocaleDropsLocalContent(content, string(existing), translation)
			} else {
				merged, preserved = MergeManagedDoc(content, string(existing))
				dropsContent = MergeDropsLocalContent(content, string(existing))
			}
			content = merged
			if string(existing) == content {
				log("unchanged: %s", "inalterado: %s", doc)
				continue
			}
			// Text the instance wrote inside an engine-owned section body has no
			// heading of its own, so the refresh overwrites it. Keep a copy
			// rather than lose it silently.
			if dropsContent && !noBackup {
				if err := os.WriteFile(dst+".pose-backup", existing, 0o644); err == nil {
					fmt.Fprintf(stderr, "[pose-install] backed up customized: %s → %s.pose-backup\n", doc, doc)
				}
			}
			if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
				fmt.Fprintf(stderr, "pose install: %v\n", err)
				return 1
			}
			if preserved {
				log("merged: %s (instance-owned sections preserved)", "mesclado: %s (seções da instância preservadas)", doc)
			} else {
				log("updated: %s", "atualizado: %s", doc)
			}
			continue
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			fmt.Fprintf(stderr, "pose install: %v\n", err)
			return 1
		}
		log("installed: %s", "instalado: %s", doc)
	}

	// 5. MCP points directly at the native binary; no shell wrapper.
	if !skipMCP {
		mcpJSON := filepath.Join(target, ".mcp.json")
		action, err := configureMCP(mcpJSON, target, projectID)
		if err != nil {
			fmt.Fprintf(stderr, "pose install: .mcp.json: %v\n", err)
			return 1
		}
		switch action {
		case "seeded":
			log("seeded: .mcp.json (server \"pose\")", "semente criada: .mcp.json (servidor \"pose\")")
		case "migrated":
			log("migrated: legacy MCP entry now uses native pose", "migrado: entrada MCP legada agora usa pose nativo")
		case "preserved":
			log("kept existing: .mcp.json (custom configuration)", "mantido existente: .mcp.json (configuração customizada)")
		}
	} else if skipMCP {
		log("MCP: skipped (--skip-mcp)", "MCP: ignorado (--skip-mcp)")
	}

	// 6. Schema stamp + instance dirs + final gate.
	svPath := filepath.Join(target, ".pose", "schema-version")
	if _, err := os.Stat(svPath); err != nil {
		_ = os.WriteFile(svPath, []byte(fmt.Sprintf("%d\n", nativeSchemaVersion)), 0o644)
		log("schema-version stamped: v%d", "schema-version gravado: v%d", nativeSchemaVersion)
	}
	if rc := cmdInit(target, io.Discard, stderr); rc != 0 {
		return rc
	}
	if rc := cmdIndex(target, nil, io.Discard, stderr); rc != 0 {
		return rc
	}
	log("running native final gate", "executando gate final nativo")
	// The operator's own --locale (or the auto-detected commandLocale) drives
	// this gate's report, not the shell's $LANG — an explicit `pose install
	// --locale en` reporting its own final gate in Portuguese because the
	// shell happens to be pt-BR would read as if the flag had failed
	// (spec pose-post-install-gate-locale).
	if rc := cmdCheckWithLocale(target, []string{"--strict"}, stdout, stderr, commandLocale); rc != 0 {
		fmt.Fprintln(stderr, text("pose install: post-install gate failed (check --strict)", "pose install: gate pós-instalação falhou (check --strict)"))
		if preExistingFailure {
			fmt.Fprintln(stderr, text(
				"pose install: this target already failed the same gate before this run touched anything (see the warning above) — this is very likely pre-existing governance debt in the target, not something this install/update caused.",
				"pose install: este alvo já falhava no mesmo gate antes desta execução tocar em qualquer coisa (veja o aviso acima) — isto é muito provavelmente dívida de governança pré-existente no alvo, não algo causado por este install/update.",
			))
		}
		// The gate runs after every machinery file, doc merge and policy
		// seed is already on disk — this command is not transactional, so
		// failing here does not undo any of it (spec
		// pose-install-gate-failure-recovery-notice, github.com/
		// oseiaspereira88/pose#18). Every overwritten *existing* file has a
		// .pose-backup copy unless --no-backup was passed; newly created
		// files have none but show up as untracked/modified in git status
		// on the git target install already requires (unless
		// --allow-non-git).
		fmt.Fprintln(stderr, text(
			"pose install: files were already written before this failure — this command does not roll back. Recover with the .pose-backup copies (unless --no-backup) and/or `git status`/`git diff` in the target.",
			"pose install: os arquivos já foram gravados antes desta falha — este comando não desfaz a operação. Recupere com as cópias .pose-backup (a menos que --no-backup tenha sido usado) e/ou `git status`/`git diff` no alvo.",
		))
		return 1
	}
	log("install complete — POSE is ready in %s", "instalação concluída — POSE pronto em %s", target)
	return 0
}

// recreateClaudeSkillSymlinks (re)links .claude/skills/* to their
// .agents/skills targets. Shared by cmdInstall and the doctor-guided
// "skills.symlinks" fix (spec pose-doctor-guided-remediation) — confined to
// .claude/skills and reversible (rerunning it is a no-op once healthy).
func recreateClaudeSkillSymlinks(target string) (linked bool, err error) {
	claudeDir := filepath.Join(target, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return false, err
	}
	linked = true
	for name, dest := range scaffold.ClaudeSkillLinks {
		link := filepath.Join(claudeDir, name)
		_ = os.Remove(link)
		if err := os.Symlink(dest, link); err != nil {
			linked = false
		}
	}
	return linked, nil
}

func configureMCP(path, target, projectID string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		payload := map[string]any{"mcpServers": map[string]any{"pose": nativeMCPEntry(target, projectID, nil)}}
		return "seeded", writeMCPJSON(path, payload)
	} else if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "preserved", nil
	}
	servers, ok := payload["mcpServers"].(map[string]any)
	if !ok {
		return "preserved", nil
	}
	changed := false
	for name, rawEntry := range servers {
		entry, ok := rawEntry.(map[string]any)
		if !ok || !legacyMCPEntry(name, entry) {
			continue
		}
		env, _ := entry["env"].(map[string]any)
		servers[name] = nativeMCPEntry(target, projectID, env)
		changed = true
	}
	if !changed {
		return "preserved", nil
	}
	return "migrated", writeMCPJSON(path, payload)
}

func legacyMCPEntry(name string, entry map[string]any) bool {
	if name != "pose" && name != "harne8-pose" {
		return false
	}
	return legacyMCPCommand(entry["command"])
}

func legacyMCPCommand(value any) bool {
	command, ok := value.(string)
	if !ok {
		return false
	}
	base := filepath.Base(filepath.Clean(command))
	return base == "pose-mcp-claude" || base == "pose-mcp"
}

func nativeMCPEntry(target, projectID string, existingEnv map[string]any) map[string]any {
	env := map[string]any{}
	for key, value := range existingEnv {
		env[key] = value
	}
	env["POSE_PROJECT_ROOT"] = target
	env["POSE_DEFAULT_PROJECT_ID"] = projectID
	return map[string]any{
		"type":    "stdio",
		"command": "pose",
		"args":    []string{"serve-mcp", "--stdio"},
		"env":     env,
	}
}

func writeMCPJSON(path string, payload map[string]any) error {
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(raw, '\n'), 0o644)
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func copyTreeInto(src fs.FS, sourceRoot, destinationRoot string, noBackup bool, stderr io.Writer, target string) error {
	return fs.WalkDir(src, sourceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(filepath.FromSlash(sourceRoot), filepath.FromSlash(path))
		if err != nil {
			return err
		}
		destination := filepath.Join(destinationRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		return copyFileWithBackup(src, path, destination, filePerm(path), noBackup, stderr, target)
	})
}

func copyFileWithBackup(src fs.FS, path, dst string, perm os.FileMode, noBackup bool, stderr io.Writer, target string) error {
	b, err := fs.ReadFile(src, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	if _, err := os.Stat(dst); err == nil {
		existing, err := os.ReadFile(dst)
		if err == nil {
			if bytes.Equal(existing, b) {
				return nil
			}
			if !noBackup {
				backupDst := dst + ".pose-backup"
				if err := os.WriteFile(backupDst, existing, perm); err == nil {
					relDst, _ := filepath.Rel(target, dst)
					fmt.Fprintf(stderr, "[pose-install] backed up customized: %s → %s.pose-backup\n", relDst, relDst)
				}
			}
		}
	}

	return os.WriteFile(dst, b, perm)
}

func copyFile(src fs.FS, path, dst string, perm os.FileMode) error {
	b, err := fs.ReadFile(src, path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, b, perm)
}

func filePerm(path string) os.FileMode {
	if strings.HasSuffix(path, ".sh") || path == "pose" || strings.HasSuffix(path, ".py") {
		return 0o755
	}
	return 0o644
}

func inTarget(dir string, fn func() int) int {
	old, err := os.Getwd()
	if err != nil {
		return 1
	}
	if err := os.Chdir(dir); err != nil {
		return 1
	}
	defer func() { _ = os.Chdir(old) }()
	return fn()
}
