package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAssessDesignRequiresSpec(t *testing.T) {
	root := newGitRepo(t)
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"assess", "design"}, &out, &errOut); code != 2 {
			t.Fatalf("missing spec exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
		}
		if !strings.Contains(errOut.String(), "--spec") {
			t.Fatalf("missing spec diagnostic: %q", errOut.String())
		}
	})
}

func TestAssessDesignRejectsUnsafeLimit(t *testing.T) {
	root := newGitRepo(t)
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"assess", "design", "--spec", "demo", "--max-files", "0"}, &out, &errOut); code != 2 {
			t.Fatalf("invalid limit exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
		}
		if !strings.Contains(errOut.String(), "positive integer") {
			t.Fatalf("missing limit diagnostic: %q", errOut.String())
		}
	})
}

func TestAssessDesignRejectsUnknownOption(t *testing.T) {
	root := newGitRepo(t)
	inDir(t, root, func() {
		var out, errOut bytes.Buffer
		if code := Main([]string{"assess", "design", "--spec", "demo", "--wat"}, &out, &errOut); code != 2 {
			t.Fatalf("unknown option exit=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
		}
		if !strings.Contains(errOut.String(), "pose assess design") {
			t.Fatalf("missing usage diagnostic: %q", errOut.String())
		}
	})
}
