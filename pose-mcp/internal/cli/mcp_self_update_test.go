// A long-running MCP server survives `pose update` (spec
// pose-mcp-server-survives-self-update).
//
// `pose serve-mcp` answers most tools by running the pose CLI, and it found
// that CLI by calling os.Executable() on every call. `pose update` renames the
// running binary to `.old`, writes the new one in its place and removes the
// `.old`. On Linux os.Executable() resolves through /proc/self/exe, so from
// then on it named the removed file: every CLI-backed tool failed with
// `fork/exec …/pose.old: no such file or directory` until the server was
// restarted. It is the defect the v2.0.0 release hit in the update handoff,
// left standing in the one caller that outlives an update.
//
// So the test runs what an operator does: start the server, update the
// binary it was started from with the real `pose update`, and call a tool.

package cli

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMCPServerRunsTheCLIAfterTheBinaryIsUpdated(t *testing.T) {
	if testing.Short() {
		t.Skip("builds two binaries")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the replacement rename is a POSIX-specific path here")
	}

	work := t.TempDir()
	fixtureBin := buildFixturePose(t, work)
	srv, _ := serveFixtureRelease(t, "99.0.0", tarGzPoseBinary(t, work, fixtureBin))
	posePath := filepath.Join(work, "install", "pose")
	buildPoseAgainstRelease(t, posePath, "2.9.0", srv.URL)

	project := filepath.Join(work, "project")
	if err := os.MkdirAll(filepath.Join(project, ".pose"), 0o755); err != nil {
		t.Fatal(err)
	}

	server := exec.Command(posePath, "serve-mcp", "--stdio")
	server.Dir = project
	server.Env = append(os.Environ(), "POSE_PROJECT_ROOT="+project, "POSE_EXECUTABLE=")
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
		_ = server.Wait()
	})
	responses := bufio.NewReader(stdout)

	// The server is up and answering before the binary moves, so what the
	// update changes is the file under a running process, not its start.
	if got := callMCP(t, stdin, responses, 1, `{"jsonrpc":"2.0","id":1,"method":"ping"}`); !strings.Contains(got, `"result"`) {
		t.Fatalf("server did not answer ping: %s", got)
	}

	update := exec.Command(posePath, "update")
	update.Dir = work
	if out, err := update.CombinedOutput(); err != nil {
		t.Fatalf("pose update: %v\n%s", err, out)
	}
	if _, err := os.Stat(posePath + ".old"); err == nil {
		t.Fatalf("the premise does not hold: %s.old outlived the update", posePath)
	}

	got := callMCP(t, stdin, responses, 2,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"pose_check","arguments":{}}}`)
	if strings.Contains(got, ".old") {
		t.Fatalf("the server ran the CLI from the name the update removed:\n%s", got)
	}
	// The binary at the path the server started from is the updated one, so
	// its output is what the tool reports.
	if !strings.Contains(got, "FIXTURE-POSE-3.0.0 args=[check") {
		t.Fatalf("the tool did not run the CLI installed at %s:\n%s", posePath, got)
	}
}

// callMCP writes one JSON-RPC request and returns the response line with id.
func callMCP(t *testing.T, stdin io.Writer, responses *bufio.Reader, id int, request string) string {
	t.Helper()
	if _, err := io.WriteString(stdin, request+"\n"); err != nil {
		t.Fatal(err)
	}
	type line struct {
		text string
		err  error
	}
	got := make(chan line, 1)
	go func() {
		text, err := responses.ReadString('\n')
		got <- line{text, err}
	}()
	select {
	case l := <-got:
		if l.err != nil {
			t.Fatalf("reading the response to request %d: %v", id, l.err)
		}
		var envelope struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(l.text), &envelope); err != nil || envelope.ID != id {
			t.Fatalf("expected the response to request %d, got: %s", id, l.text)
		}
		return l.text
	case <-time.After(60 * time.Second):
		t.Fatalf("no response to request %d", id)
		return ""
	}
}
