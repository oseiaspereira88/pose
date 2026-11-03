package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type cleanupCapture struct {
	Name                  string `json:"name"`
	File                  string `json:"file"`
	TableRevision         string `json:"table_revision"`
	CaptureRevision       string `json:"capture_revision"`
	EngineRevision        string `json:"engine_revision"`
	RawSHA256             string `json:"raw_sha256"`
	ExportSHA256          string `json:"export_sha256"`
	PrivateDetailsRemoved bool   `json:"private_details_removed"`
}

type cleanupCorpus struct {
	Capture   cleanupCapture
	Inventory LegacyInventory
}

func retainedCleanupCorpora(t *testing.T) (string, []cleanupCorpus) {
	t.Helper()
	dir := filepath.Join("testdata", "cleanup-corpora")
	raw, err := os.ReadFile(filepath.Join(dir, "provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var provenance struct {
		SchemaVersion int              `json:"schema_version"`
		Captures      []cleanupCapture `json:"captures"`
	}
	if err := json.Unmarshal(raw, &provenance); err != nil {
		t.Fatal(err)
	}
	if provenance.SchemaVersion != 1 || len(provenance.Captures) != 2 {
		t.Fatal("expected the two retained corpus captures")
	}
	var corpora []cleanupCorpus
	for _, capture := range provenance.Captures {
		if filepath.Base(capture.File) != capture.File {
			t.Fatal("capture file must stay inside its fixture directory")
		}
		raw, err := os.ReadFile(filepath.Join(dir, capture.File))
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(raw)
		if hex.EncodeToString(digest[:]) != capture.ExportSHA256 {
			t.Fatalf("%s export does not match its recorded digest", capture.Name)
		}
		decoded, err := hex.DecodeString(capture.RawSHA256)
		if err != nil || len(decoded) != sha256.Size {
			t.Fatalf("%s has no raw integrity reference", capture.Name)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if capture.PrivateDetailsRemoved {
			if _, present := fields["details"]; present {
				t.Fatal("a public summary must not publish private corpus details")
			}
		} else if capture.RawSHA256 != capture.ExportSHA256 {
			t.Fatal("the public raw capture must be retained verbatim")
		}
		var inventory LegacyInventory
		if err := json.Unmarshal(raw, &inventory); err != nil {
			t.Fatal(err)
		}
		corpora = append(corpora, cleanupCorpus{capture, inventory})
	}
	report, err := os.ReadFile(filepath.Join("..", "..", "..", ".pose", "reports", "2026-11-02-pose-v7-legacy-cleanup-both-corpora.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(report), corpora
}

// The historical table must answer to the captures, not just contain words
// that sound like a completed measurement. This validates evidence, not runtime.
func checkCleanupCorporaReport(report string, corpora []cleanupCorpus) error {
	if len(corpora) != 2 || corpora[0].Capture.Name != "pose-dist" || corpora[1].Capture.Name != "harne8" {
		return fmt.Errorf("expected pose-dist and Harne8 in table order")
	}
	for _, corpus := range corpora {
		inv, capture := corpus.Inventory, corpus.Capture
		if inv.SchemaVersion != 1 || inv.Revision != capture.CaptureRevision || capture.EngineRevision == "" {
			return fmt.Errorf("%s capture provenance is inconsistent", capture.Name)
		}
		if !inv.Complete || inv.Coverage.Losses == nil || !inv.Coverage.Complete() || len(inv.Coverage.Exclusions) != 0 || inv.Writes == nil || len(inv.Writes) != 0 {
			return fmt.Errorf("%s is not a complete read-only capture", capture.Name)
		}
		classes := map[string]bool{}
		for _, class := range inv.Classes {
			count, found := inv.Counts[class.ID]
			if class.ID == "" || classes[class.ID] || !found || count != class.Count || count < 0 {
				return fmt.Errorf("%s has inconsistent class %q", capture.Name, class.ID)
			}
			classes[class.ID] = true
		}
		if len(classes) == 0 || len(classes) != len(inv.Counts) {
			return fmt.Errorf("%s has missing classes", capture.Name)
		}
	}
	rows, headers := map[string][2]int{}, 0
	for _, line := range strings.Split(report, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		if len(cells) != 3 {
			return fmt.Errorf("malformed corpus table row: %s", line)
		}
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		if strings.Trim(cells[0], "-:") == "" {
			continue
		}
		if cells[0] == "Class" {
			headers++
			for i, corpus := range corpora {
				if !strings.Contains(strings.ToLower(cells[i+1]), corpus.Capture.Name) || !strings.Contains(cells[i+1], "`"+corpus.Capture.TableRevision+"`") {
					return fmt.Errorf("table header does not name the original measurement")
				}
			}
			continue
		}
		id := strings.Trim(cells[0], "`")
		if _, duplicate := rows[id]; duplicate {
			return fmt.Errorf("duplicate table class %s", id)
		}
		var counts [2]int
		for i, corpus := range corpora {
			count, err := strconv.Atoi(cells[i+1])
			want, found := corpus.Inventory.Counts[id]
			if err != nil || !found || count != want {
				return fmt.Errorf("%s table count for %s differs from the capture", corpus.Capture.Name, id)
			}
			counts[i] = count
		}
		rows[id] = counts
	}
	if headers != 1 || len(rows) != len(corpora[0].Inventory.Counts) || len(rows) != len(corpora[1].Inventory.Counts) {
		return fmt.Errorf("table omits captured classes or its header")
	}
	return nil
}

func TestTheCleanupPlanHoldsBothCorpora(t *testing.T) {
	report, corpora := retainedCleanupCorpora(t)
	if err := checkCleanupCorporaReport(report, corpora); err != nil {
		t.Fatal(err)
	}
}

func TestCleanupCaptureProvenanceNamesTheReproduction(t *testing.T) {
	_, corpora := retainedCleanupCorpora(t)
	public, private := corpora[0].Capture, corpora[1].Capture
	if public.TableRevision != "fcd11da9" || !strings.HasPrefix(public.CaptureRevision, "f931afe0") || public.PrivateDetailsRemoved {
		t.Fatal("the later public reproduction was confused with the original measurement")
	}
	if private.TableRevision != "ba83cbfb" || !strings.HasPrefix(private.CaptureRevision, private.TableRevision) || !private.PrivateDetailsRemoved {
		t.Fatal("the private capture must retain its actual revision and summary boundary")
	}
}

func TestCleanupCorporaRejectInvalidEvidence(t *testing.T) {
	for _, name := range []string{"false-count", "missing-row", "duplicate-row", "incomplete", "coverage-loss", "coverage-exclusion", "undeclared-coverage", "writes", "undeclared-writes", "missing-class", "duplicate-class", "inconsistent-count", "capture-revision", "table-revision"} {
		t.Run(name, func(t *testing.T) {
			report, corpora := retainedCleanupCorpora(t)
			switch name {
			case "false-count":
				report = strings.Replace(report, "| 1549 | 193 |", "| 9999 | 193 |", 1)
			case "missing-row":
				report = strings.Replace(report, "| `attestations-declared` | 1549 | 193 |\n", "", 1)
			case "duplicate-row":
				report += "\n| `attestations-declared` | 1549 | 193 |\n"
			case "incomplete":
				corpora[0].Inventory.Complete = false
			case "coverage-loss":
				corpora[0].Inventory.Coverage.Losses = append(corpora[0].Inventory.Coverage.Losses, SourceLoss{Producer: "test", Artifact: "missing.json", Kind: LossReadError})
			case "coverage-exclusion":
				corpora[0].Inventory.Coverage.Exclusions = []SourceExclusion{{Producer: "test", Pattern: "unread/*", Reason: "partial corpus", Count: 1}}
			case "undeclared-coverage":
				corpora[0].Inventory.Coverage.Losses = nil
			case "writes":
				corpora[0].Inventory.Writes = []string{"modified"}
			case "undeclared-writes":
				corpora[0].Inventory.Writes = nil
			case "missing-class":
				corpora[0].Inventory.Classes = corpora[0].Inventory.Classes[1:]
			case "duplicate-class":
				corpora[0].Inventory.Classes = append(corpora[0].Inventory.Classes, corpora[0].Inventory.Classes[0])
			case "inconsistent-count":
				corpora[0].Inventory.Classes[0].Count++
			case "capture-revision":
				corpora[0].Inventory.Revision = "not-the-capture"
			case "table-revision":
				report = strings.Replace(report, "`fcd11da9`", "`f931afe0`", 1)
			}
			if err := checkCleanupCorporaReport(report, corpora); err == nil {
				t.Fatal("invalid historical evidence was accepted")
			}
		})
	}
}
