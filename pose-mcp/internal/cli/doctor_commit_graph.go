package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// commitGraphHistoryThreshold is the history size from which a last-commit
// lookup by path is worth accelerating: below it a history walk costs a few
// milliseconds (spec pose-doctor-recommends-the-commit-graph).
var commitGraphHistoryThreshold = 500

// errCommitGraphNotApplicable marks a root the check does not apply to: not
// a Git repository, an unborn HEAD, a shallow history (Git neither reads nor
// writes a commit-graph across a shallow boundary), or a HEAD with no file to
// look up.
var errCommitGraphNotApplicable = errors.New("not a repository whose last-commit lookups a commit-graph can answer")

// commitGraphState is what doctor knows about a repository's commit-graph.
type commitGraphState struct {
	commits int
	// filtersUsed is Git's own answer: a last-commit lookup by path read
	// changed-path Bloom filters from the commit-graph it loaded.
	filtersUsed bool
	// readsDisabledBy names the configuration that keeps Git from reading
	// the filters, when the lookup used none.
	readsDisabledBy string
}

// readCommitGraphState counts HEAD's history and asks Git whether a
// last-commit lookup by path uses changed-path Bloom filters. Git reports the
// filters it consulted in its trace2 "bloom" statistics, so whichever graph it
// loads (the object directory's or an alternate's, a single file or a chain,
// of any version it accepts) is judged as Git itself uses it. It only reads,
// and never fetches: lazy fetching is off, so a partial clone reports the
// objects it lacks instead of contacting its remote.
func readCommitGraphState(root string) (commitGraphState, error) {
	if _, _, err := commitGraphGit(root, nil, "rev-parse", "--git-dir"); err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return commitGraphState{}, errCommitGraphNotApplicable
		}
		return commitGraphState{}, fmt.Errorf("repository could not be opened: %w", err)
	}
	// An unborn HEAD names a branch that does not resolve; a HEAD that
	// resolves to a commit Git cannot read is a failure.
	if _, _, err := commitGraphGit(root, nil, "rev-parse", "--verify", "--quiet", "HEAD"); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return commitGraphState{}, errCommitGraphNotApplicable
		}
		return commitGraphState{}, fmt.Errorf("HEAD could not be read: %w", err)
	}
	if _, _, err := commitGraphGit(root, nil, "rev-parse", "--verify", "HEAD^{commit}"); err != nil {
		return commitGraphState{}, fmt.Errorf("HEAD could not be read: %w", err)
	}
	shallow, _, err := commitGraphGit(root, nil, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return commitGraphState{}, fmt.Errorf("repository depth could not be read: %w", err)
	}
	if strings.TrimSpace(shallow) == "true" {
		return commitGraphState{}, errCommitGraphNotApplicable
	}
	count, _, err := commitGraphGit(root, nil, "rev-list", "--count", "HEAD")
	if err != nil {
		return commitGraphState{}, fmt.Errorf("history of HEAD could not be counted: %w", err)
	}
	commits, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil {
		return commitGraphState{}, fmt.Errorf("history of HEAD could not be counted: %w", err)
	}
	state := commitGraphState{commits: commits}
	if commits < commitGraphHistoryThreshold {
		return state, nil
	}
	listing, _, err := commitGraphGit(root, nil, "ls-tree", "-r", "--name-only", "-z", "HEAD")
	if err != nil {
		return state, fmt.Errorf("files of HEAD could not be listed: %w", err)
	}
	path, _, _ := strings.Cut(listing, "\x00")
	if path == "" {
		return commitGraphState{}, errCommitGraphNotApplicable
	}
	trace, err := os.CreateTemp("", "pose-doctor-trace2-*.json")
	if err != nil {
		return state, fmt.Errorf("Git trace could not be created: %w", err)
	}
	tracePath := trace.Name()
	trace.Close()
	defer os.Remove(tracePath)
	// The path is literal: a name with pathspec magic must not change what
	// Git looks up.
	_, stderr, err := commitGraphGit(root, []string{"GIT_TRACE2_EVENT=" + tracePath, "GIT_LITERAL_PATHSPECS=1"}, "log", "-1", "--format=%H", "--", path)
	if err != nil {
		return state, fmt.Errorf("last commit of a file could not be read: %w", err)
	}
	// Git falls back to a plain walk when its commit-graph cannot be read,
	// and says so on stderr while it succeeds.
	if stderr != "" {
		return state, fmt.Errorf("Git reported a problem while reading history: %s", stderr)
	}
	raw, err := os.ReadFile(tracePath)
	if err != nil {
		return state, fmt.Errorf("Git trace could not be read: %w", err)
	}
	state.filtersUsed = traceUsedBloomFilters(raw)
	if !state.filtersUsed {
		state.readsDisabledBy = commitGraphReadsDisabledBy(root)
	}
	return state, nil
}

// traceUsedBloomFilters reports whether a trace2 event stream carries a
// "bloom" statistics event in which some filter answered a lookup.
func traceUsedBloomFilters(raw []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for scanner.Scan() {
		var event struct {
			Category string `json:"category"`
			Key      string `json:"key"`
			Value    struct {
				Maybe         int `json:"maybe"`
				DefinitelyNot int `json:"definitely_not"`
			} `json:"value"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		if event.Category == "bloom" && event.Key == "statistics" && event.Value.Maybe+event.Value.DefinitelyNot > 0 {
			return true
		}
	}
	return false
}

// commitGraphReadsDisabledBy names the configuration that keeps Git from
// reading a commit-graph or its changed-path filters, or "" when none does:
// with it set, writing the graph cannot change the answer. Precedence follows
// Git: commitGraph.changedPathsVersion, when set, overrides the deprecated
// commitGraph.readChangedPaths.
func commitGraphReadsDisabledBy(root string) string {
	get := func(kind, key string) (string, bool) {
		out, _, err := commitGraphGit(root, nil, "config", "--type="+kind, "--get", key)
		value := strings.TrimSpace(out)
		return value, err == nil && value != ""
	}
	if value, ok := get("bool", "core.commitGraph"); ok && value == "false" {
		return "core.commitGraph=false"
	}
	if value, ok := get("int", "commitGraph.changedPathsVersion"); ok {
		if value == "0" {
			return "commitGraph.changedPathsVersion=0"
		}
		return ""
	}
	if value, ok := get("bool", "commitGraph.readChangedPaths"); ok && value == "false" {
		return "commitGraph.readChangedPaths=false"
	}
	return ""
}

// writeCommitGraph writes the repository's commit-graph with changed-path
// Bloom filters for every commit, whatever commitGraph.maxNewFilters says;
// it only writes under the Git object directory.
func writeCommitGraph(root string) error {
	_, _, err := commitGraphGit(root, nil, "commit-graph", "write", "--reachable", "--changed-paths", "--max-new-filters=-1")
	return err
}

// commitGraphGit runs Git in root, without lazy fetching and in the C
// locale, with extra environment; it returns stdout and stderr, and a
// failure carries the stderr, bounded.
func commitGraphGit(root string, env []string, args ...string) (string, string, error) {
	cmd := posemodel.RawHistoryGit(root, args...)
	// Git's messages are read below; they are read in the C locale. A
	// diagnostic never contacts a remote.
	cmd.Env = append(append(cmd.Env, "LC_ALL=C", "LANGUAGE=C", "GIT_NO_LAZY_FETCH=1"), env...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	detail := strings.TrimSpace(stderr.String())
	if len(detail) > 200 {
		detail = detail[:200] + "…"
	}
	if err != nil {
		return string(out), detail, &commitGraphGitError{args: args[0], err: err, detail: detail}
	}
	return string(out), detail, nil
}

// commitGraphGitError is a failed Git command with its stderr, bounded; it
// unwraps to the command's error, an *exec.ExitError when Git ran.
type commitGraphGitError struct {
	args   string
	err    error
	detail string
}

func (e *commitGraphGitError) Error() string {
	if e.detail == "" {
		return "git " + e.args + " " + e.err.Error()
	}
	return "git " + e.args + ": " + e.err.Error() + " " + e.detail
}

func (e *commitGraphGitError) Unwrap() error { return e.err }
