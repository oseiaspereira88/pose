package observability

import "regexp"

// secretLikePatterns is a deliberate, independent copy of the same
// deterministic, offline secret-shape scan used elsewhere in this binary
// (internal/cli's skills/doctor checks) — kept local rather than shared
// across an internal/cli <-> internal/observability import, since neither
// package needs the other's dependency surface for a handful of literal
// regexes (see ADR 2026-07-19-otel-observability-safe-by-construction-signals.md).
var secretLikePatterns = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                   // AWS access key id
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`), // PEM private key
	regexp.MustCompile(`(?i)\bgh[pousr]_[A-Za-z0-9]{20,}`),   // GitHub token shapes
	regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._-]{20,}`), // bearer/JWT-shaped token
}

// pathLikeRE matches an absolute filesystem path (Unix or Windows) of at
// least two segments — the shape of the paths POSE's own error messages
// tend to embed (spec/knowledge file paths, project roots).
var pathLikeRE = regexp.MustCompile(`(?:/[A-Za-z0-9._-]+){2,}|[A-Za-z]:\\(?:[A-Za-z0-9._-]+\\)+[A-Za-z0-9._-]+`)

const redactedPlaceholder = "[REDACTED]"

// Secrets replaces every secret-shaped substring with a fixed placeholder.
func Secrets(s string) string {
	for _, re := range secretLikePatterns {
		s = re.ReplaceAllString(s, redactedPlaceholder)
	}
	return s
}

// Paths replaces every absolute-filesystem-path-shaped substring with a
// fixed placeholder, so an emitted log line can describe *that* an error
// referenced a file without disclosing the repository layout or username.
func Paths(s string) string {
	return pathLikeRE.ReplaceAllString(s, "[PATH]")
}

// Message applies every redaction pass — the one function log/error
// fields should be piped through before being attached to a span, metric
// attribute or log record (R3: redact paths, tokens and payloads).
func Message(s string) string {
	return Paths(Secrets(s))
}
