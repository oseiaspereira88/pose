package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemediationLineageCLI(t *testing.T) {
	for _, ready := range []bool{false, true} {
		for _, tc := range []struct {
			links string
			want  int
		}{
			{"", 0}, {"spec:original@defect-fix", 0}, {"spec:absent@defect-fix", 1}, {"spec:basis@revert", 1}, {"spec:original@invented", 1},
		} {
			t.Run(tc.links+map[bool]string{true: "/ready", false: "/lint"}[ready], func(t *testing.T) {
				root := newGitRepo(t)
				writeDesignBasisSpec(t, root, "Keep the existing behavior.")
				path := filepath.Join(root, ".pose/specs/basis/spec.md")
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				content := strings.Replace(string(raw), "slug: basis", "slug: basis\nremediates: "+tc.links, 1)
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
				writeCloseoutCLIFile(t, root, ".pose/specs/2026-09-01-original.md", "---\nslug: original\nstatus: done\n---\n# Original\n")
				inDir(t, root, func() {
					var out, errOut bytes.Buffer
					args := []string{"lint-spec", "basis", "--strict"}
					if ready {
						args = append(args, "--ready-check")
					}
					if code := Main(args, &out, &errOut); code != tc.want {
						t.Fatalf("exit=%d want=%d out=%s err=%s", code, tc.want, out.String(), errOut.String())
					}
					if tc.want != 0 && !strings.Contains(out.String()+errOut.String(), "remediation-lineage/") {
						t.Fatalf("missing diagnostic: %s %s", out.String(), errOut.String())
					}
				})
			})
		}
	}
}
