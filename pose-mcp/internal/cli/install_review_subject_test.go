package cli

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	posemodel "github.com/harne8/pose-mcp/internal/pose"
)

// Spec pose-review-subject-classifies-engine-records: the source scan reads
// string patterns, so it is checked against what the engine actually writes.
// Every file a fresh install and an update leave under .pose/ has a
// review-subject class, or a spec whose change set carries it cannot be
// sealed (found in review: .pose/LICENSE and .pose/NOTICE).
func TestEveryFileTheEngineWritesUnderPoseIsClassified(t *testing.T) {
	for name, repo := range map[string]string{"install": freshInstall(t), "update": func() string { r := olderInstance(t); updateOnce(t, r); return r }()} {
		var unclassified []string
		_ = filepath.WalkDir(filepath.Join(repo, ".pose"), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(repo, path)
			rel = filepath.ToSlash(rel)
			if posemodel.ReviewSubjectPathClass(rel) == "" {
				unclassified = append(unclassified, rel)
			}
			return nil
		})
		sort.Strings(unclassified)
		if len(unclassified) > 0 {
			t.Errorf("%s leaves unclassified files under .pose/:\n%s", name, strings.Join(unclassified, "\n"))
		}
	}
}
