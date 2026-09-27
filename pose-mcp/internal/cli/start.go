package cli

// pose start — atomic start (spec pose-abm-atomic-start). Preview is the
// default and writes nothing; --apply needs the atomic_start_version
// capability and the digest of the reviewed preview.

import (
	"fmt"
	"io"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

const startUsage = "Usage: pose start spec:<slug> [--json] | --apply --digest <sha256> | --status [--json] | --cancel"

func cmdStart(root string, args []string, stdout, stderr io.Writer) int {
	r := render(stdout, stderr)
	usage := func(reason string) int {
		r.Usage(startUsage)
		if reason != "" {
			r.Failure(reason)
		}
		return 2
	}
	if len(args) == 0 || !strings.HasPrefix(args[0], "spec:") {
		return usage("a spec:<slug> scope is required")
	}
	slug := strings.TrimPrefix(args[0], "spec:")
	if posemodel.ValidateSlug(slug) != nil {
		return usage("invalid spec slug")
	}
	apply, status, cancel, jsonOutput, digest := false, false, false, false, ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--apply":
			apply = true
		case "--status":
			status = true
		case "--cancel":
			cancel = true
		case "--json":
			jsonOutput = true
		case "--digest":
			if i+1 >= len(args) {
				return usage("--digest needs a value")
			}
			i++
			digest = args[i]
		default:
			return usage("unknown option: " + args[i])
		}
	}
	store := cliGovernedStore(root)
	switch {
	case cancel:
		if err := store.CancelStart(slug); err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		r.Field("start.cancelled", "spec:"+slug)
		return 0
	case status:
		st, err := store.GetStartStatus(slug)
		if err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, st)
		}
		r.Field("start.phase", st.Phase)
		r.Field("start.status", st.Status)
		for _, node := range st.Nodes {
			r.Field("start.node."+node.ID, node.Origin)
		}
		for _, id := range st.Removed {
			r.Field("start.removed", id)
		}
		r.Field("start.obligations", strings.Join(st.Obligations, ","))
		for _, item := range st.Reconciliation {
			r.Field("start.reconcile", item)
		}
		return 0
	case apply:
		if digest == "" {
			return usage("--apply needs --digest from a reviewed preview")
		}
		plan, err := store.PreviewStart(slug)
		if err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		// Repeating a completed start is a no-op. An interrupted one still
		// previews the same plan, because the spec has not moved yet.
		if plan.Digest != digest {
			if record, ok, err := store.GetStartRecord(slug); err == nil && ok && record.PlanDigest == digest && record.Phase == "started" {
				r.Field("start.phase", record.Phase)
				r.Field("start.baseline", record.Baseline.Digest)
				return 0
			}
		}
		record, err := store.ApplyStart(plan, digest, nil)
		if err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, record)
		}
		r.Field("start.phase", record.Phase)
		r.Field("start.baseline", record.Baseline.Digest)
		return 0
	default:
		plan, err := store.PreviewStart(slug)
		if err != nil {
			r.Failure("pose start: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, plan)
		}
		r.Field("start.ready", fmt.Sprint(plan.Ready))
		if plan.Reason != "" {
			r.Field("start.reason", plan.Reason)
		}
		for _, waiting := range plan.WaitingOn {
			r.Field("start.waiting_on", waiting.Ref+": "+waiting.Reason)
		}
		r.Field("start.obligations", strings.Join(plan.Obligations, ","))
		r.Field("start.baseline", plan.Baseline.Digest)
		r.Field("start.digest", plan.Digest)
		return 0
	}
}
