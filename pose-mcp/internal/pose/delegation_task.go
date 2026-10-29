package pose

// Delegating a general task to another agent (spec pose-delegation-task).
//
// A task runs in a real directory with the access its caller chose, under the
// lifecycle every delegated run shares — active marker, live events, cancel,
// interruption, resume and handoff — but it is never governance evidence: its
// record lives under the user's state directory unless --record asks for
// .pose/delegate-runs, no review reader reads either place, and attestation
// refuses a task run (R5, R7; D1, D2).

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// delegationTaskRunsDir keeps the runs of --record inside the project.
	delegationTaskRunsDir = ".pose/delegate-runs"
	// DelegationTaskContinueMessage is what a resumed task is told when its
	// caller gives no prompt (R8).
	DelegationTaskContinueMessage = "Continue the task where you stopped. Redo the command that was interrupted, if one was. Do not repeat steps you completed unless their result must be fresh.\n"
	delegationTaskPromptMaxBytes  = 1 << 20
	delegationTaskRecordMaxBytes  = 4 << 20
)

// DelegationTaskUsageError is a request the policy or the flags refuse before
// anything runs: the command exits 2.
type DelegationTaskUsageError struct{ Msg string }

func (e *DelegationTaskUsageError) Error() string { return "pose: " + e.Msg }

func taskUsage(format string, args ...any) error {
	return &DelegationTaskUsageError{Msg: fmt.Sprintf(format, args...)}
}

// DelegationTaskRequest is what a caller asks a task to be.
type DelegationTaskRequest struct {
	Agent     string
	Access    string
	Dir       string
	Prompt    string
	AddDirs   []string
	Effort    string
	NoNetwork bool
	Record    bool
	Label     string
	Lead      string
	Routing   *DelegationRoutingRecord
	// OnStart is called once the attempt's marker is written, as for every
	// delegated run.
	OnStart func(attempt string) error
}

// DelegationTaskPlan is what a task run will do.
type DelegationTaskPlan struct {
	DelegationTaskRequest
	Config     DelegationAgent
	Template   []string
	OutsideGit bool
	Warnings   []string
	// Existing is the recorded continuation a repeated resume or handoff
	// returns instead of running again.
	Existing       *ReviewRun
	adapter        ReviewerAdapter
	envUnset       []string
	predecessor    *ReviewRun
	handoff        bool
	handoffSection string
	session        string
	processTimeout time.Duration
}

// TaskDefaultAccess is the access a task runs with when its caller names none:
// purposes.task.default_access, or read.
func (p DelegationPolicy) TaskDefaultAccess() string {
	if access := p.Purposes[DelegationPurposeTask].DefaultAccess; access != "" {
		return access
	}
	return DelegationAccessRead
}

// TaskAgentNames lists the agents that can run a task with access, in the
// order the policy declares them.
func (p DelegationPolicy) TaskAgentNames(access string) []string {
	var names []string
	for _, name := range p.DeclaredAgentNames() {
		if len(p.Agents[name].Commands[access]) > 0 {
			names = append(names, name)
		}
	}
	return names
}

// DelegationAccessWarning names what the full access disables (R2).
const DelegationAccessWarning = "--access full runs the agent with no sandbox and no approval prompts, acting on the real working directory with every permission your user has"

// PlanDelegationTask checks a request against the policy and the flags'
// rules; nothing runs and nothing is written (R1–R4).
func (s Store) PlanDelegationTask(req DelegationTaskRequest) (DelegationTaskPlan, error) {
	policy, _, found, err := s.LoadDelegationPolicy()
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	if !found {
		return DelegationTaskPlan{}, ErrNoReviewerAdapter
	}
	if req.Access == "" {
		req.Access = policy.TaskDefaultAccess()
	}
	if !containsString(delegationAccessLevels, req.Access) {
		return DelegationTaskPlan{}, taskUsage("--access takes %s", strings.Join(delegationAccessLevels, ", "))
	}
	agent, ok := policy.Agents[req.Agent]
	if !ok {
		return DelegationTaskPlan{}, taskUsage("no agent %q is configured", req.Agent)
	}
	template := agent.Commands[req.Access]
	if len(template) == 0 {
		return DelegationTaskPlan{}, taskUsage("agent %s declares no %s command", req.Agent, req.Access)
	}
	if err := validateTaskPrompt(req.Prompt); err != nil {
		return DelegationTaskPlan{}, err
	}
	dir, err := taskDirectory(req.Dir)
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	req.Dir = dir
	plan := DelegationTaskPlan{DelegationTaskRequest: req, Config: agent, Template: template, envUnset: policy.EnvUnset}
	if err := plan.checkFlags(); err != nil {
		return DelegationTaskPlan{}, err
	}
	plan.adapter, _ = policy.agentAdapter(req.Agent, DelegationPurposeTask)
	plan.adapter.Command = template
	if _, err := ResolveReviewerContainment(plan.adapter); err != nil {
		return DelegationTaskPlan{}, err
	}
	plan.OutsideGit = GitNotARepository(dir)
	if err := plan.checkOutsideGit(); err != nil {
		return DelegationTaskPlan{}, err
	}
	if req.Access == DelegationAccessFull {
		plan.Warnings = append(plan.Warnings, DelegationAccessWarning)
	}
	return plan, nil
}

// checkFlags applies the rules of --add-dir, --effort and --no-network (R3).
func (p DelegationTaskPlan) checkFlags() error {
	if p.NoNetwork {
		if p.Access == DelegationAccessWrite || p.Access == DelegationAccessFull {
			return taskUsage("--no-network cannot be enforced with --access %s, which lets the agent reach the network through what it runs", p.Access)
		}
		if len(p.Config.NoNetworkArgs) == 0 || !containsString(p.Template, "{no_network}") {
			return taskUsage("agent %s cannot enforce --no-network: it declares no no_network_args for its %s command", p.Agent, p.Access)
		}
	}
	if len(p.AddDirs) > 0 && (len(p.Config.AddDirArgs) == 0 || !containsString(p.Template, "{add_dirs}")) {
		return taskUsage("agent %s cannot take --add-dir: it declares no add_dir_args for its %s command", p.Agent, p.Access)
	}
	for i, dir := range p.AddDirs {
		abs, err := taskDirectory(dir)
		if err != nil {
			return err
		}
		p.AddDirs[i] = abs
	}
	if p.Effort != "" {
		if p.Config.Effort == nil || p.Config.Effort.Levels[p.Effort] == nil {
			return taskUsage("agent %s declares no effort level %q", p.Agent, p.Effort)
		}
		if !containsString(p.Template, "{effort}") {
			return taskUsage("agent %s's %s command takes no {effort}", p.Agent, p.Access)
		}
	}
	return nil
}

// checkOutsideGit refuses a task outside Git whose agent declares
// outside_git_args its template has no place for: they would be dropped (B7).
func (p DelegationTaskPlan) checkOutsideGit() error {
	if p.OutsideGit && len(p.Config.OutsideGitArgs) > 0 && !containsString(p.Template, "{outside_git}") {
		return taskUsage("%s is in no Git repository and agent %s declares outside_git_args, but its %s command has no {outside_git}", p.Dir, p.Agent, p.Access)
	}
	return nil
}

// taskResumeTemplate is the command that resumes a task of access: the
// agent's resume.commands for it, or resume.command for read.
func taskResumeTemplate(agent DelegationAgent, access string) []string {
	if agent.Resume == nil {
		return nil
	}
	if argv := agent.Resume.Commands[access]; len(argv) > 0 {
		return argv
	}
	if access == DelegationAccessRead {
		return agent.Resume.Command
	}
	return nil
}

func validateTaskPrompt(prompt string) error {
	switch {
	case strings.TrimSpace(prompt) == "":
		return taskUsage("the prompt is empty")
	case len(prompt) > delegationTaskPromptMaxBytes:
		return taskUsage("the prompt exceeds %d bytes", delegationTaskPromptMaxBytes)
	case !utf8.ValidString(prompt):
		return taskUsage("the prompt is not UTF-8 text")
	}
	return nil
}

func taskDirectory(dir string) (string, error) {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", taskUsage("directory %s: %v", dir, err)
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return "", taskUsage("%s is not a directory", abs)
	}
	return abs, nil
}

// TaskArgv is the argv a task runs, as it may be shown or recorded: a
// session it resumes stays {session}, never its value (review round 1, B1).
func (p DelegationTaskPlan) TaskArgv(output string) []string {
	session := ""
	if p.session != "" {
		session = "{session}"
	}
	return p.argv(output, session)
}

// argv is the access template with the task's fragments in place of
// {add_dirs}, {no_network} and {outside_git}, then the output, session and
// effort placeholders every delegated run fills.
func (p DelegationTaskPlan) argv(output, session string) []string {
	var expanded []string
	for _, arg := range p.Template {
		switch arg {
		case "{add_dirs}":
			for _, dir := range p.AddDirs {
				for _, part := range p.Config.AddDirArgs {
					if part == "{dir}" {
						part = dir
					}
					expanded = append(expanded, part)
				}
			}
		case "{no_network}":
			if p.NoNetwork {
				expanded = append(expanded, p.Config.NoNetworkArgs...)
			}
		case "{outside_git}":
			if p.OutsideGit {
				expanded = append(expanded, p.Config.OutsideGitArgs...)
			}
		default:
			expanded = append(expanded, arg)
		}
	}
	return ReviewerArgv(expanded, output, session, p.Effort, p.Config.Effort)
}

// taskStateOwner tells who owns a directory of the task-state path; tests
// replace it to stand for another user's or root's directory.
var taskStateOwner = stateDirOwner

// taskStateBase is the user's state directory: XDG_STATE_HOME when it is an
// absolute path, as the XDG Base Directory specification requires, else the
// platform default (spec pose-delegation-task-detach-and-state R5).
func taskStateBase() (string, error) {
	if base := os.Getenv("XDG_STATE_HOME"); filepath.IsAbs(base) {
		return base, nil
	}
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); filepath.IsAbs(base) {
			return base, nil
		}
	} else if home, err := os.UserHomeDir(); err == nil && filepath.IsAbs(home) {
		return filepath.Join(home, ".local", "state"), nil
	}
	return "", fmt.Errorf("pose: no state directory: set XDG_STATE_HOME to an absolute path, or HOME")
}

// taskStateDir is where the runs of a project live without --record: under
// the user's state directory, one directory per project root (D1). The base
// is the person's own and may be reached through links; it is resolved once,
// and it and every ancestor must be a directory no other user can rewrite
// (R6). The directories POSE makes below it are real, owner-only and the
// user's (R7).
func taskStateDir(root string, create bool) (string, error) {
	base, err := taskStateBase()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if create {
		if err := os.MkdirAll(base, 0700); err != nil {
			return "", err
		}
	}
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	for dir := resolved; ; dir = filepath.Dir(dir) {
		info, err := os.Lstat(dir)
		if err != nil {
			return "", err
		}
		user, byRoot := taskStateOwner(info)
		switch {
		case !info.IsDir():
			return "", fmt.Errorf("pose: the state directory %s is not a directory", dir)
		case !user && !byRoot:
			return "", fmt.Errorf("pose: the state directory %s belongs to another user; set XDG_STATE_HOME to a directory of yours", dir)
		case stateDirWritableByOthers(info, true):
			return "", fmt.Errorf("pose: another user can write %s, above the state directory; remove its group and other write permission, or set XDG_STATE_HOME to a private directory", dir)
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	sum := sha256.Sum256([]byte(abs))
	dir := resolved
	for _, name := range []string{"pose", "delegate", hex.EncodeToString(sum[:])[:16]} {
		dir = filepath.Join(dir, name)
		if create {
			if err := os.Mkdir(dir, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return "", err
			}
		}
		info, err := os.Lstat(dir)
		if err != nil {
			return "", err
		}
		user, _ := taskStateOwner(info)
		switch {
		case !info.IsDir(): // Lstat: a link is not a directory
			return "", fmt.Errorf("pose: %s is not a private directory", dir)
		case !user:
			return "", fmt.Errorf("pose: %s belongs to another user", dir)
		case stateDirWritableByOthers(info, false):
			return "", fmt.Errorf("pose: another user can write %s; remove its group and other write permission", dir)
		}
	}
	return dir, nil
}

// taskRunsDir is the directory a task's record goes to; it is created private
// and refused when it is a link.
func (s Store) taskRunsDir(record, create bool) (string, error) {
	if record {
		dir, err := ensureReviewArtifactDir(s.Root, delegationTaskRunsDir, create)
		if err == nil && create {
			err = os.Chmod(dir, 0700)
		}
		return dir, err
	}
	dir, err := taskStateDir(s.Root, create)
	if err != nil {
		return "", err
	}
	if info, err := os.Lstat(dir); err != nil {
		return "", err
	} else if create && info.Mode().Perm() != 0700 {
		if err := os.Chmod(dir, 0700); err != nil {
			return "", err
		}
	}
	return dir, nil
}

// DelegationTaskRecordDir is where a task records itself, --record or not;
// nothing is created.
func (s Store) DelegationTaskRecordDir(record bool) (string, error) {
	return s.taskRunsDir(record, false)
}

// ReadDelegationTaskRuns reads the task runs of the project from both places
// a task records itself, oldest first, with what could not be read.
func (s Store) ReadDelegationTaskRuns() ([]ReviewRun, []string) {
	var runs []ReviewRun
	var lost []string
	for _, record := range []bool{false, true} {
		dir, err := s.taskRunsDir(record, false)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				lost = append(lost, "task runs: "+err.Error())
			}
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			lost = append(lost, "task runs: "+err.Error())
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasSuffix(name, ".json") || !e.Type().IsRegular() {
				continue
			}
			raw, err := readBoundedRegular(filepath.Join(dir, name), delegationTaskRecordMaxBytes)
			if err != nil {
				lost = append(lost, name+" ("+err.Error()+")")
				continue
			}
			var run ReviewRun
			if json.Unmarshal(raw, &run) != nil || run.Purpose != DelegationPurposeTask || run.RunID+".json" != name || !strings.HasPrefix(run.RunID, "trn-") {
				lost = append(lost, name+" (not a task run)")
				continue
			}
			run.Path, run.eventsDir = filepath.Join(dir, name), dir
			runs = append(runs, run)
		}
	}
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].StartedAt != runs[j].StartedAt {
			return runs[i].StartedAt < runs[j].StartedAt
		}
		return runs[i].RunID < runs[j].RunID
	})
	return runs, lost
}

// IsDelegationTaskRun reports whether id names a recorded task run, so a
// review reader can refuse it by name rather than not finding it (R7).
func (s Store) IsDelegationTaskRun(id string) bool {
	runs, _ := s.ReadDelegationTaskRuns()
	for _, run := range runs {
		if run.RunID == id {
			return true
		}
	}
	return false
}

// ErrDelegationTaskRun refuses a task run where a review run is expected (R7,
// D2).
func ErrDelegationTaskRun(id string) error {
	return fmt.Errorf("pose: run %s is a task run: a task never produces an attestation, a verdict or evidence; run it as a review", id)
}

// ReadDelegationTaskPrompt is the prompt run was given, checked against the
// digest its record binds.
func (s Store) ReadDelegationTaskPrompt(run ReviewRun) (string, error) {
	if run.eventsDir == "" || run.PromptDigest == "" {
		return "", fmt.Errorf("pose: run %s keeps no prompt", run.RunID)
	}
	raw, err := readBoundedRegular(filepath.Join(run.eventsDir, run.RunID+".prompt"), delegationTaskPromptMaxBytes)
	if err != nil {
		return "", err
	}
	if digestBytes(raw) != run.PromptDigest {
		return "", fmt.Errorf("pose: the prompt of run %s is not the one its record binds", run.RunID)
	}
	return string(raw), nil
}

// RunDelegationTask runs plan in its directory and records the run; the
// returned run is the record (R1, R5, R6).
func (s Store) RunDelegationTask(parent context.Context, plan DelegationTaskPlan, now func() time.Time) (result ReviewRun, resultErr error) {
	if plan.Existing != nil {
		return *plan.Existing, nil
	}
	if err := parent.Err(); err != nil {
		return ReviewRun{}, err
	}
	parent, cancelRun := context.WithCancel(parent)
	defer cancelRun()
	safety, err := ResolveReviewerContainment(plan.adapter)
	if err != nil {
		return ReviewRun{}, err
	}
	dir, err := s.taskRunsDir(plan.Record, true)
	if err != nil {
		return ReviewRun{}, err
	}
	run := ReviewRun{SchemaVersion: 1, Scope: "task", Kind: DelegationPurposeTask, Purpose: DelegationPurposeTask, Adapter: plan.Agent,
		Vendor: plan.Config.Vendor, Model: plan.Config.Model, Label: plan.Label, Lead: plan.Lead, Access: plan.Access, Dir: plan.Dir,
		AddDirs: plan.AddDirs, NoNetwork: plan.NoNetwork, Recorded: plan.Record, Effort: plan.Effort, Events: plan.Config.Events, Routing: plan.Routing,
		StartedAt: now().UTC().Format(time.RFC3339Nano), ExitCode: -1, PromptDigest: digestBytes([]byte(plan.Prompt)), Safety: safety}
	if plan.predecessor == nil {
		token := make([]byte, 16)
		if _, err := rand.Read(token); err != nil {
			return ReviewRun{}, err
		}
		run.AttemptID = "tat-" + hex.EncodeToString(token)
	} else {
		prior := plan.predecessor
		run.AttemptID, run.Label, run.Lead = prior.AttemptID, prior.Label, prior.Lead
		if plan.handoff {
			run.HandsOffFrom, run.HandoffDigest, run.HandoffRenderer = prior.RunID, digestBytes([]byte(plan.handoffSection)), handoffRenderer
		} else {
			run.Resumes = prior.RunID
		}
		// A run is continued once, by a resume or a handoff (as review runs are).
		claim := filepath.Join(filepath.Dir(prior.Path), prior.RunID+".continued")
		if err := PublishFile(claim, []byte(run.StartedAt+"\n"), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
			return ReviewRun{}, fmt.Errorf("pose: run %s is already being continued", prior.RunID)
		}
	}
	stopActive, cancelled, err := s.markDelegationActive(run, cancelRun)
	if err != nil {
		return ReviewRun{}, err
	}
	defer func() { stopActive(resultErr != nil || result.RunID == "") }()
	tmp, err := os.MkdirTemp("", "pose-task-run-*")
	if err != nil {
		return ReviewRun{}, err
	}
	defer os.RemoveAll(tmp)
	var stdout, stderr []byte
	// argvSafe is set once the recorded argv was redacted with the session
	// the run printed; a run that ends before that records no argv, which a
	// template could carry the session in (review round 3, B4).
	argvSafe := false
	finish := func(status, failure string) (ReviewRun, error) {
		if !argvSafe {
			run.Argv = nil
		}
		if cancelled.Load() && status != ReviewRunCompleted {
			status, failure = ReviewRunFailed, "cancelled by pose delegate cancel"
			run.Safety.ExecutionCause = "cancelled"
		}
		run.Status, run.Failure = status, failure
		if status != ReviewRunInterrupted {
			run.Interruption = nil
		}
		run.EndedAt = now().UTC().Format(time.RFC3339Nano)
		return s.recordDelegationTask(dir, run, plan.Prompt, stdout, stderr)
	}
	if plan.OnStart != nil {
		if err := plan.OnStart(run.AttemptID); err != nil {
			return finish(ReviewRunFailed, "the run was not started: its attempt could not be handed back: "+err.Error())
		}
	}
	output := filepath.Join(tmp, "output.txt")
	argv := plan.argv(output, plan.session)
	env, secrets, err := reviewAdapterEnvironment(plan.adapter, tmp, plan.Prompt, "")
	if err != nil {
		return finish(ReviewRunFailed, "the run was not started: "+err.Error())
	}
	// The record keeps the argv with the session as {session} and every
	// authorized credential redacted (B1).
	for _, arg := range plan.TaskArgv(output) {
		red, rerr := redactReviewCapture([]byte(arg), secrets, "", 17*int64(len(arg))+1)
		if rerr != nil {
			return finish(ReviewRunFailed, "the run was not started: its argv could not be redacted")
		}
		run.Argv = append(run.Argv, string(red))
	}
	env, run.EnvRemoved = withoutParentSession(env, plan.envUnset)
	timeout := time.Duration(safety.TimeoutMS) * time.Millisecond
	if plan.processTimeout > 0 && plan.processTimeout < timeout {
		timeout = plan.processTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir, cmd.Env = plan.Dir, env
	cmd.Stdin = strings.NewReader(plan.Prompt)
	out := &reviewBoundedCapture{limit: safety.StreamBytes, cancel: cancel}
	errs := &reviewBoundedCapture{limit: safety.StreamBytes, cancel: cancel}
	cmd.Stdout, cmd.Stderr = out, errs
	pattern := ""
	if plan.Config.Session != nil {
		pattern = plan.Config.Session.Pattern
	}
	if live := s.startDelegationLive(run.AttemptID, plan.Config.Events, secrets, pattern, plan.session, safety.StreamBytes); live != nil {
		cmd.Stdout = io.MultiWriter(out, live)
		defer func() { _ = os.Remove(s.delegationLivePath(run.AttemptID)) }()
		defer live.Close()
	}
	isolateProcessGroup(cmd)
	cmd.WaitDelay = 5 * time.Second
	runErr := cmd.Run()
	terminationErr := killProcessGroup(cmd)
	if errors.Is(runErr, exec.ErrWaitDelay) && cmd.ProcessState != nil && cmd.ProcessState.Success() {
		runErr = nil
	}
	transcript, outOverflow := out.result()
	errText, errOverflow := errs.result()
	if cmd.ProcessState != nil {
		run.ExitCode = cmd.ProcessState.ExitCode()
	}
	if runErr != nil && cmd.ProcessState == nil {
		return finish(ReviewRunFailed, "the agent's command could not start: "+runErr.Error())
	}
	safety.ExecutionCause = executionCause(ctx)
	if outOverflow || errOverflow || terminationErr != nil {
		safety.CaptureComplete = false
		if terminationErr != nil {
			return finish(ReviewRunFailed, "process group termination failed; descendants may remain")
		}
		return finish(ReviewRunFailed, "the agent's output exceeded the stream budget")
	}
	safety.CaptureComplete = true
	// The final message obeys output_bytes whatever its source: a file the
	// agent wrote that is not readable within it fails the run, and so does a
	// stream final message beyond it; the stream's tail is cut to it (B3).
	raw, outputErr := ReadReviewAdapterFile(output, safety.OutputBytes)
	final := DelegationStreamFinal(plan.Config.Events, transcript)
	switch {
	case outputErr != nil && !errors.Is(outputErr, os.ErrNotExist):
		safety.CaptureComplete = false
		return finish(ReviewRunFailed, "the agent's output file is not a regular file within output_bytes")
	case outputErr == nil && len(bytes.TrimSpace(raw)) > 0:
		run.Output = string(raw)
	case strings.TrimSpace(final) != "":
		if int64(len(final)) > safety.OutputBytes {
			safety.CaptureComplete = false
			return finish(ReviewRunFailed, "the agent's final message exceeds output_bytes")
		}
		run.Output = final
	}
	// With no final message the stream's tail stands in for it, cut only
	// after the stream is redacted: a cut first could leave the end of a
	// credential or a session that no longer matches (review round 2, B1).
	tailed := run.Output == ""
	// The session, the limit and the reset are recognized on stdout and
	// stderr, each as written; a session both print is one session, two
	// different ones are refused (B2; review round 3, B2 and B3).
	status, failure, captureErr := s.classifyReviewInterruption(&run, DispatchPlan{Config: plan.adapter, session: plan.session, extraChannels: []string{string(errText)}}, transcript, runErr, ctx.Err())
	if run.Interruption != nil {
		// A task resumes with its access's command, not the review's.
		run.Interruption.Resumable = run.SessionDigest != "" && len(taskResumeTemplate(plan.Config, plan.Access)) > 0
	}
	session := run.sessionForRedaction
	run.sessionForRedaction = ""
	// The recorded argv is redacted again once the session is known: a
	// template may carry it literally.
	for i := 0; captureErr == nil && session != "" && i < len(run.Argv); i++ {
		var red []byte
		if red, captureErr = redactReviewCapture([]byte(run.Argv[i]), secrets, session, 17*int64(len(run.Argv[i]))+1); captureErr == nil {
			run.Argv[i] = string(red)
		}
	}
	argvSafe = captureErr == nil
	if captureErr == nil {
		stdout, captureErr = redactReviewCapture(transcript, secrets, session, safety.StreamBytes)
	}
	if captureErr == nil && tailed {
		limit := 20000
		if int64(limit) > safety.OutputBytes {
			limit = int(safety.OutputBytes)
		}
		run.Output = tail(string(stdout), limit)
	}
	if captureErr == nil {
		var redacted []byte
		if redacted, captureErr = redactReviewCapture([]byte(run.Output), secrets, session, safety.OutputBytes); captureErr == nil {
			run.Output = string(redacted)
		}
	}
	if captureErr == nil && run.Events != "" {
		run.eventsBytes, captureErr = redactedDelegationEvents(run.Events, transcript, run.Output, secrets, session)
	}
	if captureErr == nil {
		stderr, captureErr = redactReviewCapture(errText, secrets, session, safety.StreamBytes)
	}
	if captureErr != nil {
		run.Output, stdout, stderr, run.eventsBytes = "", nil, nil, nil
		safety.CaptureComplete = false
		return finish(ReviewRunFailed, "the agent's output could not be redacted; it was not kept")
	}
	if status == "" {
		status = ReviewRunCompleted
	}
	return finish(status, failure)
}

// recordDelegationTask publishes a task run: its record and, beside it, the
// prompt, the stream, the error stream and the normalized events, each
// owner-only and bound by digest in the record (R5).
func (s Store) recordDelegationTask(dir string, run ReviewRun, prompt string, stdout, stderr []byte) (ReviewRun, error) {
	run.RunID, run.Path = "", ""
	run.TranscriptDigest, run.StderrDigest = digestBytes(stdout), digestBytes(stderr)
	events := run.eventsBytes
	run.eventsBytes, run.EventsDigest = nil, ""
	if run.Events != "" && events != nil {
		run.EventsDigest = digestBytes(events)
	}
	identity, _ := json.Marshal(run)
	sum := sha256.Sum256(identity)
	run.RunID = "trn-" + hex.EncodeToString(sum[:])[:16]
	run.TranscriptPath = run.RunID + ".transcript"
	pretty, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return ReviewRun{}, err
	}
	files := []struct {
		name string
		data []byte
	}{{".prompt", []byte(prompt)}, {".transcript", stdout}, {".stderr", stderr}}
	if run.EventsDigest != "" {
		files = append(files, struct {
			name string
			data []byte
		}{".events.ndjson", events})
	}
	for _, f := range files {
		if err := PublishFile(filepath.Join(dir, run.RunID+f.name), f.data, 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
			return ReviewRun{}, err
		}
	}
	if err := PublishFile(filepath.Join(dir, run.RunID+".json"), append(pretty, '\n'), 0600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		return ReviewRun{}, err
	}
	run.Path, run.eventsDir = filepath.Join(dir, run.RunID+".json"), dir
	return run, nil
}

// taskContinuation finds the recorded continuation of prior, if any.
func taskContinuation(runs []ReviewRun, prior ReviewRun) *ReviewRun {
	for i := range runs {
		if runs[i].continues() == prior.RunID {
			return &runs[i]
		}
	}
	return nil
}

// taskBudgetPolicy is the policy view the attempt's continuation budget reads.
func taskBudgetPolicy(policy DelegationPolicy) ReviewersPolicy {
	view := ReviewersPolicy{Adapters: map[string]ReviewerAdapter{}}
	for name := range policy.Agents {
		view.Adapters[name], _ = policy.agentAdapter(name, DelegationPurposeTask)
	}
	return view
}

// planTaskContinuation is what a resume and a handoff of a task share: the
// predecessor, its existing continuation and the attempt's budget.
func (s Store) planTaskContinuation(id string) (DelegationPolicy, []ReviewRun, *ReviewRun, *ReviewRun, time.Duration, error) {
	policy, _, found, err := s.LoadDelegationPolicy()
	if err != nil || !found {
		if err == nil {
			err = ErrNoReviewerAdapter
		}
		return policy, nil, nil, nil, 0, err
	}
	runs, _ := s.ReadDelegationTaskRuns()
	var prior *ReviewRun
	for i := range runs {
		if runs[i].RunID == id {
			prior = &runs[i]
		}
	}
	if prior == nil {
		return policy, nil, nil, nil, 0, fmt.Errorf("pose: no task run %s", id)
	}
	if existing := taskContinuation(runs, *prior); existing != nil {
		return policy, runs, prior, existing, 0, nil
	}
	count, spent, limit, total, err := attemptContinuationBudget(runs, *prior, taskBudgetPolicy(policy))
	if err != nil {
		return policy, nil, nil, nil, 0, err
	}
	if count >= limit || spent >= total {
		return policy, nil, nil, nil, 0, refusal(DelegationReasonBudgetExhausted, fmt.Errorf("pose: the attempt's continuation budget is spent: %d continuations, %s wall-clock", count, spent))
	}
	return policy, runs, prior, nil, total - spent, nil
}

// PlanDelegationTaskResume continues task run id with its own agent and
// session; an empty prompt sends the default continuation message (R8).
func (s Store) PlanDelegationTaskResume(id, prompt string) (DelegationTaskPlan, error) {
	policy, _, prior, existing, left, err := s.planTaskContinuation(id)
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	if existing != nil {
		if existing.HandsOffFrom == prior.RunID {
			return DelegationTaskPlan{}, fmt.Errorf("pose: run %s was handed off to run %s; it is not resumed", prior.RunID, existing.RunID)
		}
		return DelegationTaskPlan{Existing: existing}, nil
	}
	if prior.Status != ReviewRunInterrupted || prior.Interruption == nil || !prior.Interruption.Resumable || prior.SessionDigest == "" {
		return DelegationTaskPlan{}, fmt.Errorf("pose: run %s did not stop resumably", prior.RunID)
	}
	agent, ok := policy.Agents[prior.Adapter]
	if !ok || agent.Resume == nil {
		return DelegationTaskPlan{}, fmt.Errorf("pose: agent %s declares no resume", prior.Adapter)
	}
	template := taskResumeTemplate(agent, prior.Access)
	if len(template) == 0 {
		return DelegationTaskPlan{}, fmt.Errorf("pose: agent %s declares no resume command for --access %s", prior.Adapter, prior.Access)
	}
	session, err := s.readReviewSession(prior.SessionDigest)
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	if prompt == "" {
		prompt = DelegationTaskContinueMessage
	}
	plan, err := s.taskContinuationPlan(policy, *prior, prior.Adapter, template, prompt, left)
	plan.session = session
	return plan, err
}

// PlanDelegationTaskHandoff continues task run id with another agent, its
// prompt the predecessor's followed by the handoff section (R8; spec
// pose-delegation-handoff).
func (s Store) PlanDelegationTaskHandoff(id, agentName string) (DelegationTaskPlan, error) {
	policy, _, prior, existing, left, err := s.planTaskContinuation(id)
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	if existing != nil {
		if existing.HandsOffFrom != prior.RunID || existing.Adapter != agentName {
			return DelegationTaskPlan{}, fmt.Errorf("pose: run %s was already continued by run %s", prior.RunID, existing.RunID)
		}
		return DelegationTaskPlan{Existing: existing}, nil
	}
	if why := handoffable(*prior); why != "" {
		return DelegationTaskPlan{}, fmt.Errorf("pose: run %s cannot be handed off: %s", prior.RunID, why)
	}
	if prior.Adapter == agentName {
		return DelegationTaskPlan{}, fmt.Errorf("pose: run %s already ran agent %s; resume it to continue the same agent", prior.RunID, agentName)
	}
	agent, ok := policy.Agents[agentName]
	if !ok || len(agent.Commands[prior.Access]) == 0 {
		return DelegationTaskPlan{}, fmt.Errorf("pose: no agent %q can run a task with --access %s", agentName, prior.Access)
	}
	original, err := s.ReadDelegationTaskPrompt(*prior)
	if err != nil {
		return DelegationTaskPlan{}, err
	}
	events, eventsErr := s.ReadDelegationEvents(*prior)
	section := renderHandoffSection(*prior, events, eventsErr)
	plan, err := s.taskContinuationPlan(policy, *prior, agentName, agent.Commands[prior.Access], original+section, left)
	plan.handoff, plan.handoffSection = true, section
	return plan, err
}

func (s Store) taskContinuationPlan(policy DelegationPolicy, prior ReviewRun, agentName string, template []string, prompt string, left time.Duration) (DelegationTaskPlan, error) {
	if err := validateTaskPrompt(prompt); err != nil {
		return DelegationTaskPlan{}, err
	}
	agent := policy.Agents[agentName]
	req := DelegationTaskRequest{Agent: agentName, Access: prior.Access, Dir: prior.Dir, Prompt: prompt, AddDirs: prior.AddDirs, NoNetwork: prior.NoNetwork, Record: prior.Recorded}
	if prior.Adapter == agentName {
		// A resume keeps the effort the run was started with (F1); another
		// agent declares its own levels.
		req.Effort = prior.Effort
	}
	plan := DelegationTaskPlan{DelegationTaskRequest: req, Config: agent, Template: template, envUnset: policy.EnvUnset, predecessor: &prior, processTimeout: left}
	if err := plan.checkFlags(); err != nil {
		return DelegationTaskPlan{}, err
	}
	plan.adapter, _ = policy.agentAdapter(agentName, DelegationPurposeTask)
	plan.adapter.Command = template
	plan.OutsideGit = GitNotARepository(prior.Dir)
	if err := plan.checkOutsideGit(); err != nil {
		return DelegationTaskPlan{}, err
	}
	if prior.Access == DelegationAccessFull {
		plan.Warnings = append(plan.Warnings, DelegationAccessWarning)
	}
	return plan, nil
}
