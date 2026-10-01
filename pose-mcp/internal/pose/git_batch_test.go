package pose

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func batchGitFixture(t testing.TB) string {
	t.Helper()
	root := t.TempDir()
	command := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
	}
	command("init", "-q")
	command("config", "user.email", "pose@example.invalid")
	command("config", "user.name", "POSE fixture")
	for name, body := range map[string][]byte{"normal.txt": []byte("content\n"), "space name.txt": {0, 1, 2, '\n'}, "empty.txt": {}, "large.txt": bytes.Repeat([]byte("x"), 8192)} {
		if err := os.WriteFile(filepath.Join(root, name), body, 0644); err != nil {
			t.Fatal(err)
		}
	}
	command("add", ".")
	command("commit", "-qm", "fixture")
	return root
}

func TestGitBatchReadsMatchGitShowAndReuseProcess(t *testing.T) {
	root := batchGitFixture(t)
	reader := &gitBatchReader{root: root}
	defer reader.close()
	pid := 0
	for _, name := range []string{"normal.txt", "space name.txt", "empty.txt", "missing.txt", "normal.txt"} {
		got, err := reader.read("HEAD:"+name, 8192)
		if name == "missing.txt" {
			if !errors.Is(err, errGitBatchMissing) {
				t.Fatalf("missing: %v", err)
			}
		} else {
			want, showErr := exec.Command("git", "-C", root, "show", "HEAD:"+name).Output()
			if err != nil || showErr != nil || !bytes.Equal(got, want) {
				t.Fatalf("%s: %v/%v: %q != %q", name, err, showErr, got, want)
			}
		}
		if pid == 0 {
			pid = reader.cmd.Process.Pid
		}
		if reader.cmd.Process.Pid != pid {
			t.Fatal("spawned a new process for another blob")
		}
	}
	reader.close()
	if reader.cmd.ProcessState == nil {
		t.Fatal("batch process not reaped")
	}
	if _, err := reader.read("HEAD:normal.txt", 8192); err == nil {
		t.Fatal("read after close succeeded")
	}
}

func TestGitBatchRejectsLimitsAndRequestInjection(t *testing.T) {
	root := batchGitFixture(t)
	for _, request := range []string{"HEAD:normal.txt\nHEAD:large.txt", "HEAD:normal.txt\x00", "HEAD:normal.txt\r"} {
		reader := &gitBatchReader{root: root}
		if _, err := reader.read(request, 8192); err == nil || reader.cmd != nil {
			t.Fatalf("unsafe request started a process: %q", request)
		}
		reader.close()
	}
	reader := &gitBatchReader{root: root}
	if _, err := reader.read("HEAD:normal.txt", -1); err == nil {
		t.Fatal("negative limit accepted")
	}
	if _, err := reader.read("HEAD:large.txt", 1); !errors.Is(err, errDesignDeltaTooLarge) {
		t.Fatalf("oversized: %v", err)
	}
	if !reader.closed || reader.cmd.ProcessState == nil {
		t.Fatal("oversized stream was not terminated/reaped")
	}
	reader.close()
}

func BenchmarkGitBlobReadProcessVsBatch(b *testing.B) {
	root := batchGitFixture(b)
	for _, batch := range []bool{false, true} {
		b.Run(fmt.Sprintf("batch=%t", batch), func(b *testing.B) {
			for iteration := 0; iteration < b.N; iteration++ {
				reader := &gitBatchReader{root: root}
				for request := 0; request < 128; request++ {
					var body []byte
					var err error
					if batch {
						body, err = reader.read("HEAD:normal.txt", 8192)
					} else {
						exists, existErr := gitObjectExists(root, "HEAD", "normal.txt")
						if !exists || existErr != nil {
							b.Fatal("object missing", existErr)
						}
						body, err = gitShowBounded(root, "HEAD", "normal.txt", 8192)
					}
					if err != nil || string(body) != "content\n" {
						b.Fatalf("read: %v %q", err, body)
					}
				}
				reader.close()
			}
		})
	}
}
