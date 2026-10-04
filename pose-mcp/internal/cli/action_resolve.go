package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdActionResolve records an answer, cancellation, waiver or invalidation
// (spec pose-action-request-resolution).
func cmdActionResolve(root, verb string, args []string, stdout, stderr io.Writer) int {
	usage := "Usage: pose action " + verb + " <act-id> --actor <principal> --request-digest <digest> --expected-revision <n> --idempotency-key <key> " +
		"[--answer <option|text>] [--evidence <ref>]... [--reason <text>] [--role <role>] [--execution <id>] [--channel <name>] [--prepared-by <p>] [--applied-by <p>] " +
		"[--confirmation-mode adopted-conclusions|authorized-operation] [--claim <project-relative json>] [--apply] [--json]"
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return usageError(stderr, usage)
	}
	res := posemodel.ActionResolution{RequestID: args[0], ExpectedRevision: -2}
	switch verb {
	case "resolve":
		res.Type = posemodel.ActionEventAnswered
	case "cancel":
		res.Type = posemodel.ActionEventCancelled
	case "waive":
		res.Type = posemodel.ActionEventWaived
	case "invalidate":
		res.Type = posemodel.ActionEventInvalidated
	}
	apply, jsonOutput, claimPath := false, false, ""
	for i := 1; i < len(args); i++ {
		flag := args[i]
		switch flag {
		case "--apply":
			apply = true
			continue
		case "--json":
			jsonOutput = true
			continue
		}
		if i+1 >= len(args) {
			return usageError(stderr, usage)
		}
		i++
		value := args[i]
		switch flag {
		case "--actor":
			res.Actor = value
		case "--request-digest":
			res.RequestDigest = value
		case "--expected-revision":
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 {
				return usageError(stderr, "--expected-revision takes the revision shown by `pose action show`")
			}
			res.ExpectedRevision = n
		case "--idempotency-key":
			res.IdempotencyKey = value
		case "--answer":
			res.Answer = value
		case "--evidence":
			res.Evidence = append(res.Evidence, value)
		case "--reason":
			res.Reason = value
		case "--role":
			res.Role = value
		case "--execution":
			res.Execution = value
		case "--channel":
			res.Channel = value
		case "--prepared-by":
			res.PreparedBy = value
		case "--applied-by":
			res.AppliedBy = value
		case "--confirmation-mode":
			res.ConfirmationMode = value
		case "--claim":
			claimPath = value
		default:
			return usageError(stderr, usage)
		}
	}
	if res.ExpectedRevision == -2 {
		return usageError(stderr, "--expected-revision is required: a write never lands on a revision the writer did not read")
	}
	if claimPath != "" {
		if err := posemodel.ValidateArtifactPath(root, claimPath, false); err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": " + err.Error())
			return 2
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claimPath)))
		if err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": " + err.Error())
			return 2
		}
		var signed struct {
			Claim    posemodel.ActionAuthorityClaim `json:"claim"`
			Envelope posemodel.ActionClaimEnvelope  `json:"envelope"`
		}
		if err := json.Unmarshal(raw, &signed); err != nil {
			render(stdout, stderr).Failure("pose action " + verb + ": invalid claim file: " + err.Error())
			return 2
		}
		res.Claim, res.Envelope = &signed.Claim, &signed.Envelope
	}
	out := render(stdout, stderr)
	store := posemodel.Store{Root: root}
	if !apply {
		view, err := store.LoadActionRequest(res.RequestID)
		if err != nil {
			out.Failure("pose action " + verb + ": " + err.Error())
			return 1
		}
		out.Field("action.preview", "true")
		renderActionView(out, view)
		out.Field("action.apply", "false (rerun with --apply to record the "+verb+")")
		return 0
	}
	view, err := store.ResolveActionRequest(res, time.Now())
	if err != nil {
		out.Failure("pose action " + verb + ": " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, view)
	}
	renderActionView(out, view)
	return 0
}
