package cli

// Human-reviewed semantic governance assist (spec pose-semantic-governance-assist):
// advisory, explainable suggestions of related follow-ups, recurrence
// patterns and knowledge — every suggestion cites its artifact, carries a
// deterministic score/rationale and declares its provider (R1). Never
// mutates lifecycle, never gates anything (Constraint). The only approved
// provider in this release is the deterministic lexical fallback already
// proven by pose-knowledge-consumption-traceability (followupSimilarity/
// followupTokens) — a real LLM-backed provider is an explicit future
// extension point (SuggestionProvider), not implemented here (see ADR).

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type governanceSuggestion struct {
	ArtifactRef string   `json:"artifact_ref"`
	Kind        string   `json:"kind"` // knowledge | followup | recurrence
	Score       float64  `json:"score"`
	Rationale   []string `json:"rationale"`
	Provider    string   `json:"provider"`
}

// approvedSuggestionProviders is an allowlist (Security: require approved
// providers). Only "lexical" — the deterministic, offline fallback — is
// approved in this release; any other name is rejected outright.
var approvedSuggestionProviders = map[string]bool{"lexical": true}

// sanitizeForPrompt is the prompt-injection defense (Security): every
// piece of candidate text is stripped of unsafe-instruction and
// secret-shaped patterns (the same scan pose-agent-skills-conformance
// applies) before it is ever compared, cited or would be handed to any
// future non-lexical provider. The lexical provider never sends text
// anywhere; this exists so a future provider inherits the same guarantee
// by construction rather than by convention.
func sanitizeForPrompt(s string) string {
	s = redactSecretShapedContent(s)
	for _, re := range unsafeSkillPatterns {
		s = re.ReplaceAllString(s, "[UNSAFE_PATTERN_REMOVED]")
	}
	return s
}

type recurringPattern struct {
	TaskSlug string
	Workflow string
	Runs     int
}

// collectRecurringPatterns mirrors cmdRecurrenceCheck's bucketing (same
// defaults: 14-day window, threshold 3 non-pass runs) as a standalone,
// side-effect-free query — kept independent of cmdRecurrenceCheck so this
// spec never risks regressing that already-shipped, separately-owned gate.
func collectRecurringPatterns(root string) []recurringPattern {
	records, _ := readHistory(root, io.Discard)
	cutoff := time.Now().UTC().AddDate(0, 0, -14)
	buckets := map[string][]historyRecord{}
	for _, r := range records {
		t, ok := parseHistoryTime(r.GeneratedAt)
		if !ok || t.Before(cutoff) || r.Outcome == "pass" {
			continue
		}
		task := r.TaskSlug
		if task == "" {
			continue
		}
		buckets[task] = append(buckets[task], r)
	}
	var out []recurringPattern
	for task, rs := range buckets {
		if len(rs) < 3 {
			continue
		}
		workflow := ""
		for _, r := range rs {
			if r.Workflow != "" {
				workflow = r.Workflow
			}
		}
		out = append(out, recurringPattern{TaskSlug: task, Workflow: workflow, Runs: len(rs)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskSlug < out[j].TaskSlug })
	return out
}

// specQueryText builds the comparison text for --for <slug>: the spec's
// Intent and Requirements sections, sanitized. Falls back to the whole
// body when section extraction finds nothing (still bounded by
// sanitization either way).
func specQueryText(root, slug string) (string, error) {
	path := filepath.Join(root, ".pose", "specs", slug, "spec.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sanitizeForPrompt(string(raw)), nil
}

func computeSemanticSuggestions(root, forSpec, query string, top int) ([]governanceSuggestion, int) {
	restrictedFiltered := 0
	queryTokens := followupTokens(normalizeFollowup(query))
	var out []governanceSuggestion

	// Knowledge: sensitivity filtered before any retrieval (R2).
	if artifacts, err := loadKnowledgeArtifacts(root); err == nil {
		for _, a := range artifacts {
			if a.Sensitivity == "restricted" {
				restrictedFiltered++
				continue
			}
			body := sanitizeForPrompt(a.Body)
			score := followupSimilarity(query, body)
			shared := sharedTokens(queryTokens, body)
			if score > 0 && len(shared) > 0 {
				out = append(out, governanceSuggestion{
					ArtifactRef: "knowledge:" + a.Slug, Kind: "knowledge",
					Score: score, Rationale: shared, Provider: "lexical",
				})
			}
		}
	}

	// Follow-ups: every OTHER spec's open follow-ups (never the target
	// spec suggesting itself).
	for _, f := range collectFollowups(root) {
		if f.Spec == forSpec || f.RawDisposition != "" && f.RawDisposition != "open" {
			continue
		}
		body := sanitizeForPrompt(f.Text)
		score := followupSimilarity(query, body)
		shared := sharedTokens(queryTokens, body)
		if score > 0 && len(shared) > 0 {
			out = append(out, governanceSuggestion{
				ArtifactRef: "spec:" + f.Spec + "#followup", Kind: "followup",
				Score: score, Rationale: shared, Provider: "lexical",
			})
		}
	}

	// Recurrence patterns.
	for _, p := range collectRecurringPatterns(root) {
		body := sanitizeForPrompt(p.TaskSlug + " " + p.Workflow)
		score := followupSimilarity(query, body)
		shared := sharedTokens(queryTokens, body)
		if score > 0 && len(shared) > 0 {
			out = append(out, governanceSuggestion{
				ArtifactRef: "recurrence:" + p.TaskSlug, Kind: "recurrence",
				Score: score, Rationale: shared, Provider: "lexical",
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ArtifactRef < out[j].ArtifactRef
	})
	if top > 0 && len(out) > top {
		out = out[:top]
	}
	return out, restrictedFiltered
}

func sharedTokens(queryTokens map[string]bool, body string) []string {
	var shared []string
	for token := range followupTokens(normalizeFollowup(body)) {
		if queryTokens[token] {
			shared = append(shared, token)
		}
	}
	sort.Strings(shared)
	if len(shared) > 6 {
		shared = shared[:6]
	}
	return shared
}

func cmdSemanticSuggest(root string, args []string, stdout, stderr io.Writer) int {
	forSpec, freeQuery, provider := "", "", "lexical"
	top := 5
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--for":
			if i+1 >= len(args) {
				return usageError(stderr, "pose semantic-suggest: --for requires a spec slug")
			}
			i++
			forSpec = args[i]
		case "--query":
			if i+1 >= len(args) {
				return usageError(stderr, "pose semantic-suggest: --query requires text")
			}
			i++
			freeQuery = args[i]
		case "--provider":
			if i+1 >= len(args) {
				return usageError(stderr, "pose semantic-suggest: --provider requires a value")
			}
			i++
			provider = args[i]
		case "--top":
			if i+1 >= len(args) {
				return usageError(stderr, "pose semantic-suggest: --top requires a value")
			}
			i++
			n, e := strconv.Atoi(args[i])
			if e != nil || n < 1 {
				return usageError(stderr, "pose semantic-suggest: --top must be a positive integer")
			}
			top = n
		case "--json":
			jsonOut = true
		default:
			return usageError(stderr, "Usage: pose semantic-suggest (--for <spec-slug>|--query \"text\") [--top N] [--provider lexical] [--json]")
		}
	}
	if !approvedSuggestionProviders[provider] {
		fmt.Fprintf(stderr, "pose semantic-suggest: provider %q is not approved (allowed: lexical)\n", provider)
		return 2
	}
	if forSpec == "" && freeQuery == "" {
		return usageError(stderr, "Usage: pose semantic-suggest (--for <spec-slug>|--query \"text\") [--top N] [--provider lexical] [--json]")
	}

	query := freeQuery
	if forSpec != "" {
		q, err := specQueryText(root, forSpec)
		if err != nil {
			fmt.Fprintf(stderr, "pose semantic-suggest: %v\n", err)
			return 1
		}
		query = q
	}

	suggestions, restrictedFiltered := computeSemanticSuggestions(root, forSpec, query, top)

	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(map[string]any{
			"for":                 forSpec,
			"suggestions":         suggestions,
			"restricted_filtered": restrictedFiltered,
		})
		return 0
	}
	fmt.Fprintf(stdout, "# Semantic governance suggestions — ADVISORY, %s ranking\n", provider)
	fmt.Fprintf(stdout, "# for=%q candidates=%d restricted_filtered=%d\n", forSpec, len(suggestions), restrictedFiltered)
	if len(suggestions) == 0 {
		fmt.Fprintln(stdout, "(no candidates)")
	}
	for _, s := range suggestions {
		fmt.Fprintf(stdout, "- %s kind:%s score:%.2f provider:%s rationale:shared-terms[%s]\n",
			s.ArtifactRef, s.Kind, s.Score, s.Provider, strings.Join(s.Rationale, " "))
	}
	fmt.Fprintln(stdout, "\nAdvisory only: suggestions never gate or auto-apply lifecycle. Confirm relevance, then record your decision with 'pose suggest-feedback'.")
	return 0
}

type suggestionFeedback struct {
	RecordedAt  string  `json:"recorded_at"`
	ForSpec     string  `json:"for_spec"`
	ArtifactRef string  `json:"artifact_ref"`
	Kind        string  `json:"kind"`
	Decision    string  `json:"decision"` // accept | reject
	Score       float64 `json:"score,omitempty"`
	Provider    string  `json:"provider"`
}

var validFeedbackDecision = map[string]bool{"accept": true, "reject": true}

// cmdSuggestFeedback records an accept/reject decision without the
// candidate's text or rationale (R3: feed evaluation without training on
// restricted content — there is never any content in this record to
// begin with, restricted or not).
func cmdSuggestFeedback(root string, args []string, stdout, stderr io.Writer) int {
	var fb suggestionFeedback
	fb.Provider = "lexical"
	scoreStr := ""
	for i := 0; i < len(args); i++ {
		if i+1 >= len(args) {
			return usageError(stderr, "Usage: pose suggest-feedback --for <spec-slug> --ref <artifact-ref> --kind knowledge|followup|recurrence --decision accept|reject [--score N] [--provider lexical]")
		}
		v := args[i+1]
		switch args[i] {
		case "--for":
			fb.ForSpec = v
		case "--ref":
			fb.ArtifactRef = v
		case "--kind":
			fb.Kind = v
		case "--decision":
			fb.Decision = v
		case "--score":
			scoreStr = v
		case "--provider":
			fb.Provider = v
		default:
			return usageError(stderr, "pose suggest-feedback: unknown flag "+args[i])
		}
		i++
	}
	if fb.ForSpec == "" || fb.ArtifactRef == "" {
		fmt.Fprintln(stderr, "pose suggest-feedback: --for and --ref are required")
		return 2
	}
	if !validFeedbackDecision[fb.Decision] {
		fmt.Fprintln(stderr, "pose suggest-feedback: --decision must be accept|reject")
		return 2
	}
	if !approvedSuggestionProviders[fb.Provider] {
		fmt.Fprintf(stderr, "pose suggest-feedback: provider %q is not approved (allowed: lexical)\n", fb.Provider)
		return 2
	}
	if scoreStr != "" {
		n, err := strconv.ParseFloat(scoreStr, 64)
		if err != nil {
			fmt.Fprintln(stderr, "pose suggest-feedback: --score must be a number")
			return 2
		}
		fb.Score = n
	}
	now := time.Now().UTC()
	fb.RecordedAt = now.Format(time.RFC3339)
	line, _ := json.Marshal(fb)
	path := filepath.Join(root, ".pose", "reports", "history", "semantic-feedback-"+now.Format("2006-01")+".jsonl")
	if err := appendEvent(path, line); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintf(stdout, "feedback recorded: for=%s ref=%s decision=%s\n", fb.ForSpec, fb.ArtifactRef, fb.Decision)
	return 0
}
