package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// cmdAction is `pose action` (specs pose-action-requests and
// pose-action-request-resolution): open, read and resolve material requests
// to an actor. Writes are preview by default and need --apply.
func cmdAction(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose action <open|list|show|resolve|cancel|waive|invalidate> ..."
	if len(args) == 0 {
		return usageError(stderr, usage)
	}
	switch args[0] {
	case "open":
		return cmdActionOpen(root, args[1:], stdout, stderr)
	case "list":
		return cmdActionList(root, args[1:], stdout, stderr)
	case "show":
		return cmdActionShow(root, args[1:], stdout, stderr)
	case "resolve", "cancel", "waive", "invalidate":
		return cmdActionResolve(root, args[0], args[1:], stdout, stderr)
	default:
		return usageError(stderr, usage)
	}
}

func cmdActionOpen(root string, args []string, stdout, stderr io.Writer) int {
	const usage = "Usage: pose action open --origin <spec:slug|milestone:r/id|roadmap:slug> --kind <decision|approval|input|external-operation|acceptance> --question <text> " +
		"--requested-by <principal> [--execution <id>] [--recipient <principal> | --recipient-role <role>] [--option <id>=<consequence>]... [--recommend <id>] " +
		"--target <kind:ID|self|xref:...>... --effect <phase>:<block|advisory>... [--context <text>] [--subject <kind:ID|path:file>] [--supersedes <act-id>] [--apply] [--json]"
	var r posemodel.ActionRequest
	apply, jsonOutput := false, false
	for i := 0; i < len(args); i++ {
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
		case "--origin":
			r.Origin = value
		case "--kind":
			r.Kind = value
		case "--question":
			r.Question = value
		case "--context":
			r.Context = value
		case "--requested-by":
			r.RequestedBy.Principal = value
		case "--execution":
			r.RequestedBy.Execution = value
		case "--recipient":
			r.Recipient.Principal = value
		case "--recipient-role":
			r.Recipient.Role = value
		case "--option":
			id, consequence, ok := strings.Cut(value, "=")
			if !ok {
				return usageError(stderr, "--option takes <id>=<consequence>")
			}
			r.Options = append(r.Options, posemodel.ActionOption{ID: strings.TrimSpace(id), Consequence: strings.TrimSpace(consequence)})
		case "--recommend":
			r.Recommend = value
		case "--target":
			target, err := posemodel.ParseActionTarget(value)
			if err != nil {
				return usageError(stderr, "invalid --target "+value)
			}
			r.Targets = append(r.Targets, target)
		case "--effect":
			phase, mode, ok := strings.Cut(value, ":")
			if !ok {
				return usageError(stderr, "--effect takes <phase>:<block|advisory>")
			}
			r.Effects = append(r.Effects, posemodel.ObligationEffect{Phase: phase, Mode: mode})
		case "--subject":
			if path, ok := strings.CutPrefix(value, "path:"); ok {
				r.Subject = &posemodel.ActionSubject{Path: path}
			} else {
				node, err := posemodel.ParseActionTarget(value)
				if err != nil {
					return usageError(stderr, "invalid --subject "+value)
				}
				r.Subject = &posemodel.ActionSubject{Node: node}
			}
		case "--supersedes":
			r.Supersedes = value
		default:
			return usageError(stderr, usage)
		}
	}
	store := posemodel.Store{Root: root}
	out := render(stdout, stderr)
	if !apply {
		prepared, err := store.PrepareActionRequest(r, time.Now())
		if err != nil {
			out.Failure("pose action open: " + err.Error())
			return 1
		}
		if jsonOutput {
			return writeJSON(stdout, prepared)
		}
		out.Field("action.preview", "true")
		out.Field("action.id", prepared.ID)
		out.Field("action.request_digest", prepared.RequestDigest)
		out.Field("action.apply", "false (rerun with --apply to record it)")
		return 0
	}
	view, err := store.OpenActionRequest(r, time.Now())
	if err != nil {
		out.Failure("pose action open: " + err.Error())
		return 1
	}
	identityWarning := actionIdentityFallback(store)
	if jsonOutput {
		if identityWarning != "" {
			// stderr keeps the JSON on stdout parseable.
			render(stderr, stderr).Field("action.limitation", identityWarning)
		}
		return writeJSON(stdout, view)
	}
	renderActionView(out, view)
	if identityWarning != "" {
		out.Field("action.limitation", identityWarning)
	}
	return 0
}

// actionIdentityFallback says when the request was qualified with a project
// id derived from the directory name: the journal keeps that xref, so a clone
// in another directory would record a different project (found by the
// agency-readiness pilot rehearsal).
func actionIdentityFallback(store posemodel.Store) string {
	snapshot := store.CurrentObligationSnapshot()
	for _, limitation := range snapshot.Limitations {
		if strings.Contains(limitation, "fell back to the directory name") {
			return "the request is qualified as " + snapshot.Project + ", derived from the directory name; declare it in .pose/project.json so every checkout records the same project"
		}
	}
	return ""
}

func cmdActionList(root string, args []string, stdout, stderr io.Writer) int {
	state, jsonOutput, present, actor := "", false, false, ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--present":
			present = true
		case "--actor":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose action list [--state <state>] [--present [--actor <id|role>]] [--json]")
			}
			i++
			actor = args[i]
		case "--state":
			if i+1 >= len(args) {
				return usageError(stderr, "Usage: pose action list [--state <state>] [--json]")
			}
			i++
			state = args[i]
		default:
			return usageError(stderr, "Usage: pose action list [--state <state>] [--json]")
		}
	}
	views, err := posemodel.Store{Root: root}.ListActionRequests()
	if err != nil {
		render(stdout, stderr).Failure("pose action list: " + err.Error())
		return 1
	}
	if present {
		// Grouped for one conversation (spec pose-action-request-presentation):
		// requests that restrict start or execution first; release-only ones
		// can wait. Each keeps its own id, digest and answer.
		presentation := posemodel.PresentActionRequests(views, actor)
		if jsonOutput {
			return writeJSON(stdout, presentation)
		}
		out := render(stdout, stderr)
		out.Field("actions.present", fmt.Sprintf("%d request(s) to answer now, %d that can wait for their phase", presentation.InterruptNow, presentation.CanWait))
		for _, group := range presentation.Groups {
			when := "can wait until " + group.EarliestPhase
			if group.Interrupt {
				when = "needed before " + group.EarliestPhase
			}
			out.Field("actions.group", group.Origin+" — "+when)
			for _, item := range group.Requests {
				out.Field("  "+item.ID, fmt.Sprintf("%s: %s (restricts %s; answer against %s at revision %d)", item.Kind, item.Question, strings.Join(item.Restricts, ","), item.RequestDigest, item.Revision))
				for _, option := range item.Options {
					marker := ""
					if option.ID == item.Recommend {
						marker = " (recommended)"
					}
					out.Field("    option."+option.ID, option.Consequence+marker)
				}
			}
		}
		return 0
	}
	filtered := []posemodel.ActionRequestView{}
	for _, v := range views {
		if state == "" || v.State == state {
			filtered = append(filtered, v)
		}
	}
	if jsonOutput {
		return writeJSON(stdout, filtered)
	}
	out := render(stdout, stderr)
	out.Field("actions.count", strconv.Itoa(len(filtered)))
	for _, v := range filtered {
		out.Field("  "+v.Request.ID, fmt.Sprintf("%s %s satisfaction=%s origin=%s — %s", v.Request.Kind, v.State, v.Satisfaction, v.Request.Origin, v.Request.Question))
	}
	return 0
}

func cmdActionShow(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return usageError(stderr, "Usage: pose action show <act-id> [--json]")
	}
	view, err := posemodel.Store{Root: root}.LoadActionRequest(args[0])
	if err != nil {
		render(stdout, stderr).Failure("pose action show: " + err.Error())
		return 1
	}
	if len(args) > 1 && args[1] == "--json" {
		return writeJSON(stdout, view)
	}
	renderActionView(render(stdout, stderr), view)
	return 0
}

// renderActionView shows exactly the content an answer is bound to: the
// question, the options with their consequences, what the request restricts,
// and the digest a resolution must name.
func renderActionView(out interface{ Field(string, string) }, v posemodel.ActionRequestView) {
	r := v.Request
	out.Field("action.id", r.ID)
	out.Field("action.state", fmt.Sprintf("%s satisfaction=%s revision=%d", v.State, v.Satisfaction, v.Revision))
	out.Field("action.kind", r.Kind)
	out.Field("action.origin", r.Origin)
	out.Field("action.question", r.Question)
	if r.Context != "" {
		out.Field("action.context", r.Context)
	}
	for _, option := range r.Options {
		marker := ""
		if option.ID == r.Recommend {
			marker = " (recommended)"
		}
		out.Field("action.option."+option.ID, option.Consequence+marker)
	}
	who := "unassigned"
	switch {
	case r.Recipient.Principal != "":
		who = r.Recipient.Principal
	case r.Recipient.Role != "":
		who = "role:" + r.Recipient.Role
	}
	out.Field("action.recipient", who)
	for _, target := range r.Targets {
		out.Field("action.target", target.String())
	}
	for _, effect := range r.Effects {
		out.Field("action.effect", effect.Phase+"/"+effect.Mode)
	}
	out.Field("action.request_digest", r.RequestDigest)
	if v.State == posemodel.ActionStateAnswered {
		out.Field("action.answer", fmt.Sprintf("%s by %s (%s)", v.Answer, v.AnsweredBy, v.Assurance))
	}
	for _, limitation := range v.Limitations {
		out.Field("action.limitation", limitation)
	}
}
