package pose

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const reviewGitlinkMetadataDir = "gitlinks"

type reviewGitlinkSetup struct {
	ctx                context.Context
	cancel             context.CancelFunc
	hooks              string
	metadata           string
	metadataOwned      os.FileInfo
	maxCommittedBytes  int64
	count              int
	committedBytes     int64
	captureOverflow    bool
	terminationFailure bool
}

func (s *reviewGitlinkSetup) materialize(origin, worktree, commit string, entries []ReviewBundleSubjectEntry) error {
	for _, entry := range entries {
		if entry.Class != "submodule" {
			continue
		}
		path := reviewBundleEntryPath(entry)
		raw, err := s.git(64<<10, "--literal-pathspecs", "-C", worktree, "ls-tree", "-l", "-z", commit, "--", path)
		if err != nil {
			return fmt.Errorf("read sealed gitlink %q: %w", path, err)
		}
		pin, name, _, err := reviewGitlinkRecord(bytes.TrimSuffix(raw, []byte{0}))
		if err != nil || name != path || pin == "" || digestBytes([]byte(pin)) != entry.Digest {
			return fmt.Errorf("sealed gitlink %q does not match the subject", path)
		}
		if err := s.checkout(origin, worktree, path, pin); err != nil {
			return err
		}
	}
	return nil
}

func reviewGitlinkRecord(record []byte) (pin, path string, size int64, err error) {
	metadata, path, ok := strings.Cut(string(record), "\t")
	if !ok {
		return "", "", 0, fmt.Errorf("sealed tree record has no path")
	}
	fields := strings.Fields(metadata)
	if len(fields) != 4 || strings.ContainsRune(path, 0) {
		return "", "", 0, fmt.Errorf("invalid sealed tree record")
	}
	if fields[0] != "160000" {
		size, err := strconv.ParseInt(fields[3], 10, 64)
		if fields[1] != "blob" || err != nil || size < 0 {
			return "", "", 0, fmt.Errorf("sealed tree blob size unavailable")
		}
		return "", path, size, nil
	}
	if fields[1] != "commit" || !isGitRevision(fields[2]) {
		return "", "", 0, fmt.Errorf("invalid sealed gitlink")
	}
	return fields[2], path, 0, nil
}

func (s *reviewGitlinkSetup) checkout(origin, worktree, path, pin string) error {
	source, err := dispatchInputPath(origin, path)
	if err != nil {
		return err
	}
	target, err := dispatchInputPath(worktree, path)
	if err != nil {
		return err
	}
	if _, err := dispatchWorktreeIdentity(source); err != nil {
		return err
	}
	children, err := os.ReadDir(target)
	if err != nil || len(children) != 0 {
		return fmt.Errorf("sealed gitlink %q has no empty checkout directory", path)
	}
	// Charge the exact shallow tree before transferring its objects. This also
	// witnesses the pin's availability in the selected local repository.
	raw, err := s.git(gitTreeListingMaxBytes, "-C", source, "ls-tree", "-r", "-l", "-z", pin)
	if err != nil {
		return fmt.Errorf("read local tree for sealed gitlink %q: %w", path, err)
	}
	type nestedGitlink struct{ path, pin string }
	var nested []nestedGitlink
	var treeBytes int64
	for record := range bytes.SplitSeq(bytes.TrimSuffix(raw, []byte{0}), []byte{0}) {
		if len(record) == 0 {
			continue
		}
		nestedPin, nestedPath, size, err := reviewGitlinkRecord(record)
		if err != nil {
			return fmt.Errorf("inspect local tree for sealed gitlink %q: %w", path, err)
		}
		if size > s.maxCommittedBytes-s.committedBytes-treeBytes {
			return fmt.Errorf("sealed gitlink %q exceeds the snapshot byte ceiling", path)
		}
		treeBytes += size
		if nestedPin != "" {
			nested = append(nested, nestedGitlink{path: nestedPath, pin: nestedPin})
		}
	}
	s.committedBytes += treeBytes
	if s.metadataOwned == nil {
		if err := os.Mkdir(s.metadata, 0700); err != nil {
			return fmt.Errorf("private gitlink metadata directory unavailable")
		}
		s.metadataOwned, err = os.Lstat(s.metadata)
		if err != nil {
			return fmt.Errorf("private gitlink metadata identity unavailable")
		}
	}
	s.count++
	gitDir := filepath.Join(s.metadata, strconv.Itoa(s.count))
	objectFormat := "sha1"
	if len(pin) == 64 {
		objectFormat = "sha256"
	}
	if _, err := s.git(64<<10, "-C", target, "init", "-q", "--object-format="+objectFormat, "--template="+s.hooks, "--separate-git-dir="+gitDir); err != nil {
		return fmt.Errorf("initialize private Git directory for %q: %w", path, err)
	}
	// Git writes an absolute marker on init. Preserve a reproducible checkout
	// snapshot by expressing the path within the owned runtime directory.
	marker, err := filepath.Rel(target, gitDir)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, ".git"), []byte("gitdir: "+filepath.ToSlash(marker)+"\n"), 0600); err != nil {
		return err
	}
	if _, err := s.git(64<<10, "-C", target, "fetch", "-q", "--depth=1", "--no-tags", "--no-recurse-submodules", "--no-write-fetch-head", "--no-auto-maintenance", "--", source, pin); err != nil {
		return fmt.Errorf("fetch local objects for sealed gitlink %q: %w", path, err)
	}
	if _, err := s.git(64<<10, "-C", target, "checkout", "-q", "--detach", pin, "--"); err != nil {
		return fmt.Errorf("check out sealed gitlink %q: %w", path, err)
	}
	for _, child := range nested {
		if err := s.checkout(source, target, child.path, child.pin); err != nil {
			return err
		}
	}
	return nil
}

func (s *reviewGitlinkSetup) git(limit int64, args ...string) ([]byte, error) {
	command := gitCommand(append([]string{"-c", "core.hooksPath=" + s.hooks, "-c", "core.attributesFile=" + os.DevNull}, args...)...)
	env := command.Env[:0]
	for _, variable := range command.Env {
		name, _, _ := strings.Cut(variable, "=")
		if !strings.HasPrefix(name, "GIT_") {
			env = append(env, variable)
		}
	}
	command.Env = append(env,
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_ALLOW_PROTOCOL=file",
		"GIT_TERMINAL_PROMPT=0", "GIT_ATTR_NOSYSTEM=1", "GIT_OPTIONAL_LOCKS=0",
		"GIT_NO_REPLACE_OBJECTS=1", "GIT_GRAFT_FILE="+os.DevNull,
	)
	cmd := reviewSetupCommand(s.ctx, command)
	output := &reviewBoundedCapture{limit: limit, cancel: s.cancel}
	cmd.Stdout, cmd.Stderr = output, output
	runErr := cmd.Run()
	data, overflow := output.result()
	terminationErr := killProcessGroup(cmd)
	s.captureOverflow = overflow
	s.terminationFailure = terminationErr != nil
	if runErr != nil || terminationErr != nil || overflow || s.ctx.Err() != nil {
		return nil, fmt.Errorf("gitlink setup command failed: %w", errors.Join(runErr, terminationErr, s.ctx.Err()))
	}
	return data, nil
}
