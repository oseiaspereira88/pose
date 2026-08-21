// Package cli implements the native-only unified `pose` binary.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harne8/pose-mcp/internal/version"
)

// Version mirrors the authoritative release version. The release pipeline
// stamps internal/version via -ldflags (spec pose-version-contract).
var Version = version.Version

// Main is the entrypoint used by cmd/pose. It returns the process exit code.
func Main(args []string, stdout, stderr io.Writer) int {
	return mainWithUsage(args, stdout, stderr)
}

func mainCommand(args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleFor(stderr)
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	switch cmd {
	case "version", "--version", "-v":
		target := ""
		if len(args) > 0 {
			target = args[0]
		}
		return cmdVersion(stdout, target)
	case "help", "-h", "--help":
		return cmdHelp(stdout)
	case "init":
		if len(args) > 0 && args[0] == "--wizard" {
			root, err := projectRoot()
			if err != nil {
				fmt.Fprintf(stderr, "pose init: %v\n", err)
				return 1
			}
			return cmdInitWizard(root, args[1:], stdout, stderr)
		}
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose init: %v\n", err)
			return 1
		}
		return cmdInit(root, stdout, stderr)
	case "new-spec":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-spec: %v\n", err)
			return 1
		}
		return cmdNewSpec(root, args, stdout, stderr)
	case "new-roadmap":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-roadmap: %v\n", err)
			return 1
		}
		return cmdNewRoadmap(root, args, stdout, stderr)
	case "new-adr":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-adr: %v\n", err)
			return 1
		}
		return cmdNewADR(root, args, stdout, stderr)
	case "new-knowledge":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose new-knowledge: %v\n", err)
			return 1
		}
		return cmdNewKnowledge(root, args, stdout, stderr)
	case "followups":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose followups: %v\n", err)
			return 1
		}
		return cmdFollowups(root, args, stdout, stderr)
	case "amend":
		return cmdAmend(args, stdout, stderr)
	case "assess":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose assess: %v\n", err)
			return 1
		}
		return cmdAssess(root, args, stdout, stderr)
	case "state":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose state: %v\n", err)
			return 1
		}
		return cmdState(root, args, stdout, stderr)
	case "docs-init":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose docs-init: %v\n", err)
			return 1
		}
		return cmdDocsInit(root, args, stdout, stderr)
	case "docs-check":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose docs-check: %v\n", err)
			return 1
		}
		return cmdDocsCheck(root, args, stdout, stderr)
	case "docs-review":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose docs-review: %v\n", err)
			return 1
		}
		return cmdDocsReview(root, args, stdout, stderr)
	case "docs-sync":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose docs-sync: %v\n", err)
			return 1
		}
		return cmdDocsSync(root, args, stdout, stderr)
	case "report-limitation", "feedback":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose report-limitation: %v\n", err)
			return 1
		}
		return cmdReportLimitation(root, args, stdout, stderr)
	case "report":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose report: %v\n", err)
			return 1
		}
		return cmdReport(root, args, stdout, stderr)
	case "validate":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose validate: %v\n", err)
			return 1
		}
		return cmdValidate(root, args, stdout, stderr)
	case "check":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose check: %v\n", err)
			return 1
		}
		return cmdCheck(root, args, stdout, stderr)
	case "review", "review-plan", "review-check", "closeout-check", "close", "continuous-closeout", "artifact-check", "artifact-backfill", "surface-check", "roadmap-check":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose %s: %v\n", cmd, err)
			return 1
		}
		switch cmd {
		case "review":
			return cmdReview(root, args, stdout, stderr)
		case "review-plan":
			return cmdReviewPlan(root, args, stdout, stderr)
		case "review-check":
			return cmdReviewCheck(root, args, stdout, stderr)
		case "closeout-check":
			return cmdCloseoutCheck(root, args, stdout, stderr)
		case "continuous-closeout":
			return cmdContinuousCloseout(root, args, stdout, stderr)
		case "artifact-check":
			return cmdArtifactCheck(root, args, stdout, stderr)
		case "artifact-backfill":
			return cmdArtifactBackfill(root, args, stdout, stderr)
		case "surface-check":
			return cmdSurfaceCheck(root, args, stdout, stderr)
		case "roadmap-check":
			return cmdRoadmapCheck(root, args, stdout, stderr)
		default:
			return cmdClose(root, args, stdout, stderr)
		}
	case "update", "index", "knowledge-check", "knowledge-housekeeping", "knowledge-usage", "knowledge-suggest", "reports-housekeeping", "recurrence-check", "recurrence-effect", "hooks", "suggest", "stats", "usage", "stacks", "skills-check", "record-deployment", "record-incident", "dora-metrics", "adoption-metrics", "events-housekeeping", "semantic-suggest", "suggest-feedback", "portfolio-projection", "reconcile-evidence":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose %s: %v\n", cmd, err)
			return 1
		}
		switch cmd {
		case "update":
			return cmdUpdate(root, args, stdout, stderr)
		case "index":
			return cmdIndex(root, args, stdout, stderr)
		case "knowledge-check":
			return cmdKnowledgeCheck(root, args, stdout, stderr)
		case "knowledge-housekeeping":
			return cmdKnowledgeHousekeeping(root, args, stdout, stderr)
		case "knowledge-usage":
			return cmdKnowledgeUsage(root, stdout, stderr)
		case "knowledge-suggest":
			return cmdKnowledgeSuggest(root, args, stdout, stderr)
		case "reports-housekeeping":
			return cmdReportsHousekeeping(root, args, stdout, stderr)
		case "recurrence-check":
			return cmdRecurrenceCheck(root, args, stdout, stderr)
		case "recurrence-effect":
			return cmdRecurrenceEffect(root, args, stdout, stderr)
		case "skills-check":
			return cmdSkillsCheck(root, args, stdout, stderr)
		case "stacks":
			return cmdStacks(root, args, stdout, stderr)
		case "hooks":
			return cmdHooks(root, args, stdout, stderr)
		case "suggest":
			return cmdSuggest(root, args, stdout, stderr)
		case "usage":
			return cmdUsage(root, args, stdout, stderr)
		case "record-deployment":
			return cmdRecordDeployment(root, args, stdout, stderr)
		case "record-incident":
			return cmdRecordIncident(root, args, stdout, stderr)
		case "dora-metrics":
			return cmdDORAMetrics(root, args, stdout, stderr)
		case "adoption-metrics":
			return cmdAdoptionMetrics(root, args, stdout, stderr)
		case "events-housekeeping":
			return cmdEventsHousekeeping(root, args, stdout, stderr)
		case "semantic-suggest":
			return cmdSemanticSuggest(root, args, stdout, stderr)
		case "suggest-feedback":
			return cmdSuggestFeedback(root, args, stdout, stderr)
		case "portfolio-projection":
			return cmdPortfolioProjection(root, args, stdout, stderr)
		case "reconcile-evidence":
			return cmdReconcileEvidence(root, args, stdout, stderr)
		default:
			return cmdStats(root, args, stdout, stderr)
		}
	case "release-notes":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose release-notes: %v\n", err)
			return 1
		}
		return cmdReleaseNotes(root, args, stdout, stderr)
	case "release":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose release: %v\n", err)
			return 1
		}
		return cmdRelease(root, args, stdout, stderr)
	case "release-package-manifests":
		return cmdReleasePackageManifests(args, stdout, stderr)
	case "install":
		return cmdInstall(args, stdout, stderr)
	case "doctor":
		return cmdDoctor(args, stdout, stderr)
	case "import":
		defer emitTelemetry("import")
		return cmdImport(args, stdout, stderr)
	case "extension":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose extension: %v\n", err)
			return 1
		}
		return cmdExtension(root, args, stdout, stderr)
	case "contribute":
		root, err := projectRoot()
		if err != nil {
			fmt.Fprintf(stderr, "pose contribute: %v\n", err)
			return 1
		}
		return cmdContribute(root, args, stdout, stderr)
	case "lint-spec":
		defer emitTelemetry("lint-spec")
		return cmdLintSpec(args, stdout, stderr)
	case "history-check":
		defer emitTelemetry("history-check")
		return cmdHistoryCheck(args, stdout, stderr)
	case "telemetry":
		return cmdTelemetry(args, stdout, stderr)
	case "serve-mcp":
		// Blocking; wiring lives in internal/bootstrap.
		runServeMCP(args)
		return 0
	}

	fmt.Fprintf(stderr, "%s: %s\n", cliText(locale, "Unknown command", "Comando desconhecido"), cmd)
	fmt.Fprintln(stderr, cliText(locale, "Run 'pose help' to see available commands.", "Execute 'pose help' para ver os comandos disponíveis."))
	return 2
}

// projectRoot resolves the git toplevel, falling back to the current directory.
func projectRoot() (string, error) {
	return projectRootAt(".")
}

// projectRootAt resolves the git top-level starting from dir, falling back
// to dir's own absolute path when dir isn't inside a git work tree (spec
// pose-instance-engine-version-tracking R2) — the same fallback shape
// projectRoot() already had for ".".
func projectRootAt(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// cmdVersion prints the binary version and, when an instance is resolvable
// (target, or the current directory when target is ""), its schema and —
// when recorded — the engine version that last delivered its machinery
// (spec pose-instance-engine-version-tracking).
func cmdVersion(w io.Writer, target string) int {
	fmt.Fprintf(w, "pose %s\n", Version)
	dir := "."
	if target != "" {
		dir = target
	}
	if root, err := projectRootAt(dir); err == nil {
		sv := filepath.Join(root, ".pose", "schema-version")
		if b, err := os.ReadFile(sv); err == nil {
			fmt.Fprintf(w, "schema: %s\n", strings.TrimSpace(string(b)))
		} else if _, err := os.Stat(filepath.Join(root, ".pose")); err == nil {
			fmt.Fprintf(w, "schema: unversioned (run 'pose update')\n")
		}
		if ev := instanceEngineVersion(root); ev != "" {
			fmt.Fprintf(w, "instance last updated by: pose %s\n", ev)
		}
	}
	return 0
}

func cmdHelp(w io.Writer) int {
	fmt.Fprint(w, cliText(cliLocaleValue(), helpTextEN, helpTextPtBR))
	return 0
}

const helpTextEN = `POSE - Project Operating Standard for Engineering

Usage: pose <command> [options]

Native binary:
  version                             Binary and instance schema versions
  init                                Ensure the minimum POSE repository structure
  serve-mcp [--stdio]                 Start the POSE MCP server (POSE_* environment)
  doctor [--json]                     Diagnose installation and instance health
  report-limitation --title "..."     Report an engine limitation, bug or proposal
                                      [--kind limitation|bug|suggestion] [--submit]
  install <dir> [--locale tag] [...]  Install embedded POSE without cloning
  import <spec-kit|openspec> <path>   Import external specs [--dry-run]
  telemetry <enable|disable|status>   Anonymous opt-in telemetry

Scaffolds:
  new-spec <slug>                     Create a feature spec scaffold
  new-roadmap <slug>                  Create a governed roadmap
  new-adr "<title>"                   Create an ADR
  new-knowledge <type> <slug>         Create a handoff, note, or decision log

Deterministic gates:
  check | validate | knowledge-check | recurrence-check | lint-spec |
  followups | amend | history-check | skills-check | review-check | closeout-check |
  artifact-check | surface-check | roadmap-check

Governed closeout:
  review-plan <scope> [--explain]     Resolve component-aware criteria and tools
  review bundle <scope> [--seal]      Prepare or seal an immutable review subject
  review attest <bundle-id> [...]     Record/import a separate attestation
  review verify <scope|bundle-id>     Verify freshness and closeout readiness
  review record <scope> [...]         Compatibility entrypoint
  close <scope>                         Apply a review-gated lifecycle transition
  continuous-closeout <action>          Persist and project a terminal scope
  artifact-backfill --from-git          Propose explicit historical provenance

Release lifecycle:
  release <plan|prepare|check|notes|record|status|open-next|backfill>
  release-notes --version vX.Y.Z        Compatibility alias for immutable notes

Project state:
  state [init|refresh|diff]           Native project-state artifact (bare = validate)

Discovery and metrics:
  suggest | stats | usage | recurrence-effect | stacks
  record-deployment | record-incident | dora-metrics | adoption-metrics
  semantic-suggest | suggest-feedback | portfolio-projection | reconcile-evidence

Extensions:
  extension install <dir> [--target <dir>] [--dry-run] [--yes] [--force] [--allow-unsigned]
  extension list [--json] | extension remove <id> [...] | extension verify <dir>

Contributions:
  contribute <enable|disable|status|stage|list> [--target <dir>] [--json]

Artifacts and maintenance:
  index | report | update [--dry-run] | knowledge-housekeeping |
  knowledge-usage | knowledge-suggest | reports-housekeeping | events-housekeeping | hooks

All commands execute in the Go binary without Bash or Python fallbacks.
`

const helpTextPtBR = `POSE - Project Operating Standard for Engineering

Uso: pose <comando> [opções]

Nativos (binário):
  version                             Versão do binário + schema da instância
  init                                Garante a estrutura mínima do POSE no repo
  serve-mcp [--stdio]                 Sobe o servidor MCP do POSE (env POSE_*)
  doctor [--json]                     Diagnóstico da instalação/instância
  report-limitation --title "..."     Relata limitação, bug ou sugestão do engine
                                      [--kind limitation|bug|suggestion] [--submit]
  install <dir> [--locale tag] [...]  Instala o POSE embutido num repo (sem clone)
  import <spec-kit|openspec> <path>   Importa specs externas [--dry-run]
  telemetry <enable|disable|status>   Telemetria anônima OPT-IN (nada é enviado
                                      sem opt-in E POSE_TELEMETRY_URL)

Scaffold:
  new-spec <slug>                     Cria scaffold de spec por feature
  new-roadmap <slug>                  Cria roadmap governado em .pose/roadmaps/
  new-adr "<título>"                  Cria ADR com template padrão
  new-knowledge <type> <slug>         Cria handoff/note/decision-log

Gates determinísticos:
  check | validate | knowledge-check | recurrence-check | lint-spec |
  followups | amend | history-check | skills-check | review-check | closeout-check |
  artifact-check | surface-check | roadmap-check

Fechamento governado:
  review-plan <escopo> [--explain]     Resolve critérios e ferramentas por componente
  review bundle <escopo> [--seal]      Prepara ou sela o sujeito imutável do review
  review attest <bundle-id> [...]      Registra/importa atestação separada
  review verify <escopo|bundle-id>     Verifica atualidade e prontidão para fechamento
  review record <escopo> [...]         Entrada de compatibilidade
  close <escopo>                        Aplica transição de ciclo de vida governada
  continuous-closeout <ação>            Persiste e projeta um escopo terminal
  artifact-backfill --from-git          Propõe proveniência histórica explícita

Ciclo de release:
  release <plan|prepare|check|notes|record|status|open-next|backfill>
  release-notes --version vX.Y.Z        Alias para notas imutáveis

Estado do projeto:
  state [init|refresh|diff]           Artefato nativo de project-state (sem subcomando = validar)

Descoberta e métricas:
  suggest | stats | usage | recurrence-effect | stacks
  record-deployment | record-incident | dora-metrics | adoption-metrics
  semantic-suggest | suggest-feedback | portfolio-projection | reconcile-evidence

Extensões:
  extension install <dir> [--target <dir>] [--dry-run] [--yes] [--force] [--allow-unsigned]
  extension list [--json] | extension remove <id> [...] | extension verify <dir>

Contribuições:
  contribute <enable|disable|status|stage|list> [--target <dir>] [--json]

Geração de artefatos:
  index | report

Manutenção:
  update [--dry-run] | knowledge-housekeeping | knowledge-usage |
  knowledge-suggest | reports-housekeeping | events-housekeeping | hooks

Todos os comandos executam no binário Go, sem fallbacks Bash ou Python.
'pose help' completo por comando: consulte o POSE.md da instância.
`
