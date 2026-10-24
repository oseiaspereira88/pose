package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestPolicyScopeCLIRefusesInvalidConfiguration(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		root := t.TempDir()
		mustWrite(t, filepath.Join(root, ".pose/policy/review.json"), `{"schema_version":2,"enabled":true,"identity_assurance":{"spce":"declared"}}`)
		args := []string{"state", "--governance"}
		if jsonMode {
			args = append(args, "--json")
		}
		var out, errOut bytes.Buffer
		code := 0
		inDir(t, root, func() { code = Main(args, &out, &errOut) })
		if code == 0 || !strings.Contains(errOut.String(), `identity_assurance["spce"]`) {
			t.Fatalf("invalid policy not diagnosed: code=%d out=%s err=%s", code, out.String(), errOut.String())
		}
		if strings.Contains(out.String(), `effective=true`) || strings.Contains(out.String(), `"effective":true`) {
			t.Fatalf("false protection: %s", out.String())
		}
	}
}
