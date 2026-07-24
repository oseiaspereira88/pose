package pose

// Shared, deterministic, offline security scan (originally spec
// pose-agent-skills-conformance, extended to governed docs by spec
// pose-docs-governance-contract R2/Segurança): a schema-valid skill or a
// well-formed doc can still tell a reader to do something unsafe, or an
// author can paste a real credential into prose just as easily as code.
// Defense in depth — never a replacement for the dedicated gitleaks gate.

import "regexp"

// UnsafeContentPatterns flag instructions that would push a reader toward
// unreviewed remote code execution or disabled safety checks.
var UnsafeContentPatterns = []*regexp.Regexp{
	regexp.MustCompile(`curl[^\n]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`),
	regexp.MustCompile(`wget[^\n]*\|\s*(sudo\s+)?(sh|bash|zsh)\b`),
	regexp.MustCompile(`\brm\s+-rf\s+/(\s|$)`),
	regexp.MustCompile(`--no-verify\b`),
	regexp.MustCompile(`(?i)\bdisable\s+(ssl|tls)\s+verif`),
}

// SecretLikePatterns match common credential shapes.
var SecretLikePatterns = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                   // AWS access key id
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), // PEM private key
	regexp.MustCompile(`(?i)\bgh[pousr]_[A-Za-z0-9]{20,}`),   // GitHub token shapes
}

// ScanContentSecurity applies both pattern sets to content, returning one
// message per distinct match. Pure and side-effect-free.
func ScanContentSecurity(content []byte) []string {
	var issues []string
	for _, re := range UnsafeContentPatterns {
		if re.Match(content) {
			issues = append(issues, "matches unsafe instruction pattern "+re.String())
		}
	}
	for _, re := range SecretLikePatterns {
		if re.Match(content) {
			issues = append(issues, "matches secret-shaped pattern "+re.String())
		}
	}
	return issues
}
