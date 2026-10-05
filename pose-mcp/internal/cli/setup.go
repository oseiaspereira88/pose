package cli

// pose setup — one place to see what is in force, what is new and what is
// missing, and the one next step (spec pose-setup-command). At a terminal it
// offers each open step and performs it only after an explicit yes; without
// one, or with --json, it only reports, with the exact commands, so an agent
// can put the same questions to the person it works for.

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
	"github.com/harne8/pose-mcp/internal/version"
)

const setupUsage = "Usage: pose setup [--json] [--no-input]"

// setupInput returns where answers come from and whether a person is there
// to give them. Tests replace it with scripted answers.
var setupInput = func() (io.Reader, bool) {
	info, err := os.Stdin.Stat()
	return os.Stdin, err == nil && info.Mode()&os.ModeCharDevice != 0
}

type setupStep struct {
	ID      string `json:"id"`
	Area    string `json:"area"`
	State   string `json:"state"` // done | todo | optional
	Summary string `json:"summary"`
	Command string `json:"command,omitempty"`
	// performable steps can be done by setup itself after a yes.
	performable bool
	capability  string
}

type setupYou struct {
	Suggested  string   `json:"suggested,omitempty"`
	Source     string   `json:"source,omitempty"`
	Registered bool     `json:"registered"`
	Roles      []string `json:"roles"`
	Keys       int      `json:"keys"`
	KeyFile    string   `json:"key_file,omitempty"`
}

type setupPlan struct {
	SchemaVersion   int                         `json:"schema_version"`
	Engine          string                      `json:"engine"`
	ReviewedVersion string                      `json:"reviewed_version,omitempty"`
	Project         string                      `json:"project_id"`
	ProjectDeclared bool                        `json:"project_declared"`
	Assurance       string                      `json:"identity_assurance"`
	You             setupYou                    `json:"you"`
	Principals      []identityView              `json:"principals"`
	Hook            bool                        `json:"pre_commit_hook"`
	InForce         []string                    `json:"in_force"`
	New             []posemodel.CapabilityState `json:"new"`
	NeedsSetup      []posemodel.CapabilityState `json:"needs_setup"`
	Available       []string                    `json:"available"`
	Onboarding      string                      `json:"onboarding_spec,omitempty"`
	Steps           []setupStep                 `json:"steps"`
	Next            *setupStep                  `json:"next,omitempty"`
}

// candidateKeyFile returns the user's public key POSE would suggest
// registering, preferring a security key.
func candidateKeyFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, name := range []string{"id_ed25519_sk.pub", "id_ed25519.pub"} {
		path := filepath.Join(home, ".ssh", name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func buildSetupPlan(root string) (setupPlan, error) {
	engine := version.ReleaseBase()
	plan := setupPlan{SchemaVersion: 1, Engine: engine, InForce: []string{}, New: []posemodel.CapabilityState{}, NeedsSetup: []posemodel.CapabilityState{}, Available: []string{}, Principals: []identityView{}}
	decisions, err := posemodel.ReadAdoptionDecisions(root)
	if err != nil {
		return plan, err
	}
	plan.ReviewedVersion = decisions.ReviewedVersion
	id, declared, err := posemodel.ReadProjectFile(root)
	if err != nil {
		return plan, err
	}
	plan.Project, plan.ProjectDeclared = posemodel.DefaultProjectID(root), declared && id != ""
	policy, err := posemodel.LoadActionPolicy(root)
	if err != nil {
		return plan, err
	}
	plan.Assurance = policy.IdentityAssurance
	for _, principal := range policy.Principals() {
		view := identityView{Principal: principal, Roles: policy.RolesOf(principal), Keys: []identityKeyView{}}
		if view.Roles == nil {
			view.Roles = []string{}
		}
		for _, key := range policy.KeysFor(principal) {
			view.Keys = append(view.Keys, identityKeyView{Type: key.Type, Fingerprint: key.Fingerprint(), Comment: key.Comment, Presence: key.ProvesPresence()})
		}
		plan.Principals = append(plan.Principals, view)
	}
	plan.You.Suggested, plan.You.Source = suggestedPrincipal(root)
	plan.You.Roles = []string{}
	if plan.You.Suggested != "" {
		plan.You.Roles = policy.RolesOf(plan.You.Suggested)
		if plan.You.Roles == nil {
			plan.You.Roles = []string{}
		}
		plan.You.Keys = len(policy.KeysFor(plan.You.Suggested))
		plan.You.Registered = plan.You.Keys > 0
	}
	plan.You.KeyFile = candidateKeyFile()
	_, hookErr := os.Lstat(filepath.Join(root, ".git", "hooks", "pre-commit"))
	plan.Hook = hookErr == nil

	states, err := posemodel.CapabilityStates(root)
	if err != nil {
		return plan, err
	}
	review, err := posemodel.CapabilitiesToReview(root, engine)
	if err != nil {
		return plan, err
	}
	isNew := map[string]bool{}
	for _, state := range review {
		isNew[state.ID] = true
		plan.New = append(plan.New, state)
	}
	agency := false
	for _, state := range states {
		switch {
		case state.State == posemodel.CapabilityOn:
			plan.InForce = append(plan.InForce, state.ID)
			agency = agency || state.ID == "agency-readiness"
		case state.State == posemodel.CapabilityNeedsSetup:
			plan.NeedsSetup = append(plan.NeedsSetup, state)
		case !isNew[state.ID] && state.State == posemodel.CapabilityOff:
			plan.Available = append(plan.Available, state.ID)
		}
	}

	// Steps, in the order a newcomer meets them.
	step := setupStep{ID: "identity.project", Area: "identity", State: "done", Summary: "project id " + plan.Project + " declared in .pose/project.json"}
	if !plan.ProjectDeclared {
		step.State, step.Summary, step.Command = "todo", "project id "+plan.Project+" is derived from the directory name, so another checkout would be another project", "pose update"
	}
	plan.Steps = append(plan.Steps, step)

	step = setupStep{ID: "identity.maintainer", Area: "identity", performable: true}
	rolesHeld := 0
	for _, list := range policy.Roles {
		rolesHeld += len(list)
	}
	keyArg := "<file.pub>"
	if plan.You.KeyFile != "" {
		keyArg = plan.You.KeyFile
	}
	who := plan.You.Suggested
	if who == "" {
		who = "human:<you>"
	}
	switch {
	case plan.You.Registered && len(plan.You.Roles) > 0:
		step.State, step.Summary, step.performable = "done", who+" holds "+strings.Join(plan.You.Roles, ", ")+" and can prove answers with a registered key", false
	case rolesHeld == 0 && agency:
		step.State, step.Summary = "todo", "nobody holds a role, so a request addressed to the maintainer has nobody to answer it"
	case plan.Assurance == posemodel.ReviewIdentityAssuranceVerified && !plan.You.Registered:
		step.State, step.Summary = "todo", "identity assurance is verified and "+who+" has no registered key to prove answers"
	default:
		step.State, step.Summary = "optional", who+" can register a key to sign answers (verified without any service)"
	}
	if step.State != "done" {
		step.Command = "pose identity add " + who + " --key " + keyArg + " --role maintainer --apply"
		if plan.You.KeyFile == "" {
			step.Summary += " (no SSH public key in ~/.ssh: create one with `ssh-keygen -t ed25519-sk`, or `-t ed25519` without a security key)"
		}
	}
	plan.Steps = append(plan.Steps, step)

	step = setupStep{ID: "hooks.pre-commit", Area: "safety", State: "done", Summary: "pre-commit gate installed"}
	if !plan.Hook {
		step.State, step.Summary, step.Command, step.performable = "todo", "no pre-commit gate: a commit is not checked before it lands", "pose hooks install", true
	}
	plan.Steps = append(plan.Steps, step)

	for _, state := range plan.New {
		recommendation := "decide"
		if state.DefaultForNew {
			recommendation = "recommended: adopt (on in new instances)"
		}
		summary := state.ID + " — " + state.Effect + " (since " + state.IntroducedIn + "; " + recommendation + ")"
		if state.State == posemodel.CapabilityDeferred {
			summary += "; deferred earlier: " + state.Decision.Reason
		}
		if state.Missing != "" {
			summary += "; " + state.Missing
		}
		plan.Steps = append(plan.Steps, setupStep{ID: "capability:" + state.ID, Area: "capabilities", State: "todo", Summary: summary,
			Command: "pose adopt " + state.ID + " --apply  |  --decline --reason <why> --apply  |  --defer --reason <why> --apply", performable: true, capability: state.ID})
	}

	if dirty, err := setupUncommitted(root); err == nil && dirty != "" {
		plan.Steps = append(plan.Steps, setupStep{ID: "commit", Area: "lifecycle", State: "todo", Summary: dirty, Command: "git add -A && git commit -m \"Adopt POSE\""})
	}
	// The onboarding spec tracks these steps (spec pose-onboarding-spec):
	// started first, closed once nothing else is open.
	if rel := findOnboardingSpec(root); rel != "" {
		if spec, err := (posemodel.Store{Root: root}).GetSpec(onboardingSlug); err == nil {
			open := 0
			for _, other := range plan.Steps {
				if other.State == "todo" {
					open++
				}
			}
			step := setupStep{ID: "onboarding", Area: "lifecycle", State: "done", Summary: rel + " is " + spec.Status}
			switch spec.Status {
			case "draft":
				step.State, step.Summary, step.Command = "todo", rel+" tracks these steps as governed work: start it first", "pose start spec:"+onboardingSlug
				plan.Steps = append([]setupStep{step}, plan.Steps...)
			case "in-progress":
				if open == 0 {
					step.State, step.Summary, step.Command = "todo", rel+": every step is done — run its checks, fill the requirement trace and close it", "pose close spec:"+onboardingSlug
				} else {
					step.State, step.Summary = "optional", rel+" is in progress; the open steps above complete it"
				}
				plan.Steps = append(plan.Steps, step)
			default:
				plan.Steps = append(plan.Steps, step)
			}
			plan.Onboarding = rel
		}
	}
	for i := range plan.Steps {
		if plan.Steps[i].State == "todo" {
			plan.Next = &plan.Steps[i]
			break
		}
	}
	if plan.Next == nil {
		for i := range plan.Steps {
			if plan.Steps[i].State == "optional" {
				plan.Next = &plan.Steps[i]
				break
			}
		}
	}
	return plan, nil
}

// setupUncommitted says why the POSE configuration is not committed yet, or
// "" when it is.
func setupUncommitted(root string) (string, error) {
	if err := exec.Command("git", "-C", root, "rev-parse", "--verify", "-q", "HEAD").Run(); err != nil {
		return "the repository has no commit yet: commit the installed instance so every checkout reads the same configuration", nil
	}
	out, err := exec.Command("git", "-C", root, "status", "--porcelain", "--", ".pose", "AGENTS.md", "POSE.md").Output()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(out)) != "" {
		return "POSE configuration has uncommitted changes: commit them so every checkout reads the same configuration", nil
	}
	return "", nil
}

func cmdSetup(root string, args []string, stdout, stderr io.Writer) int {
	jsonOutput, noInput := false, false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--no-input":
			noInput = true
		default:
			return usageError(stderr, setupUsage)
		}
	}
	out := render(stdout, stderr)
	plan, err := buildSetupPlan(root)
	if err != nil {
		out.Failure("pose setup: " + err.Error())
		return 1
	}
	if jsonOutput {
		return writeJSON(stdout, plan)
	}
	renderSetupPlan(out.Field, out.Section, plan)
	input, tty := setupInput()
	if noInput || !tty {
		if plan.Next != nil {
			out.Hint("run the commands above, or `pose setup` at a terminal to be asked step by step")
		}
		return 0
	}
	return runSetupInteractive(root, plan, bufio.NewReader(input), stdout, stderr)
}

func renderSetupPlan(rawField func(string, string), section func(string), plan setupPlan) {
	// Every open step carries its command, not only the next one.
	commands := map[string]string{}
	for _, step := range plan.Steps {
		if step.State != "done" && step.Command != "" {
			commands["setup."+step.ID] = step.Command
			if step.capability != "" {
				commands["setup.new."+step.capability] = step.Command
			}
		}
	}
	field := func(name, value string) {
		rawField(name, value)
		if command, ok := commands[name]; ok && name != "setup.next" {
			rawField(name+".command", command)
		}
	}
	section("Identity")
	for _, step := range plan.Steps {
		if step.Area == "identity" {
			field("setup."+step.ID, step.State+" — "+step.Summary)
		}
	}
	if plan.You.Suggested != "" {
		field("setup.you", plan.You.Suggested+" (from "+plan.You.Source+"; a name, not a proof — a registered key is the proof)")
	}
	for _, principal := range plan.Principals {
		line := "roles: " + strings.Join(principal.Roles, ", ")
		if len(principal.Roles) == 0 {
			line = "roles: none"
		}
		line += "; keys: " + strconv.Itoa(len(principal.Keys))
		field("setup.principal."+principal.Principal, line)
	}
	section("Safety")
	for _, step := range plan.Steps {
		if step.Area == "safety" {
			field("setup."+step.ID, step.State+" — "+step.Summary)
		}
	}
	section("Capabilities")
	field("setup.in_force", strings.Join(plan.InForce, ", "))
	reviewed := plan.ReviewedVersion
	if reviewed == "" {
		reviewed = "never"
	}
	if len(plan.New) == 0 {
		field("setup.new", "none since the last review ("+reviewed+")")
	}
	for _, step := range plan.Steps {
		if step.Area == "capabilities" {
			field("setup.new."+step.capability, step.Summary)
		}
	}
	for _, state := range plan.NeedsSetup {
		field("setup.needs_setup."+state.ID, state.Missing)
	}
	if len(plan.Available) > 0 {
		sort.Strings(plan.Available)
		field("setup.available", strings.Join(plan.Available, ", ")+" — off; `pose adopt --list` explains each")
	}
	for _, step := range plan.Steps {
		if step.Area == "lifecycle" {
			section("Lifecycle")
			field("setup."+step.ID, step.State+" — "+step.Summary)
		}
	}
	section("Next")
	if plan.Next == nil {
		field("setup.next", "nothing to set up — `pose new-spec <slug>` starts governed work")
		return
	}
	field("setup.next", plan.Next.Summary)
	field("setup.next.command", plan.Next.Command)
}

// runSetupInteractive offers each open step and performs it only after an
// explicit yes.
func runSetupInteractive(root string, plan setupPlan, in *bufio.Reader, stdout, stderr io.Writer) int {
	out := render(stdout, stderr)
	ask := func(question string) string {
		_, _ = io.WriteString(stdout, question)
		line, _ := in.ReadString('\n')
		return strings.TrimSpace(line)
	}
	yes := func(answer string) bool {
		switch strings.ToLower(answer) {
		case "y", "yes", "s", "sim":
			return true
		}
		return false
	}
	date := time.Now().UTC().Format(time.DateOnly)
	decided := false
	for _, step := range plan.Steps {
		if !step.performable || step.State == "done" {
			continue
		}
		switch {
		case step.ID == "hooks.pre-commit":
			if !yes(ask("Install the pre-commit gate (pose hooks install)? [y/N] ")) {
				continue
			}
			if code := cmdHooks(root, []string{"install"}, stdout, stderr); code != 0 {
				return code
			}
		case step.ID == "identity.maintainer":
			principal := plan.You.Suggested
			if answer := ask("Register a key to prove your answers? Principal [" + principal + "] (empty keeps it, '-' skips): "); answer == "-" {
				continue
			} else if answer != "" {
				principal = answer
			}
			if principal == "" {
				out.Hint("skipped: no principal")
				continue
			}
			keyFile := plan.You.KeyFile
			if answer := ask("Public key file [" + keyFile + "]: "); answer != "" {
				keyFile = answer
			}
			if keyFile == "" {
				out.Hint("skipped: no public key; create one with `ssh-keygen -t ed25519-sk` and run `pose setup` again")
				continue
			}
			if !yes(ask("Register " + keyFile + " for " + principal + " with role maintainer? [y/N] ")) {
				continue
			}
			if code := cmdIdentityAdd(root, []string{principal, "--key", keyFile, "--role", "maintainer", "--apply"}, stdout, stderr); code != 0 {
				return code
			}
		case step.capability != "":
			answer := strings.ToLower(ask(step.Summary + "\n  [a]dopt, [d]ecline, [l]ater (defer), [s]kip? "))
			switch answer {
			case "a", "adopt":
				if code := cmdAdopt(root, []string{step.capability, "--apply"}, stdout, stderr); code != 0 {
					return code
				}
				decided = true
			case "d", "decline", "l", "later", "defer":
				flag := "--decline"
				if strings.HasPrefix(answer, "l") || answer == "defer" {
					flag = "--defer"
				}
				reason := ask("Reason (kept so the decision can be revisited): ")
				if reason == "" {
					out.Hint("skipped: a decision needs a reason")
					continue
				}
				if code := cmdAdopt(root, []string{step.capability, flag, "--reason", reason, "--date", date, "--apply"}, stdout, stderr); code != 0 {
					return code
				}
				decided = true
			}
		}
	}
	if decided {
		advanceReviewedVersion(root)
	}
	after, err := buildSetupPlan(root)
	if err != nil {
		out.Failure("pose setup: " + err.Error())
		return 1
	}
	if after.Next != nil {
		out.Field("setup.next", after.Next.Summary)
		out.Field("setup.next.command", after.Next.Command)
	} else {
		out.Field("setup.next", "nothing to set up — `pose new-spec <slug>` starts governed work")
	}
	return 0
}

// advanceReviewedVersion records this engine's release as reviewed once no
// capability is left to decide for it.
func advanceReviewedVersion(root string) {
	engine := version.ReleaseBase()
	if pending, err := posemodel.CapabilitiesToReview(root, engine); err == nil && len(pending) == 0 {
		_ = posemodel.SetReviewedVersion(root, engine)
	}
}
