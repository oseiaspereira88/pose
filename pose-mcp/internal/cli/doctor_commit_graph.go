package cli

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// commitGraphHistoryThreshold is the history size from which a last-commit
// lookup by path is worth accelerating: below it a history walk costs a few
// milliseconds (spec pose-doctor-recommends-the-commit-graph).
var commitGraphHistoryThreshold = 500

// commitGraphState is what doctor knows about a repository's commit-graph.
type commitGraphState struct {
	commits      int
	present      bool
	changedPaths bool
}

// readCommitGraphState counts HEAD's history and reads the commit-graph of
// the repository at root: a single file or a chain of layers, each of which
// must carry the changed-path Bloom filters (chunks BIDX and BDAT). It only
// reads.
func readCommitGraphState(root string) (commitGraphState, error) {
	count, err := commitGraphGit(root, "rev-list", "--count", "HEAD")
	if err != nil {
		return commitGraphState{}, err
	}
	commits, err := strconv.Atoi(strings.TrimSpace(count))
	if err != nil {
		return commitGraphState{}, err
	}
	state := commitGraphState{commits: commits}
	single, err := commitGraphGit(root, "rev-parse", "--path-format=absolute", "--git-path", "objects/info/commit-graph")
	if err != nil {
		return state, err
	}
	chain, err := commitGraphGit(root, "rev-parse", "--path-format=absolute", "--git-path", "objects/info/commit-graphs/commit-graph-chain")
	if err != nil {
		return state, err
	}
	layers := []string{}
	if raw, err := os.ReadFile(strings.TrimSpace(chain)); err == nil {
		dir := filepath.Dir(strings.TrimSpace(chain))
		scanner := bufio.NewScanner(bytes.NewReader(raw))
		for scanner.Scan() {
			if hash := strings.TrimSpace(scanner.Text()); hash != "" {
				layers = append(layers, filepath.Join(dir, "graph-"+hash+".graph"))
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return state, err
	}
	if len(layers) == 0 {
		if _, err := os.Stat(strings.TrimSpace(single)); err == nil {
			layers = append(layers, strings.TrimSpace(single))
		} else if !errors.Is(err, os.ErrNotExist) {
			return state, err
		}
	}
	if len(layers) == 0 {
		return state, nil
	}
	state.present, state.changedPaths = true, true
	for _, layer := range layers {
		has, err := commitGraphLayerHasChangedPaths(layer)
		if err != nil || !has {
			state.changedPaths = false
			break
		}
	}
	return state, nil
}

// commitGraphLayerHasChangedPaths reads one commit-graph file's header and
// chunk table: signature CGPH, version, hash version, chunk count, base
// count, then one 12-byte entry (4-byte id, 8-byte offset) per chunk plus a
// terminating entry.
func commitGraphLayerHasChangedPaths(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	header := make([]byte, 8)
	if _, err := io.ReadFull(file, header); err != nil {
		return false, err
	}
	if string(header[:4]) != "CGPH" {
		return false, fmt.Errorf("%s is not a commit-graph", path)
	}
	chunks := int(header[6])
	table := make([]byte, 12*(chunks+1))
	if _, err := io.ReadFull(file, table); err != nil {
		return false, err
	}
	found := map[string]bool{}
	for i := 0; i < chunks; i++ {
		entry := table[12*i : 12*i+12]
		if binary.BigEndian.Uint64(entry[4:]) == 0 {
			return false, fmt.Errorf("%s has a chunk table entry with no offset", path)
		}
		found[string(entry[:4])] = true
	}
	return found["BIDX"] && found["BDAT"], nil
}

// writeCommitGraph writes the repository's commit-graph with changed-path
// Bloom filters; it only writes under the Git object directory.
func writeCommitGraph(root string) error {
	cmd := posemodel.RawHistoryGit(root, "commit-graph", "write", "--reachable", "--changed-paths")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit-graph write: %v %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func commitGraphGit(root string, args ...string) (string, error) {
	cmd := posemodel.RawHistoryGit(root, args...)
	out, err := cmd.Output()
	return string(out), err
}
