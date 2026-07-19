package mcpserver

// Exact catalog conformance tests (spec pose-mcp-catalog-conformance):
// R1 golden contract over tool IDs and input schemas, R2 optional-tool
// activation, R3 docs and registry metadata checked against the same catalog.

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update", false, "rewrite testdata/tool-catalog.golden.json from the runtime catalog")

const goldenPath = "testdata/tool-catalog.golden.json"

// catalogDocument is the frozen public contract: the exact tools/list payload
// plus the per-tool governance record.
func catalogDocument(t *testing.T) []byte {
	t.Helper()
	doc := map[string]any{
		"catalog_version": 1,
		"tools":           toolDefinitions(),
		"governance":      catalogGovernance,
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshaling catalog: %v", err)
	}
	return append(b, '\n')
}

// R1: runtime catalog must match the reviewed golden byte-for-byte.
func TestCatalogMatchesGolden(t *testing.T) {
	got := catalogDocument(t)
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden (run `go test -run Golden -update` once, then review the diff): %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("tool catalog drifted from the reviewed golden.\nIf the change is intentional, run `go test ./internal/mcpserver -run Golden -update`, review the diff and update docs; removals or incompatible schema changes require an ADR and a release note.")
	}
}

// Every advertised tool has exactly one governance record, and every
// governance record points at an advertised tool. Optional tools declare an
// activation condition; required tools must not.
func TestCatalogGovernanceBijection(t *testing.T) {
	advertised := map[string]bool{}
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if name == "" {
			t.Fatalf("tool definition without name: %v", def)
		}
		if advertised[name] {
			t.Fatalf("duplicate tool %q in catalog", name)
		}
		advertised[name] = true
		gov, ok := catalogGovernance[name]
		if !ok {
			t.Errorf("tool %q has no governance record (risk class)", name)
			continue
		}
		switch gov.Risk {
		case RiskRead, RiskGate, RiskExternal:
		default:
			t.Errorf("tool %q has invalid risk class %q", name, gov.Risk)
		}
		if gov.Optional && gov.Activation == "" {
			t.Errorf("optional tool %q must declare an activation condition", name)
		}
		if !gov.Optional && gov.Activation != "" {
			t.Errorf("required tool %q must not declare an activation condition", name)
		}
		schema, _ := def["inputSchema"].(map[string]any)
		if schema == nil || schema["type"] != "object" {
			t.Errorf("tool %q inputSchema must be a JSON Schema object", name)
		}
	}
	for name := range catalogGovernance {
		if !advertised[name] {
			t.Errorf("governance record %q references a tool that is not advertised", name)
		}
	}
}

// Runtime enforcement must reject calls that violate a tool's declared
// required arguments (negative/schema path). Optional conductor tools are
// exercised separately: their activation error takes precedence.
func TestCatalogRequiredArgumentsEnforced(t *testing.T) {
	ts := newTestServer(t, "")
	id := 100
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if strings.HasPrefix(name, "conductor_") {
			continue
		}
		schema, _ := def["inputSchema"].(map[string]any)
		required, _ := schema["required"].([]string)
		if len(required) == 0 {
			continue
		}
		id++
		req := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":{}}}`, id, name)
		_, out := post(t, ts, req)
		isErr, _ := out.Result["isError"].(bool)
		if !isErr {
			t.Errorf("tool %q accepted a call missing required arguments %v", name, required)
			continue
		}
		content, _ := out.Result["content"].([]any)
		text := ""
		if len(content) > 0 {
			text, _ = content[0].(map[string]any)["text"].(string)
		}
		if !strings.Contains(text, "required argument") {
			t.Errorf("tool %q missing-argument error should mention 'required argument', got: %s", name, text)
		}
	}
}

// R3: the public MCP documentation lists exactly the advertised catalog —
// no undocumented tools, no documented ghosts.
func TestCatalogDocsConformance(t *testing.T) {
	raw, err := os.ReadFile("../../../docs-site/docs/mcp.md")
	if err != nil {
		t.Fatalf("reading mcp.md: %v", err)
	}
	re := regexp.MustCompile(`\b(?:pose|conductor)_[a-z_]+\b`)
	documented := map[string]bool{}
	for _, m := range re.FindAllString(string(raw), -1) {
		documented[m] = true
	}
	var missing, ghosts []string
	for _, def := range toolDefinitions() {
		name, _ := def["name"].(string)
		if !documented[name] {
			missing = append(missing, name)
		}
		delete(documented, name)
	}
	for name := range documented {
		ghosts = append(ghosts, name)
	}
	sort.Strings(missing)
	sort.Strings(ghosts)
	if len(missing) > 0 {
		t.Errorf("tools advertised but undocumented in docs-site/docs/mcp.md: %v", missing)
	}
	if len(ghosts) > 0 {
		t.Errorf("tools documented in docs-site/docs/mcp.md but not advertised: %v", ghosts)
	}
}

// R3: registry metadata (server.json) must agree with the runtime server
// identity and offer the stdio transport the binary implements.
func TestCatalogRegistryConformance(t *testing.T) {
	raw, err := os.ReadFile("../../server.json")
	if err != nil {
		t.Fatalf("reading server.json: %v", err)
	}
	var doc struct {
		Name     string `json:"name"`
		Packages []struct {
			Transport struct {
				Type string `json:"type"`
			} `json:"transport"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing server.json: %v", err)
	}
	if doc.Name != "io.github.oseiaspereira88/pose-mcp" {
		t.Errorf("server.json name = %q, want io.github.oseiaspereira88/pose-mcp", doc.Name)
	}
	if len(doc.Packages) == 0 {
		t.Fatal("server.json declares no packages")
	}
	for i, p := range doc.Packages {
		if p.Transport.Type != "stdio" {
			t.Errorf("package %d transport = %q, want stdio", i, p.Transport.Type)
		}
	}
}
