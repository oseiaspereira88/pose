// Composition contract (spec pose-composition-contract).
//
// The harne8 monorepo composes this repository's images and restates, in its own
// compose file, facts this repository owns: build contexts, the internal port
// and the environment variable names the service is configured through. Nothing
// reconciled the copies, so a rename here failed nothing there — the container
// started with a default and nobody was told.
//
// This publishes those facts as `composition-contract.json` and keeps the file
// honest. The environment surface is *derived*, not restated: the prefixed keys
// come from mcpenforce.ConfigEnvSuffixes — the list ConfigFromEnv itself reads —
// and the rest from the literals in this module's source. Writing that list by
// hand would reproduce, one layer down, the defect this contract closes.
//
// Regenerate after an intentional change:
//
//	UPDATE_COMPOSITION_CONTRACT=1 go -C pose-mcp test ./internal/version/ -run CompositionContract
package version_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	mcpenforce "github.com/harne8/mcp-enforce"
)

const contractComment = "Facts this repository owns about composing its images: build context, " +
	"Dockerfile, exposed port and the environment variables each service reads. " +
	"A consumer (harne8's docker-compose) should derive from this rather than restate it. " +
	"Environment names are derived from the source, including the prefixed keys that exist " +
	"nowhere as literals; regenerate with UPDATE_COMPOSITION_CONTRACT=1 (spec pose-composition-contract)."

var (
	literalEnvRe = regexp.MustCompile(`os\.Getenv\("((?:POSE|HARNE8|GF)_[A-Z_]+)"\)`)
	exposeRe     = regexp.MustCompile(`(?m)^EXPOSE\s+(\d+)`)
)

// poseMCPEnvironment derives the full configuration surface of pose-mcp: the
// literal reads in this module, plus every prefixed key the enforcement library
// reads on its behalf.
func poseMCPEnvironment(t *testing.T, root string) []string {
	t.Helper()
	names := map[string]bool{}
	for _, key := range mcpenforce.ConfigEnvSuffixes {
		names["POSE_MCP_"+key] = true
	}
	walkGoSources(t, filepath.Join(root, "pose-mcp"), names)
	// Declared by the image itself rather than read through os.Getenv.
	names["POSE_MCP_ADDR"] = true
	return sortedKeys(names)
}

func sidecarEnvironment(t *testing.T, root string) []string {
	t.Helper()
	names := map[string]bool{}
	for _, key := range mcpenforce.ConfigEnvSuffixes {
		names["GF_SIDECAR_"+key] = true
	}
	walkGoSources(t, filepath.Join(root, "mcp-enforce"), names)
	names["GF_SIDECAR_ADDR"] = true
	return sortedKeys(names)
}

func walkGoSources(t *testing.T, dir string, into map[string]bool) {
	t.Helper()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for _, m := range literalEnvRe.FindAllStringSubmatch(string(raw), -1) {
			into[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func exposedPort(t *testing.T, dockerfile string) int {
	t.Helper()
	raw, err := os.ReadFile(dockerfile)
	if err != nil {
		t.Fatalf("reading %s: %v", dockerfile, err)
	}
	m := exposeRe.FindAllStringSubmatch(string(raw), -1)
	if len(m) == 0 {
		t.Fatalf("%s declares no EXPOSE", dockerfile)
	}
	// The last EXPOSE is the runtime stage's; earlier ones would be builder stages.
	port := 0
	for _, digit := range m[len(m)-1][1] {
		port = port*10 + int(digit-'0')
	}
	return port
}

func TestCompositionContract(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "composition-contract.json")

	want := map[string]any{
		"schema_version": 1,
		"comment":        contractComment,
		"images": []any{
			map[string]any{
				"name": "pose-mcp",
				// Built from the repository root: pose-mcp consumes the sibling
				// mcp-enforce module through `replace => ../mcp-enforce`, which
				// must be present in the build context.
				"dockerfile":    "pose-mcp/Dockerfile",
				"build_context": ".",
				"port":          exposedPort(t, filepath.Join(root, "pose-mcp", "Dockerfile")),
				"environment":   poseMCPEnvironment(t, root),
			},
			map[string]any{
				"name": "mcp-enforce-sidecar",
				// Built from its own directory: the Dockerfile copies go.mod from
				// the context root.
				"dockerfile":    "mcp-enforce/Dockerfile",
				"build_context": "mcp-enforce",
				"port":          exposedPort(t, filepath.Join(root, "mcp-enforce", "Dockerfile")),
				"environment":   sidecarEnvironment(t, root),
			},
		},
	}

	rendered, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("rendering contract: %v", err)
	}
	rendered = append(rendered, '\n')

	if os.Getenv("UPDATE_COMPOSITION_CONTRACT") != "" {
		if err := os.WriteFile(path, rendered, 0o644); err != nil {
			t.Fatalf("writing contract: %v", err)
		}
		t.Log("composition-contract.json regenerated")
		return
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading composition-contract.json: %v — regenerate with UPDATE_COMPOSITION_CONTRACT=1", err)
	}

	var got, expected any
	if err := json.Unmarshal(onDisk, &got); err != nil {
		t.Fatalf("parsing composition-contract.json: %v", err)
	}
	if err := json.Unmarshal(rendered, &expected); err != nil {
		t.Fatalf("parsing rendered contract: %v", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("composition-contract.json disagrees with this repository — a consumer composing from it would configure the service wrongly.\n"+
			"Regenerate with UPDATE_COMPOSITION_CONTRACT=1 after confirming the change is intended.\n\non disk:\n%s\nderived:\n%s",
			string(onDisk), string(rendered))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", "..", ".."))
}

// TestDockerfilePortIsSingleSourced pins EXPOSE as the authoritative port.
//
// The internal port was written down four times: the Dockerfile's ENV, its
// EXPOSE, the compose mapping in the monorepo, and — once the contract shipped —
// a fourth copy. Adding a declaration that merely agrees today is how the
// duplication this project keeps closing gets created.
//
// EXPOSE is the source: the contract derives from it, and the ENV default in the
// same file must agree with it. The compose mapping is the consumer's copy and
// is the reason the contract exists.
func TestDockerfilePortIsSingleSourced(t *testing.T) {
	root := repoRoot(t)
	addrRe := regexp.MustCompile(`(?m)^ENV\s+\w*_ADDR=:(\d+)`)
	for _, df := range []string{"pose-mcp/Dockerfile", "mcp-enforce/Dockerfile"} {
		path := filepath.Join(root, df)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", df, err)
		}
		m := addrRe.FindStringSubmatch(string(raw))
		if m == nil {
			t.Errorf("%s declares no ENV *_ADDR default — the port would live only in EXPOSE and the consumer's mapping", df)
			continue
		}
		exposed := exposedPort(t, path)
		if fmt.Sprintf("%d", exposed) != m[1] {
			t.Errorf("%s: ENV *_ADDR is :%s but EXPOSE is %d — the image advertises one port and defaults to another, and the published contract derives from EXPOSE",
				df, m[1], exposed)
		}
	}
}
