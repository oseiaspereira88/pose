package pose

import (
	"context"
	"regexp"
	"strings"
)

// A read-view operation reruns every Git command it captured when it ends, to prove the
// output did not change while the operation ran. Some commands cannot change: their output is
// a pure function of object ids, or of the commit HEAD named when they ran. For those the
// rerun proves nothing the commit does not already prove, and it is the dearest part of a
// check (spec pose-governance-audit-skips-pure-git-reads).

var governanceFullObjectID = regexp.MustCompile(`^[0-9a-f]{40}$`)

// governancePureGitRead classifies a captured command. byObject means the output depends on
// object ids alone; byHead means it depends on the commit HEAD named when it ran, which the
// capture reads before and after the command. Anything not listed exactly is neither, and
// keeps its rerun: a ref name, --all, a pretty format that names refs, or a flag the
// classification does not know can all make the output depend on mutable state.
func governancePureGitRead(args []string) (byObject, byHead bool) {
	if len(args) == 0 {
		return false, false
	}
	switch args[0] {
	case "show":
		// show <40-hex>:<path>
		if len(args) == 2 {
			id, path, ok := strings.Cut(args[1], ":")
			return ok && governanceFullObjectID.MatchString(id) && path != "", false
		}
	case "cat-file":
		// cat-file blob <40-hex>
		return len(args) == 3 && args[1] == "blob" && governanceFullObjectID.MatchString(args[2]), false
	case "log":
		// log -1 --format=%H -- <path...>: the last commit that changed the paths, from HEAD.
		if len(args) >= 5 && args[1] == "-1" && args[2] == "--format=%H" && args[3] == "--" {
			return false, true
		}
	case "ls-tree":
		// ls-tree [flags] <40-hex> [-- <path...>]
		i := 1
		for i < len(args) && strings.HasPrefix(args[i], "-") && args[i] != "--" {
			switch args[i] {
			case "-z", "-r", "-t", "-l", "--name-only", "--full-tree":
			default:
				return false, false
			}
			i++
		}
		if i >= len(args) {
			return false, false
		}
		tree := args[i]
		i++
		if i < len(args) && args[i] != "--" {
			return false, false
		}
		// ls-tree HEAD is cheap to rerun and dearer to qualify (two HEAD reads): it keeps its rerun.
		return governanceFullObjectID.MatchString(tree), false
	}
	return false, false
}

// governanceAuditGit runs a captured command again for the final audit; tests replace it to
// count the reruns.
var governanceAuditGit = runGovernanceGitCommandOutputContext

// governanceHeadAt reads HEAD of a root directly, outside any read view, so that no
// observation is recorded and the answer is the repository's now.
func governanceHeadAt(ctx context.Context, files Files, root string) string {
	stdout, _, err := runGovernanceGitCommandOutputContext(ctx, files, root, []string{"rev-parse", "--verify", "HEAD"}, 4096)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(stdout))
}
