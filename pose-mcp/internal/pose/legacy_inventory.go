package pose

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// LegacyInventory is the read-only dry-run of a future major's cleanup (spec
// pose-v7-legacy-cleanup-plan). It counts the representations a 7.0 could
// retire or convert and says, per class, what the conversion would do and
// what must hold first. It writes nothing and decides nothing: 6.x applies
// no breaking change, and old records keep their meaning, signature and
// auditability — sealed bundles and attestations are never rewritten.
type LegacyInventory struct {
	SchemaVersion int                 `json:"schema_version"`
	Revision      string              `json:"revision,omitempty"`
	Classes       []LegacyClass       `json:"classes"`
	Risks         []string            `json:"risks"`
	Writes        []string            `json:"writes"`
	Limitations   []string            `json:"limitations"`
	Counts        map[string]int      `json:"counts"`
	Details       map[string][]string `json:"details,omitempty"`
}

// LegacyClass is one family of legacy representation.
type LegacyClass struct {
	ID      string `json:"id"`
	Count   int    `json:"count"`
	Action  string `json:"action"`
	Removal string `json:"removal_criterion"`
}

// Dry-run actions.
const (
	LegacyKeep    = "keep-read-only"
	LegacyConvert = "convert"
	LegacyRefuse  = "refuse-until-resolved"
	LegacyNone    = "none"
)

// InventoryLegacy reads the repository.
func (s Store) InventoryLegacy() (LegacyInventory, error) {
	inv := LegacyInventory{SchemaVersion: 1, Revision: gitHeadAtRoot(s.Root), Counts: map[string]int{}, Details: map[string][]string{}, Writes: []string{},
		Limitations: []string{
			"a dry-run: nothing is written, and no apply exists in 6.x",
			"counts describe this repository only; consumers' corpora must run their own dry-run before a removal is decided",
		}}
	add := func(id string, count int, action, removal string) {
		inv.Classes = append(inv.Classes, LegacyClass{ID: id, Count: count, Action: action, Removal: removal})
		inv.Counts[id] = count
	}

	// Spec layouts and statuses.
	specs, err := s.ListSpecs("", "")
	if err != nil {
		return inv, err
	}
	layouts := map[string]int{}
	blocked := 0
	amendmentsFlat := 0
	for _, sp := range specs {
		rel, _ := filepath.Rel(filepath.Join(s.Root, ".pose", "specs"), sp.Path)
		rel = filepath.ToSlash(rel)
		switch {
		case !strings.Contains(rel, "/"):
			layouts["flat"]++
			if _, err := os.Stat(AmendmentsPath(sp.Path)); err == nil {
				amendmentsFlat++
			}
		case datePrefixRE.MatchString(rel):
			layouts["folder"]++
		default:
			layouts["legacy-folder"]++
			inv.Details["legacy-folder"] = append(inv.Details["legacy-folder"], rel)
		}
		if sp.Status == "blocked" {
			blocked++
			inv.Details["status-blocked"] = append(inv.Details["status-blocked"], sp.Slug)
		}
	}
	add("spec-layout-flat", layouts["flat"], LegacyNone, "current layout; nothing to retire")
	add("spec-layout-folder", layouts["folder"], LegacyKeep, "folder specs stay readable; new specs are flat by default")
	add("spec-layout-legacy-folder", layouts["legacy-folder"], LegacyConvert, "undated folders convert to a dated flat file when every reference resolves to the new path; zero remaining in consumers' dry-runs")
	add("status-blocked", blocked, LegacyConvert, "blocked converts to in-progress plus an operational wait (action request or external wait); removable when no consumer dry-run reports a blocked spec and the adoption metrics v2 have shipped for one minor")
	add("flat-spec-amendment-journals", amendmentsFlat, LegacyNone, "current representation")

	// Review policy: legacy adoption keys and unknown keys.
	raw, err := os.ReadFile(filepath.Join(s.Root, ".pose", "policy", "review.json"))
	if err == nil {
		legacyKeys := 0
		for _, contract := range reviewContracts {
			source, err := ContractAdoptionSourceOf(raw, contract.ID)
			if err != nil {
				inv.Risks = append(inv.Risks, "review policy: "+err.Error())
				continue
			}
			if source.Source == ContractAdoptionLegacy || source.Source == ContractAdoptionLegacyEmpty || source.LegacyShadowed {
				legacyKeys++
				detail := contract.ID + " via " + source.LegacyField
				if source.LegacyShadowed {
					detail += " (shadowed by contract_adoptions)"
				}
				inv.Details["policy-legacy-adoption-keys"] = append(inv.Details["policy-legacy-adoption-keys"], detail)
			}
		}
		add("policy-legacy-adoption-keys", legacyKeys, LegacyConvert, "each legacy key moves into contract_adoptions with the same date, so the cutoff keeps its meaning; removable after one minor in which doctor reports review.adoption-source for every consumer")
		unknown := unknownPolicyKeys(raw)
		inv.Details["policy-unknown-keys"] = unknown
		add("policy-unknown-keys", len(unknown), LegacyRefuse, "an unknown key must not silence an expected enforcement: a major refuses it until it is renamed or removed")
		for _, key := range unknown {
			inv.Risks = append(inv.Risks, "review policy key "+key+" is not read by this engine; if it was meant to enable a gate, that gate is off")
		}
	}

	// Sealed bundles and attestations: never rewritten.
	bundles, err := s.ListReviewBundles("")
	if err != nil {
		return inv, err
	}
	sealed, withoutContracts := 0, 0
	for _, b := range bundles {
		if b.State != "sealed" {
			continue
		}
		sealed++
		if len(b.Payload.GoverningContracts) == 0 {
			withoutContracts++
		}
	}
	add("sealed-bundles", sealed, LegacyKeep, "never removed: a sealed bundle is the subject of a signed or recorded judgment")
	add("sealed-bundles-without-governing-contracts", withoutContracts, LegacyKeep, "read under the contracts their seal date implies; never re-sealed")
	attestations, err := s.ListReviewAttestations("")
	if err != nil {
		return inv, err
	}
	supplements, _ := s.ListReviewAttributionSupplements("")
	supplemented := map[string]bool{}
	for _, sup := range supplements {
		supplemented[sup.AttestationID] = true
	}
	signed, declared, undifferentiated := 0, 0, 0
	for _, a := range attestations {
		if a.Envelope != nil {
			signed++
		} else {
			declared++
		}
		if a.Attribution == nil && !supplemented[a.AttestationID] {
			undifferentiated++
		}
	}
	add("attestations-verified", signed, LegacyKeep, "never removed")
	add("attestations-declared", declared, LegacyKeep, "declared identity stays disclosed as declared; a major may require verified identity for new attestations only")
	add("attestations-legacy-undifferentiated", undifferentiated, LegacyKeep, "rendered as legacy-undifferentiated; supplements may clarify roles without altering the signed record")

	// Interrupted transfers.
	interrupted := 0
	if entries, err := os.ReadDir(filepath.Join(s.Root, ".pose", "transfers")); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || !specTransferIDRE.MatchString(entry.Name()) {
				continue
			}
			// Status is per project: this repository is done with a transfer
			// once its own side reached its terminal phase — activated as the
			// destination, source-retired as the source.
			plan, err := readSpecTransferPlan(transferPlanFile(s.Root, entry.Name()))
			if err != nil {
				interrupted++
				inv.Details["interrupted-transfers"] = append(inv.Details["interrupted-transfers"], entry.Name()+" (plan unreadable)")
				continue
			}
			project := s.CurrentObligationSnapshot().Project
			terminal := "activated"
			if plan.Source.Project == project && plan.Destination.Project != project {
				terminal = "source-retired"
			}
			status, err := ReadSpecTransferStatus(s, entry.Name(), project)
			if err != nil || status.Phase != terminal {
				interrupted++
				inv.Details["interrupted-transfers"] = append(inv.Details["interrupted-transfers"], entry.Name())
			}
		}
	}
	add("interrupted-transfers", interrupted, LegacyRefuse, "a major's migration refuses to run while a transfer is incomplete; resume or roll it back first")
	if interrupted > 0 {
		inv.Risks = append(inv.Risks, "an incomplete transfer would be read differently after a migration; resolve it first")
	}

	for key := range inv.Details {
		sort.Strings(inv.Details[key])
		if len(inv.Details[key]) == 0 {
			delete(inv.Details, key)
		}
	}
	if inv.Risks == nil {
		inv.Risks = []string{}
	}
	return inv, nil
}

// unknownPolicyKeys lists top-level review policy keys the typed policy does
// not read.
func unknownPolicyKeys(raw []byte) []string {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	known := map[string]bool{}
	t := reflect.TypeOf(ReviewPolicy{})
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			known[name] = true
		}
	}
	var unknown []string
	for key := range doc {
		if !known[key] && !strings.HasPrefix(key, "_") && key != "$schema" {
			unknown = append(unknown, key)
		}
	}
	sort.Strings(unknown)
	return unknown
}
