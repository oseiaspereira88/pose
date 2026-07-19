package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type followup struct {
	Spec           string `json:"spec"`
	SpecStatus     string `json:"spec_status"`
	RawDisposition string `json:"raw_disposition"`
	Target         string `json:"target"`
	Text           string `json:"text"`
	// Ownership metadata (spec pose-followup-ownership-sla): parsed from a
	// trailing "(owner:@x crit:high review:YYYY-MM-DD by:@y)" group. Owner is
	// "unowned" when the group is absent (legacy follow-ups).
	Owner       string `json:"owner"`
	Criticality string `json:"criticality,omitempty"`
	Review      string `json:"review,omitempty"`
	By          string `json:"by,omitempty"`
	MetaErr     string `json:"meta_error,omitempty"`
}

type nearDuplicateMember struct {
	Spec        string `json:"spec"`
	Text        string `json:"text"`
	Disposition string `json:"disposition"`
}

type nearDuplicateCandidate struct {
	Members []nearDuplicateMember `json:"members"`
	Specs   []string              `json:"specs"`
}

var followupBullet = regexp.MustCompile(`^\s*-\s+(.*\S)\s*$`)
var followupDisposition = regexp.MustCompile(`^\[\s*([a-z-]+)(?:\s*:\s*([^\]]+))?\s*\]\s*(.*)$`)
var followupHTMLComment = regexp.MustCompile(`(?s)<!--.*?-->`)
var followupMetaGroup = regexp.MustCompile(`\(([^()]*\bowner:[^()]*)\)\s*$`)
var followupReviewDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
var followupCriticality = map[string]bool{"low": true, "medium": true, "high": true}

// parseFollowupMeta extracts the trailing ownership group from a follow-up
// text. Returns the stripped text and the parsed fields; metaErr describes a
// malformed group ("" when valid or absent).
func parseFollowupMeta(text string) (stripped, owner, crit, review, by, metaErr string) {
	m := followupMetaGroup.FindStringSubmatchIndex(text)
	if m == nil {
		return strings.TrimSpace(text), "unowned", "", "", "", ""
	}
	group := text[m[2]:m[3]]
	stripped = strings.TrimSpace(text[:m[0]])
	owner = "unowned"
	for _, field := range strings.Fields(group) {
		key, value, ok := strings.Cut(field, ":")
		if !ok || value == "" {
			return stripped, owner, crit, review, by, "malformed ownership field '" + field + "' (use key:value)"
		}
		switch key {
		case "owner":
			owner = value
		case "crit":
			if !followupCriticality[value] {
				return stripped, owner, crit, review, by, "invalid crit '" + value + "' (use low|medium|high)"
			}
			crit = value
		case "review":
			if !followupReviewDate.MatchString(value) {
				return stripped, owner, crit, review, by, "invalid review date '" + value + "' (use YYYY-MM-DD)"
			}
			review = value
		case "by":
			by = value
		default:
			return stripped, owner, crit, review, by, "unknown ownership field '" + key + "' (use owner|crit|review|by)"
		}
	}
	if crit == "" || review == "" {
		return stripped, owner, crit, review, by, "incomplete ownership group (declare owner, crit and review together)"
	}
	return stripped, owner, crit, review, by, ""
}

func cmdFollowups(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	all, jsonOut, scopeSet, threshold := false, false, false, 60
	overdueOnly, failOverdue, ownerFilter := false, false, ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--overdue":
			overdueOnly = true
		case "--fail-overdue":
			failOverdue = true
		case "--owner":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --owner requires an alias.", "Erro: --owner exige um alias."))
				return 2
			}
			i++
			ownerFilter = args[i]
		case "--open":
			if scopeSet && all {
				fmt.Fprintln(stderr, cliText(locale, "Error: --open and --all are mutually exclusive.", "Erro: --open e --all são mutuamente exclusivos."))
				return 2
			}
			scopeSet = true
		case "--all":
			if scopeSet && !all {
				fmt.Fprintln(stderr, cliText(locale, "Error: --open and --all are mutually exclusive.", "Erro: --open e --all são mutuamente exclusivos."))
				return 2
			}
			all, scopeSet = true, true
		case "--json":
			jsonOut = true
		case "--similarity":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, cliText(locale, "Error: --similarity requires an integer from 0 to 100.", "Erro: --similarity exige inteiro 0..100."))
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 || value > 100 {
				fmt.Fprintln(stderr, cliText(locale, "Error: --similarity requires an integer from 0 to 100.", "Erro: --similarity exige inteiro 0..100."))
				return 2
			}
			threshold = value
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: unknown option: %s\n", "Erro: opção desconhecida: %s\n"), args[i])
			return 2
		}
	}
	allEntries := collectFollowups(root)
	today := followupToday()
	openEntries := make([]followup, 0, len(allEntries))
	overdueEntries := make([]followup, 0)
	unowned := 0
	for _, entry := range allEntries {
		if entry.RawDisposition == "" || entry.RawDisposition == "open" {
			openEntries = append(openEntries, entry)
			if entry.Owner == "unowned" {
				unowned++
			}
			if entry.Review != "" && entry.Review < today {
				overdueEntries = append(overdueEntries, entry)
			}
		}
	}
	selected := openEntries
	if all {
		selected = allEntries
	}
	if overdueOnly {
		selected = overdueEntries
	}
	if ownerFilter != "" {
		filtered := make([]followup, 0, len(selected))
		for _, entry := range selected {
			if entry.Owner == ownerFilter {
				filtered = append(filtered, entry)
			}
		}
		selected = filtered
	}
	candidates := clusterFollowups(allEntries, float64(threshold)/100)
	if jsonOut {
		specs := map[string]bool{}
		for _, entry := range allEntries {
			specs[entry.Spec] = true
		}
		payload := map[string]any{
			"total": len(allEntries), "open": len(openEntries), "specs": len(specs),
			"overdue": len(overdueEntries), "unowned": unowned,
			"similarity_threshold": threshold, "items": selected,
			"near_duplicate_candidates": candidates,
		}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: serializing follow-ups: %v\n", "Erro: serializar follow-ups: %v\n"), err)
			return 1
		}
		if failOverdue && len(overdueEntries) > 0 {
			return 1
		}
		return 0
	}
	label := "open follow-ups"
	if all {
		label = "all follow-ups"
	}
	if overdueOnly {
		label = "overdue follow-ups"
	}
	fmt.Fprintf(stdout, "# POSE follow-ups — %s\n# total=%d open=%d overdue=%d unowned=%d specs=%d\n\n", label, len(allEntries), len(openEntries), len(overdueEntries), unowned, uniqueFollowupSpecs(allEntries))
	if len(selected) == 0 {
		fmt.Fprintln(stdout, "(none)")
	}
	for _, entry := range selected {
		disposition := entry.RawDisposition
		if disposition == "" {
			disposition = "open"
		}
		tag := "[" + disposition + "]"
		if entry.Target != "" {
			tag = "[" + disposition + ": " + entry.Target + "]"
		}
		meta := "owner:" + entry.Owner
		if entry.Criticality != "" {
			meta += " crit:" + entry.Criticality
		}
		if entry.Review != "" {
			meta += " review:" + entry.Review
			if entry.Review < today && (entry.RawDisposition == "" || entry.RawDisposition == "open") {
				meta += " OVERDUE"
			}
		}
		fmt.Fprintf(stdout, "- %s %s (%s)\n    %s\n", entry.Spec, tag, meta, entry.Text)
	}
	if failOverdue && len(overdueEntries) > 0 {
		fmt.Fprintf(stdout, "\nResultado: FALHA (%d follow-up(s) com review vencido)\n", len(overdueEntries))
		return 1
	}
	if len(candidates) > 0 {
		fmt.Fprintf(stdout, "\n## Near-duplicate candidates (%d) — lexical similarity >= %d/100\n", len(candidates), threshold)
		for index, candidate := range candidates {
			fmt.Fprintf(stdout, "\n[%d] specs: %s\n", index+1, strings.Join(candidate.Specs, ", "))
			for _, member := range candidate.Members {
				fmt.Fprintf(stdout, "    - (%s [%s]) %s\n", member.Spec, member.Disposition, member.Text)
			}
		}
	}
	return 0
}

func collectFollowups(root string) []followup {
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*", "spec.md"))
	sort.Strings(paths)
	entries := []followup{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		body := followupHTMLComment.ReplaceAllString(string(raw), "")
		status := frontmatterStatus(body)
		inFinal, inFollowups := false, false
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "## ") {
				heading := strings.TrimSpace(strings.TrimLeft(line, "#0123456789. "))
				inFinal = strings.HasPrefix(strings.ToLower(heading), "final report")
				inFollowups = false
				continue
			}
			if inFinal && strings.HasPrefix(line, "### ") {
				inFollowups = strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "###"))), "follow-up")
				continue
			}
			if !inFollowups {
				continue
			}
			match := followupBullet.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			text, disposition, target := match[1], "", ""
			if parsed := followupDisposition.FindStringSubmatch(text); parsed != nil {
				disposition, target, text = parsed[1], strings.TrimSpace(parsed[2]), strings.TrimSpace(parsed[3])
			}
			stripped, owner, crit, review, by, metaErr := parseFollowupMeta(text)
			if stripped != "" {
				entries = append(entries, followup{
					Spec: filepath.Base(filepath.Dir(path)), SpecStatus: status,
					RawDisposition: disposition, Target: target, Text: stripped,
					Owner: owner, Criticality: crit, Review: review, By: by, MetaErr: metaErr,
				})
			}
		}
	}
	return entries
}

// followupToday returns today's UTC date (YYYY-MM-DD) for overdue math.
// Test override: POSE_FOLLOWUP_TODAY (dogfood determinism, never documented
// as a user surface).
func followupToday() string {
	if v := os.Getenv("POSE_FOLLOWUP_TODAY"); followupReviewDate.MatchString(v) {
		return v
	}
	return time.Now().UTC().Format("2006-01-02")
}

func frontmatterStatus(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "unset"
	}
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			break
		}
		if strings.HasPrefix(line, "status:") {
			return strings.TrimSpace(strings.SplitN(strings.TrimPrefix(line, "status:"), "#", 2)[0])
		}
	}
	return "unset"
}

func normalizeFollowup(text string) string {
	var builder strings.Builder
	space := true
	for _, char := range strings.ToLower(strings.ReplaceAll(text, "`", " ")) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			space = false
		} else if !space {
			builder.WriteByte(' ')
			space = true
		}
	}
	return strings.TrimSpace(builder.String())
}

var followupStopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "uma": true, "para": true,
	"por": true, "com": true, "sem": true, "que": true, "não": true, "nao": true,
	"mais": true, "cada": true, "when": true, "onde": true, "como": true,
}

func followupTokens(text string) map[string]bool {
	tokens := map[string]bool{}
	for _, token := range strings.Fields(text) {
		if len([]rune(token)) > 2 && !followupStopwords[token] {
			tokens[token] = true
		}
	}
	return tokens
}

func followupSimilarity(left, right string) float64 {
	left, right = normalizeFollowup(left), normalizeFollowup(right)
	leftTokens, rightTokens := followupTokens(left), followupTokens(right)
	intersection, union := 0, len(leftTokens)
	for token := range rightTokens {
		if leftTokens[token] {
			intersection++
		} else {
			union++
		}
	}
	jaccard := 0.0
	if union > 0 {
		jaccard = float64(intersection) / float64(union)
	}
	sequence := ratcliffRatio([]rune(left), []rune(right))
	if sequence > jaccard {
		return sequence
	}
	return jaccard
}

func ratcliffRatio(left, right []rune) float64 {
	if len(left)+len(right) == 0 {
		return 1
	}
	return 2 * float64(matchingRunes(left, right)) / float64(len(left)+len(right))
}

func matchingRunes(left, right []rune) int {
	bestLength, bestLeft, bestRight := 0, 0, 0
	for i := range left {
		for j := range right {
			length := 0
			for i+length < len(left) && j+length < len(right) && left[i+length] == right[j+length] {
				length++
			}
			if length > bestLength {
				bestLength, bestLeft, bestRight = length, i, j
			}
		}
	}
	if bestLength == 0 {
		return 0
	}
	return bestLength + matchingRunes(left[:bestLeft], right[:bestRight]) + matchingRunes(left[bestLeft+bestLength:], right[bestRight+bestLength:])
}

func clusterFollowups(entries []followup, threshold float64) []nearDuplicateCandidate {
	parents := make([]int, len(entries))
	for index := range parents {
		parents[index] = index
	}
	var find func(int) int
	find = func(index int) int {
		if parents[index] != index {
			parents[index] = find(parents[index])
		}
		return parents[index]
	}
	for left := range entries {
		for right := left + 1; right < len(entries); right++ {
			if entries[left].Spec != entries[right].Spec && followupSimilarity(entries[left].Text, entries[right].Text) >= threshold {
				parents[find(left)] = find(right)
			}
		}
	}
	groups := map[int][]followup{}
	for index, entry := range entries {
		groups[find(index)] = append(groups[find(index)], entry)
	}
	keys := make([]int, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	result := []nearDuplicateCandidate{}
	for _, key := range keys {
		group, specs := groups[key], map[string]bool{}
		for _, entry := range group {
			specs[entry.Spec] = true
		}
		if len(specs) < 2 {
			continue
		}
		candidate := nearDuplicateCandidate{}
		for _, entry := range group {
			disposition := entry.RawDisposition
			if disposition == "" {
				disposition = "open"
			}
			candidate.Members = append(candidate.Members, nearDuplicateMember{entry.Spec, entry.Text, disposition})
		}
		for spec := range specs {
			candidate.Specs = append(candidate.Specs, spec)
		}
		sort.Strings(candidate.Specs)
		result = append(result, candidate)
	}
	return result
}

func uniqueFollowupSpecs(entries []followup) int {
	specs := map[string]bool{}
	for _, entry := range entries {
		specs[entry.Spec] = true
	}
	return len(specs)
}
