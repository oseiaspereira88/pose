package cli

// pose doctor — installation and instance diagnostics (spec pose-doctor,
// pose-doctor-guided-remediation). Diagnosis is always read-only; mutation
// only happens under --fix, defaults to a dry-run preview, and requires
// --yes to actually apply — never silently.

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
	"strconv"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/scaffold"
)

// doctorSchemaVersion versions doctor's own JSON output shape (distinct
// from the POSE instance schema-version). Existing fields are never
// renamed or removed across versions — only added — so consumers pinned to
// v1 keep working; bump this only for a genuinely breaking shape change.
const doctorSchemaVersion = 1

// remediationClass distinguishes what an operator (or an automated caller)
// can do about a finding (spec pose-doctor-guided-remediation R2):
//   - "n/a": informational, level is ok.
//   - "fixable": pose itself can apply a confined, reversible fix via
//     `pose doctor --fix`.
//   - "detectable": diagnosed with a concrete hint, but the fix is outside
//     doctor's confined-mutation scope (e.g. `pose upgrade`, a manual
//     `pose install`) and must be run explicitly by the operator.
//   - "blocked": outside POSE's control entirely (missing external tool).
const (
	remediationNA         = "n/a"
	remediationFixable    = "fixable"
	remediationDetectable = "detectable"
	remediationBlocked    = "blocked"
)

type doctorFinding struct {
	Check             string `json:"check"`
	Level             string `json:"level"` // ok | warn | error
	Message           string `json:"message"`
	Hint              string `json:"hint,omitempty"`
	Evidence          string `json:"evidence,omitempty"`
	RemediationClass  string `json:"remediation_class"`
	FixCode           string `json:"fix_code,omitempty"`
	DiagnosticScope   string `json:"diagnostic_scope,omitempty"`
	ConnectionChecked *bool  `json:"connection_checked,omitempty"`
}

// doctorFix is a confined, reversible, idempotent remediation action. Every
// entry here is safe to apply blind (no destructive or irreversible
// operation, never touches instance content) — that confinement is what
// lets `pose doctor --fix --yes` apply it without per-action confirmation
// once the operator has already opted into --yes.
type doctorFix struct {
	describe string
	apply    func(root string) error
}

var doctorFixRegistry = map[string]doctorFix{
	"hooks.pre-commit": {
		describe: "install the pre-commit hook (pose hooks install)",
		apply: func(root string) error {
			var out, errB bytes.Buffer
			if code := cmdHooks(root, []string{"install"}, &out, &errB); code != 0 {
				return fmt.Errorf("%s", strings.TrimSpace(errB.String()))
			}
			return nil
		},
	},
	"mcp.config": {
		describe: "regenerate .mcp.json to point at the native pose binary",
		apply: func(root string) error {
			projectID := "proj." + filepath.Base(root)
			_, err := configureMCP(filepath.Join(root, ".mcp.json"), root, projectID)
			return err
		},
	},
	"skills.symlinks": {
		describe: "recreate .claude/skills symlinks",
		apply: func(root string) error {
			linked, err := recreateClaudeSkillSymlinks(root)
			if err != nil {
				return err
			}
			if !linked {
				return fmt.Errorf("filesystem does not support symlinks")
			}
			return nil
		},
	},
}

// classifyFinding derives the guided-remediation class from the check code
// and level, so every call site of add() stays a plain (check, level,
// message, hint) tuple — the classification lives in one place.
func classifyFinding(check, level string) (class, fixCode string) {
	if level == "ok" {
		return remediationNA, ""
	}
	if _, ok := doctorFixRegistry[check]; ok {
		return remediationFixable, check
	}
	switch check {
	case "deps.git", "deps.go":
		return remediationBlocked, ""
	}
	return remediationDetectable, ""
}

// redactSecretShapedContent is the same offline, deterministic defense-in-depth scan
// pose skills-check applies to skill content (secretLikePatterns,
// skills_check.go) — doctor never prints a secret even if a future check
// starts echoing file content as evidence (security requirement: never
// print secrets).
func redactSecretShapedContent(s string) string {
	for _, re := range secretLikePatterns {
		s = re.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	text := func(english, portuguese string) string { return cliText(locale, english, portuguese) }
	jsonOut, fix, yes := false, false, false
	only := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--json":
			jsonOut = true
		case "--fix":
			fix = true
		case "--dry-run":
			// explicit alias for --fix's default (no-op if --yes not set)
		case "--yes":
			yes = true
		case "--only":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, text("Error: --only requires a check code", "Erro: --only exige um código de check"))
				return 2
			}
			i++
			only = args[i]
		case "-h", "--help":
			fmt.Fprintln(stdout, text(
				"Usage: pose doctor [--json] [--fix [--yes] [--only <check>]] — read-only diagnostics, optional guided remediation",
				"Uso: pose doctor [--json] [--fix [--yes] [--only <check>]] — diagnóstico read-only, remediação guiada opcional"))
			return 0
		default:
			fmt.Fprintf(stderr, text("Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), a)
			return 2
		}
	}
	if yes && !fix {
		fmt.Fprintln(stderr, text("Error: --yes requires --fix", "Erro: --yes exige --fix"))
		return 2
	}
	if only != "" && !fix {
		fmt.Fprintln(stderr, text("Error: --only requires --fix", "Erro: --only exige --fix"))
		return 2
	}
	if only != "" {
		if _, ok := doctorFixRegistry[only]; !ok {
			fmt.Fprintf(stderr, text("Error: %q is not a fixable check code (available: %s)\n", "Erro: %q não é um código de check corrigível (disponíveis: %s)\n"), only, strings.Join(fixableCodes(), ", "))
			return 2
		}
	}

	root, findings := runDoctorDiagnostics(locale)
	if !fix {
		return doctorReport(root, findings, jsonOut, stdout, locale)
	}

	var candidates []string
	for _, f := range findings {
		if f.RemediationClass != remediationFixable {
			continue
		}
		if only != "" && f.FixCode != only {
			continue
		}
		candidates = append(candidates, f.FixCode)
	}

	if !yes {
		return doctorFixPreview(findings, candidates, jsonOut, stdout, locale)
	}
	return doctorFixApply(root, findings, candidates, jsonOut, stdout, stderr, locale)
}

func fixableCodes() []string {
	codes := make([]string, 0, len(doctorFixRegistry))
	for c := range doctorFixRegistry {
		codes = append(codes, c)
	}
	return codes
}

// runDoctorDiagnostics is the read-only diagnostic pass, factored out so
// --fix --yes can call it again after applying fixes (recheck, R3).
func runDoctorDiagnostics(locale cliLocale) (root string, findings []doctorFinding) {
	text := func(english, portuguese string) string { return cliText(locale, english, portuguese) }
	add := func(check, level, message, hint string) {
		class, fixCode := classifyFinding(check, level)
		findings = append(findings, doctorFinding{
			Check:            check,
			Level:            level,
			Message:          redactSecretShapedContent(message),
			Hint:             redactSecretShapedContent(hint),
			Evidence:         redactSecretShapedContent(message),
			RemediationClass: class,
			FixCode:          fixCode,
		})
	}
	markStaticMCPConfig := func() {
		checked := false
		findings[len(findings)-1].DiagnosticScope = "static-configuration"
		findings[len(findings)-1].ConnectionChecked = &checked
	}

	// 1. Binary + toolchain deps.
	add("binary", "ok", fmt.Sprintf("pose %s", Version), "")
	if _, err := exec.LookPath("git"); err != nil {
		add("deps.git", "error", text("git not found in PATH", "git não encontrado no PATH"), text("install git; POSE uses it to resolve the project root", "instale git — o POSE resolve o root do projeto por ele"))
	} else {
		add("deps.git", "ok", text("git available", "git disponível"), "")
	}
	if _, err := exec.LookPath("go"); err != nil {
		add("deps.go", "warn", text("go not found (optional; needed only to rebuild MCP)", "go não encontrado (opcional: só para rebuild do MCP)"), "")
	} else {
		add("deps.go", "ok", text("go available", "go disponível"), "")
	}

	// 2. Instance.
	r, err := projectRoot()
	if err != nil {
		add("instance.root", "error", fmt.Sprintf(text("could not resolve root: %v", "root não resolvido: %v"), err), "")
		return "", findings
	}
	root = r
	add("instance.root", "ok", root, "")
	poseDir := filepath.Join(root, ".pose")
	if fi, err := os.Stat(poseDir); err != nil || !fi.IsDir() {
		add("instance.pose-dir", "error", text(".pose/ not found — POSE is not installed in this repository", ".pose/ ausente — este repo não tem POSE instalado"),
			text("run the POSE distribution install.sh", "rode o install.sh da distribuição POSE"))
		return root, findings
	}
	add("instance.pose-dir", "ok", text(".pose/ present", ".pose/ presente"), "")

	// 3. Native engine contract.
	add("engine.native", "ok", text("native Go engine active; no script runtime required", "motor Go nativo ativo; runtime de scripts não é necessário"), "")

	// 4. Schema version.
	svPath := filepath.Join(poseDir, "schema-version")
	engineVersion := engineSchemaVersion(root)
	if b, err := os.ReadFile(svPath); err != nil {
		add("schema.version", "warn", text("instance has no .pose/schema-version", "instância sem .pose/schema-version"), text("run 'pose update'", "rode 'pose update'"))
	} else {
		instance := strings.TrimSpace(string(b))
		n, convErr := strconv.Atoi(instance)
		switch {
		case convErr != nil:
			add("schema.version", "error", fmt.Sprintf(text("invalid schema-version: %q", "schema-version inválido: %q"), instance), text("run 'pose update'", "rode 'pose update'"))
		case engineVersion > 0 && n > engineVersion:
			add("schema.version", "error",
				fmt.Sprintf(text("instance v%d is newer than engine v%d", "instância v%d é mais nova que o motor v%d"), n, engineVersion),
				text("update the POSE engine; downgrade is unsupported", "atualize o motor POSE (não há downgrade)"))
		case engineVersion > 0 && n < engineVersion:
			add("schema.version", "warn",
				fmt.Sprintf(text("instance v%d is behind engine v%d", "instância v%d atrás do motor v%d"), n, engineVersion), text("run 'pose update'", "rode 'pose update'"))
		default:
			add("schema.version", "ok", "schema v"+instance, "")
		}
	}

	// 5. Skills symlinks (.claude/skills → .agents/skills).
	claudeSkills := filepath.Join(root, ".claude", "skills")
	if entries, err := os.ReadDir(claudeSkills); err == nil {
		broken := 0
		names := []string{}
		for _, e := range entries {
			link := filepath.Join(claudeSkills, e.Name())
			if fi, err := os.Lstat(link); err == nil && fi.Mode()&os.ModeSymlink != 0 {
				if _, err := os.Stat(link); err != nil {
					broken++
					names = append(names, e.Name())
				}
			}
		}
		if broken > 0 {
			add("skills.symlinks", "error", fmt.Sprintf(text("%d broken symlink(s) under .claude/skills: %s", "%d symlink(s) quebrado(s) em .claude/skills: %s"), broken, strings.Join(names, ", ")), text("rerun install.sh to recreate symlinks", "re-rode o install.sh para recriar os symlinks"))
		} else {
			add("skills.symlinks", "ok", text("skill symlinks are healthy", "symlinks de skills íntegros"), "")
		}
	}

	// 6. MCP uses the same native binary directly.
	if b, err := os.ReadFile(filepath.Join(root, ".mcp.json")); err != nil {
		add("mcp.config", "warn", text("static .mcp.json configuration not found", "configuração estática .mcp.json ausente"), text("run 'pose install', then restart/reconnect the client and call pose_mcp_context", "rode 'pose install', reinicie/reconecte o cliente e chame pose_mcp_context"))
		markStaticMCPConfig()
	} else if strings.Contains(string(b), `"command": "pose"`) {
		add("mcp.config", "ok", text("static MCP configuration points to the native pose binary", "configuração estática MCP aponta para o binário pose nativo"), text("active connection not checked; call pose_mcp_context after workspace or configuration changes", "conexão ativa não verificada; chame pose_mcp_context após mudar workspace ou configuração"))
		markStaticMCPConfig()
	} else {
		add("mcp.config", "warn", text("static MCP configuration does not use the native pose command", "configuração estática MCP não usa o comando pose nativo"), text("regenerate .mcp.json with 'pose install', restart/reconnect, then call pose_mcp_context", "regenere .mcp.json com 'pose install', reinicie/reconecte e chame pose_mcp_context"))
		markStaticMCPConfig()
	}

	// 7. Git hooks.
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	if _, err := os.Lstat(hook); err != nil {
		add("hooks.pre-commit", "warn", text("pre-commit hook not installed", "pre-commit não instalado"), text("run 'pose hooks install' for an automatic commit gate", "rode 'pose hooks install' para gate automático no commit"))
	} else {
		add("hooks.pre-commit", "ok", text("pre-commit hook installed", "pre-commit instalado"), "")
	}

	// 8. Delivery-integrity / artifact-contract policy roots (issue #17): a
	// v1.2.0 `pose install`/`pose update --force` could seed
	// .pose/policy/{delivery,artifacts}.json with roots copied verbatim from
	// pose-mcp's own source tree (fixed at the source in
	// internal/scaffold/distpolicy — see NeutralPolicyTemplates). An instance
	// that already seeded those files before the fix keeps the contaminated
	// content — `pose install`/`update` never overwrite an existing policy
	// file — so detect it here instead of relying on a fresh
	// install/update to self-heal.
	if dp, err := posemodel.LoadDeliveryPolicy(root); err == nil && dp.Enabled && len(dp.Roots) > 0 {
		var missing []string
		for _, r := range dp.Roots {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(r.Path))); err != nil {
				missing = append(missing, r.Path)
			}
		}
		if len(missing) == len(dp.Roots) {
			add("policy.delivery-roots", "warn",
				fmt.Sprintf(text("delivery policy roots do not exist in this project: %s", "raízes da política de entrega não existem neste projeto: %s"), strings.Join(missing, ", ")),
				text("check .pose/policy/delivery.json — roots may have been inherited from a contaminated pose-mcp v1.2.0 template (github.com/oseiaspereira88/pose issue #17); replace with this project's own module paths",
					".pose/policy/delivery.json — as roots podem ter vindo de um template pose-mcp v1.2.0 contaminado (issue #17); substitua pelos caminhos reais deste projeto"))
		} else {
			add("policy.delivery-roots", "ok", text("delivery policy roots resolve in this project", "raízes da política de entrega resolvem neste projeto"), "")
		}
	}
	if ap, err := posemodel.LoadArtifactPolicy(root); err == nil && ap.Enabled && len(ap.GovernedRoots) > 0 {
		var missing []string
		for _, p := range ap.GovernedRoots {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
				missing = append(missing, p)
			}
		}
		if len(missing) == len(ap.GovernedRoots) {
			add("policy.artifact-roots", "warn",
				fmt.Sprintf(text("artifact policy governed_roots do not exist in this project: %s", "governed_roots da política de artefatos não existem neste projeto: %s"), strings.Join(missing, ", ")),
				text("check .pose/policy/artifacts.json — governed_roots may have been inherited from a contaminated pose-mcp v1.2.0 template (github.com/oseiaspereira88/pose issue #17); replace with this project's own source directories",
					".pose/policy/artifacts.json — governed_roots podem ter vindo de um template pose-mcp v1.2.0 contaminado (issue #17); substitua pelos diretórios reais deste projeto"))
		} else {
			add("policy.artifact-roots", "ok", text("artifact policy governed_roots resolve in this project", "governed_roots da política de artefatos resolvem neste projeto"), "")
		}
	}

	// 8a. Instance config completeness. A `.pose/schema-version` stamp means
	// this instance considers itself current, so the subsystems every other
	// current instance has should exist too. Before spec
	// pose-upgrade-path-audit-fixes, a plain `pose update`
	// (without --force) refreshed AGENTS.md/POSE.md to reference these
	// subsystems without ever seeding them on an old instance, so `Result:
	// SUCCESS` could be followed immediately by `pose check --strict`
	// failing with broken references — and this same doctor run reported no
	// error. Detected here so an instance a stale binary already left in
	// that state surfaces it, not just prevented going forward.
	if _, err := os.Stat(filepath.Join(root, ".pose", "schema-version")); err == nil {
		var missing []string
		for _, rel := range []string{
			filepath.Join(".pose", "policy", "delivery.json"),
			filepath.Join(".pose", "policy", "artifacts.json"),
			filepath.Join(".pose", "indexes", "spec-graph.json"),
			filepath.Join(".pose", "indexes", "extensions.lock.json"),
		} {
			if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
				missing = append(missing, filepath.ToSlash(rel))
			}
		}
		if len(missing) > 0 {
			add("instance.config-completeness", "warn",
				fmt.Sprintf(text("subsystems referenced by this instance's manuals were never seeded: %s", "subsistemas referenciados pelos manuais desta instância nunca foram semeados: %s"), strings.Join(missing, ", ")),
				text("run 'pose update' to seed them (safe: additive-only, never overwrites an existing file)", "rode 'pose update' para semeá-los (seguro: só adiciona, nunca sobrescreve um arquivo existente)"))
		} else {
			add("instance.config-completeness", "ok", text("subsystems referenced by this instance's manuals are all seeded", "subsistemas referenciados pelos manuais desta instância estão todos semeados"), "")
		}
	}

	// 9. Review-scope recorded change set coverage (issue #17, comments 2-3):
	// `pose review bundle <scope> --seal` reads its subject exclusively from
	// graph.ChangeSets (`.pose/indexes/delivery-integrity.json`), which
	// `pose index` builds solely from change sets persisted via
	// `pose report --spec <slug> --change-from <rev> --change-to <rev>`
	// (loadRecordedChangeSets, reports/history/*.jsonl). A commit carrying
	// the `POSE-Spec: <slug>` trailer does NOT by itself make a spec
	// sealable — that trailer only feeds resolveGitChangeSet's live
	// auto-discovery fallback used by `pose artifact-check`/
	// `artifact-backfill --from-git` when called without --from/--to, which
	// is a different, ephemeral code path that never persists into the
	// index. (This corrects the original issue #17 investigation, which
	// attributed the review-bundle blocking to missing trailers; verified
	// empirically while implementing this check — spec
	// pose-review-scope-trailer-check Decision 2.) Warn before someone hits
	// "no immutable attributed change set exists" at seal time.
	if specs, err := (posemodel.Store{Root: root}).ListSpecs("", ""); err == nil {
		recorded := map[string]bool{}
		for _, set := range loadRecordedChangeSets(root) {
			if set.Spec != "" {
				recorded[set.Spec] = true
			}
		}
		if trailers, err := allCommitsWithSpecTrailers(root); err == nil {
			for spec := range trailers {
				recorded[spec] = true
			}
		}
		var untraceable []string
		for _, spec := range specs {
			if len(spec.Delivers) == 0 {
				continue
			}
			if spec.Status != "in-progress" && spec.Status != "done" {
				continue
			}
			if recorded[spec.Slug] {
				continue
			}
			untraceable = append(untraceable, spec.Slug)
		}
		if len(untraceable) > 0 {
			sort.Strings(untraceable)
			add("review.scope-change-set", "warn",
				fmt.Sprintf(text("%d spec(s) with delivers: have no change set recorded in .pose/reports/history or Git POSE-Spec trailers: %s",
					"%d spec(s) com delivers: não têm nenhum change set registrado em .pose/reports/history ou trailers Git POSE-Spec: %s"),
					len(untraceable), strings.Join(untraceable, ", ")),
				text("commit changes with a 'POSE-Spec: <slug>' trailer or run 'pose report --spec <slug> --change-from <rev> --change-to <rev>' to record the delivering range before sealing (github.com/oseiaspereira88/pose issue #17)",
					"comite as mudanças com o trailer 'POSE-Spec: <slug>' ou rode 'pose report --spec <slug> --change-from <rev> --change-to <rev>' para registrar o range antes de selar (issue #17)"))
		} else {
			add("review.scope-change-set", "ok", text("every in-progress/done spec with delivers: has an attributed change set", "toda spec em andamento/concluída com delivers: tem change set atribuído"), "")
		}
	}

	// 10. Retired machinery still on disk (spec
	// pose-domain-rule-extension-migration, ADR
	// retired-machinery-files-stay-on-disk-never-auto-migrated-by-pose-update):
	// deliverMachinery only ever walks the *current* engine source's
	// machineryRoots, so a file the engine used to ship but no longer does
	// (moved out to become an extension) is simply never revisited — it
	// stays on disk untouched and un-updated. That is the deliberate,
	// non-breaking compatibility strategy; this check is the discoverability
	// half of it, since nothing else would ever tell an instance the file it
	// still has is no longer receiving updates.
	if delivered := loadMachineryManifest(root); len(delivered) > 0 {
		current := map[string]bool{}
		for _, mroot := range machineryRoots {
			_ = fs.WalkDir(scaffold.Dist(), mroot, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				current[path] = true
				return nil
			})
		}
		var retired []string
		for path := range delivered {
			if current[path] {
				continue
			}
			if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(path))); statErr != nil {
				continue // instance already removed it — nothing to flag
			}
			retired = append(retired, path)
		}
		sort.Strings(retired)
		if len(retired) > 0 {
			hints := make([]string, 0, len(retired))
			for _, path := range retired {
				if rest, ok := strings.CutPrefix(path, ".pose/rules/"); ok {
					if slug, ok := strings.CutSuffix(rest, ".md"); ok {
						hints = append(hints, fmt.Sprintf("%s -> pose extension install pose-rule-%s", path, slug))
						continue
					}
				}
				hints = append(hints, path)
			}
			add("machinery.retired-on-disk", "warn",
				fmt.Sprintf(text("%d machinery file(s) present but no longer delivered by this engine version: %s",
					"%d arquivo(s) de machinery presente(s) mas não mais entregue(s) por esta versão do engine: %s"),
					len(retired), strings.Join(retired, ", ")),
				text("content is untouched and still valid, it just no longer receives engine updates; install the matching extension to resume updates: "+strings.Join(hints, "; "),
					"o conteúdo está intacto e continua válido, só não recebe mais atualizações do engine; instale a extensão correspondente para retomar: "+strings.Join(hints, "; ")))
		} else {
			add("machinery.retired-on-disk", "ok", text("no retired machinery content found", "nenhum conteúdo de machinery aposentado encontrado"), "")
		}
	}

	// 11. Stack-matched rule extension not installed (spec
	// pose-adaptive-rule-delivery): module-metadata.json already knows
	// each module's stack, which may resolve to a matching rule extension
	// (resolveRuleExtension) that is not installed. Advisory only — no
	// fetch/catalog mechanism exists to install it automatically
	// (Decision 1), so this only ever recommends by name.
	if raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json")); err == nil {
		var doc struct {
			Modules map[string]map[string]string `json:"modules"`
		}
		if json.Unmarshal(raw, &doc) == nil {
			var recommend []string
			paths := make([]string, 0, len(doc.Modules))
			for p := range doc.Modules {
				paths = append(paths, p)
			}
			sort.Strings(paths)
			for _, modPath := range paths {
				stack := doc.Modules[modPath]["domain"]
				if stack == "" {
					continue
				}
				extID, ok := resolveRuleExtension(root, modPath, stack)
				if !ok {
					continue
				}
				ruleFile, ok := ruleExtensionFile[extID]
				if !ok || ruleExtensionInstalled(root, ruleFile) {
					continue
				}
				recommend = append(recommend, fmt.Sprintf("%s (%s) -> pose extension install <path-to-%s>", modPath, stack, extID))
			}
			if len(recommend) > 0 {
				add("rules.stack-extension-available", "warn",
					fmt.Sprintf(text("%d module(s) match a rule extension not yet installed: %s",
						"%d módulo(s) combinam com uma extensão de rule ainda não instalada: %s"),
						len(recommend), strings.Join(recommend, "; ")),
					text("obtain the extension package and run 'pose extension install <path>' — see AGENTS.md's Domain rules section",
						"obtenha o pacote da extensão e rode 'pose extension install <path>' — veja a seção Domain rules do AGENTS.md"))
			} else {
				add("rules.stack-extension-available", "ok", text("no unmatched rule extensions for detected modules", "nenhuma extensão de rule pendente para os módulos detectados"), "")
			}

			// 11a. Orphaned module-metadata entries (specs
			// pose-upgrade-path-audit-fixes,
			// pose-stack-detection-consolidation): discovery is additive-only
			// and never removes an entry, so a path that stopped existing —
			// a stale self-referential entry from an old scaffold version, a
			// typo/case mismatch, a module that was deleted — stays forever,
			// silently. Root ("") is exempt: it is a valid module-metadata
			// key that legitimately has no directory of its own to stat.
			var orphans []string
			modPaths := make([]string, 0, len(doc.Modules))
			for p := range doc.Modules {
				modPaths = append(modPaths, p)
			}
			sort.Strings(modPaths)
			for _, modPath := range modPaths {
				if modPath == "" || modPath == "." {
					continue
				}
				if !confinedRelativePath(modPath) {
					continue
				}
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(modPath))); err != nil {
					orphans = append(orphans, modPath)
				}
			}
			if len(orphans) > 0 {
				add("module-metadata.orphan-entries", "warn",
					fmt.Sprintf(text("%d module-metadata.json entr(y/ies) do not resolve to a directory in this project: %s",
						"%d entrada(s) de module-metadata.json não resolvem a um diretório neste projeto: %s"),
						len(orphans), strings.Join(orphans, ", ")),
					text("check .pose/indexes/module-metadata.json — the module may have been deleted, renamed, or (github.com/oseiaspereira88/pose issue #21-adjacent) inherited from an old scaffold that leaked its own modules; remove the entry or fix the path",
						"confira .pose/indexes/module-metadata.json — o módulo pode ter sido apagado, renomeado, ou (adjacente à issue #21) herdado de um scaffold antigo que vazou seus próprios módulos; remova a entrada ou corrija o caminho"))
			} else {
				add("module-metadata.orphan-entries", "ok", text("every module-metadata.json entry resolves to a real directory", "toda entrada de module-metadata.json resolve a um diretório real"), "")
			}
		}
	}

	// 12. Redundant root-plus-child monorepo execution (spec
	// pose-monorepo-validation-advisory): a root module whose configured
	// checks delegate to workspace members (npm's --workspaces, Cargo's
	// --workspace) alongside child modules POSE also validates directly
	// runs those children's tests twice. The fix already exists —
	// moduleOverrides.<path>.replaceDefaultChecks — this is purely
	// discoverability, never an automatic skip (spec Decision 1).
	if raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json")); err == nil {
		if matrix, err := parseValidationMatrix(raw); err == nil {
			if modules, err := discoverValidationModules(root); err == nil {
				rootStack := ""
				var children []string
				for _, m := range modules {
					if m.Rel == "." {
						rootStack = m.Stack
					} else {
						children = append(children, m.Rel)
					}
				}
				delegates := false
				// The delegation flag (npm's --workspaces) typically lives
				// inside the root package.json's own "scripts" values, not
				// in POSE's generic per-stack check args — `npm test
				// --if-present` (POSE's own invocation) reads
				// scripts.test, and THAT string is what actually contains
				// "--workspaces". Checking POSE's own check definitions
				// alone (as an earlier version of this check did) never
				// finds it, since those are stack-generic and never
				// contain the flag themselves.
				if rootStack == "node" {
					if raw, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
						var pkg struct {
							Scripts map[string]string `json:"scripts"`
						}
						if json.Unmarshal(raw, &pkg) == nil {
							for _, script := range pkg.Scripts {
								if strings.Contains(script, "--workspace") {
									delegates = true
								}
							}
						}
					}
				}
				// Cargo has no scripts indirection layer: a workspace
				// delegation would appear directly in POSE's own
				// configured check args for the root (moduleOverrides["."]
				// or the generic "rust" stack, if customized to invoke
				// `cargo test --workspace`).
				rootChecks := matrix.Stacks[rootStack].Checks
				if override, ok := matrix.ModuleOverrides["."]; ok {
					rootChecks = override.Checks
				}
				for _, c := range rootChecks {
					for _, a := range c.Args {
						if strings.Contains(a, "--workspace") {
							delegates = true
						}
					}
				}
				var flagged []string
				if delegates {
					for _, childRel := range children {
						if o, ok := matrix.ModuleOverrides[childRel]; ok && o.ReplaceDefaultChecks {
							continue // already resolved
						}
						flagged = append(flagged, childRel)
					}
					sort.Strings(flagged)
				}
				if len(flagged) > 0 {
					examples := make([]string, 0, len(flagged))
					for _, rel := range flagged {
						examples = append(examples, fmt.Sprintf(`"%s": {"replaceDefaultChecks": true, "checks": []}`, rel))
					}
					add("validate.redundant-workspace-execution", "warn",
						fmt.Sprintf(text("root module's checks delegate to workspace members (--workspace flag) while %d child module(s) are also validated directly, likely running the same tests twice: %s",
							"os checks do módulo raiz delegam para os membros do workspace (flag --workspace) enquanto %d módulo(s) filho(s) também são validados diretamente, provavelmente rodando os mesmos testes duas vezes: %s"),
							len(flagged), strings.Join(flagged, ", ")),
						text("add to .pose/indexes/validation-matrix.json's moduleOverrides to stop the direct run and keep only the root's delegated one: "+strings.Join(examples, ", "),
							"adicione em moduleOverrides de .pose/indexes/validation-matrix.json para parar a execução direta e manter só a delegada pelo root: "+strings.Join(examples, ", ")))
				} else {
					add("validate.redundant-workspace-execution", "ok", text("no redundant root-plus-child execution detected", "nenhuma execução redundante raiz+filho detectada"), "")
				}
			}
		}
	}

	// 12. Invisible governance failures (spec
	// pose-diagnose-invisible-governance-failures). Both of the below are cases
	// where POSE silently reduces what it knows and reports a downstream symptom
	// instead of the upstream loss, so the operator ends up reading the engine's
	// source to find a cause POSE already had.

	// 12a. A review profile may name an evidence class that `pose validate`
	// refuses to register on a check. Nothing cross-validates the two
	// vocabularies, so the divergence only shows up as a bundle resolving
	// evidence=0, with no indication that the class was unsatisfiable by
	// construction.
	//
	// Only the profiles policy actually selects. A profile sitting on disk that
	// no scope resolves to and no overlay list names builds no plan, so
	// reporting it is noise an operator has to investigate to dismiss — the case
	// that appears the moment a project owns its profiles and leaves the shipped
	// ones in place.
	if policy, policyErr := (posemodel.Store{Root: root}).GetReviewPolicy(); policyErr == nil {
		selected := map[string]bool{}
		for _, ref := range policy.Profiles {
			selected[strings.SplitN(ref, "@", 2)[0]] = true
		}
		for _, ref := range policy.OverlayProfiles {
			selected[strings.SplitN(ref, "@", 2)[0]] = true
		}
		unreachable := map[string][]string{}
		for id := range selected {
			if id == "" {
				continue
			}
			entryName := id + ".json"
			raw, readErr := os.ReadFile(filepath.Join(root, ".pose", "review-profiles", entryName))
			if readErr != nil {
				continue
			}
			entry := struct{ name string }{entryName}
			var profile struct {
				Criteria []struct {
					EvidenceClasses []string `json:"evidence_classes"`
				} `json:"criteria"`
				Tools []struct {
					EvidenceClasses []string `json:"evidence_classes"`
				} `json:"tools"`
			}
			if json.Unmarshal(raw, &profile) != nil {
				continue
			}
			seen := map[string]bool{}
			for _, criterion := range profile.Criteria {
				for _, class := range criterion.EvidenceClasses {
					if !posemodel.ValidEvidenceClasses[class] && !seen[class] {
						seen[class] = true
						unreachable[entry.name] = append(unreachable[entry.name], class)
					}
				}
			}
			for _, tool := range profile.Tools {
				for _, class := range tool.EvidenceClasses {
					if !posemodel.ValidEvidenceClasses[class] && !seen[class] {
						seen[class] = true
						unreachable[entry.name] = append(unreachable[entry.name], class)
					}
				}
			}
		}
		if len(unreachable) > 0 {
			names := make([]string, 0, len(unreachable))
			for file, classes := range unreachable {
				sort.Strings(classes)
				names = append(names, fmt.Sprintf("%s (%s)", file, strings.Join(classes, ", ")))
			}
			sort.Strings(names)
			add("review.evidence-vocabulary", "warn",
				fmt.Sprintf(text("%d selected review profile(s) demand evidence classes no registered check may emit: %s",
					"%d review profile(s) selecionado(s) exigem classes de evidência que nenhum check registrado pode emitir: %s"),
					len(unreachable), strings.Join(names, "; ")),
				text("pose validate only accepts the classes in the delivery contract's vocabulary; a criterion demanding another one can never be satisfied by a real check, leaving auto-attest as the only path that completes — reconcile the profile or register a check that emits the class",
					"o pose validate só aceita as classes do vocabulário do contrato de entrega; um critério que exige outra jamais é satisfeito por check real, e sobra o auto-attest como único caminho que completa — reconcilie o profile ou registre um check que emita a classe"))
		} else {
			add("review.evidence-vocabulary", "ok", text("every evidence class the selected review profiles demand can be emitted by a check", "toda classe de evidência exigida pelos review profiles selecionados pode ser emitida por um check"), "")
		}

		// 12f. The vocabulary check above asks whether a class *could* be
		// emitted by some check somewhere. This asks whether one actually is,
		// here. The two came apart when `go vet` moved from `build` to `lint`:
		// `build` stayed a perfectly valid class, the profiles kept demanding
		// it, and no registered check produced it any more — a gate depending
		// on evidence this repository does not generate, with nothing saying so
		// (spec pose-report-a-demanded-class-nothing-produces).
		// A criterion's `evidence_classes` is a disjunction: the attestation cites
		// one of them, and the first match satisfies it. So the unit that can be
		// unsatisfiable is the criterion, not the class — reporting classes one
		// by one said `e2e` was unproduced while every criterion listing it also
		// accepted `integration` or `unit`, both of which are produced. A true
		// statement that names nothing to fix, which is how a check earns being
		// ignored (spec pose-class-producers-reads-the-disjunction).
		type classDemand struct {
			where   string
			classes []string
		}
		// A profile selected by a language this repository does not have can never
		// be selected for any component, so its criteria are not ones anyone here
		// will be held to. Reporting them is the second kind of false alarm this
		// check had: `frontend-review` demands `a11y`, and there is no JavaScript
		// or TypeScript module for it to apply to.
		//
		// Only the language selector is evaluated. The others need the
		// component-aware planner, and a partial evaluation can only make this
		// quieter, never wrongly loud: a profile is skipped solely when it names
		// languages and none of them is present.
		domains := map[string]bool{}
		if raw, metaErr := os.ReadFile(filepath.Join(root, ".pose", "indexes", "module-metadata.json")); metaErr == nil {
			var metadata struct {
				Modules map[string]struct {
					Domain string `json:"domain"`
				} `json:"modules"`
			}
			if json.Unmarshal(raw, &metadata) == nil {
				for _, module := range metadata.Modules {
					if module.Domain != "" {
						domains[module.Domain] = true
					}
				}
			}
		}
		demands := []classDemand{}
		for id := range selected {
			if id == "" {
				continue
			}
			raw, readErr := os.ReadFile(filepath.Join(root, ".pose", "review-profiles", id+".json"))
			if readErr != nil {
				continue
			}
			var profile struct {
				Selectors struct {
					Languages []string `json:"languages"`
				} `json:"selectors"`
				Criteria []struct {
					ID              string   `json:"id"`
					EvidenceClasses []string `json:"evidence_classes"`
				} `json:"criteria"`
				Tools []struct {
					ID              string   `json:"id"`
					EvidenceClasses []string `json:"evidence_classes"`
				} `json:"tools"`
			}
			if json.Unmarshal(raw, &profile) != nil {
				continue
			}
			if len(profile.Selectors.Languages) > 0 && len(domains) > 0 {
				applies := false
				for _, language := range profile.Selectors.Languages {
					if domains[language] {
						applies = true
						break
					}
				}
				if !applies {
					continue
				}
			}
			for _, criterion := range profile.Criteria {
				if len(criterion.EvidenceClasses) > 0 {
					demands = append(demands, classDemand{id + "/" + criterion.ID, criterion.EvidenceClasses})
				}
			}
			for _, tool := range profile.Tools {
				if len(tool.EvidenceClasses) > 0 {
					demands = append(demands, classDemand{id + "/tool:" + tool.ID, tool.EvidenceClasses})
				}
			}
		}
		if len(demands) > 0 {
			produced := map[string]bool{}
			if raw, matrixErr := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json")); matrixErr == nil {
				if matrix, parseErr := parseValidationMatrix(raw); parseErr == nil {
					for _, stack := range matrix.Stacks {
						for _, check := range stack.Checks {
							if class := strings.TrimSpace(check.EvidenceClass); class != "" {
								produced[class] = true
							}
						}
					}
					for _, override := range matrix.ModuleOverrides {
						for _, check := range override.Checks {
							if class := strings.TrimSpace(check.EvidenceClass); class != "" {
								produced[class] = true
							}
						}
					}
				}
			}
			unsatisfiable := []string{}
			for _, demand := range demands {
				satisfiable := false
				for _, class := range demand.classes {
					if produced[class] {
						satisfiable = true
						break
					}
				}
				if !satisfiable {
					unsatisfiable = append(unsatisfiable, demand.where+" ("+strings.Join(demand.classes, "|")+")")
				}
			}
			sort.Strings(unsatisfiable)
			if len(unsatisfiable) > 0 {
				add("validate.class-producers", "warn",
					fmt.Sprintf(text("%d review criterion/criteria demand evidence no registered check produces: %s",
						"%d critério(s) de review exigem evidência que nenhum check registrado produz: %s"),
						len(unsatisfiable), strings.Join(unsatisfiable, ", ")),
					text("none of the classes a criterion accepts is emitted here, so it can only be satisfied by evidence this repository does not generate — register a check that emits one of them in .pose/indexes/validation-matrix.json, or reconcile the profile",
						"nenhuma das classes que o critério aceita é emitida aqui, então ele só é satisfeito por evidência que este repositório não gera — registre em .pose/indexes/validation-matrix.json um check que emita uma delas, ou reconcilie o profile"))
			} else {
				add("validate.class-producers", "ok",
					text("every review criterion the selected profiles declare accepts a class some registered check produces", "todo critério de review declarado pelos profiles selecionados aceita uma classe que algum check registrado produz"), "")
			}
		}
	}

	// 12c. `.pose/policy/` is not machinery, so an update delivers a stricter
	// engine and never a statement of when the instance received it. Without a
	// date, rules introduced after a closeout was recorded judge it anyway, and
	// an operator sees a wall of failures about work nobody touched. The
	// contracts come from the registry, so a future one is reported here
	// without this block learning about it.
	if raw, err := os.ReadFile(filepath.Join(root, ".pose", "policy", "review.json")); err == nil {
		var policy posemodel.ReviewPolicy
		if json.Unmarshal(raw, &policy) == nil {
			// Legacy attempts live in `.pose/reviews/*.md` — the directory
			// Store.ListReviewAttempts reads. An earlier version of this check
			// globbed `.pose/review-attempts/`, which nothing writes, so the
			// pre-bundle instance it exists to diagnose always counted zero and
			// got an ok. Its test seeded an attestation, which does exist, and
			// passed over the dead path.
			attestations, _ := filepath.Glob(filepath.Join(root, ".pose", "review-attestations", "*.json"))
			attempts, _ := filepath.Glob(filepath.Join(root, ".pose", "reviews", "*.md"))
			history := len(attestations) + len(attempts)
			unrecorded := []string{}
			summaries := map[string]string{}
			for _, contract := range posemodel.ReviewContracts() {
				summaries[contract.ID] = contract.Summary
				if !policy.ContractAdoptionRecorded(contract.ID) {
					unrecorded = append(unrecorded, contract.ID)
				}
			}
			sort.Strings(unrecorded)
			switch {
			case len(unrecorded) == 0:
				add("review.contract-adoption", "ok",
					text("every governance contract records the date this instance received it",
						"todo contrato de governança registra a data em que esta instância o recebeu"), "")
			case history == 0:
				add("review.contract-adoption", "ok",
					text("no recorded review predates a governance contract", "nenhuma review registrada precede um contrato de governança"), "")
			default:
				details := make([]string, 0, len(unrecorded))
				for _, id := range unrecorded {
					details = append(details, fmt.Sprintf("%s (%s)", id, summaries[id]))
				}
				add("review.contract-adoption", "warn",
					fmt.Sprintf(text("%d recorded review(s) may predate %d governance contract(s) this instance records no adoption date for: %s",
						"%d review(s) registrada(s) pode(m) preceder %d contrato(s) de governança sem data de adoção nesta instância: %s"),
						history, len(unrecorded), strings.Join(details, "; ")),
					text("a review recorded before a contract existed fails it without having become less considered — `pose update` stamps today's date into contract_adoptions, or set it by hand to the day this instance received the contract; an explicitly empty value means judge my whole history by the current contract",
						"uma review registrada antes de um contrato existir reprova sem ter se tornado menos criteriosa — o `pose update` carimba a data de hoje em contract_adoptions, ou defina à mão o dia em que esta instância recebeu o contrato; um valor explicitamente vazio significa julgue todo o meu histórico pelo contrato atual"))
			}
		}
	}

	// 12e. A review profile left at schema v1 is exempt from the closed rule and
	// evidence catalogs v2 enforces — the check that stops a profile demanding a
	// class no check may emit. Two profiles shipped at v1 for long enough that
	// every instance installed in that window still carries them, and the
	// migration that should have moved them reported success without doing so
	// (spec pose-doctor-reports-a-profile-left-behind).
	if entries, err := os.ReadDir(filepath.Join(root, ".pose", "review-profiles")); err == nil {
		behind := []string{}
		examined := 0
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			raw, readErr := os.ReadFile(filepath.Join(root, ".pose", "review-profiles", entry.Name()))
			if readErr != nil {
				continue
			}
			var profile struct {
				SchemaVersion int `json:"schema_version"`
			}
			if json.Unmarshal(raw, &profile) != nil {
				continue
			}
			examined++
			if profile.SchemaVersion < posemodel.ReviewPolicySchemaVersion {
				behind = append(behind, fmt.Sprintf("%s (v%d)", entry.Name(), profile.SchemaVersion))
			}
		}
		sort.Strings(behind)
		if len(behind) > 0 {
			add("review.profile-schema", "warn",
				fmt.Sprintf(text("%d review profile(s) are below schema v%d: %s",
					"%d review profile(s) abaixo do schema v%d: %s"), len(behind), posemodel.ReviewPolicySchemaVersion, strings.Join(behind, ", ")),
				text("a schema-v1 profile is exempt from the closed rule and evidence catalogs, so it can demand a class no check may emit — run `pose update` to migrate it",
					"um profile schema-v1 fica isento dos catálogos fechados de rules e classes, então pode exigir uma classe que nenhum check emite — rode `pose update` para migrá-lo"))
		} else if examined > 0 {
			add("review.profile-schema", "ok",
				fmt.Sprintf(text("every review profile is at schema v%d", "todo review profile está no schema v%d"), posemodel.ReviewPolicySchemaVersion), "")
		}
	}

	// 12d. Every policy decoder ignores keys it does not know, so a newer field
	// never breaks an older binary reading the same repository. What that gives
	// up is telling an operator that a key they wrote is not one the engine
	// reads: a misspelling takes the default and the setting silently does
	// nothing. Reported here, where it costs a finding instead of a refusal.
	//
	// The review policy was the first to get this, because that is where the
	// strict decoder had to be dropped. The exposure is identical in every
	// other policy the engine models, and identical is the point: an operator
	// who learns the finding exists for one file has no way to know it does not
	// exist for the next.
	for _, policy := range policyKeyChecks() {
		var raw []byte
		var err error
		for _, candidate := range policy.paths() {
			if raw, err = os.ReadFile(filepath.Join(root, ".pose", "policy", candidate)); err == nil {
				break
			}
		}
		if err != nil {
			continue
		}
		var document map[string]json.RawMessage
		if json.Unmarshal(raw, &document) != nil {
			continue
		}
		known := map[string]bool{}
		for _, key := range policy.known {
			known[key] = true
		}
		unknown := []string{}
		for key := range document {
			if !known[key] && !posemodel.PolicyKeyIsAnnotation(key) {
				unknown = append(unknown, key)
			}
		}
		sort.Strings(unknown)
		if len(unknown) > 0 {
			add(policy.check, "warn",
				fmt.Sprintf(text("the %s policy carries %d key(s) this engine does not read: %s",
					"a política de %s carrega %d chave(s) que este engine não lê: %s"), policy.label, len(unknown), strings.Join(unknown, ", ")),
				fmt.Sprintf(text("a key the engine does not model is ignored, so its setting has no effect — check the spelling, or remove it; the keys read are: %s",
					"uma chave que o engine não modela é ignorada, então o ajuste não tem efeito — confira a grafia, ou remova; as chaves lidas são: %s"),
					strings.Join(policy.known, ", ")))
		} else {
			add(policy.check, "ok",
				fmt.Sprintf(text("every key in the %s policy is one this engine reads", "toda chave da política de %s é lida por este engine"), policy.label), "")
		}
	}

	// 12b. A check with no evidenceClass still runs and still passes, but its
	// result carries no class, so review evidence collection discards it. The
	// module looks covered and contributes nothing.
	if raw, err := os.ReadFile(filepath.Join(root, ".pose", "indexes", "validation-matrix.json")); err == nil {
		if matrix, parseErr := parseValidationMatrix(raw); parseErr == nil {
			unclassed := []string{}
			for name, override := range matrix.ModuleOverrides {
				for _, check := range override.Checks {
					if strings.TrimSpace(check.EvidenceClass) == "" {
						unclassed = append(unclassed, name+"/"+check.Name)
					}
				}
			}
			for name, stack := range matrix.Stacks {
				for _, check := range stack.Checks {
					if strings.TrimSpace(check.EvidenceClass) == "" {
						unclassed = append(unclassed, "stack:"+name+"/"+check.Name)
					}
				}
			}
			sort.Strings(unclassed)
			if len(unclassed) > 0 {
				add("validate.evidence-class-coverage", "warn",
					fmt.Sprintf(text("%d registered check(s) declare no evidenceClass, so their results are discarded when a review collects evidence: %s",
						"%d check(s) registrado(s) não declaram evidenceClass, então os resultados deles são descartados quando uma review coleta evidência: %s"),
						len(unclassed), strings.Join(unclassed, ", ")),
					text("declare the class each check actually produces in .pose/indexes/validation-matrix.json — a passing check with no class leaves the module looking covered while contributing nothing to a review",
						"declare em .pose/indexes/validation-matrix.json a classe que cada check de fato produz — um check verde sem classe deixa o módulo parecendo coberto sem contribuir nada para uma review"))
			} else {
				add("validate.evidence-class-coverage", "ok", text("every registered check declares an evidence class", "todo check registrado declara uma classe de evidência"), "")
			}
		}
	}

	return root, findings
}

func engineSchemaVersion(root string) int {
	return nativeSchemaVersion
}

func countLevels(findings []doctorFinding) (errors, warns int) {
	for _, f := range findings {
		switch f.Level {
		case "error":
			errors++
		case "warn":
			warns++
		}
	}
	return
}

func renderFindingsText(findings []doctorFinding, stdout io.Writer, locale cliLocale) {
	for _, f := range findings {
		icon := map[string]string{"ok": "✓", "warn": "!", "error": "✗"}[f.Level]
		suffix := ""
		if f.RemediationClass == remediationFixable {
			suffix = cliText(locale, " (fixable: pose doctor --fix)", " (corrigível: pose doctor --fix)")
		}
		fmt.Fprintf(stdout, "[%s] %-18s %s%s\n", icon, f.Check, f.Message, suffix)
		if f.Hint != "" {
			fmt.Fprintf(stdout, "      ↳ %s\n", f.Hint)
		}
	}
}

func printSummaryLine(errors, warns int, stdout io.Writer, locale cliLocale) {
	fmt.Fprintf(stdout, cliText(locale, "\ndoctor: %d error(s), %d warning(s)\n", "\ndoctor: %d erro(s), %d aviso(s)\n"), errors, warns)
}

func doctorReport(root string, findings []doctorFinding, jsonOut bool, stdout io.Writer, locale cliLocale) int {
	errors, warns := countLevels(findings)
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"doctor_schema_version": doctorSchemaVersion,
			"findings":              findings,
			"errors":                errors,
			"warnings":              warns,
		})
	} else {
		renderFindingsText(findings, stdout, locale)
		printSummaryLine(errors, warns, stdout, locale)
		PrintContributorDoctorHint(root, stdout, locale)
	}
	if errors > 0 {
		return 1
	}
	return 0
}

type doctorFixCandidate struct {
	Check    string `json:"check"`
	Describe string `json:"describe"`
}

// doctorFixPreview is the default `pose doctor --fix` behavior: report the
// diagnostics, then list what each fixable finding *would* do — no
// mutation (constraint: default to advice or dry-run).
func doctorFixPreview(findings []doctorFinding, candidates []string, jsonOut bool, stdout io.Writer, locale cliLocale) int {
	errors, warns := countLevels(findings)
	planned := make([]doctorFixCandidate, 0, len(candidates))
	for _, c := range candidates {
		planned = append(planned, doctorFixCandidate{c, doctorFixRegistry[c].describe})
	}
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"doctor_schema_version": doctorSchemaVersion,
			"findings":              findings,
			"errors":                errors,
			"warnings":              warns,
			"fix": map[string]any{
				"mode":       "dry-run",
				"candidates": planned,
			},
		})
	} else {
		renderFindingsText(findings, stdout, locale)
		printSummaryLine(errors, warns, stdout, locale)
		if len(candidates) == 0 {
			fmt.Fprintln(stdout, cliText(locale, "\nfix: nothing fixable right now.", "\nfix: nada corrigível no momento."))
		} else {
			fmt.Fprintln(stdout, cliText(locale, "\n[DRY-RUN] would apply:", "\n[DRY-RUN] aplicaria:"))
			for _, c := range planned {
				fmt.Fprintf(stdout, "  - %s: %s\n", c.Check, c.Describe)
			}
			fmt.Fprintln(stdout, cliText(locale, "Result: DRY-RUN — no changes applied. Re-run with --fix --yes to apply.", "Resultado: DRY-RUN — nenhuma mudança aplicada. Rode com --fix --yes para aplicar."))
		}
	}
	if errors > 0 {
		return 1
	}
	return 0
}

type doctorFixResult struct {
	Check  string `json:"check"`
	Status string `json:"status"` // applied | error
	Detail string `json:"detail,omitempty"`
}

// doctorFixApply applies every candidate fix, then reruns diagnostics
// (recheck, R3) and reports whether each targeted check now reads ok —
// applying it twice in a row is a no-op by construction, since every
// registered fix is idempotent.
func doctorFixApply(root string, before []doctorFinding, candidates []string, jsonOut bool, stdout, stderr io.Writer, locale cliLocale) int {
	if len(candidates) == 0 {
		errors, warns := countLevels(before)
		if jsonOut {
			enc := json.NewEncoder(stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]any{
				"doctor_schema_version": doctorSchemaVersion,
				"findings":              before,
				"errors":                errors,
				"warnings":              warns,
				"fix":                   map[string]any{"mode": "apply", "results": []doctorFixResult{}},
			})
		} else {
			renderFindingsText(before, stdout, locale)
			printSummaryLine(errors, warns, stdout, locale)
			fmt.Fprintln(stdout, cliText(locale, "\nfix: nothing fixable right now.", "\nfix: nada corrigível no momento."))
		}
		return 0
	}

	var results []doctorFixResult
	for _, c := range candidates {
		if err := doctorFixRegistry[c].apply(root); err != nil {
			results = append(results, doctorFixResult{c, "error", redactSecretShapedContent(err.Error())})
		} else {
			results = append(results, doctorFixResult{c, "applied", ""})
		}
	}

	_, after := runDoctorDiagnostics(locale)
	stillBroken := map[string]bool{}
	for _, f := range after {
		if f.Level != "ok" {
			stillBroken[f.Check] = true
		}
	}
	allResolved := true
	for _, c := range candidates {
		if stillBroken[c] {
			allResolved = false
		}
	}

	errors, warns := countLevels(after)
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{
			"doctor_schema_version": doctorSchemaVersion,
			"findings":              after,
			"errors":                errors,
			"warnings":              warns,
			"fix": map[string]any{
				"mode":    "apply",
				"results": results,
			},
		})
	} else {
		renderFindingsText(after, stdout, locale)
		printSummaryLine(errors, warns, stdout, locale)
		fmt.Fprintln(stdout, cliText(locale, "\n[APPLIED]", "\n[APLICADO]"))
		for _, r := range results {
			detail := ""
			if r.Detail != "" {
				detail = " (" + r.Detail + ")"
			}
			fmt.Fprintf(stdout, "  - %s: %s%s\n", r.Check, r.Status, detail)
		}
		if allResolved {
			fmt.Fprintln(stdout, cliText(locale, "Result: SUCCESS — recheck confirms every targeted finding is now ok.", "Resultado: SUCESSO — a reverificação confirma que todo achado alvo agora está ok."))
		} else {
			fmt.Fprintln(stdout, cliText(locale, "Result: PARTIAL — recheck still finds an issue on at least one targeted check.", "Resultado: PARCIAL — a reverificação ainda encontra um problema em pelo menos um check alvo."))
		}
	}
	if !allResolved {
		return 1
	}
	for _, r := range results {
		if r.Status == "error" {
			return 1
		}
	}
	return 0
}
