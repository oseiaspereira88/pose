package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-governed-capabilities-default-on-new-instances.

func readInstanceReviewPolicy(t *testing.T, repo string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repo, ".pose", "policy", "review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func installedInstance(t *testing.T) (string, string) {
	t.Helper()
	repo := newGitRepo(t)
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp"}, &out, &errB); code != 0 {
		t.Fatalf("install exit=%d err=%s", code, errB.String())
	}
	return repo, out.String()
}

func TestGovernedCapabilitiesAreAdoptedByAFreshInstall(t *testing.T) {
	repo, log := installedInstance(t)
	today := time.Now().UTC().Format(time.DateOnly)
	doc := readInstanceReviewPolicy(t, repo)
	for _, capability := range posemodel.GovernedCapabilities() {
		if !posemodel.GovernedCapabilityAdopted(doc, capability) {
			t.Fatalf("a fresh install did not adopt %s: %v", capability.ID, doc)
		}
		if capability.DateKey != "" && doc[capability.DateKey] != today {
			t.Fatalf("%s is not dated with the install day: %v", capability.DateKey, doc[capability.DateKey])
		}
		if !strings.Contains(log, "capability (adoption): "+capability.ID) {
			t.Fatalf("the install did not log the adoption of %s:\n%s", capability.ID, log)
		}
	}
	dates, _ := doc["overlay_adopted_at"].(map[string]any)
	if dates["structural-materiality@1"] != today {
		t.Fatalf("the structural overlay is not dated with the install day: %v", doc["overlay_adopted_at"])
	}
	if _, err := (posemodel.Store{Root: repo}).GetReviewPolicy(); err != nil {
		t.Fatalf("the installed policy is refused by the reader: %v", err)
	}
}

func TestGovernedCapabilitiesAreNeverAdoptedOverAnExistingPolicy(t *testing.T) {
	repo, _ := installedInstance(t)
	path := filepath.Join(repo, ".pose", "policy", "review.json")
	doc := readInstanceReviewPolicy(t, repo)
	for _, capability := range posemodel.GovernedCapabilities() {
		posemodel.RetireGovernedCapability(doc, capability)
	}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errB bytes.Buffer
	if code := cmdInstall([]string{repo, "--skip-mcp", "--force"}, &out, &errB); code != 0 {
		t.Fatalf("reinstall exit=%d err=%s", code, errB.String())
	}
	after := readInstanceReviewPolicy(t, repo)
	for _, capability := range posemodel.GovernedCapabilities() {
		if posemodel.GovernedCapabilityAdopted(after, capability) {
			t.Fatalf("install over an existing policy adopted %s", capability.ID)
		}
	}

	// An update that seeds a missing policy still adopts nothing.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errB.Reset()
	if code := cmdUpdate(repo, []string{"--no-self"}, &out, &errB); code != 0 {
		t.Fatalf("update exit=%d err=%s", code, errB.String())
	}
	seeded := readInstanceReviewPolicy(t, repo)
	for _, capability := range posemodel.GovernedCapabilities() {
		if posemodel.GovernedCapabilityAdopted(seeded, capability) {
			t.Fatalf("update adopted %s", capability.ID)
		}
	}
}

func TestAdoptTogglesACapabilityThroughTheReader(t *testing.T) {
	repo, _ := installedInstance(t)
	path := filepath.Join(repo, ".pose", "policy", "review.json")
	doc := readInstanceReviewPolicy(t, repo)
	for _, capability := range posemodel.GovernedCapabilities() {
		posemodel.RetireGovernedCapability(doc, capability)
	}
	raw, _ := json.MarshalIndent(doc, "", "  ")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (int, string, string) {
		var out, errB bytes.Buffer
		code := cmdAdopt(repo, args, &out, &errB)
		return code, out.String(), errB.String()
	}

	code, out, _ := run("causality-closeout", "--date", "2026-10-06")
	if code != 0 || !strings.Contains(out, "set causality_closeout_adopted_at=2026-10-06") || !strings.Contains(out, "adopt.apply=false") {
		t.Fatalf("preview: code=%d out=%s", code, out)
	}
	if posemodel.GovernedCapabilityAdopted(readInstanceReviewPolicy(t, repo), mustCapability(t, "causality-closeout")) {
		t.Fatal("a preview wrote the policy")
	}
	if code, out, errOut := run("causality-closeout", "--date", "2026-10-06", "--apply"); code != 0 {
		t.Fatalf("apply: code=%d out=%s err=%s", code, out, errOut)
	}
	policy, err := (posemodel.Store{Root: repo}).GetReviewPolicy()
	if err != nil || policy.CausalityCloseoutVersion != 1 || policy.CausalityCloseoutAdoptedAt != "2026-10-06" || policy.OverlayAdoptedAt["structural-materiality@1"] != "2026-10-06" {
		t.Fatalf("adopt --apply did not write a readable adoption: %+v err=%v", policy, err)
	}
	if code, out, _ := run("causality-closeout", "--apply"); code != 0 || !strings.Contains(out, "already adopted") {
		t.Fatalf("adopting twice: code=%d out=%s", code, out)
	}
	if code, out, errOut := run("causality-closeout", "--off", "--apply"); code != 0 || !strings.Contains(out, "remove causality_closeout_version") {
		t.Fatalf("off: code=%d out=%s err=%s", code, out, errOut)
	}
	if policy, _ := (posemodel.Store{Root: repo}).GetReviewPolicy(); policy.CausalityCloseoutVersion != 0 || len(policy.OverlayAdoptedAt) != 0 {
		t.Fatalf("--off left the capability adopted: %+v", policy)
	}
	if code, _, errOut := run("causality-closeout", "--date", "06/10/2026", "--apply"); code == 0 || !strings.Contains(errOut, "YYYY-MM-DD") {
		t.Fatalf("a malformed date was written: code=%d err=%s", code, errOut)
	}
	if code, _, errOut := run("time-travel"); code == 0 || !strings.Contains(errOut, "agency-readiness, atomic-start, causality-closeout, contract-nodes") {
		t.Fatalf("an unknown capability was not refused with the known ones: code=%d err=%s", code, errOut)
	}
}

func TestDoctorWarnsWhenAgencyReadinessHasNoPrincipal(t *testing.T) {
	repo, _ := installedInstance(t)
	finding, found := findDoctorFinding(runDoctorJSON(t, repo), "actions.roles")
	if !found || finding.Level != "warn" || !strings.Contains(finding.Message, "no principal holds a role") {
		t.Fatalf("doctor did not warn about the empty role map: found=%v %+v", found, finding)
	}
	actions := filepath.Join(repo, ".pose", "policy", "actions.json")
	if err := os.WriteFile(actions, []byte(`{"schema_version":1,"roles":{"maintainer":["human:someone"]},"identity_assurance":"declared"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if finding, _ := findDoctorFinding(runDoctorJSON(t, repo), "actions.roles"); finding.Level != "ok" {
		t.Fatalf("doctor still warns once a principal holds a role: %+v", finding)
	}
}

func TestAdoptIsDocumented(t *testing.T) {
	for _, rel := range []string{"../../../POSE.md", "../../../locales/pt-BR/POSE.md", "../scaffold/dist/POSE.md", "../scaffold/dist/locales/pt-BR/POSE.md"} {
		raw, err := os.ReadFile(filepath.FromSlash(rel))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "pose adopt <capability>") {
			t.Fatalf("%s does not document pose adopt", rel)
		}
	}
	if _, ok := commandHelpCatalog["adopt"]; !ok {
		t.Fatal("pose adopt has no command help")
	}
}

func mustCapability(t *testing.T, id string) posemodel.GovernedCapability {
	t.Helper()
	capability, ok := posemodel.LookupGovernedCapability(id)
	if !ok {
		t.Fatalf("unknown capability %s", id)
	}
	return capability
}
