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
	"unicode"
)

type followup struct {
	Spec           string `json:"spec"`
	SpecStatus     string `json:"spec_status"`
	RawDisposition string `json:"raw_disposition"`
	Target         string `json:"target"`
	Text           string `json:"text"`
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

func cmdFollowups(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	all, jsonOut, scopeSet, threshold := false, false, false, 60
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
	openEntries := make([]followup, 0, len(allEntries))
	for _, entry := range allEntries {
		if entry.RawDisposition == "" || entry.RawDisposition == "open" {
			openEntries = append(openEntries, entry)
		}
	}
	selected := openEntries
	if all {
		selected = allEntries
	}
	candidates := clusterFollowups(allEntries, float64(threshold)/100)
	if jsonOut {
		specs := map[string]bool{}
		for _, entry := range allEntries {
			specs[entry.Spec] = true
		}
		payload := map[string]any{
			"total": len(allEntries), "open": len(openEntries), "specs": len(specs),
			"similarity_threshold": threshold, "items": selected,
			"near_duplicate_candidates": candidates,
		}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: serializing follow-ups: %v\n", "Erro: serializar follow-ups: %v\n"), err)
			return 1
		}
		return 0
	}
	label := "open follow-ups"
	if all {
		label = "all follow-ups"
	}
	fmt.Fprintf(stdout, "# POSE follow-ups — %s\n# total=%d open=%d specs=%d\n\n", label, len(allEntries), len(openEntries), uniqueFollowupSpecs(allEntries))
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
		fmt.Fprintf(stdout, "- %s %s\n    %s\n", entry.Spec, tag, entry.Text)
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
			if text != "" {
				entries = append(entries, followup{filepath.Base(filepath.Dir(path)), status, disposition, target, text})
			}
		}
	}
	return entries
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
