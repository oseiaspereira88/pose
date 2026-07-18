package cli

// pose install — clone-free native installer (spec pose-cli-embed-standalone):
// installs the embedded POSE distribution into a target repository and seeds
// MCP configuration that invokes this same binary through `pose serve-mcp`.

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/crisol/pose-mcp/internal/scaffold"
)

func cmdInstall(args []string, stdout, stderr io.Writer) int {
	commandLocale := cliLocaleValue()
	text := func(english, portuguese string) string { return cliText(commandLocale, english, portuguese) }
	var target, projectName, projectID, locale string
	locale = "en"
	force, skipMCP, allowNonGit := false, false, false

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
		case "-h", "--help":
			fmt.Fprintln(stdout, text("Usage: pose install <target-dir> [--project-name n] [--project-id id] [--locale tag] [--force] [--skip-mcp] [--allow-non-git]", "Uso: pose install <target-dir> [--project-name n] [--project-id id] [--locale tag] [--force] [--skip-mcp] [--allow-non-git]"))
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
	if projectName == "" {
		projectName = filepath.Base(target)
	}
	if projectID == "" {
		projectID = "proj." + projectName
	}
	log := func(english, portuguese string, a ...any) {
		fmt.Fprintf(stdout, "[pose-install] "+text(english, portuguese)+"\n", a...)
	}
	log("target:       %s", "alvo:         %s", target)
	log("project name: %s", "nome do projeto: %s", projectName)
	log("project id:   %s", "id do projeto:   %s", projectID)

	dist := scaffold.Dist()

	// 1. Native machinery: this binary is the runtime.
	for _, dir := range []string{".pose/workflows", ".pose/rules", ".pose/templates", ".agents/skills"} {
		if err := copyTree(dist, dir, target); err != nil {
			fmt.Fprintf(stderr, text("pose install: merging %s: %v\n", "pose install: mesclando %s: %v\n"), dir, err)
			return 1
		}
		log("machinery (merged): %s", "maquinário (mesclado): %s", dir)
	}

	// .claude/skills symlinks (embed cannot carry them).
	claudeDir := filepath.Join(target, ".claude", "skills")
	if err := os.MkdirAll(claudeDir, 0o755); err == nil {
		linked := true
		for name, dest := range scaffold.ClaudeSkillLinks {
			link := filepath.Join(claudeDir, name)
			_ = os.Remove(link)
			if err := os.Symlink(dest, link); err != nil {
				linked = false
			}
		}
		if linked {
			log("machinery: .claude/skills (symlinks)", "maquinário: .claude/skills (symlinks)")
		} else {
			log("warning: could not create .claude/skills symlinks (unsupported filesystem?) — Claude Code discovery degraded", "aviso: não foi possível criar symlinks em .claude/skills (filesystem sem suporte?) — descoberta do Claude Code degradada")
		}
	}

	// 2. Config indexes: seed only when absent.
	idxEntries, _ := fs.ReadDir(dist, ".pose/indexes")
	_ = os.MkdirAll(filepath.Join(target, ".pose", "indexes"), 0o755)
	for _, e := range idxEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		dst := filepath.Join(target, ".pose", "indexes", e.Name())
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := copyFile(dist, ".pose/indexes/"+e.Name(), dst, 0o644); err == nil {
			log("index (seed): %s", "índice (semente): %s", e.Name())
		}
	}

	// 3. Legal texts vendored under .pose/.
	_ = copyFile(dist, "LICENSE", filepath.Join(target, ".pose", "LICENSE"), 0o644)
	_ = copyFile(dist, "NOTICE", filepath.Join(target, ".pose", "NOTICE"), 0o644)
	log("vendored: .pose/LICENSE, .pose/NOTICE", "incorporados: .pose/LICENSE, .pose/NOTICE")

	// 4. Root docs with locale + placeholders.
	docsPrefix := ""
	if locale != "en" {
		if _, err := fs.Stat(dist, "locales/"+locale); err == nil {
			docsPrefix = "locales/" + locale + "/"
			log("locale: %s (docs/templates localized)", "locale: %s (docs/templates localizados)", locale)
			if tmplEntries, err := fs.ReadDir(dist, "locales/"+locale+"/templates"); err == nil {
				for _, e := range tmplEntries {
					_ = copyFile(dist, "locales/"+locale+"/templates/"+e.Name(),
						filepath.Join(target, ".pose", "templates", e.Name()), 0o644)
				}
				log("machinery (locale override): .pose/templates", "maquinário (override de locale): .pose/templates")
			}
			for _, editorialDir := range []string{".pose/workflows", ".pose/rules", ".agents/skills"} {
				source := "locales/" + locale + "/" + editorialDir
				if _, err := fs.Stat(dist, source); err != nil {
					continue
				}
				if err := copyTreeInto(dist, source, filepath.Join(target, filepath.FromSlash(editorialDir))); err != nil {
					fmt.Fprintf(stderr, text("pose install: locale overlay %s: %v\n", "pose install: overlay de locale %s: %v\n"), editorialDir, err)
					return 1
				}
				log("machinery (locale override): %s", "maquinário (override de locale): %s", editorialDir)
			}
		} else {
			log("locale '%s' not available — falling back to en", "locale '%s' indisponível — fallback para en", locale)
		}
	}
	replacer := strings.NewReplacer("{{PROJECT_NAME}}", projectName, "{{PROJECT_ID}}", projectID)
	for _, doc := range []string{"AGENTS.md", "POSE.md"} {
		dst := filepath.Join(target, doc)
		if _, err := os.Stat(dst); err == nil && !force {
			log("kept existing: %s (use --force to overwrite)", "mantido existente: %s (use --force para sobrescrever)", doc)
			continue
		}
		b, err := fs.ReadFile(dist, docsPrefix+doc)
		if err != nil {
			b, err = fs.ReadFile(dist, doc)
		}
		if err != nil {
			fmt.Fprintf(stderr, "pose install: %s: %v\n", doc, err)
			return 1
		}
		if err := os.WriteFile(dst, []byte(replacer.Replace(string(b))), 0o644); err != nil {
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
	if rc := cmdCheck(target, []string{"--strict"}, stdout, stderr); rc != 0 {
		fmt.Fprintln(stderr, text("pose install: post-install gate failed (check --strict)", "pose install: gate pós-instalação falhou (check --strict)"))
		return 1
	}
	log("install complete — POSE is ready in %s", "instalação concluída — POSE pronto em %s", target)
	return 0
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
	if name != "pose" && name != "crisol-pose" {
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

func copyTree(src fs.FS, root, targetBase string) error {
	return fs.WalkDir(src, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dst := filepath.Join(targetBase, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copyFile(src, path, dst, filePerm(path))
	})
}

func copyTreeInto(src fs.FS, sourceRoot, destinationRoot string) error {
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
		return copyFile(src, path, destination, filePerm(path))
	})
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
