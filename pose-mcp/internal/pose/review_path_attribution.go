package pose

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ReviewBundlePathAttribution records what this scope's own commits did to one
// subject path.
//
// A subject's Base and Head span every commit between the first and the last
// attributed one, so a diff between them also carries what other specs, and
// releases, did to the same path in between. The structural assessment of a
// field pilot found that this is where its false positives came from: a release
// bump, an ADR or a matrix reconciliation landed in the range and was reported
// as this scope's structure. Each segment here is a run of attributed commits
// whose content chains without interruption, so comparing its endpoints shows
// what the scope did and nothing else.
//
// Segments name blobs, not commits. A blob id is content identity: a rebase or a
// squash that keeps the content keeps the ids, so the structural input digest
// still excludes provider refs, as its ADR requires.
type ReviewBundlePathAttribution struct {
	Path     string                    `json:"path"`
	Segments []ReviewBundleBlobSegment `json:"segments"`
}

// ReviewBundleBlobSegment is one uninterrupted run of attributed commits on a
// path. An empty blob means the path was absent on that side.
type ReviewBundleBlobSegment struct {
	BeforeMode string `json:"before_mode,omitempty"`
	Before     string `json:"before,omitempty"`
	AfterMode  string `json:"after_mode,omitempty"`
	After      string `json:"after,omitempty"`
	// SharedWith names the other specs whose POSE-Spec trailer is on a commit
	// of this run. Such a commit is attributed to this scope and to them at
	// once, so its content cannot be split by commit: the reviewer is asked to
	// confirm the fact belongs here instead.
	SharedWith []string `json:"shared_with,omitempty"`
}

const (
	reviewAttributionMaxCommits = 5000
	reviewAttributionMaxBytes   = 64 << 20
)

type attributedChange struct {
	order              int
	beforeMode, before string
	afterMode, after   string
	sharedWith         []string
}

// reviewBundleAttribution reads, in one bounded Git call, what each attributed
// commit did to the subject paths. It returns nil when Git cannot answer; the
// structural assessment then compares Base and Head as before, and the range
// observation already says whether that range is contaminated.
func reviewBundleAttribution(root string, sets []ChangeSet, allowed map[string]bool, entries []ReviewBundleSubjectEntry) []ReviewBundlePathAttribution {
	wanted := map[string]bool{}
	for _, entry := range entries {
		// A rename joins two paths; a commit-level view would have to follow it
		// across runs. Renames keep the range comparison, and their path fact.
		if entry.Action == "renamed" || entry.Path == "" {
			continue
		}
		wanted[entry.Path] = true
	}
	commits := []string{}
	seen := map[string]bool{}
	for _, set := range sets {
		for _, commit := range set.Commits {
			if !seen[commit] && designDeltaRevisionRE.MatchString(commit) {
				seen[commit] = true
				commits = append(commits, commit)
			}
		}
	}
	if len(wanted) == 0 || len(commits) == 0 || len(commits) > reviewAttributionMaxCommits {
		return nil
	}
	raw, err := gitAttributionLog(root, commits)
	if err != nil {
		return nil
	}
	changes, ok := parseAttributionLog(raw, wanted, allowed)
	if !ok {
		return nil
	}
	paths := make([]string, 0, len(changes))
	for path := range changes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	result := []ReviewBundlePathAttribution{}
	for _, path := range paths {
		list := changes[path]
		sort.SliceStable(list, func(i, j int) bool { return list[i].order < list[j].order })
		segments := []ReviewBundleBlobSegment{}
		for _, change := range list {
			last := len(segments) - 1
			if last >= 0 && segments[last].After == change.before && segments[last].AfterMode == change.beforeMode {
				segments[last].After, segments[last].AfterMode = change.after, change.afterMode
				segments[last].SharedWith = uniqueSorted(append(segments[last].SharedWith, change.sharedWith...))
				continue
			}
			segments = append(segments, ReviewBundleBlobSegment{
				BeforeMode: change.beforeMode, Before: change.before,
				AfterMode: change.afterMode, After: change.after,
				SharedWith: uniqueSorted(change.sharedWith),
			})
		}
		for i := range segments {
			if len(segments[i].SharedWith) == 0 {
				segments[i].SharedWith = nil
			}
		}
		result = append(result, ReviewBundlePathAttribution{Path: path, Segments: segments})
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func gitAttributionLog(root string, commits []string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	// --no-walk shows exactly the given commits; --no-renames splits a rename
	// into a removal and an addition so each path's blobs stay on that path.
	cmd := exec.CommandContext(ctx, "git", "-C", root, "-c", "core.quotePath=false", "log", "--no-walk", "--stdin",
		"--raw", "--no-abbrev", "--no-renames", "--no-color", "--format=%x01%H %ct%n%B%x02")
	cmd.Stdin = strings.NewReader(strings.Join(commits, "\n") + "\n")
	var out bytes.Buffer
	cmd.Stdout = &limitedWriter{buf: &out, limit: reviewAttributionMaxBytes}
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

type limitedWriter struct {
	buf   *bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.buf.Len()+len(p) > w.limit {
		return 0, fmt.Errorf("attribution log exceeds %d bytes", w.limit)
	}
	return w.buf.Write(p)
}

// parseAttributionLog keeps, for each wanted path, the blob change of every
// commit that touched it, ordered by commit time. Commits that touch no wanted
// path contribute nothing. ok is false when the output is not what was asked.
func parseAttributionLog(raw []byte, wanted, allowed map[string]bool) (map[string][]attributedChange, bool) {
	type commitChange struct {
		at     int64
		index  int
		shared []string
		lines  []string
	}
	parsed := []commitChange{}
	for index, chunk := range strings.Split(string(raw), "\x01") {
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		bodyEnd := strings.Index(chunk, "\x02")
		newline := strings.Index(chunk, "\n")
		if bodyEnd < 0 || newline < 0 || newline > bodyEnd {
			return nil, false
		}
		header := strings.Fields(chunk[:newline])
		if len(header) != 2 {
			return nil, false
		}
		at, err := strconv.ParseInt(header[1], 10, 64)
		if err != nil {
			return nil, false
		}
		shared := []string{}
		for _, line := range strings.Split(chunk[newline+1:bodyEnd], "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "POSE-Spec:") {
				continue
			}
			if spec := strings.TrimSpace(strings.TrimPrefix(line, "POSE-Spec:")); spec != "" && !allowed[spec] {
				shared = append(shared, spec)
			}
		}
		lines := []string{}
		for _, line := range strings.Split(chunk[bodyEnd+1:], "\n") {
			if strings.HasPrefix(line, ":") {
				lines = append(lines, line)
			}
		}
		parsed = append(parsed, commitChange{at: at, index: index, shared: shared, lines: lines})
	}
	sort.SliceStable(parsed, func(i, j int) bool {
		if parsed[i].at != parsed[j].at {
			return parsed[i].at < parsed[j].at
		}
		return parsed[i].index < parsed[j].index
	})
	changes := map[string][]attributedChange{}
	for order, commit := range parsed {
		for _, line := range commit.lines {
			// :<old mode> <new mode> <old oid> <new oid> <status>\t<path>
			tab := strings.Index(line, "\t")
			if tab < 0 {
				return nil, false
			}
			fields := strings.Fields(line[1:tab])
			path := line[tab+1:]
			if len(fields) != 5 {
				return nil, false
			}
			if !wanted[path] {
				continue
			}
			change := attributedChange{order: order, sharedWith: commit.shared}
			if !gitOIDIsZero(fields[2]) {
				change.beforeMode, change.before = fields[0], fields[2]
			}
			if !gitOIDIsZero(fields[3]) {
				change.afterMode, change.after = fields[1], fields[3]
			}
			changes[path] = append(changes[path], change)
		}
	}
	return changes, true
}

// gitOIDIsZero holds for SHA-1 and SHA-256 repositories alike.
func gitOIDIsZero(oid string) bool { return strings.Trim(oid, "0") == "" }
