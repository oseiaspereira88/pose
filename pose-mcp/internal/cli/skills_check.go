package cli

// Agent Skills conformance (spec pose-agent-skills-conformance): CI-checked
// structural validation, POSE compatibility metadata and a deterministic,
// offline security scan for every shipped skill. Discovery/workflow
// conformance fixtures live in skills_check_test.go against this repo's own
// .agents/skills/ tree (dogfooding).

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/harne8/pose-mcp/internal/scaffold"
)

var skillLinkRE = regexp.MustCompile(`\]\(([^)]+)\)`)

// unsafeSkillPatterns flag instructions that would push an agent toward
// unreviewed remote code execution — a schema-valid skill can still tell an
// agent to do something unsafe (spec's own stated technical risk).
var unsafeSkillPatterns = []*regexp.Regexp{
	regexp.MustCompile(`curl[^\n]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`),
	regexp.MustCompile(`wget[^\n]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`),
	regexp.MustCompile(`\brm\s+-rf\s+/(\s|$)`),
	regexp.MustCompile(`--no-verify\b`),
	regexp.MustCompile(`(?i)\bdisable\s+(ssl|tls)\s+verif`),
}

// secretLikePatterns are a deterministic, offline, defense-in-depth scan —
// not a substitute for the dedicated gitleaks gate in CI (spec
// pose-ossf-security-baseline); it exists because a skill file is prose an
// author can paste a real credential into just as easily as code.
var secretLikePatterns = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                   // AWS access key id
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), // PEM private key
	regexp.MustCompile(`(?i)\bgh[pousr]_[A-Za-z0-9]{20,}`),   // GitHub token shapes
}

type skillIssue struct {
	Skill    string `json:"skill"`
	Severity string `json:"severity"` // error | warning
	Message  string `json:"message"`
}

// checkSkillFrontmatter validates required Agent Skills fields plus POSE's
// additive compatibility metadata (R2): pose_schema_range, clients,
// capabilities. Returns issues; never panics on malformed input.
func checkSkillFrontmatter(slug string, fm map[string]string) []skillIssue {
	var issues []skillIssue
	req := func(key string) {
		if strings.TrimSpace(fm[key]) == "" {
			issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("missing required frontmatter field %q", key)})
		}
	}
	req("name")
	req("description")
	req("when_to_use")
	if fm["name"] != "" && fm["name"] != slug {
		issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("frontmatter name %q does not match directory %q", fm["name"], slug)})
	}
	req("pose_schema_range")
	if r := fm["pose_schema_range"]; r != "" {
		if _, _, err := parseSchemaRange(r); err != nil {
			issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("invalid pose_schema_range %q: %v", r, err)})
		}
	}
	req("clients")
	req("capabilities")
	return issues
}

func parseSchemaRange(r string) (min, max int, err error) {
	lo, hi, ok := strings.Cut(r, "-")
	if !ok {
		return 0, 0, fmt.Errorf("expected \"min-max\"")
	}
	min, err = strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		return 0, 0, err
	}
	max, err = strconv.Atoi(strings.TrimSpace(hi))
	if err != nil {
		return 0, 0, err
	}
	if min > max || min < 1 {
		return 0, 0, fmt.Errorf("min must be >=1 and <= max")
	}
	return min, max, nil
}

func splitCSVList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// checkSkillLinks resolves every markdown link in body relative to the
// skill's directory and requires the target to exist and stay inside root
// (path escape, part of the Security requirement).
func checkSkillLinks(root, slug, skillDir, body string) []skillIssue {
	var issues []skillIssue
	for _, m := range skillLinkRE.FindAllStringSubmatch(body, -1) {
		target := m[1]
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") {
			continue // external URL or in-page anchor
		}
		target = strings.SplitN(target, "#", 2)[0]
		if target == "" {
			continue
		}
		abs := filepath.Join(skillDir, filepath.FromSlash(target))
		rel, err := filepath.Rel(root, abs)
		if err != nil || !confinedRelativePath(rel) {
			issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("linked resource escapes the repository: %s", target)})
			continue
		}
		if _, err := os.Stat(abs); err != nil {
			issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("linked resource not found: %s", target)})
		}
	}
	return issues
}

func checkSkillSecurity(slug, body string) []skillIssue {
	var issues []skillIssue
	for _, re := range unsafeSkillPatterns {
		if re.MatchString(body) {
			issues = append(issues, skillIssue{slug, "error", fmt.Sprintf("instructs an unsafe pattern matching %q", re.String())})
		}
	}
	for _, re := range secretLikePatterns {
		if re.MatchString(body) {
			issues = append(issues, skillIssue{slug, "error", "content matches a known secret-shaped pattern — remove it"})
		}
	}
	return issues
}

// checkSkillClients cross-validates a declared "claude-code" client against
// the actual .claude/skills symlink registry the installer materializes —
// a client cannot be declared supported without a real link surface.
func checkSkillClients(slug string, clients []string) []skillIssue {
	var issues []skillIssue
	wantsClaude := false
	for _, c := range clients {
		if c == "claude-code" {
			wantsClaude = true
		}
	}
	if wantsClaude {
		if _, ok := scaffold.ClaudeSkillLinks[slug]; !ok {
			issues = append(issues, skillIssue{slug, "error", "declares client \"claude-code\" but has no .claude/skills symlink registered in scaffold.ClaudeSkillLinks"})
		}
	}
	return issues
}

// checkOneSkill runs every conformance check for a single skill directory.
func checkOneSkill(root, slug, dir string) []skillIssue {
	path := filepath.Join(dir, "SKILL.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		return []skillIssue{{slug, "error", "SKILL.md not found (layout)"}}
	}
	body := string(raw)
	fm, ferr := readFlatFrontmatter(path)
	if ferr != nil {
		return []skillIssue{{slug, "error", fmt.Sprintf("frontmatter: %v", ferr)}}
	}
	issues := checkSkillFrontmatter(slug, fm)
	issues = append(issues, checkSkillLinks(root, slug, dir, body)...)
	issues = append(issues, checkSkillSecurity(slug, body)...)
	issues = append(issues, checkSkillClients(slug, splitCSVList(fm["clients"]))...)
	return issues
}

func cmdSkillsCheck(root string, args []string, stdout, stderr io.Writer) int {
	mode := "strict"
	for _, a := range args {
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		default:
			return usageError(stderr, "Usage: pose skills-check [--strict|--tolerant]")
		}
	}
	skillsDir := filepath.Join(root, ".agents", "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	var all []skillIssue
	for _, slug := range slugs {
		all = append(all, checkOneSkill(root, slug, filepath.Join(skillsDir, slug))...)
	}
	errors, warnings := 0, 0
	for _, iss := range all {
		if iss.Severity == "error" {
			errors++
		} else {
			warnings++
		}
		fmt.Fprintf(stdout, "[%s] %s: %s\n", strings.ToUpper(iss.Severity), iss.Skill, iss.Message)
	}
	fmt.Fprintf(stdout, "skills.checked=%d\nskills.errors=%d\nskills.warnings=%d\n", len(slugs), errors, warnings)
	if errors > 0 {
		fmt.Fprintln(stdout, "Result: FAILURE")
		if mode == "strict" {
			return 1
		}
		fmt.Fprintln(stdout, "Result: TOLERATED_FAILURE")
		return 0
	}
	fmt.Fprintln(stdout, "Result: SUCCESS")
	return 0
}
