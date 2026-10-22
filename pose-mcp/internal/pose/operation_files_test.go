package pose

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// operationFixture is a small tree with the layouts an update meets: a
// directory reached through a link, a link to a file, a dangling link inside
// the tree and one outside it, and a read-only file.
func operationFixture(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "repo")
	outside = filepath.Join(base, "outside")
	for _, dir := range []string{"a", "c", "ro"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	writeOp(t, filepath.Join(root, "a", "x.md"), "x\n", 0o644)
	writeOp(t, filepath.Join(root, "c", "target.md"), "target\n", 0o644)
	writeOp(t, filepath.Join(root, "c", "private.md"), "private\n", 0o600)
	for link, target := range map[string]string{
		"b":                "a",
		"c/link.md":        "target.md",
		"c/dangling.md":    "missing.md",
		"c/external.md":    filepath.Join(outside, "external.md"),
		"c/external-gone":  filepath.Join(outside, "gone", "file.md"),
		"c/link-to-parent": "..",
	} {
		if err := os.Symlink(target, filepath.Join(root, filepath.FromSlash(link))); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	return root, outside
}

func writeOp(t *testing.T, path, content string, perm fs.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
}

// treeState records every entry under dirs without following links.
func treeState(t *testing.T, dirs ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				out[path] = "error:" + err.Error()
				return nil
			}
			info, err := os.Lstat(path)
			if err != nil {
				return nil
			}
			switch {
			case info.Mode()&fs.ModeSymlink != 0:
				target, _ := os.Readlink(path)
				out[path] = "link:" + target
			case info.IsDir():
				out[path] = fmt.Sprintf("dir:%o", info.Mode().Perm())
			default:
				raw, err := os.ReadFile(path)
				sum := sha256.Sum256(raw)
				state := fmt.Sprintf("file:%o:%s", info.Mode().Perm(), hex.EncodeToString(sum[:8]))
				if err != nil {
					state = fmt.Sprintf("file:%o:unreadable", info.Mode().Perm())
				}
				out[path] = state
			}
			return nil
		})
	}
	return out
}

func sameTree(t *testing.T, before, after map[string]string) {
	t.Helper()
	diff := []string{}
	for path, state := range after {
		if before[path] != state {
			diff = append(diff, path+": "+before[path]+" -> "+state)
		}
	}
	for path, state := range before {
		if _, ok := after[path]; !ok {
			diff = append(diff, path+": "+state+" -> removed")
		}
	}
	if len(diff) > 0 {
		sort.Strings(diff)
		t.Fatalf("the preview changed the filesystem:\n%s", strings.Join(diff, "\n"))
	}
}

func changeLines(changes []PreviewChange) []string {
	out := []string{}
	for _, c := range changes {
		line := c.Verb + ": " + c.Path
		if c.Through != "" {
			line += " (through " + c.Through + ")"
		}
		out = append(out, line)
	}
	return out
}

// A write is read back through every path that reaches it, the original is
// untouched, and the net change names the location and the link it was
// written through.
func TestUpdateOperationReadsItsOwnWritesThroughAliases(t *testing.T) {
	root, outside := operationFixture(t)
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Unix(1_700_000_000, 0))
	if err := p.WriteFile(filepath.Join(root, "b", "new.md"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if raw, err := p.ReadFile(filepath.Join(root, "a", "new.md")); err != nil || string(raw) != "new\n" {
		t.Fatalf("read through the other name: %q %v", raw, err)
	}
	if raw, err := p.ReadFile(filepath.Join(root, "c", "link-to-parent", "a", "new.md")); err != nil || string(raw) != "new\n" {
		t.Fatalf("read through a link to the parent: %q %v", raw, err)
	}
	entries, err := p.ReadDir(filepath.Join(root, "a"))
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, ",") != "new.md,x.md" {
		t.Fatalf("listing: %v", names)
	}
	info, err := p.Stat(filepath.Join(root, "b", "new.md"))
	if err != nil || info.Size() != 4 || info.IsDir() || info.Name() != "new.md" {
		t.Fatalf("stat: %+v %v", info, err)
	}
	if got := strings.Join(changeLines(p.Changes(root)), "\n"); got != "create: a/new.md (through b/new.md)" {
		t.Fatalf("changes:\n%s", got)
	}
	sameTree(t, before, treeState(t, root, outside))
}

// A direct write follows a link to a file and changes its target; a
// publication replaces the link itself. The distinction is what makes an
// overwrite and an atomic rename different effects.
func TestUpdateOperationDistinguishesDirectWritesFromPublication(t *testing.T) {
	root, outside := operationFixture(t)
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Now())
	if err := p.WriteFile(filepath.Join(root, "c", "link.md"), []byte("direct\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if raw, _ := p.ReadFile(filepath.Join(root, "c", "target.md")); string(raw) != "direct\n" {
		t.Fatalf("a direct write did not reach the target: %q", raw)
	}
	if info, err := p.Lstat(filepath.Join(root, "c", "link.md")); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("a direct write replaced the link: %v %v", info, err)
	}
	if err := p.Publish(filepath.Join(root, "c", "link.md"), []byte("published\n"), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny}); err != nil {
		t.Fatal(err)
	}
	if info, err := p.Lstat(filepath.Join(root, "c", "link.md")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("a publication kept the link: %v %v", info, err)
	}
	if raw, _ := p.ReadFile(filepath.Join(root, "c", "target.md")); string(raw) != "direct\n" {
		t.Fatalf("the publication wrote through the link: %q", raw)
	}
	got := strings.Join(changeLines(p.Changes(root)), "\n")
	want := "replace link with file: c/link.md\nmodify: c/target.md (through c/link.md)"
	if got != want {
		t.Fatalf("changes:\n%s\nwant:\n%s", got, want)
	}
	sameTree(t, before, treeState(t, root, outside))
}

// Writing through a dangling link creates its target where the link points,
// inside the tree or outside it, in the view only. An exclusive creation
// refuses the dangling link, as O_EXCL does.
func TestUpdateOperationNeverWritesThroughABrokenLink(t *testing.T) {
	root, outside := operationFixture(t)
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Now())
	if err := p.WriteFile(filepath.Join(root, "c", "dangling.md"), []byte("in\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := p.WriteFile(filepath.Join(root, "c", "external.md"), []byte("out\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := p.WriteFile(filepath.Join(root, "c", "external-gone"), []byte("x"), 0o644)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a dangling link into a missing directory: %v", err)
	}
	if err := p.CreateExclusive(filepath.Join(root, "c", "dangling.md"), []byte("x"), 0o644, true); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("exclusive creation over a link: %v", err)
	}
	changes := p.Changes(root)
	got := strings.Join(changeLines(changes), "\n")
	if !strings.Contains(got, "create: c/missing.md (through c/dangling.md)") || !strings.Contains(got, "create: "+filepath.Join(outside, "external.md")+" (through c/external.md)") {
		t.Fatalf("changes:\n%s", got)
	}
	outsideCount := 0
	for _, c := range changes {
		if c.Outside {
			outsideCount++
		}
	}
	if outsideCount != 1 {
		t.Fatalf("outside changes: %+v", changes)
	}
	refused := 0
	for _, op := range p.Operations() {
		if op.Refused != "" {
			refused++
		}
	}
	if refused != 2 {
		t.Fatalf("the ledger records %d refusals: %+v", refused, p.Operations())
	}
	sameTree(t, before, treeState(t, root, outside))
}

// A directory that can be searched but not listed is read by name, its
// listing fails as the operation's would, a writable file in it is
// overwritten in place and a new entry in it is refused.
func TestUpdateOperationReadsAnExecuteOnlyDirectoryByName(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root, outside := operationFixture(t)
	dir := filepath.Join(root, "a")
	if err := os.Chmod(dir, 0o100); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Now())
	if raw, err := p.ReadFile(filepath.Join(dir, "x.md")); err != nil || string(raw) != "x\n" {
		t.Fatalf("read by name: %q %v", raw, err)
	}
	if _, err := p.ReadDir(dir); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("listing an execute-only directory: %v", err)
	}
	if _, err := os.ReadDir(dir); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("fixture: the directory can be listed: %v", err)
	}
	if err := p.WriteFile(filepath.Join(dir, "x.md"), []byte("rewritten\n"), 0o644); err != nil {
		t.Fatalf("overwriting a writable file in place: %v", err)
	}
	if err := p.WriteFile(filepath.Join(dir, "x.md.pose-backup"), []byte("x\n"), 0o644); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("a new entry in a directory that cannot be written: %v", err)
	}
	if err := p.Publish(filepath.Join(dir, "x.md"), []byte("y"), 0o644, PublishDerived, PublishExpectation{}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("a publication needs the directory: %v", err)
	}
	if got := strings.Join(changeLines(p.Changes(root)), "\n"); got != "modify: a/x.md" {
		t.Fatalf("changes:\n%s", got)
	}
	sameTree(t, before, treeState(t, root, outside))
}

// The native operations, run on a twin of the tree, change exactly what the
// preview says they change.
func TestUpdateOperationPreviewMatchesTheNativeOperations(t *testing.T) {
	run := func(files Files, root string) []error {
		errs := []error{}
		do := func(err error) { errs = append(errs, err) }
		do(files.WriteFile(filepath.Join(root, "b", "new.md"), []byte("new\n"), 0o644))
		do(files.WriteFile(filepath.Join(root, "c", "link.md"), []byte("direct\n"), 0o644))
		do(files.Publish(filepath.Join(root, "c", "private.md"), []byte("private\n"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAny}))
		do(files.Publish(filepath.Join(root, "a", "x.md"), []byte("x\n"), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny}))
		do(files.MkdirAll(filepath.Join(root, "b", "deep", "er"), 0o755))
		do(files.CreateExclusive(filepath.Join(root, "b", "deep", "er", "once.md"), []byte("1"), 0o644, true))
		do(files.CreateExclusive(filepath.Join(root, "b", "deep", "er", "once.md"), []byte("2"), 0o644, true))
		do(files.Remove(filepath.Join(root, "c", "dangling.md")))
		do(files.Symlink("target.md", filepath.Join(root, "c", "dangling.md")))
		do(files.Symlink("target.md", filepath.Join(root, "c", "link.md")))
		do(files.Publish(filepath.Join(root, "c", "fresh.md"), []byte("f"), 0o600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}))
		do(files.Publish(filepath.Join(root, "c", "fresh.md"), []byte("f"), 0o600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}))
		do(files.Publish(filepath.Join(root, "c", "fresh.md"), []byte("g"), 0o600, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}))
		do(files.WriteFile(filepath.Join(root, "a", "back.md"), []byte("tmp"), 0o644))
		do(files.Remove(filepath.Join(root, "a", "back.md")))
		do(files.RemoveAll(filepath.Join(root, "ro")))
		do(files.Mkdir(filepath.Join(root, "c"), 0o755))
		return errs
	}
	rootApply, outsideApply := operationFixture(t)
	rootPreview, outsidePreview := operationFixture(t)
	beforeApply := treeState(t, rootApply, outsideApply)
	beforePreview := treeState(t, rootPreview, outsidePreview)
	applied := run(NativeFiles(), rootApply)
	p := NewPreviewFiles(time.Now())
	previewed := run(p, rootPreview)
	for i := range applied {
		if (applied[i] == nil) != (previewed[i] == nil) || (applied[i] != nil && errorClass(applied[i]) != errorClass(previewed[i])) {
			t.Fatalf("operation %d: applied %v, previewed %v", i, applied[i], previewed[i])
		}
	}
	sameTree(t, beforePreview, treeState(t, rootPreview, outsidePreview))
	want := []string{}
	afterApply := treeState(t, rootApply, outsideApply)
	for path, state := range afterApply {
		was, existed := beforeApply[path]
		rel, _ := filepath.Rel(rootApply, path)
		rel = filepath.ToSlash(rel)
		switch {
		case !existed && strings.HasPrefix(state, "dir:"):
			want = append(want, "create directory: "+rel)
		case !existed && strings.HasPrefix(state, "link:"):
			want = append(want, "create link: "+rel)
		case !existed:
			want = append(want, "create: "+rel)
		case strings.HasPrefix(was, "link:") && strings.HasPrefix(state, "file:"):
			want = append(want, "replace link with file: "+rel)
		case strings.HasPrefix(was, "link:") && was != state:
			want = append(want, "replace link: "+rel)
		case strings.HasPrefix(was, "file:") && was != state && was[:9] != state[:9] && was[9:] == state[9:]:
			want = append(want, "change mode: "+rel)
		case was != state && strings.HasPrefix(state, "file:"):
			want = append(want, "modify: "+rel)
		}
	}
	for path, was := range beforeApply {
		if _, ok := afterApply[path]; !ok {
			rel, _ := filepath.Rel(rootApply, path)
			if strings.HasPrefix(was, "dir:") {
				want = append(want, "remove directory: "+filepath.ToSlash(rel))
			} else {
				want = append(want, "remove: "+filepath.ToSlash(rel))
			}
		}
	}
	got := []string{}
	for _, c := range p.Changes(rootPreview) {
		got = append(got, c.Verb+": "+c.Path)
	}
	sort.Strings(want)
	sort.Strings(got)
	t.Logf("parity:\n%s", strings.Join(got, "\n"))
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preview:\n%s\napplied:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func errorClass(err error) string {
	switch {
	case errors.Is(err, fs.ErrExist):
		return "exist"
	case errors.Is(err, fs.ErrNotExist):
		return "not-exist"
	case errors.Is(err, fs.ErrPermission):
		return "permission"
	case errors.Is(err, ErrPublishConflict):
		return "conflict"
	}
	return err.Error()
}

// Writing the original bytes back is no change, and a location written many
// times is one change.
func TestUpdateOperationCountsNetChangesOnce(t *testing.T) {
	root, outside := operationFixture(t)
	p := NewPreviewFiles(time.Now())
	path := filepath.Join(root, "a", "x.md")
	for i := 0; i < 3; i++ {
		if err := p.Publish(path, []byte(fmt.Sprintf("v%d\n", i)), 0o644, PublishDerived, PublishExpectation{}); err != nil {
			t.Fatal(err)
		}
	}
	if got := changeLines(p.Changes(root)); len(got) != 1 || got[0] != "modify: a/x.md" {
		t.Fatalf("three writes: %v", got)
	}
	if err := p.WriteFile(path, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := changeLines(p.Changes(root)); len(got) != 0 {
		t.Fatalf("writing the original back: %v", got)
	}
	if err := p.Publish(filepath.Join(root, "c", "private.md"), []byte("private\n"), 0o644, PublishDerived, PublishExpectation{}); err != nil {
		t.Fatal(err)
	}
	if got := changeLines(p.Changes(root)); len(got) != 1 || got[0] != "change mode: c/private.md" {
		t.Fatalf("a publication with another mode: %v", got)
	}
	_ = outside
}

// Expectations are checked against the view: a conflict is refused with the
// publisher's error, an identical immutable retry is no change.
func TestUpdateOperationChecksPublicationExpectations(t *testing.T) {
	root, _ := operationFixture(t)
	p := NewPreviewFiles(time.Now())
	path := filepath.Join(root, "a", "x.md")
	err := p.Publish(path, []byte("other"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectDigest, Digest: digestHex([]byte("stale"))})
	var conflict *PublishConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("a stale digest: %v", err)
	}
	if err := p.Publish(path, []byte("next"), 0o644, PublishAuthoritative, ExpectBytes([]byte("x\n"), true)); err != nil {
		t.Fatalf("the digest the writer read: %v", err)
	}
	if err := p.Publish(path, []byte("again"), 0o644, PublishAuthoritative, ExpectBytes([]byte("x\n"), true)); !errors.As(err, &conflict) {
		t.Fatalf("the staged bytes are the current ones: %v", err)
	}
	if err := p.Publish(path, []byte("next"), 0o644, PublishAuthoritative, PublishExpectation{Mode: ExpectAbsent}); err != nil {
		t.Fatalf("an identical immutable retry: %v", err)
	}
	ops := p.Operations()
	if !ops[len(ops)-1].Unchanged {
		t.Fatalf("the retry is not recorded as unchanged: %+v", ops[len(ops)-1])
	}
}

// A refusal the operating system would give is returned as the operation's
// error and recorded; nothing after it is staged at that location.
func TestUpdateOperationReturnsTheRefusalTheSystemWouldGive(t *testing.T) {
	root, outside := operationFixture(t)
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Now())
	p.RefuseWritesUnder(filepath.Join(root, "c"), syscall.EROFS)
	err := p.Publish(filepath.Join(root, "c", "target.md"), []byte("x"), 0o644, PublishDerived, PublishExpectation{})
	if !errors.Is(err, syscall.EROFS) {
		t.Fatalf("publication on a read-only location: %v", err)
	}
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != filepath.Join(root, "c", "target.md") {
		t.Fatalf("the error does not name the path: %#v", err)
	}
	if err := p.WriteFile(filepath.Join(root, "a", "ok.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("elsewhere: %v", err)
	}
	if got := changeLines(p.Changes(root)); len(got) != 1 || got[0] != "create: a/ok.md" {
		t.Fatalf("changes: %v", got)
	}
	ops := p.Operations()
	if ops[0].Refused == "" || ops[0].Op != "publish" {
		t.Fatalf("the refusal is not in the ledger: %+v", ops)
	}
	sameTree(t, before, treeState(t, root, outside))
}

// A removed link reads as absent until a new one is created, and a removed
// directory hides what it held.
func TestUpdateOperationReplacesLinksAndRemovesTrees(t *testing.T) {
	root, outside := operationFixture(t)
	before := treeState(t, root, outside)
	p := NewPreviewFiles(time.Now())
	link := filepath.Join(root, "c", "link.md")
	if err := p.Remove(link); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Lstat(link); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a removed link: %v", err)
	}
	if err := p.Symlink("private.md", link); err != nil {
		t.Fatal(err)
	}
	if raw, err := p.ReadFile(link); err != nil || string(raw) != "private\n" {
		t.Fatalf("through the new link: %q %v", raw, err)
	}
	if err := p.Remove(filepath.Join(root, "a")); !errors.Is(err, syscall.ENOTEMPTY) {
		t.Fatalf("removing a directory with entries: %v", err)
	}
	if err := p.RemoveAll(filepath.Join(root, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Stat(filepath.Join(root, "b", "x.md")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("through a link into a removed directory: %v", err)
	}
	if err := p.MkdirAll(filepath.Join(root, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if entries, err := p.ReadDir(filepath.Join(root, "a")); err != nil || len(entries) != 0 {
		t.Fatalf("a directory created again shows the original's entries: %v %v", entries, err)
	}
	got := strings.Join(changeLines(p.Changes(root)), "\n")
	want := "remove: a/x.md\nreplace link: c/link.md"
	if got != want {
		t.Fatalf("changes:\n%s\nwant:\n%s", got, want)
	}
	sameTree(t, before, treeState(t, root, outside))
}

// The helpers that walk and match go through the view.
func TestUpdateOperationWalksAndGlobsTheView(t *testing.T) {
	root, _ := operationFixture(t)
	p := NewPreviewFiles(time.Now())
	if err := p.Publish(filepath.Join(root, "a", "sub", "y.md"), []byte("y"), 0o644, PublishDerived, PublishExpectation{}); err != nil {
		t.Fatal(err)
	}
	if err := p.Remove(filepath.Join(root, "a", "x.md")); err != nil {
		t.Fatal(err)
	}
	matches, err := GlobIn(p, filepath.Join(root, "a", "*", "*.md"))
	if err != nil || len(matches) != 1 || matches[0] != filepath.Join(root, "a", "sub", "y.md") {
		t.Fatalf("glob: %v %v", matches, err)
	}
	walked := []string{}
	if err := WalkDirIn(p, filepath.Join(root, "a"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		walked = append(walked, filepath.ToSlash(rel))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(walked, ",") != "a,a/sub,a/sub/y.md" {
		t.Fatalf("walk: %v", walked)
	}
	file, err := p.Open(filepath.Join(root, "a", "sub", "y.md"))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(file)
	_ = file.Close()
	if string(raw) != "y" {
		t.Fatalf("open: %q", raw)
	}
}

// One preview is safe for the concurrent readers a gate runs.
func TestUpdateOperationIsSafeForConcurrentReaders(t *testing.T) {
	root, _ := operationFixture(t)
	p := NewPreviewFiles(time.Now())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := filepath.Join(root, "a", fmt.Sprintf("n%d.md", i))
			if err := p.WriteFile(name, []byte("n"), 0o644); err != nil {
				t.Error(err)
			}
			if _, err := p.ReadDir(filepath.Join(root, "b")); err != nil {
				t.Error(err)
			}
			_, _ = p.ReadFile(filepath.Join(root, "c", "link.md"))
		}(i)
	}
	wg.Wait()
	if got := len(p.Changes(root)); got != 8 {
		t.Fatalf("changes: %d", got)
	}
}

// Atomic replacement cannot turn a directory entry into a regular file. The
// preview must refuse at the same boundary, leaving its children visible.
func TestUpdateOperationPublicationRefusesADirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "target")
	writeOp(t, filepath.Join(path, "child"), "keep\n", 0o644)
	p := NewPreviewFiles(time.Now())
	previewErr := p.Publish(path, []byte("replacement"), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny})
	actualErr := NativeFiles().Publish(path, []byte("replacement"), 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny})
	if actualErr == nil {
		t.Fatal("fixture: native replacement accepted a directory")
	}
	if previewErr == nil {
		t.Fatal("preview accepted a replacement native publication refuses")
	}
	if raw, err := p.ReadFile(filepath.Join(path, "child")); err != nil || string(raw) != "keep\n" {
		t.Fatalf("refused publication hid child: %q %v", raw, err)
	}
	if changes := p.Changes(root); len(changes) != 0 {
		t.Fatalf("refused publication changed: %v", changes)
	}
}
