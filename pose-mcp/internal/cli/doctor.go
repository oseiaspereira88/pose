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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
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
		return doctorReport(findings, jsonOut, stdout, locale)
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
				fmt.Sprintf(text("%d spec(s) with delivers: have no change set recorded in .pose/reports/history — sealing a review bundle for them will fail with \"no immutable attributed change set exists\": %s",
					"%d spec(s) com delivers: não têm nenhum change set registrado em .pose/reports/history — selar um review bundle para elas vai falhar com \"no immutable attributed change set exists\": %s"),
					len(untraceable), strings.Join(untraceable, ", ")),
				text("run 'pose report --spec <slug> --change-from <rev> --change-to <rev>' to record the delivering range before sealing (a commit trailer alone does not persist one; github.com/oseiaspereira88/pose issue #17)",
					"rode 'pose report --spec <slug> --change-from <rev> --change-to <rev>' para registrar o range antes de selar (só o trailer do commit não persiste nada; issue #17)"))
		} else {
			add("review.scope-change-set", "ok", text("every in-progress/done spec with delivers: has a recorded change set", "toda spec em andamento/concluída com delivers: tem change set registrado"), "")
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

func doctorReport(findings []doctorFinding, jsonOut bool, stdout io.Writer, locale cliLocale) int {
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
