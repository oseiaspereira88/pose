package version

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Execute the workflow's notes selection with a recording command stub. A
// tagged run must never fall back to mutable preview notes, even if the
// prepared manifest is missing. Only an unprepared, non-publishing snapshot
// uses the plan/preview path.
func TestReleaseSnapshotNotesKeepTaggedPublicationFrozen(t *testing.T) {
	raw, err := os.ReadFile("../../../.github/workflows/release.yml")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	start := strings.Index(source, "      - name: Release notes from changelog fragments\n        run: |\n")
	if start < 0 {
		t.Fatal("notes step missing")
	}
	section := source[start:]
	section = section[strings.Index(section, "        run: |\n")+len("        run: |\n"):]
	section = strings.SplitN(section, "      - run:", 2)[0]
	lines := strings.Split(strings.TrimRight(section, "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimPrefix(lines[i], "          ")
	}
	script := strings.Join(lines, "\n")
	// Keep temporary output local to this test instead of the runner's /tmp.
	for _, tc := range []struct {
		name, ref    string
		prepared     bool
		want, absent string
	}{
		{"snapshot before cut", "refs/heads/main", false, "release-notes --preview", "release notes"},
		{"prepared snapshot", "refs/heads/main", true, "release check", "--preview"},
		{"tag without manifest", "refs/tags/v6.0.0", false, "release check", "--preview"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			bin := filepath.Join(root, "bin")
			if err := os.Mkdir(bin, 0755); err != nil {
				t.Fatal(err)
			}
			log := filepath.Join(root, "calls")
			if err := os.WriteFile(filepath.Join(bin, "go"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$CALLS\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
			if tc.prepared {
				dir := filepath.Join(root, ".pose/releases/v6.0.0")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command("bash", "-c", strings.ReplaceAll(script, "/tmp/notes.md", filepath.Join(root, "notes.md")))
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "PATH="+bin+":"+os.Getenv("PATH"), "GITHUB_REF="+tc.ref, "RELEASE_VERSION=v6.0.0", "CALLS="+log)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("notes selection: %v %s", err, out)
			}
			calls, _ := os.ReadFile(log)
			if !strings.Contains(string(calls), tc.want) || strings.Contains(string(calls), tc.absent) {
				t.Fatalf("wrong notes path: %s", calls)
			}
		})
	}
}
