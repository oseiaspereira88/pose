// Self-update end-to-end (spec pose-self-update-handoff-rehearsal).
//
// The download, binary replacement and handoff to the replaced binary had
// never been executed by anything but a real release. The v2.0.0 run was the
// first, and it failed there: performSelfUpdate had renamed the running binary
// to `.old`, and the handoff then called os.Executable() and got that removed
// name. A unit test over the pieces would not have caught it — the defect was
// in the seam between them, and only running the whole sequence reaches it.
//
// So this test runs the whole sequence: a local server stands in for the
// release API and the asset host, a real pose binary is built pointed at it,
// and the assertion is that the binary on disk afterwards is the downloaded
// one and that the *new* binary is what finished the update.

package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixtureNextRelease is the "newer pose" the update downloads. It reports the
// arguments it was handed, which is what proves the handoff reached it and
// carried --no-self.
const fixtureNextRelease = `package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("FIXTURE-POSE-3.0.0 args=%v\n", os.Args[1:])
}
`

func TestSelfUpdateDownloadsReplacesAndHandsOff(t *testing.T) {
	if testing.Short() {
		t.Skip("builds two binaries")
	}
	if runtime.GOOS == "windows" {
		t.Skip("the replacement rename is a POSIX-specific path here")
	}

	const latest = "99.0.0"
	const current = "2.9.0"

	work := t.TempDir()
	fixtureBin := buildFixturePose(t, work)
	srv, served := serveFixtureRelease(t, latest, tarGzPoseBinary(t, work, fixtureBin))

	// The binary under test is built at an older version and pointed at the
	// local server, then run from a directory it is free to overwrite.
	posePath := filepath.Join(work, "install", "pose")
	buildPoseAgainstRelease(t, posePath, current, srv.URL)

	cmd := exec.Command(posePath, "update")
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	got := string(out)
	if err != nil {
		t.Fatalf("pose update: %v\n%s", err, got)
	}

	if served.api == 0 || served.asset == 0 {
		t.Fatalf("the release API and asset were not both fetched: api=%d asset=%d\n%s", served.api, served.asset, got)
	}

	// The handoff: the output has to come from the downloaded binary, with
	// --no-self, and not from the process that started the update.
	if !strings.Contains(got, "FIXTURE-POSE-3.0.0 args=[update --no-self") {
		t.Fatalf("the replaced binary did not finish the update; output was:\n%s", got)
	}

	// And the replacement itself landed where the next invocation will find
	// it — not beside it, and with no `.old` left behind.
	replaced, err := os.ReadFile(posePath)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(fixtureBin)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(replaced, expected) {
		t.Errorf("the binary at %s is not the downloaded one", posePath)
	}
	if _, err := os.Stat(posePath + ".old"); err == nil {
		t.Errorf("the backup at %s.old outlived a successful update", posePath)
	}
}

// fixtureReleaseRequests counts what the local release server was asked for.
type fixtureReleaseRequests struct {
	api   int
	asset int
}

// serveFixtureRelease stands in for both the release API and the asset host:
// it announces latest and serves archive under the name the real release uses.
func serveFixtureRelease(t *testing.T, latest string, archive []byte) (*httptest.Server, *fixtureReleaseRequests) {
	t.Helper()
	assetName := fmt.Sprintf("pose_%s_%s_%s.tar.gz", latest, runtime.GOOS, runtime.GOARCH)
	served := &fixtureReleaseRequests{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/releases/latest"):
			served.api++
			_ = json.NewEncoder(w).Encode(map[string]string{"tag_name": "v" + latest})
		case strings.HasSuffix(r.URL.Path, "/"+assetName):
			served.asset++
			w.Header().Set("Content-Type", "application/gzip")
			_, _ = w.Write(archive)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, served
}

// buildPoseAgainstRelease builds the real pose binary at posePath, reporting
// version current and fetching its updates from releaseURL.
func buildPoseAgainstRelease(t *testing.T, posePath, current, releaseURL string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(posePath), 0o755); err != nil {
		t.Fatal(err)
	}
	pkg := "github.com/harne8/pose-mcp/internal"
	build := exec.Command("go", "build",
		"-ldflags", strings.Join([]string{
			"-X " + pkg + "/version.Version=" + current,
			"-X " + pkg + "/cli.releaseAPIBase=" + releaseURL,
			"-X " + pkg + "/cli.releaseDownloadBase=" + releaseURL,
		}, " "),
		"-o", posePath, "github.com/harne8/pose-mcp/cmd/pose")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the pose binary under test: %v\n%s", err, out)
	}
}

func buildFixturePose(t *testing.T, work string) string {
	t.Helper()
	src := filepath.Join(work, "fixture")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "main.go"), []byte(fixtureNextRelease), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "go.mod"), []byte("module posefixture\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(work, "pose-next")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", bin, ".")
	build.Dir = src
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the release fixture: %v\n%s", err, out)
	}
	return bin
}

func tarGzPoseBinary(t *testing.T, work, bin string) []byte {
	t.Helper()
	body, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	// Named exactly as the release archive names it: extractPoseBinary looks
	// for `pose` at the root, so a layout change here is a real regression.
	if err := tw.WriteHeader(&tar.Header{Name: "pose", Mode: 0o755, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
