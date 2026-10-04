package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/harne8/pose-mcp/internal/pose"
)

// cmdStateAttention renders what needs attention (spec pose-state-attention):
// coverage limits first, then what waits on a person or role, then what
// restricts each phase, then residual debt. Read-only; the same domain
// function backs the MCP `pose_obligations` tool.
func cmdStateAttention(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose state --attention [--scope <ref>] [--actor <principal|role>] [--phase <phase>] [--kind <category>] [--json]"
	var q pose.ObligationQuery
	jsonOutput := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--attention":
		case "--json":
			jsonOutput = true
		case "--scope", "--actor", "--phase", "--kind", "--state":
			if i+1 >= len(args) {
				return usageError(stderr, usage)
			}
			i++
			switch args[i-1] {
			case "--scope":
				q.Scope = args[i]
			case "--actor":
				q.Actor = args[i]
			case "--phase":
				q.Phase = args[i]
			case "--kind":
				q.Category = args[i]
			case "--state":
				q.Satisfaction = args[i]
			}
		default:
			return usageError(stderr, usage)
		}
	}
	actor := q.Actor
	// The actor filters what is shown first; every obligation stays in the
	// report so phase blocking is never hidden by the actor view.
	q.Actor = ""
	report, err := pose.Store{Root: root}.ProjectObligations(q)
	if err != nil {
		render(stdout, stderr).Failure("pose state --attention: " + err.Error())
		return 1
	}
	attention := pose.BuildAttention(report, actor)
	if jsonOutput {
		return writeJSON(stdout, struct {
			pose.ObligationReport
			Attention pose.Attention `json:"attention"`
		}{report, attention})
	}
	byID := map[string]pose.Obligation{}
	for _, o := range report.Obligations {
		byID[o.ID] = o
	}
	out := render(stdout, stderr)
	if attention.Incomplete {
		limits := []string{}
		for _, c := range attention.Coverage {
			limits = append(limits, c.Producer+"="+c.State)
		}
		if !report.Snapshot.Coherent {
			limits = append(limits, "snapshot=incoherent")
		}
		out.Field("attention.coverage", "INCOMPLETE — "+strings.Join(limits, ", ")+"; an empty group below does not mean nothing is owed there")
	} else {
		out.Field("attention.coverage", "complete")
	}
	for _, limitation := range report.Snapshot.Limitations {
		out.Field("attention.limitation", limitation)
	}
	who := actor
	if who == "" {
		who = "a person or role"
	}
	out.Field("attention.for_actor", fmt.Sprintf("%d item(s) need %s", len(attention.ForActor), who))
	for _, id := range attention.ForActor {
		out.Field("  "+id, attentionLine(byID[id]))
	}
	for _, phase := range []string{pose.PhaseStart, pose.PhaseExecution, pose.PhaseReview, pose.PhaseCloseout, pose.PhaseRelease} {
		ids := attention.Blocking[phase]
		out.Field("attention.blocks."+phase, fmt.Sprintf("%d obligation(s)", len(ids)))
		for _, id := range ids {
			out.Field("  "+id, attentionLine(byID[id]))
		}
	}
	out.Field("attention.residual", fmt.Sprintf("%d advisory follow-up(s); they restrict no phase", len(attention.Residual)))
	shown := attention.Residual
	if len(shown) > 10 {
		shown = shown[:10]
	}
	for _, id := range shown {
		out.Field("  "+id, attentionLine(byID[id]))
	}
	if len(attention.Residual) > len(shown) {
		out.Field("attention.residual.more", fmt.Sprintf("%d more; `pose followups --open` lists them", len(attention.Residual)-len(shown)))
	}
	out.Field("attention.snapshot", fmt.Sprintf("%s revision=%s dirty=%t policy=%s", report.Snapshot.Project, shortRev(report.Snapshot.SourceRevision), report.Snapshot.WorktreeDirty, shortRev(strings.TrimPrefix(report.Snapshot.PolicyDigest, "sha256:"))))
	return 0
}

func attentionLine(o pose.Obligation) string {
	who := "unassigned"
	switch {
	case o.Recipient.Principal != "":
		who = o.Recipient.Principal
	case o.Recipient.Role != "":
		who = "role:" + o.Recipient.Role
	}
	phases := []string{}
	for _, effect := range o.Effects {
		phases = append(phases, effect.Phase+"/"+effect.Mode)
	}
	target := o.Source.Ref.String()
	if len(o.Targets) > 0 {
		target = o.Targets[0].String()
	}
	return fmt.Sprintf("[%s %s] %s — needs %s; restricts %s; target %s; satisfied when %s", o.Category, o.ReasonCode, o.Message, who, strings.Join(phases, ","), target, o.Condition)
}

func shortRev(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
