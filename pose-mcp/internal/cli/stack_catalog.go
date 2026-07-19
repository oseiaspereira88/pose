package cli

// Polyglot stack catalog (spec pose-stack-catalog-expansion): maintained
// detection profiles for Python and .NET alongside the existing Node.js, Go,
// Rust and Java markers. Detection is offline, bounded and deterministic —
// it reads directory entries and marker filenames, never executes project
// files. Each profile declares its prerequisite native tool, a priority used
// to resolve multiple managers in the same directory (higher-priority marker
// wins; conflicts are reported, never silently dropped) and the confidence
// of that resolution.

import (
	"os/exec"
	"path/filepath"
	"sort"
)

type stackProfile struct {
	ID           string `json:"id"`
	Stack        string `json:"stack"`   // maps to a validation-matrix.json stacks key
	Manager      string `json:"manager"` // "" when the stack has a single manager (e.g. dotnet)
	Marker       string `json:"marker"`  // exact filename, or a "*.ext" suffix pattern
	Prerequisite string `json:"prerequisite"`
	// Priority resolves multiple present markers in one directory: lower
	// wins. Ties are not expected within a stack (profiles are ordered).
	Priority int `json:"priority"`
}

// stackCatalog is the maintained profile registry (R2). Node/Go/Rust/Java
// keep their existing single-marker, single-manager behavior unchanged
// (compatibility) and are listed here too so `pose stacks` reports one
// complete, queryable catalog instead of a partial one.
var stackCatalog = []stackProfile{
	{ID: "node", Stack: "node", Marker: "package.json", Prerequisite: "npm", Priority: 1},
	{ID: "go", Stack: "go", Marker: "go.mod", Prerequisite: "go", Priority: 1},
	{ID: "rust", Stack: "rust", Marker: "Cargo.toml", Prerequisite: "cargo", Priority: 1},
	{ID: "java-maven", Stack: "java", Manager: "maven", Marker: "pom.xml", Prerequisite: "mvn", Priority: 1},
	{ID: "java-gradle", Stack: "java", Manager: "gradle", Marker: "build.gradle", Prerequisite: "gradle", Priority: 2},
	{ID: "java-gradle-kts", Stack: "java", Manager: "gradle", Marker: "build.gradle.kts", Prerequisite: "gradle", Priority: 2},
	{ID: "python-poetry", Stack: "python", Manager: "poetry", Marker: "poetry.lock", Prerequisite: "poetry", Priority: 1},
	{ID: "python-pipenv", Stack: "python", Manager: "pipenv", Marker: "Pipfile", Prerequisite: "pipenv", Priority: 2},
	{ID: "python-pip", Stack: "python", Manager: "pip", Marker: "requirements.txt", Prerequisite: "pytest", Priority: 3},
	{ID: "python-setuptools", Stack: "python", Manager: "setuptools", Marker: "setup.py", Prerequisite: "pytest", Priority: 4},
	{ID: "python-pep517", Stack: "python", Manager: "pep517", Marker: "pyproject.toml", Prerequisite: "pytest", Priority: 5},
	{ID: "dotnet-solution", Stack: "dotnet", Manager: "dotnet", Marker: "*.sln", Prerequisite: "dotnet", Priority: 1},
	{ID: "dotnet-project", Stack: "dotnet", Manager: "dotnet", Marker: "*.csproj", Prerequisite: "dotnet", Priority: 2},
	{ID: "dotnet-project-fs", Stack: "dotnet", Manager: "dotnet", Marker: "*.fsproj", Prerequisite: "dotnet", Priority: 2},
	{ID: "dotnet-project-vb", Stack: "dotnet", Manager: "dotnet", Marker: "*.vbproj", Prerequisite: "dotnet", Priority: 2},
}

func stackMarkerMatches(name, marker string) bool {
	if ext, ok := isSuffixMarker(marker); ok {
		return filepath.Ext(name) == ext
	}
	return name == marker
}

func isSuffixMarker(marker string) (string, bool) {
	if len(marker) > 2 && marker[0] == '*' && marker[1] == '.' {
		return marker[1:], true
	}
	return "", false
}

// stackDetection is one profile's resolution outcome in a directory.
type stackDetection struct {
	Profile           stackProfile `json:"profile"`
	Winner            bool         `json:"winner"`
	Confidence        string       `json:"confidence"` // high | medium (conflict)
	PrerequisiteFound bool         `json:"prerequisite_found"`
	OverrideHint      string       `json:"override_hint"`
}

// detectStackProfiles matches every catalog profile against entry names
// found directly in dir (no recursion, no file execution) and resolves
// conflicts by priority. Multiple stacks (e.g. node + python in the same
// directory) are independent and each gets its own winner.
func detectStackProfiles(dirEntryNames []string) []stackDetection {
	byStack := map[string][]stackProfile{}
	for _, p := range stackCatalog {
		for _, name := range dirEntryNames {
			if stackMarkerMatches(name, p.Marker) {
				byStack[p.Stack] = append(byStack[p.Stack], p)
				break
			}
		}
	}
	var out []stackDetection
	stacks := make([]string, 0, len(byStack))
	for s := range byStack {
		stacks = append(stacks, s)
	}
	sort.Strings(stacks)
	for _, s := range stacks {
		profiles := byStack[s]
		sort.Slice(profiles, func(i, j int) bool { return profiles[i].Priority < profiles[j].Priority })
		conflict := len(profiles) > 1
		for i, p := range profiles {
			_, err := exec.LookPath(p.Prerequisite)
			confidence := "high"
			if conflict {
				confidence = "medium"
			}
			hint := "override via .pose/indexes/validation-matrix.json moduleOverrides"
			out = append(out, stackDetection{
				Profile: p, Winner: i == 0, Confidence: confidence,
				PrerequisiteFound: err == nil, OverrideHint: hint,
			})
		}
	}
	return out
}
