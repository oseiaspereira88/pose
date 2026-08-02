package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

func cmdReviewCheck(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, ok := parseScopeCheckArgs("review-check", args, stderr)
	if !ok {
		return 2
	}
	eval, err := (posemodel.Store{Root: root}).ReviewCheck(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review-check: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, eval); code != 0 {
			return code
		}
		if eval.Required && !eval.Approved {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "review.scope=%s\nreview.required=%t\nreview.profile=%s\nreview.digest=%s\nreview.fresh=%t\nreview.approved=%t\n", eval.Scope, eval.Required, eval.Profile, eval.ScopeDigest, eval.Fresh, eval.Approved)
	for _, warning := range eval.Warnings {
		fmt.Fprintf(stdout, "[WARN] %s\n", warning)
	}
	for _, blocker := range eval.Blockers {
		fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
	}
	if eval.Required && !eval.Approved {
		return 1
	}
	return 0
}

func cmdCloseoutCheck(root string, args []string, stdout, stderr io.Writer) int {
	ref, jsonOutput, ok := parseScopeCheckArgs("closeout-check", args, stderr)
	if !ok {
		return 2
	}
	state, err := (posemodel.Store{Root: root}).GetCloseoutState(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose closeout-check: %v\n", err)
		return 1
	}
	if jsonOutput {
		if code := writeJSON(stdout, state); code != 0 {
			return code
		}
		if !state.Terminal {
			return 1
		}
		return 0
	}
	fmt.Fprintf(stdout, "closeout.scope=%s\ncloseout.digest=%s\ncloseout.lifecycle_done=%t\ncloseout.review_approved=%t\ncloseout.terminal=%t\ncloseout.next_action=%s\n", state.Scope, state.ScopeDigest, state.LifecycleDone, state.Review.Approved, state.Terminal, state.NextAction)
	for _, blocker := range state.Blockers {
		fmt.Fprintf(stderr, "[ERROR] %s\n", blocker)
	}
	if !state.Terminal {
		return 1
	}
	return 0
}

func parseScopeCheckArgs(command string, args []string, stderr io.Writer) (string, bool, bool) {
	jsonOutput := false
	ref := ""
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(arg, "-") || ref != "" {
				fmt.Fprintf(stderr, "Usage: pose %s <spec:slug|milestone:roadmap/id|roadmap:slug> [--json]\n", command)
				return "", false, false
			}
			ref = arg
		}
	}
	if ref == "" {
		fmt.Fprintf(stderr, "Usage: pose %s <spec:slug|milestone:roadmap/id|roadmap:slug> [--json]\n", command)
		return "", false, false
	}
	return ref, jsonOutput, true
}

func writeJSON(w io.Writer, value any) int {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return 1
	}
	return 0
}

func cmdReview(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "record" {
		fmt.Fprintln(stderr, "Usage: pose review record <scope> --reviewer <execution-id> --decision <approved|approved-with-reservations|changes-requested|rejected> --evidence <ref> [--finding 'ID|severity|disposition|action|evidence'] [--apply]")
		return 2
	}
	return cmdReviewRecord(root, args[1:], stdout, stderr)
}

func cmdReviewRecord(root string, args []string, stdout, stderr io.Writer) int {
	var ref, reviewer, decision string
	var evidence, findings []string
	apply := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--reviewer", "--decision", "--evidence", "--finding":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "pose review record: missing option value")
				return 2
			}
			i++
			switch args[i-1] {
			case "--reviewer":
				reviewer = args[i]
			case "--decision":
				decision = args[i]
			case "--evidence":
				evidence = append(evidence, args[i])
			case "--finding":
				findings = append(findings, args[i])
			}
		case "--apply":
			apply = true
		default:
			if strings.HasPrefix(args[i], "-") || ref != "" {
				fmt.Fprintf(stderr, "pose review record: unexpected argument %q\n", args[i])
				return 2
			}
			ref = args[i]
		}
	}
	if ref == "" || reviewer == "" || decision == "" || len(evidence) == 0 {
		fmt.Fprintln(stderr, "pose review record: scope, reviewer, decision and at least one evidence ref are required")
		return 2
	}
	allowedDecision := map[string]bool{"approved": true, "approved-with-reservations": true, "changes-requested": true, "rejected": true}
	if !allowedDecision[decision] || strings.ContainsAny(reviewer, "\r\n") {
		fmt.Fprintln(stderr, "pose review record: invalid decision or reviewer")
		return 2
	}
	store := posemodel.Store{Root: root}
	profile, err := store.ReviewProfileForScope(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	digest, err := store.ScopeDigest(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	attempts, err := store.ListReviewAttempts(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	supersedes := ""
	if len(attempts) > 0 {
		supersedes = attempts[len(attempts)-1].ReviewID
	}
	now := time.Now().UTC().Truncate(time.Second)
	sum := sha256.Sum256([]byte(ref + digest + reviewer + now.Format(time.RFC3339)))
	reviewID := "rvw-" + now.Format("20060102T150405Z") + "-" + hex.EncodeToString(sum[:4])
	sort.Strings(evidence)
	content, err := renderReviewAttempt(reviewID, ref, digest, profile, reviewer, decision, now.Format(time.RFC3339), supersedes, evidence, findings)
	if err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 2
	}
	path := filepath.Join(root, ".pose", "reviews", reviewID+".md")
	if !apply {
		fmt.Fprintf(stdout, "review.plan=record\nreview.id=%s\nreview.scope=%s\nreview.digest=%s\nreview.path=%s\nreview.apply=false\n", reviewID, ref, digest, filepath.ToSlash(strings.TrimPrefix(path, root+string(os.PathSeparator))))
		return 0
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	if err := writeFileExclusive(path, []byte(content)); err != nil {
		fmt.Fprintf(stderr, "pose review record: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Review recorded: %s\n", path)
	return 0
}

func renderReviewAttempt(id, scope, digest string, profile posemodel.ReviewProfile, reviewer, decision, reviewedAt, supersedes string, evidence, findings []string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nschema_version: 1\nreview_id: %s\nscope: %s\nscope_digest: %s\nprofile: %s\nreviewer: %s\ndecision: %s\nreviewed_at: %s\n", id, scope, digest, profile.Ref(), reviewer, decision, reviewedAt)
	if supersedes == "" {
		b.WriteString("supersedes:\n")
	} else {
		fmt.Fprintf(&b, "supersedes: %s\n", supersedes)
	}
	fmt.Fprintf(&b, "evidence_refs: [%s]\n---\n\n## Criteria\n", strings.Join(evidence, ", "))
	for _, criterion := range profile.Criteria {
		fmt.Fprintf(&b, "- %s [passed] evidence:%s\n", criterion.ID, evidence[0])
	}
	b.WriteString("\n## Findings\n")
	for _, raw := range findings {
		parts := strings.Split(raw, "|")
		if len(parts) < 5 || posemodel.ValidateSlug(parts[0]) != nil {
			return "", fmt.Errorf("finding must be ID|severity|disposition|action|evidence")
		}
		fmt.Fprintf(&b, "- %s [%s] severity:%s action:%s evidence:%s", parts[0], parts[2], parts[1], strings.ReplaceAll(parts[3], " ", "_"), strings.ReplaceAll(parts[4], " ", "_"))
		if len(parts) > 5 && parts[5] != "" {
			fmt.Fprintf(&b, " owner:%s", parts[5])
		}
		if len(parts) > 6 && parts[6] != "" {
			fmt.Fprintf(&b, " rationale:%s", strings.ReplaceAll(parts[6], " ", "_"))
		}
		if len(parts) > 7 && parts[7] != "" {
			fmt.Fprintf(&b, " review_by:%s", parts[7])
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func writeFileExclusive(path string, content []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func cmdClose(root string, args []string, stdout, stderr io.Writer) int {
	ref, _, ok := parseScopeCheckArgs("close", args, stderr)
	if !ok {
		return 2
	}
	store := posemodel.Store{Root: root}
	state, err := store.GetCloseoutState(ref)
	if err != nil {
		fmt.Fprintf(stderr, "pose close: %v\n", err)
		return 1
	}
	if !state.Review.Approved || len(state.Children) > 0 && hasOpenChild(state.Children) {
		fmt.Fprintf(stderr, "pose close: scope is not eligible; next action: %s\n", state.NextAction)
		return 1
	}
	scope, _ := posemodel.ParseScopeRef(ref)
	if scope.Kind == "milestone" {
		fmt.Fprintf(stdout, "Milestone closeout verified: %s\n", ref)
		return 0
	}
	var path string
	if scope.Kind == "spec" {
		sp, err := store.GetSpec(scope.Slug)
		if err != nil {
			fmt.Fprintf(stderr, "pose close: %v\n", err)
			return 1
		}
		path = sp.Path
	} else {
		path = filepath.Join(root, ".pose", "roadmaps", scope.Slug+".md")
	}
	if err := applyLifecycleDone(path, scope.Kind == "spec"); err != nil {
		fmt.Fprintf(stderr, "pose close: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Closed: %s\n", ref)
	return 0
}

type continuousCloseoutSelection struct {
	SchemaVersion int    `json:"schema_version"`
	Scope         string `json:"scope"`
	StartedAt     string `json:"started_at"`
	Status        string `json:"status"`
}

func cmdContinuousCloseout(root string, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"status"}
	}
	path := filepath.Join(root, ".pose", "continuous-closeout.json")
	switch args[0] {
	case "start":
		if len(args) < 2 || len(args) > 3 || len(args) == 3 && args[2] != "--apply" {
			fmt.Fprintln(stderr, "Usage: pose continuous-closeout start <scope> [--apply]")
			return 2
		}
		ref := args[1]
		store := posemodel.Store{Root: root}
		policy, err := store.GetReviewPolicy()
		if err != nil || !policy.Enabled || !policy.ContinuousCloseout {
			fmt.Fprintln(stderr, "pose continuous-closeout: continuous mode is not enabled by review policy")
			return 1
		}
		if _, err := store.GetCloseoutState(ref); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		selection := continuousCloseoutSelection{SchemaVersion: 1, Scope: ref, StartedAt: time.Now().UTC().Truncate(time.Second).Format(time.RFC3339), Status: "active"}
		if len(args) == 2 {
			fmt.Fprintf(stdout, "continuous.scope=%s\ncontinuous.apply=false\n", ref)
			return 0
		}
		raw, _ := json.MarshalIndent(selection, "", "  ")
		raw = append(raw, '\n')
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		if _, err := os.Stat(path); err == nil {
			fmt.Fprintln(stderr, "pose continuous-closeout: an active terminal scope already exists")
			return 1
		}
		if err := writeFileExclusive(path, raw); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Continuous closeout started: %s\n", ref)
		return 0
	case "status":
		raw, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			fmt.Fprintln(stdout, "continuous.active=false")
			return 0
		}
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		var selection continuousCloseoutSelection
		if err := json.Unmarshal(raw, &selection); err != nil || selection.SchemaVersion != 1 || selection.Status != "active" {
			fmt.Fprintln(stderr, "pose continuous-closeout: invalid persisted selection")
			return 1
		}
		state, err := (posemodel.Store{Root: root}).GetCloseoutState(selection.Scope)
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "continuous.active=true\ncontinuous.scope=%s\ncontinuous.terminal=%t\ncontinuous.next_action=%s\n", selection.Scope, state.Terminal, state.NextAction)
		if !state.Terminal {
			return 1
		}
		return 0
	case "complete":
		if len(args) != 2 || args[1] != "--apply" {
			fmt.Fprintln(stderr, "Usage: pose continuous-closeout complete --apply")
			return 2
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		var selection continuousCloseoutSelection
		if json.Unmarshal(raw, &selection) != nil {
			fmt.Fprintln(stderr, "pose continuous-closeout: invalid persisted selection")
			return 1
		}
		state, err := (posemodel.Store{Root: root}).GetCloseoutState(selection.Scope)
		if err != nil || !state.Terminal {
			fmt.Fprintf(stderr, "pose continuous-closeout: terminal success is not satisfied; next action: %s\n", state.NextAction)
			return 1
		}
		if err := os.Remove(path); err != nil {
			fmt.Fprintf(stderr, "pose continuous-closeout: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Continuous closeout completed: %s\n", selection.Scope)
		return 0
	default:
		fmt.Fprintln(stderr, "Usage: pose continuous-closeout <start|status|complete> [...]")
		return 2
	}
}

func hasOpenChild(children []posemodel.CloseoutState) bool {
	for _, child := range children {
		if !child.Terminal {
			return true
		}
	}
	return false
}

func applyLifecycleDone(path string, spec bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(raw)
	if !strings.HasPrefix(text, "---\n") {
		return fmt.Errorf("artifact has no flat frontmatter")
	}
	text = replaceFrontmatterValue(text, "status", "done")
	if spec {
		text = replaceFrontmatterValue(text, "completed_at", time.Now().UTC().Format("2006-01-02"))
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pose-close-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.WriteString(text); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func replaceFrontmatterValue(content, key, value string) string {
	end := strings.Index(content[4:], "\n---")
	if end < 0 {
		return content
	}
	end += 4
	head, tail := content[:end], content[end:]
	lines := strings.Split(head, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(line, key+":") {
			lines[i] = key + ": " + value
			found = true
		}
	}
	if !found {
		lines = append(lines, key+": "+value)
	}
	return strings.Join(lines, "\n") + tail
}
