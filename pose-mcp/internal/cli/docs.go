package cli

// Docs governance CLI (spec pose-docs-governance-contract): `docs-init`
// scaffolds the opt-in manifest (write layer, ADR-003); `docs-check` is a
// thin formatter over the read-only pose.Store.CheckDocs checker.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harne8/pose-mcp/internal/pose"
)

func cmdDocsInit(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	profile := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--profile":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --profile requires a value", "Erro: --profile exige um valor"))
				return 2
			}
			i++
			profile = args[i]
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if profile != "" && !pose.ValidDocsProfile(profile) {
		fmt.Fprintf(stderr, cliText(locale, "Error: unknown profile %q (choose library|service|cli|monorepo)\n",
			"Erro: perfil %q desconhecido (escolha library|service|cli|monorepo)\n"), profile)
		return 2
	}
	store := pose.Store{Root: root}
	if store.HasDocsManifest() {
		fmt.Fprintf(stderr, cliText(locale,
			"Error: %s already exists; edit it instead of re-initializing\n",
			"Erro: %s já existe; edite-o em vez de re-inicializar\n"), store.DocsManifestPath())
		return 1
	}
	roots := pose.DocsProfileRoots(profile)
	manifest := pose.DocsManifest{SchemaVersion: pose.DocsManifestSchema, Profile: profile, Roots: roots, DefaultReviewDays: 180, Entries: []pose.DocsEntry{}}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-init: %v\n", err)
		return 1
	}
	path := store.DocsManifestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "pose docs-init: %v\n", err)
		return 1
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		fmt.Fprintf(stderr, "pose docs-init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, cliText(locale,
		"Docs manifest created: %s (roots=%s; run `pose docs-check` to see undeclared docs and start declaring them)\n",
		"Manifesto de docs criado: %s (roots=%s; rode `pose docs-check` para ver docs não declaradas e começar a declará-las)\n"),
		path, strings.Join(roots, ","))
	return 0
}

var docsRuleExplanations = map[string]string{
	"missing":             "A doc declared in the manifest's entries no longer exists at that path — the manifest drifted ahead of the tree.",
	"undeclared":          "A file exists under a declared root but is not in entries — the tree drifted ahead of the manifest.",
	"missing_frontmatter": "A declared doc has no YAML frontmatter block with both title and doc_type.",
	"broken_link":         "A relative markdown link inside a declared doc does not resolve to a real file under the project root.",
	"broken_reference":    "A typed reference (spec:/adr:/knowledge:/doc:/...) inside a declared doc does not resolve locally.",
	"stale":               "A declared doc is past its review_after date, or past default_review_days since the commit that last touched it.",
	"security":            "A declared doc matches an unsafe-instruction or secret-shaped pattern — defense in depth, not a substitute for gitleaks.",
}

func cmdDocsCheck(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	asJSON := false
	explain := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			asJSON = true
		case "--explain":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --explain requires a rule name", "Erro: --explain exige um nome de regra"))
				return 2
			}
			i++
			explain = args[i]
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	if explain != "" {
		text, ok := docsRuleExplanations[explain]
		if !ok {
			fmt.Fprintf(stderr, cliText(locale, "Error: unknown rule %q\n", "Erro: regra %q desconhecida\n"), explain)
			return 2
		}
		fmt.Fprintf(stdout, "%s: %s\n", explain, text)
		return 0
	}

	store := pose.Store{Root: root}
	if !store.HasDocsManifest() {
		fmt.Fprintln(stderr, cliText(locale,
			"Error: no docs manifest found (run `pose docs-init`)",
			"Erro: nenhum manifesto de docs encontrado (rode `pose docs-init`)"))
		return 1
	}
	manifest, err := store.LoadDocsManifest()
	if err != nil {
		fmt.Fprintf(stderr, "pose docs-check: %v\n", err)
		return 1
	}
	result := store.CheckDocs(context.Background(), manifest)

	if asJSON {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(stdout, string(encoded))
	} else {
		printDocsCheckHuman(stdout, result, locale)
	}
	if result.Totals.Errors > 0 {
		return 1
	}
	return 0
}

func printDocsCheckHuman(stdout io.Writer, result pose.DocsCheckResult, locale cliLocale) {
	fmt.Fprintf(stdout, cliText(locale,
		"docs: declared=%d undeclared=%d stale=%d errors=%d warnings=%d\n",
		"docs: declaradas=%d não-declaradas=%d vencidas=%d erros=%d avisos=%d\n"),
		result.Totals.Declared, result.Totals.Undeclared, result.Totals.Stale, result.Totals.Errors, result.Totals.Warnings)
	types := make([]string, 0, len(result.Totals.ByType))
	for t := range result.Totals.ByType {
		if t != "" {
			types = append(types, t)
		}
	}
	sort.Strings(types)
	if len(types) > 0 {
		var parts []string
		for _, t := range types {
			parts = append(parts, fmt.Sprintf("%s=%d", t, result.Totals.ByType[t]))
		}
		fmt.Fprintf(stdout, cliText(locale, "by doc_type: %s\n", "por doc_type: %s\n"), strings.Join(parts, " "))
	}
	for _, issue := range result.Issues {
		fmt.Fprintf(stdout, "[%s] %s: %s: %s\n", strings.ToUpper(issue.Severity), issue.Path, issue.Rule, issue.Message)
	}
	for _, pending := range result.ReviewPending {
		for _, trigger := range pending.Triggers {
			fmt.Fprintf(stdout, cliText(locale,
				"[REVIEW_PENDING] %s: since %s, trigger %s\n",
				"[REVISÃO_PENDENTE] %s: desde %s, gatilho %s\n"), pending.Doc, trigger.Since, trigger.Trigger)
		}
	}
	if result.Totals.Errors == 0 {
		fmt.Fprintln(stdout, cliText(locale, "Result: SUCCESS", "Resultado: SUCESSO"))
	} else {
		fmt.Fprintf(stdout, cliText(locale, "Result: FAILURE (%d error(s))\n", "Resultado: FALHA (%d erro(s))\n"), result.Totals.Errors)
	}
}
