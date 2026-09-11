// A stdio MCP server stops when asked (spec pose-stdio-server-honours-sigterm).
//
// `pose serve-mcp --stdio` caught SIGTERM with signal.NotifyContext and then
// blocked reading stdin, checking the context only after the next line arrived.
// So SIGTERM left it running, and the next request — the one a client sends to
// find out whether the server is alive — closed the connection unanswered.
// Observed stopping this repository's own server to pick up 5.0.1.

package cli

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStdioMCPServerExitsOnSIGTERMWhileIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM is POSIX")
	}
	work := t.TempDir()
	posePath := filepath.Join(work, "pose")
	buildPoseAgainstRelease(t, posePath, "5.0.1", "http://127.0.0.1:1")
	project := filepath.Join(work, "project")
	if err := os.MkdirAll(filepath.Join(project, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}

	server := exec.Command(posePath, "serve-mcp", "--stdio")
	server.Dir = project
	server.Env = append(os.Environ(), "POSE_PROJECT_ROOT="+project)
	stdin, err := server.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := server.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = stdin.Close()
		_ = server.Process.Kill()
	})
	// Answering first proves the server is up and idle in its read loop when
	// the signal arrives, rather than still starting.
	if got := callMCP(t, stdin, bufio.NewReader(stdout), 1, `{"jsonrpc":"2.0","id":1,"method":"ping"}`); !strings.Contains(got, `"result"`) {
		t.Fatalf("server did not answer ping: %s", got)
	}

	if err := server.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	exited := make(chan error, 1)
	go func() { exited <- server.Wait() }()
	select {
	case err := <-exited:
		if err != nil {
			t.Fatalf("a requested shutdown exited with an error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server was still running 10s after SIGTERM, with stdin open and no request pending")
	}
}
