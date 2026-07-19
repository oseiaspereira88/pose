package cli

// Knowledge consumption traceability (spec pose-knowledge-consumption-traceability):
// specs cite consumed knowledge with stable `knowledge:<slug>` refs;
// `knowledge-usage` projects citations to inform owner review WITHOUT
// extending TTL; `knowledge-suggest` offers deterministic, explainable,
// sensitivity-filtered advisory retrieval that always requires human
// confirmation. No semantic backend is involved or required.

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var knowledgeRefRE = regexp.MustCompile(`\bknowledge:([a-z0-9](?:[a-z0-9._-]*[a-z0-9])?)`)

type knowledgeArtifact struct {
	File        string
	Slug        string
	Owner       string
	Sensitivity string
	ExpiresAt   string
	Body        string
}

func loadKnowledgeArtifacts(root string) ([]knowledgeArtifact, error) {
	dir := filepath.Join(root, ".pose", "knowledge")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var artifacts []knowledgeArtifact
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || strings.EqualFold(e.Name(), "README.md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		fm, err := readFlatFrontmatter(path)
		if err != nil {
			continue // schema errors belong to knowledge-check
		}
		raw, _ := os.ReadFile(path)
		artifacts = append(artifacts, knowledgeArtifact{
			File: e.Name(), Slug: fm["slug"], Owner: fm["owner"],
			Sensitivity: fm["sensitivity"], ExpiresAt: fm["expires_at"], Body: string(raw),
		})
	}
	return artifacts, nil
}

// collectKnowledgeRefs scans every spec body for knowledge:<slug> citations.
// Returns slug → citing spec slugs (sorted, unique).
func collectKnowledgeRefs(root string) map[string][]string {
	refs := map[string]map[string]bool{}
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*", "spec.md"))
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		spec := filepath.Base(filepath.Dir(path))
		for _, m := range knowledgeRefRE.FindAllStringSubmatch(string(raw), -1) {
			if refs[m[1]] == nil {
				refs[m[1]] = map[string]bool{}
			}
			refs[m[1]][spec] = true
		}
	}
	out := map[string][]string{}
	for slug, specs := range refs {
		for s := range specs {
			out[slug] = append(out[slug], s)
		}
		sort.Strings(out[slug])
	}
	return out
}

// validateKnowledgeRefs reports dangling knowledge:<slug> citations (R1:
// stable references must resolve to a governed artifact).
func validateKnowledgeRefs(root string, stderr io.Writer) int {
	artifacts, err := loadKnowledgeArtifacts(root)
	if err != nil {
		return 0 // no knowledge dir: nothing to validate
	}
	known := map[string]bool{}
	for _, a := range artifacts {
		known[a.Slug] = true
	}
	failures := 0
	for slug, specs := range collectKnowledgeRefs(root) {
		if !known[slug] {
			fmt.Fprintf(stderr, "[ERROR] knowledge ref: knowledge:%s cited by %s does not resolve to a governed artifact\n", slug, strings.Join(specs, ", "))
			failures++
		}
	}
	return failures
}

// cmdKnowledgeUsage projects citation signals per artifact. Signals inform
// the owner's review decision; TTL (expires_at) is never modified here.
func cmdKnowledgeUsage(root string, stdout, stderr io.Writer) int {
	artifacts, err := loadKnowledgeArtifacts(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	refs := collectKnowledgeRefs(root)
	fmt.Fprintln(stdout, "# POSE knowledge usage — citation signals (TTL is never extended automatically)")
	cited := 0
	for _, a := range artifacts {
		specs := refs[a.Slug]
		if len(specs) > 0 {
			cited++
		}
		fmt.Fprintf(stdout, "- %s owner:%s expires:%s citations:%d", a.Slug, a.Owner, a.ExpiresAt, len(specs))
		if len(specs) > 0 {
			fmt.Fprintf(stdout, " cited_by:%s", strings.Join(specs, ","))
		}
		fmt.Fprintln(stdout)
	}
	fmt.Fprintf(stdout, "usage.artifacts=%d\nusage.cited=%d\n", len(artifacts), cited)
	return 0
}

// cmdKnowledgeSuggest ranks non-restricted knowledge against a query using
// the same deterministic lexical engine as follow-up clustering. Output is
// advisory: rationale is exposed and confirmation is always required.
func cmdKnowledgeSuggest(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: pose knowledge-suggest <query terms...>")
		return 2
	}
	query := strings.Join(args, " ")
	artifacts, err := loadKnowledgeArtifacts(root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	type suggestion struct {
		artifact knowledgeArtifact
		score    float64
		shared   []string
	}
	queryTokens := followupTokens(normalizeFollowup(query))
	var suggestions []suggestion
	restricted := 0
	for _, a := range artifacts {
		if a.Sensitivity == "restricted" {
			restricted++ // sensitivity filter precedes any retrieval
			continue
		}
		score := followupSimilarity(query, a.Body)
		var shared []string
		for token := range followupTokens(normalizeFollowup(a.Body)) {
			if queryTokens[token] {
				shared = append(shared, token)
			}
		}
		sort.Strings(shared)
		if score > 0 && len(shared) > 0 {
			suggestions = append(suggestions, suggestion{a, score, shared})
		}
	}
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].score != suggestions[j].score {
			return suggestions[i].score > suggestions[j].score
		}
		return suggestions[i].artifact.Slug < suggestions[j].artifact.Slug
	})
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}
	fmt.Fprintf(stdout, "# POSE knowledge suggestions — ADVISORY, deterministic lexical ranking\n")
	fmt.Fprintf(stdout, "# query=%q candidates=%d restricted_filtered=%d\n", query, len(suggestions), restricted)
	if len(suggestions) == 0 {
		fmt.Fprintln(stdout, "(no candidates)")
	}
	for _, s := range suggestions {
		rationale := s.shared
		if len(rationale) > 6 {
			rationale = rationale[:6]
		}
		fmt.Fprintf(stdout, "- %s score:%.2f sensitivity:%s rationale:shared-terms[%s]\n", s.artifact.Slug, s.score, s.artifact.Sensitivity, strings.Join(rationale, " "))
	}
	fmt.Fprintln(stdout, "\nConfirm relevance before citing: suggestions never gate or auto-apply; cite with knowledge:<slug> after human review.")
	return 0
}
