package cli

// pose adopt — decide a capability from the catalog: turn it on or off, or
// record that this project declined or deferred it (specs
// pose-governed-capabilities-default-on-new-instances, pose-capability-catalog).
// A new instance adopts the catalog's defaults at install; an existing one
// decides each capability here, without knowing which policy keys, dates,
// overlays or schema versions go together.

import (
	"fmt"
	"io"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

const adoptUsage = "Usage: pose adopt --list [--json] | pose adopt <capability> [--off | --decline --reason <text> | --defer --reason <text>] [--date YYYY-MM-DD] [--apply] | pose adopt --request <act-id> [--apply]"

func cmdAdopt(root string, args []string, stdout, stderr io.Writer) int {
	id, mode, reason, apply, list, jsonOutput, requestID := "", "on", "", false, false, false, ""
	date := time.Now().UTC().Format(time.DateOnly)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--list":
			list = true
		case "--json":
			jsonOutput = true
		case "--off":
			mode = "off"
		case "--decline":
			mode = posemodel.AdoptionDeclined
		case "--defer":
			mode = posemodel.AdoptionDeferred
		case "--apply":
			apply = true
		case "--date", "--reason", "--request":
			if i+1 >= len(args) {
				return usageError(stderr, adoptUsage)
			}
			switch args[i] {
			case "--date":
				date = args[i+1]
			case "--reason":
				reason = args[i+1]
			default:
				requestID = args[i+1]
			}
			i++
		default:
			if strings.HasPrefix(args[i], "--") || id != "" {
				return usageError(stderr, adoptUsage)
			}
			id = args[i]
		}
	}
	out := render(stdout, stderr)
	if list {
		states, err := posemodel.CapabilityStates(root)
		if err != nil {
			out.Failure("pose adopt: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, states)
		}
		for _, state := range states {
			line := fmt.Sprintf("%s — %s (since %s", state.State, state.Effect, state.IntroducedIn)
			if state.DefaultForNew {
				line += ", on in new instances"
			}
			line += ")"
			if state.Missing != "" {
				line += "; " + state.Missing
			}
			if state.Decision != nil {
				line += fmt.Sprintf("; %s on %s: %s", state.Decision.Decision, state.Decision.Date, state.Decision.Reason)
			}
			out.Field("adopt."+state.ID, line)
		}
		return 0
	}
	if requestID != "" {
		// An answered configuration-review request decides the capability
		// it asked about (spec pose-update-configuration-review).
		if id != "" || mode != "on" || reason != "" {
			return usageError(stderr, adoptUsage)
		}
		req, err := loadReviewRequest(root, requestID)
		if err != nil {
			out.Failure("pose adopt: " + err.Error())
			return 1
		}
		out.Field("adopt.request", requestID)
		out.Field("adopt.capability", req.Capability)
		out.Field("adopt.answer", req.View.Answer+" by "+req.View.AnsweredBy+" ("+req.View.Assurance+")")
		if reviewAnswerApplied(root, req) {
			out.Field("adopt.result", "already applied; nothing to change")
			return 0
		}
		out.Field("adopt.apply", boolString(apply))
		if !apply {
			return 0
		}
		result, err := applyReviewAnswer(root, req, date)
		if err != nil {
			out.Failure("pose adopt: " + err.Error())
			return 1
		}
		out.Field("adopt.result", result)
		return 0
	}
	if id == "" {
		return usageError(stderr, adoptUsage)
	}
	entry, known := posemodel.LookupCatalogEntry(id)
	if !known {
		out.Failure("pose adopt: unknown capability " + id + "; governed capabilities: " + strings.Join(posemodel.CatalogIDs(), ", "))
		return 2
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		out.Failure("pose adopt: --date must be YYYY-MM-DD, got " + date)
		return 2
	}
	docs, err := posemodel.LoadPolicyDocs(root)
	if err != nil {
		out.Failure("pose adopt: " + err.Error())
		return 1
	}
	out.Field("adopt.capability", entry.ID+" — "+entry.Effect)
	out.Field("adopt.action", mode)

	if mode == posemodel.AdoptionDeclined || mode == posemodel.AdoptionDeferred {
		if entry.Adopted(docs) {
			out.Failure("pose adopt: " + entry.ID + " is on; turn it off with `pose adopt " + entry.ID + " --off --apply` before recording that it was " + mode)
			return 1
		}
		if strings.TrimSpace(reason) == "" {
			flag := map[string]string{posemodel.AdoptionDeclined: "--decline", posemodel.AdoptionDeferred: "--defer"}[mode]
			out.Failure("pose adopt: " + flag + " needs --reason, so the decision can be revisited")
			return 2
		}
		out.Field("adopt.change", "record "+entry.ID+" as "+mode+" on "+date+": "+reason)
		out.Field("adopt.apply", boolString(apply))
		if !apply {
			return 0
		}
		if err := posemodel.RecordAdoptionDecisionFor(root, entry.ID, posemodel.AdoptionDecision{Decision: mode, Reason: reason, Date: date, Version: version.ReleaseBase()}); err != nil {
			out.Failure("pose adopt: " + err.Error())
			return 1
		}
		advanceReviewedVersion(root)
		return 0
	}

	var changes []string
	if mode == "off" {
		if blocker := posemodel.CatalogRetireBlocker(docs, entry); blocker != "" {
			out.Failure("pose adopt: " + blocker)
			return 1
		}
		changes = entry.Retire(docs)
	} else {
		if !entry.Adopted(docs) {
			if blocker := posemodel.CatalogAdoptBlocker(root, docs, entry); blocker != "" {
				out.Failure("pose adopt: " + blocker)
				return 1
			}
		}
		changes = entry.Adopt(docs, date)
	}
	if len(changes) == 0 {
		if mode == "off" {
			out.Field("adopt.result", entry.ID+" is not adopted; nothing to change")
		} else {
			out.Field("adopt.result", entry.ID+" is already adopted; its date is kept, so nothing is re-judged")
		}
		return 0
	}
	for _, change := range changes {
		out.Field("adopt.change", change)
	}
	// The readers decide, before anything is written: a policy they would
	// refuse is never left behind.
	if _, _, _, _, err := docs.Rendered(posemodel.Store{Root: root}); err != nil {
		out.Failure("pose adopt: the resulting policy would be refused: " + strings.TrimPrefix(err.Error(), "pose: "))
		return 1
	}
	out.Field("adopt.apply", boolString(apply))
	if !apply {
		return 0
	}
	if err := docs.Write(root, posemodel.Store{Root: root}); err != nil {
		out.Failure("pose adopt: " + err.Error())
		return 1
	}
	if mode == "on" {
		_ = posemodel.ClearAdoptionDecision(root, entry.ID)
	}
	advanceReviewedVersion(root)
	return 0
}

// adoptGovernedCapabilitiesAtInstall adopts every capability the catalog turns
// on for a new instance, dated today, in a review policy this install has just
// created. It is never called for a policy that existed before the install,
// and never by `pose update`: an instance already under way decides when.
func adoptGovernedCapabilitiesAtInstall(target string, now time.Time, log func(english, portuguese string, a ...any)) {
	docs, err := posemodel.LoadPolicyDocs(target)
	if err != nil {
		return
	}
	date := now.UTC().Format(time.DateOnly)
	// A new instance has reviewed this engine's catalog by adopting its
	// defaults; only what a later engine introduces is new to it.
	_ = posemodel.SetReviewedVersion(target, version.ReleaseBase())
	adopted := []string{}
	for _, entry := range posemodel.CapabilityCatalog() {
		if !entry.DefaultForNew || entry.Adopted(docs) {
			continue
		}
		if posemodel.CatalogAdoptBlocker(target, docs, entry) != "" {
			continue
		}
		if len(entry.Adopt(docs, date)) > 0 {
			adopted = append(adopted, entry.ID)
		}
	}
	if len(adopted) == 0 {
		return
	}
	if err := docs.Write(target, posemodel.Store{Root: target}); err != nil {
		log("warning: governed capabilities not adopted, the policy would be refused: %v", "aviso: capacidades governadas não adotadas, a política seria recusada: %v", err)
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
