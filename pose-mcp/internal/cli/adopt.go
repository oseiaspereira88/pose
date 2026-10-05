package cli

// pose adopt — turn a governed capability on or off in this instance's review
// policy (spec pose-governed-capabilities-default-on-new-instances). A new
// instance adopts every governed capability at install; an existing one
// decides when, and this is how it says so without knowing which version key,
// adoption date and overlay entry go together.

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const adoptUsage = "Usage: pose adopt <capability> [--off] [--date YYYY-MM-DD] [--apply]"

func cmdAdopt(root string, args []string, stdout, stderr io.Writer) int {
	id, off, apply, date := "", false, false, time.Now().UTC().Format(time.DateOnly)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--off":
			off = true
		case "--apply":
			apply = true
		case "--date":
			if i+1 >= len(args) {
				return usageError(stderr, adoptUsage)
			}
			i++
			date = args[i]
		default:
			if strings.HasPrefix(args[i], "--") || id != "" {
				return usageError(stderr, adoptUsage)
			}
			id = args[i]
		}
	}
	if id == "" {
		return usageError(stderr, adoptUsage)
	}
	out := render(stdout, stderr)
	capability, known := posemodel.LookupGovernedCapability(id)
	if !known {
		out.Failure("pose adopt: unknown capability " + id + "; governed capabilities: " + strings.Join(posemodel.GovernedCapabilityIDs(), ", "))
		return 2
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		out.Failure("pose adopt: --date must be YYYY-MM-DD, got " + date)
		return 2
	}
	path := filepath.Join(root, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		out.Failure("pose adopt: " + err.Error())
		return 1
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		out.Failure("pose adopt: invalid .pose/policy/review.json: " + err.Error())
		return 1
	}
	if !off && capability.Overlay != "" {
		profile := strings.SplitN(capability.Overlay, "@", 2)[0]
		if _, err := os.Stat(filepath.Join(root, ".pose", "review-profiles", profile+".json")); err != nil {
			out.Failure("pose adopt: " + id + " needs the review profile " + capability.Overlay + ", which this instance does not have; run `pose update` to install the shipped profiles")
			return 1
		}
	}
	var changes []string
	if off {
		changes = posemodel.RetireGovernedCapability(doc, capability)
	} else {
		changes = posemodel.AdoptGovernedCapability(doc, capability, date)
	}
	action := "on"
	if off {
		action = "off"
	}
	out.Field("adopt.capability", capability.ID+" — "+capability.Summary)
	out.Field("adopt.action", action)
	if len(changes) == 0 {
		if off {
			out.Field("adopt.result", capability.ID+" is not adopted; nothing to change")
		} else {
			out.Field("adopt.result", capability.ID+" is already adopted; its date is kept, so nothing is re-judged")
		}
		return 0
	}
	for _, change := range changes {
		out.Field("adopt.change", change)
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		out.Failure("pose adopt: " + err.Error())
		return 1
	}
	updated = append(updated, '\n')
	// The reader decides, before anything is written: a policy it would refuse
	// is never left behind.
	if _, err := (posemodel.Store{Root: root}).ParseReviewPolicyDocument(updated); err != nil {
		out.Failure("pose adopt: the resulting review policy would be refused: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	out.Field("adopt.apply", boolString(apply))
	if !apply {
		return 0
	}
	if err := writeAtomic(path, updated, 0o644); err != nil {
		out.Failure("pose adopt: " + err.Error())
		return 1
	}
	return 0
}

// adoptGovernedCapabilitiesAtInstall adopts every governed capability, dated
// today, in a review policy this install has just created. It is never called
// for a policy that existed before the install, and never by `pose update`:
// an instance already under way decides when with `pose adopt`.
func adoptGovernedCapabilitiesAtInstall(target string, now time.Time, log func(english, portuguese string, a ...any)) {
	path := filepath.Join(target, ".pose", "policy", "review.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return
	}
	date := now.UTC().Format(time.DateOnly)
	adopted := []string{}
	for _, capability := range posemodel.GovernedCapabilities() {
		if len(posemodel.AdoptGovernedCapability(doc, capability, date)) > 0 {
			adopted = append(adopted, capability.ID)
		}
	}
	if len(adopted) == 0 {
		return
	}
	updated, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return
	}
	updated = append(updated, '\n')
	if _, err := (posemodel.Store{Root: target}).ParseReviewPolicyDocument(updated); err != nil {
		log("warning: governed capabilities not adopted, the policy would be refused: %v", "aviso: capacidades governadas não adotadas, a política seria recusada: %v", err)
		return
	}
	if writeAtomic(path, updated, 0o644) != nil {
		return
	}
	for _, id := range adopted {
		log("capability (adoption): %s from %s — `pose adopt %s --off --apply` turns it off", "capacidade (adoção): %s desde %s — `pose adopt %s --off --apply` desliga", id, date, id)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
