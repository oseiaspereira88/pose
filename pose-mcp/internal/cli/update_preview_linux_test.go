//go:build linux

package cli

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

// Observe filesystem mutation events, including transient publisher staging and
// locks that final content snapshots miss. This is an inotify mutation audit,
// not a syscall audit. Reads (including atime updates) are excluded.
func TestUpdatePreviewHasNoPhysicalMutationEvents(t *testing.T) {
	// Build before installing watches: compiler/cache effects are test setup,
	// not activity of the command being audited. Run the actual CLI binary.
	executable := filepath.Join(t.TempDir(), "pose")
	build := exec.Command("go", "build", "-buildvcs=false", "-o", executable, "./cmd/pose")
	build.Dir = "../.."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build preview CLI: %v %s", err, out)
	}
	for _, force := range []bool{false, true} {
		t.Run(map[bool]string{false: "plain", true: "force"}[force], func(t *testing.T) {
			root := staleInstance(t)
			outside := t.TempDir()
			temp := t.TempDir()
			t.Setenv("TMPDIR", temp)
			mustWrite(t, filepath.Join(outside, "sentinel"), "untouched\n")
			for _, link := range []string{"security.md", "documentation-style.md"} {
				path := filepath.Join(root, ".pose/rules", link)
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(outside, link), path); err != nil {
					t.Fatal(err)
				}
			}
			internalLink := filepath.Join(root, ".pose/rules/delivery-evidence.md")
			if err := os.Remove(internalLink); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../internal-missing.md", internalLink); err != nil {
				t.Fatal(err)
			}
			// Force reaches default adoption and exclusive onboarding creation.
			if force {
				if err := os.Remove(filepath.Join(root, ".pose/policy/review.json")); err != nil {
					t.Fatal(err)
				}
				paths, _ := filepath.Glob(filepath.Join(root, ".pose/specs/*-pose-onboarding.md"))
				for _, path := range paths {
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
				}
			}

			// Exercise B38 historical version-source validation inside the force
			// final gate. Prepare and tag before watching; preview must read the
			// committed snapshot without creating even transient temp files.
			if force {
				releaseRoot := cliReleaseRepo(t)
				for _, rel := range []string{"VERSION", ".pose/policy/release.json", ".pose/changelogs/v1.0.0/alpha.md", ".pose/changelogs/v1.0.0.md", ".pose/releases/v1.0.0/manifest.json"} {
					raw, err := os.ReadFile(filepath.Join(releaseRoot, filepath.FromSlash(rel)))
					if err != nil {
						t.Fatal(err)
					}
					mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), string(raw))
				}
				policy := filepath.Join(root, ".pose/policy/release.json")
				raw, err := os.ReadFile(policy)
				if err != nil {
					t.Fatal(err)
				}
				// Adopted policy reaches release judgment. Keep the cut's original
				// policy bytes in the tag, then adopt in today's worktree.
				for _, args := range [][]string{{"add", "-A"}, {"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture release"}, {"-c", "tag.gpgsign=false", "tag", "v1.0.0"}} {
					if out, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
						t.Fatalf("git fixture: %v %s", err, out)
					}
				}
				raw = []byte(strings.Replace(string(raw), `"adopted_at":""`, `"adopted_at":"2026-10-01"`, 1))
				raw = []byte(strings.Replace(string(raw), `"provider":""`, `"provider":"github"`, 1))
				raw = []byte(strings.Replace(string(raw), `"repository":""`, `"repository":"example/project"`, 1))
				mustWrite(t, policy, string(raw))
			}
			fd, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
			if err != nil {
				t.Fatal(err)
			}
			defer unix.Close(fd)
			const mask = unix.IN_MODIFY | unix.IN_ATTRIB | unix.IN_CLOSE_WRITE | unix.IN_MOVED_FROM | unix.IN_MOVED_TO | unix.IN_CREATE | unix.IN_DELETE | unix.IN_DELETE_SELF | unix.IN_MOVE_SELF
			watches := map[int]string{}
			for _, base := range []string{root, outside, temp} {
				err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						return nil
					}
					wd, err := unix.InotifyAddWatch(fd, path, mask)
					if err != nil {
						return err
					}
					watches[wd] = path
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"update", "--dry-run"}
			if force {
				args = append(args, "--force")
			}
			cmd := exec.Command(executable, args...)
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "POSE_USAGE_DISABLED=", "POSE_USAGE_SUPPRESS_CLI=", "POSE_USAGE_DIR="+filepath.Join(outside, "usage"))
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("preview subprocess: %v %s", err, out)
			}

			buf := make([]byte, 1024*1024)
			n, err := unix.Read(fd, buf)
			if err != nil && !errors.Is(err, unix.EAGAIN) {
				t.Fatal(err)
			}
			if n > 0 {
				events := []string{}
				for offset := 0; offset+16 <= n; {
					wd := int(int32(binary.LittleEndian.Uint32(buf[offset:])))
					mask := binary.LittleEndian.Uint32(buf[offset+4:])
					size := int(binary.LittleEndian.Uint32(buf[offset+12:]))
					name := strings.TrimRight(string(buf[offset+16:offset+16+size]), "\x00")
					events = append(events, fmt.Sprintf("%s mask=%#x", filepath.Join(watches[wd], name), mask))
					offset += 16 + size
				}
				t.Fatalf("preview generated physical mutation events: %v; output:\n%s", events, out)
			}
			if _, err := os.Stat(filepath.Join(root, ".pose/internal-missing.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("dangling internal target changed: %v", err)
			}
			if _, err := os.Stat(filepath.Join(outside, "security.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("dangling external target changed: %v", err)
			}
			if _, err := os.Stat(filepath.Join(outside, "documentation-style.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("dangling external target changed: %v", err)
			}
		})
	}
}
