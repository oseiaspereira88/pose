package cli

// Public claims contract (spec pose-public-claims-contract).
//
// POSE's own public surfaces drifted: the landing page, the homepage, the
// structured metadata and thirteen docs pages described 1.4.x while the
// product shipped 1.7.10, and the README carried a "What's new in v1.4.3"
// section. Fixing the strings is an hour of work that decays immediately;
// what was missing is a gate tying a public claim to a released fact.
//
// Surfaces are declared, never discovered. A scanner that flags every
// version-shaped string in the repository would fail on changelogs, historical
// assessments and release notes — all of which are supposed to name old
// versions. Only what a surface *claims about the current product* is checked.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Version-claim shapes. Each carries the version in group 1 so one extractor
// serves prose, shell pins and URLs alike.
var publicVersionPatterns = []*regexp.Regexp{
	// Prose: "POSE 1.7.10", "POSE v1.7", "POSE v1.4.3".
	regexp.MustCompile(`\bPOSE\s+v?(\d+\.\d+(?:\.\d+)?)\b`),
	// Release lines: "POSE 6.x". The version is the major alone, which
	// versionMatchesRelease compares at that precision.
	regexp.MustCompile(`\bPOSE\s+v?(\d+)\.x\b`),
	// Shell/PowerShell install pins: `V=1.7.10`, `$V = "1.7.10"`.
	regexp.MustCompile(`\$?V\s*=\s*"?(\d+\.\d+\.\d+)"?`),
	// Pinned release asset URLs.
	regexp.MustCompile(`releases/download/v(\d+\.\d+\.\d+)/`),
	// Action pins: `pose-action@v1.7.10`, including HTML-escaped `&#64;`.
	regexp.MustCompile(`pose-action(?:@|&#64;)v(\d+\.\d+\.\d+)`),
	// Structured metadata.
	regexp.MustCompile(`"softwareVersion"\s*:\s*"(\d+\.\d+(?:\.\d+)?)"`),
}

const (
	// versionClaimsNone: the surface must carry no version claim at all. This
	// is the evergreen case, and it has to be enforceable rather than merely
	// recommended — most of the observed drift was not a maintenance failure
	// but a design failure, where the version sat somewhere it had no reason
	// to be.
	versionClaimsNone = "none"
	// versionClaimsCurrent: the surface may name a version, but only the
	// released one (an install pin, an Action pin).
	versionClaimsCurrent = "current-only"
)

type publicSurface struct {
	Path          string `json:"path"`
	VersionClaims string `json:"version_claims"`
	Note          string `json:"note,omitempty"`
}

type publicClaimsContract struct {
	SchemaVersion int `json:"schema_version"`
	Product       struct {
		Name          string `json:"name"`
		Category      string `json:"category"`
		License       string `json:"license"`
		Repository    string `json:"repository"`
		DocsCanonical string `json:"docs_canonical"`
	} `json:"product"`
	// VersionSource is a repo-relative JSON file carrying `engine_version`.
	// Reading the released version from an artifact the release gate already
	// produces keeps this check offline, which matters because it has to run
	// before a release exists.
	VersionSource string          `json:"version_source"`
	DocsHosts     publicDocsHosts `json:"docs_hosts"`
	Surfaces      []publicSurface `json:"surfaces"`
}

type publicDocsHosts struct {
	Canonical  string   `json:"canonical"`
	Deprecated []string `json:"deprecated"`
}

type publicClaimFinding struct {
	Surface  string `json:"surface"`
	Claim    string `json:"claim"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func loadPublicClaims(root string) (publicClaimsContract, error) {
	var c publicClaimsContract
	raw, err := os.ReadFile(filepath.Join(root, ".pose", "public", "claims.json"))
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(raw, &c); err != nil {
		return c, fmt.Errorf("parsing .pose/public/claims.json: %w", err)
	}
	if c.SchemaVersion != 1 {
		return c, fmt.Errorf("unsupported public claims schema %d", c.SchemaVersion)
	}
	return c, nil
}

// releasedVersion reads engine_version from the declared version source.
func releasedVersion(root, source string) (string, error) {
	if strings.TrimSpace(source) == "" {
		return "", fmt.Errorf("version_source is empty")
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source)))
	if err != nil {
		return "", err
	}
	var doc struct {
		EngineVersion string `json:"engine_version"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("parsing %s: %w", source, err)
	}
	if strings.TrimSpace(doc.EngineVersion) == "" {
		return "", fmt.Errorf("%s declares no engine_version", source)
	}
	return doc.EngineVersion, nil
}

// publicVersionProvenance separates three facts the gate used to collapse
// into `released_version` (spec pose-public-claims-publication-provenance):
// the candidate the local compatibility metadata names, whether a release
// manifest for it was prepared (and how far its lifecycle got), and the most
// recent version whose publication is proven by retained release events.
// Local metadata supports candidate consistency only; it never proves that a
// version was published.
type publicVersionProvenance struct {
	CandidateVersion      string `json:"candidate_version"`
	CandidateSource       string `json:"candidate_source"`
	CandidateState        string `json:"candidate_state"`
	PreparedVersion       string `json:"prepared_version,omitempty"`
	PreparedSource        string `json:"prepared_source,omitempty"`
	PublishedVersion      string `json:"published_version,omitempty"`
	PublishedSource       string `json:"published_source,omitempty"`
	PublishedVersionState string `json:"published_version_state"`
}

// resolvePublicVersionProvenance reads only retained, local release records:
// the gate stays offline and still runs before a release exists.
func resolvePublicVersionProvenance(root, source, candidate string) publicVersionProvenance {
	out := publicVersionProvenance{CandidateVersion: candidate, CandidateSource: source, CandidateState: "unprepared", PublishedVersionState: "unproven"}
	tag := "v" + strings.TrimPrefix(candidate, "v")
	status, err := (posemodel.Store{Root: root}).GetReleaseStatus("")
	if err != nil || status == nil {
		out.CandidateState = "unknown"
		out.PublishedVersionState = "unknown"
		return out
	}
	for _, release := range status.Releases {
		if release.Version == tag {
			out.CandidateState = release.State
			out.PreparedVersion = strings.TrimPrefix(release.Version, "v")
			out.PreparedSource = ".pose/releases/" + release.Version + "/manifest.json"
		}
		// Releases are sorted newest first; the first with retained
		// publication evidence is the latest proven publication.
		if out.PublishedVersion == "" && (release.State == "published" || release.State == "verified") && len(release.Gaps) == 0 {
			out.PublishedVersion = strings.TrimPrefix(release.Version, "v")
			out.PublishedSource = ".pose/releases/" + release.Version + "/events.jsonl"
			out.PublishedVersionState = release.State
		}
	}
	return out
}

// versionClaimsIn returns every distinct version a surface claims, in stable
// order so output does not churn between runs.
func versionClaimsIn(body string) []string {
	seen := map[string]bool{}
	for _, re := range publicVersionPatterns {
		for _, m := range re.FindAllStringSubmatch(body, -1) {
			seen[m[1]] = true
		}
	}
	out := make([]string, 0, len(seen))
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// versionMatchesRelease compares a claim against the released version at the
// precision the claim uses: "1.7" is satisfied by 1.7.10, "1.7.9" is not.
// Without this, dropping a patch digit would read as drift rather than as the
// deliberate choice to name a release line.
func versionMatchesRelease(claim, release string) bool {
	c := strings.Split(claim, ".")
	r := strings.Split(release, ".")
	if len(c) > len(r) {
		return false
	}
	for i := range c {
		if c[i] != r[i] {
			return false
		}
	}
	return true
}

func checkPublicSurface(root string, s publicSurface, c publicClaimsContract, release string) []publicClaimFinding {
	var out []publicClaimFinding
	add := func(claim, msg string) {
		out = append(out, publicClaimFinding{Surface: s.Path, Claim: claim, Severity: "error", Message: msg})
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(s.Path)))
	if err != nil {
		add("existence", fmt.Sprintf("declared surface is unreadable: %v", err))
		return out
	}
	body := string(raw)

	switch s.VersionClaims {
	case versionClaimsNone:
		for _, v := range versionClaimsIn(body) {
			add("version", fmt.Sprintf(
				"declares version %s, but this surface is declared evergreen — a marketing surface should not age on every patch; move the version next to the install command or drop it",
				v))
		}
	case versionClaimsCurrent:
		for _, v := range versionClaimsIn(body) {
			if !versionMatchesRelease(v, release) {
				add("version", fmt.Sprintf(
					"declares version %s but the released version is %s", v, release))
			}
		}
	default:
		add("contract", fmt.Sprintf(
			"unknown version_claims %q (expected %q or %q)", s.VersionClaims, versionClaimsNone, versionClaimsCurrent))
	}

	for _, host := range c.DocsHosts.Deprecated {
		if host == "" || !strings.Contains(body, host) {
			continue
		}
		add("docs-route", fmt.Sprintf(
			"links to %s, which is not the canonical documentation host (%s) — two live documentation sites split ranking and leave readers unable to tell which is current",
			host, c.DocsHosts.Canonical))
	}
	return out
}

func cmdPublicClaims(root string, args []string, stdout, stderr io.Writer) int {
	mode, asJSON := "strict", false
	for _, a := range args {
		switch a {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--json":
			asJSON = true
		default:
			return usageError(stderr, "Usage: pose public-claims [--strict|--tolerant] [--json]")
		}
	}

	contract, err := loadPublicClaims(root)
	if errors.Is(err, os.ErrNotExist) {
		// Still an error — the gate cannot run and saying otherwise would let a
		// pipeline report claims as verified when nothing was checked. What
		// changes is what the operator is told: `open ...: no such file or
		// directory` names a symptom and hides that the contract is opt-in, that
		// nothing scaffolds it, and how to start one.
		fmt.Fprintln(stderr, "pose public-claims: this instance declares no public claims contract")
		fmt.Fprintln(stderr, "  .pose/public/claims.json is absent. The contract is opt-in and no scaffold")
		fmt.Fprintln(stderr, "  creates it: it declares which surfaces make claims about the current")
		fmt.Fprintln(stderr, "  product, so one that contradicts a released fact fails this gate instead")
		fmt.Fprintln(stderr, "  of ageing quietly. Surfaces are declared, never discovered — scanning for")
		fmt.Fprintln(stderr, "  version-shaped strings would flag changelogs and release notes, which are")
		fmt.Fprintln(stderr, "  supposed to name old versions.")
		// `.pose/public` is not in instanceDirs and would not survive a clone if
		// it were, since Git does not track empty directories. The instruction
		// creates it, so it works on a fresh install and on a checkout alike.
		fmt.Fprintln(stderr, "  To start: mkdir -p .pose/public && cp .pose/templates/public-claims.json .pose/public/claims.json")
		return 2
	}
	if err != nil {
		fmt.Fprintf(stderr, "pose public-claims: %v\n", err)
		return 2
	}
	release, err := releasedVersion(root, contract.VersionSource)
	if err != nil {
		fmt.Fprintf(stderr, "pose public-claims: %v\n", err)
		return 2
	}

	surfaces := append([]publicSurface(nil), contract.Surfaces...)
	sort.Slice(surfaces, func(i, j int) bool { return surfaces[i].Path < surfaces[j].Path })

	findings := make([]publicClaimFinding, 0)
	for _, s := range surfaces {
		findings = append(findings, checkPublicSurface(root, s, contract, release)...)
	}

	provenance := resolvePublicVersionProvenance(root, contract.VersionSource, release)
	if asJSON {
		payload := struct {
			SchemaVersion int `json:"schema_version"`
			// Release is the legacy field: it always held the candidate read
			// from local metadata, never a proven publication. It keeps that
			// value for one minor; read candidate_version and
			// published_version instead.
			Release          string                  `json:"released_version"`
			DeprecatedFields map[string]string       `json:"deprecated_fields"`
			Provenance       publicVersionProvenance `json:"version_provenance"`
			Surfaces         int                     `json:"surfaces_checked"`
			Findings         []publicClaimFinding    `json:"findings"`
		}{1, release, map[string]string{"released_version": "holds the local candidate version, not a proven publication; use version_provenance.candidate_version and version_provenance.published_version"}, provenance, len(surfaces), findings}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			fmt.Fprintf(stderr, "pose public-claims: %v\n", err)
			return 2
		}
	} else {
		for _, f := range findings {
			fmt.Fprintf(stdout, "[%s] %s: %s: %s\n", strings.ToUpper(f.Severity), f.Surface, f.Claim, f.Message)
		}
		// The surfaces are checked against the candidate. Whether that
		// candidate was prepared or published is reported separately and
		// only from retained release records.
		out := render(stdout, stderr)
		out.Field("public-claims.candidate_version", provenance.CandidateVersion)
		out.Field("public-claims.candidate_state", provenance.CandidateState)
		if provenance.PublishedVersion != "" {
			out.Field("public-claims.published_version", provenance.PublishedVersion+" ("+provenance.PublishedVersionState+", "+provenance.PublishedSource+")")
		} else {
			out.Field("public-claims.published_version", provenance.PublishedVersionState)
		}
		fmt.Fprintf(stdout, "public-claims.surfaces=%d\n", len(surfaces))
		fmt.Fprintf(stdout, "public-claims.errors=%d\n", len(findings))
	}

	if len(findings) > 0 {
		if !asJSON {
			fmt.Fprintln(stdout, "Result: FAILURE")
		}
		if mode == "strict" {
			return 1
		}
		if !asJSON {
			fmt.Fprintln(stdout, "Result: TOLERATED_FAILURE")
		}
		return 0
	}
	if !asJSON {
		fmt.Fprintln(stdout, "Result: SUCCESS")
	}
	return 0
}
