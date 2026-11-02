package pose

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
)

// Spec pose-spec-facts-evidence-binding: the facts and the attribution of change sets read the
// trailers of the history in one way.

// SpecTrailerLogArgs are the arguments of the `git log` whose output ParseSpecTrailerLog reads:
// every ref, oldest first, the commits that mention a POSE-Spec trailer, hash and full body.
var SpecTrailerLogArgs = []string{"log", "--all", "--reverse", "--grep=POSE-Spec:", "--format=%H%x00%B%x00"}

// ParseSpecTrailerLog maps each spec slug to the commits that carry its trailer. A trailer is a
// line of the body that starts with `POSE-Spec:` after trimming; the slug is the rest of the
// line, trimmed. It does not use Git's trailer parser.
func ParseSpecTrailerLog(out []byte) map[string][]string {
	parts := bytes.Split(out, []byte{0})
	specCommits := map[string][]string{}
	for i := 0; i+1 < len(parts); i += 2 {
		sha, body := strings.TrimSpace(string(parts[i])), string(parts[i+1])
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "POSE-Spec:") {
				spec := strings.TrimSpace(strings.TrimPrefix(line, "POSE-Spec:"))
				if spec != "" {
					specCommits[spec] = append(specCommits[spec], sha)
				}
			}
		}
	}
	return specCommits
}

// RawHistoryGit is a git command on root's raw object history: replace refs
// (--no-replace-objects) and grafts (GIT_GRAFT_FILE at the always-empty null
// device, which overrides both the repository's graft file and one the
// environment names) never change which commits carry a trailer, what a change
// set holds, or what a later removal is judged on (spec
// pose-delivery-later-removal R3, review round 4 LR-R4-01).
func RawHistoryGit(root string, args ...string) *exec.Cmd {
	return gitCommand(append([]string{"--no-replace-objects", "-C", root}, args...)...)
}

// RawHistoryGitContext is RawHistoryGit under ctx.
func RawHistoryGitContext(ctx context.Context, root string, args ...string) *exec.Cmd {
	return gitCommandContext(ctx, append([]string{"--no-replace-objects", "-C", root}, args...)...)
}

// RawGit is RawHistoryGit for a caller whose arguments already say where to read.
func RawGit(args ...string) *exec.Cmd {
	return gitCommand(append([]string{"--no-replace-objects"}, args...)...)
}

// SpecTrailerCommits reads the history of root and returns the commits that carry the trailer of
// slug, the way the attribution of change sets does. An error is a history that could not be
// read, which is not a history without commits.
func SpecTrailerCommits(root, slug string) ([]string, error) {
	out, err := RawHistoryGit(root, SpecTrailerLogArgs...).Output()
	if err != nil {
		return nil, err
	}
	return ParseSpecTrailerLog(out)[slug], nil
}
