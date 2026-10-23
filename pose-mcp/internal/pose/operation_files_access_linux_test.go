//go:build linux

package pose

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestUpdateOperationVirtualRegularFileSubprocess(t *testing.T) {
	if os.Getenv("POSE_B29_VIRTUAL_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestUpdateOperationVirtualRegularFileSubprocess$", "-test.v")
		cmd.Env = append(os.Environ(), "POSE_B29_VIRTUAL_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("virtual-file child: %v\n%s", err, out)
		}
		return
	}
	path := "/proc/self/comm"
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Skip("proc comm unavailable")
	}
	p := NewPreviewFiles(time.Now())
	data := []byte("a long value that the kernel will truncate")
	pe := p.WriteFile(path, data, 0o644)
	// This changes only the disposable test child's kernel task name.
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Skipf("kernel control unavailable: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil || string(raw) == string(data) {
		t.Fatalf("virtual-file control: %q %v", raw, err)
	}
	if !errors.Is(pe, ErrPreviewUnsupported) || len(p.Gaps()) == 0 {
		t.Fatalf("virtual regular file fabricated ordinary bytes: %v %v; kernel %q", pe, p.Gaps(), raw)
	}
}

// Only this disposable child changes its process-wide umask.
func TestUpdateOperationRestrictiveUmaskSubprocess(t *testing.T) {
	if os.Getenv("POSE_B29_PERMISSION_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestUpdateOperationRestrictiveUmaskSubprocess$", "-test.v")
		cmd.Env = append(os.Environ(), "POSE_B29_PERMISSION_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("unprivileged child: %v\n%s", err, out)
		}
		return
	}
	if os.Geteuid() == 0 {
		t.Fatal("run permission regression as an unprivileged user")
	}
	for _, mask := range []int{0111, 0400} {
		t.Run(fmt.Sprintf("%04o", mask), func(t *testing.T) {
			root := t.TempDir()
			previous := syscall.Umask(mask)
			defer syscall.Umask(previous)
			p := NewPreviewFiles(time.Now())
			native := NativeFiles()
			path := filepath.Join(root, "entry")
			if mask == 0111 {
				if err := p.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				if err := native.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(path, 0700)
				for _, op := range []string{"write", "stat", "read", "list"} {
					child := filepath.Join(path, "child")
					var pe, ne error
					switch op {
					case "write":
						pe = p.WriteFile(child, []byte("x"), 0600)
						ne = native.WriteFile(child, []byte("x"), 0600)
					case "stat":
						_, pe = p.Stat(child)
						_, ne = native.Stat(child)
					case "read":
						_, pe = p.ReadFile(child)
						_, ne = native.ReadFile(child)
					case "list":
						_, pe = p.ReadDir(path)
						_, ne = native.ReadDir(path)
					}
					if errors.Is(pe, ErrPreviewUnsupported) {
						if len(p.Gaps()) == 0 {
							t.Fatal("missing gap")
						}
						continue
					}
					if (pe == nil) != (ne == nil) {
						t.Errorf("%s preview %v native %v", op, pe, ne)
					}
				}
			} else {
				if err := p.CreateExclusive(path, []byte("x"), 0600, false); err != nil {
					t.Fatal(err)
				}
				if err := native.CreateExclusive(path, []byte("x"), 0600, false); err != nil {
					t.Fatal(err)
				}
				for _, open := range []bool{false, true} {
					var pe, ne error
					if open {
						f, e := p.Open(path)
						pe = e
						if f != nil {
							f.Close()
						}
						f, e = native.Open(path)
						ne = e
						if f != nil {
							f.Close()
						}
					} else {
						_, pe = p.ReadFile(path)
						_, ne = native.ReadFile(path)
					}
					if errors.Is(pe, ErrPreviewUnsupported) {
						if len(p.Gaps()) == 0 {
							t.Fatal("missing gap")
						}
						continue
					}
					if (pe == nil) != (ne == nil) {
						t.Errorf("open=%v preview %v native %v", open, pe, ne)
					}
				}
			}
		})
	}
	t.Run("physical-search-before-dotdot", func(t *testing.T) {
		root := t.TempDir()
		locked := filepath.Join(root, "locked")
		if err := os.Mkdir(locked, 0000); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(locked, 0700)
		writeOp(t, filepath.Join(root, "victim"), "original", 0600)
		link := filepath.Join(root, "link")
		if err := os.Symlink(root+"/locked/../victim", link); err != nil {
			t.Fatal(err)
		}
		p := NewPreviewFiles(time.Now())
		_, pe := p.ReadFile(link)
		_, ne := os.ReadFile(link)
		if !errors.Is(ne, os.ErrPermission) {
			t.Fatalf("native control: %v", ne)
		}
		if !errors.Is(pe, os.ErrPermission) && !(errors.Is(pe, ErrPreviewUnsupported) && len(p.Gaps()) > 0) {
			t.Fatalf("physical search skipped: %v", pe)
		}
	})
	for _, recursive := range []bool{false, true} {
		t.Run(fmt.Sprintf("unreadable-empty-removal-%v", recursive), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "empty")
			if err := os.Mkdir(path, 0000); err != nil {
				t.Fatal(err)
			}
			p := NewPreviewFiles(time.Now())
			var pe, ne error
			if recursive {
				pe = p.RemoveAll(path)
				ne = os.RemoveAll(path)
			} else {
				pe = p.Remove(path)
				ne = os.Remove(path)
			}
			if ne != nil {
				t.Fatalf("native empty-directory removal: %v", ne)
			}
			if !errors.Is(pe, ErrPreviewUnsupported) || len(p.Gaps()) == 0 {
				t.Fatalf("unobserved emptiness must remain incomplete: %v %v", pe, p.Gaps())
			}
		})
	}
	t.Run("default-acl-creation", func(t *testing.T) {
		parent := t.TempDir()
		// Linux POSIX ACL xattr v2: owner read, group read, other none.
		acl := make([]byte, 4+3*8)
		binary.LittleEndian.PutUint32(acl, 2)
		for n, tag := range []uint16{1, 4, 32} {
			entry := acl[4+n*8:]
			binary.LittleEndian.PutUint16(entry, tag)
			if n != 2 {
				binary.LittleEndian.PutUint16(entry[2:], 4)
			}
			binary.LittleEndian.PutUint32(entry[4:], ^uint32(0))
		}
		if err := syscall.Setxattr(parent, "system.posix_acl_default", acl, 0); err != nil {
			t.Skipf("default ACL unavailable: %v", err)
		}
		path := filepath.Join(parent, "child")
		p := NewPreviewFiles(time.Now())
		pe := p.Mkdir(path, 0o755)
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
		defer os.Chmod(path, 0o700)
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0o100 != 0 {
			t.Fatalf("native inherited ACL control: %v %v", info, err)
		}
		if !errors.Is(pe, ErrPreviewUnsupported) || len(p.Gaps()) == 0 {
			t.Fatalf("inherited mode guessed: %v %v", pe, p.Gaps())
		}
	})

}
