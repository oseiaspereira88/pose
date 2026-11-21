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

type reviewGitlinkSetup struct {
	ctx                context.Context
	cancel             context.CancelFunc
	hooks              string
	metadata           string
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
			return err
		}
		pin, name, err := reviewGitlinkRecord(bytes.TrimSuffix(raw, []byte{0}))
		if err != nil || name != path || pin == "" || digestBytes([]byte(pin)) != entry.Digest {
			return fmt.Errorf("sealed gitlink %q does not match the subject", path)
		}
		if err := s.checkout(origin, worktree, path, pin); err != nil {
			return err
		}
	}
	return nil
}

func reviewGitlinkRecord(record []byte) (pin, path string, err error) {
	metadata, path, ok := strings.Cut(string(record), "\t")
	if !ok {
		return "", "", fmt.Errorf("sealed tree record has no path")
	}
	fields := strings.Fields(metadata)
	if len(fields) != 4 || strings.ContainsRune(path, 0) {
		return "", "", fmt.Errorf("invalid sealed tree record")
	}
	if fields[0] != "160000" {
		return "", path, nil
	}
	if fields[1] != "commit" || !isGitRevision(fields[2]) {
		return "", "", fmt.Errorf("invalid sealed gitlink")
	}
	return fields[2], path, nil
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
	if err := os.MkdirAll(s.metadata, 0700); err != nil {
		return fmt.Errorf("private gitlink metadata directory unavailable")
	}
	s.count++
	gitDir := filepath.Join(s.metadata, strconv.Itoa(s.count))
	objectFormat := "sha1"
	if len(pin) == 64 {
		objectFormat = "sha256"
	}
	if _, err := s.git(64<<10, "-C", target, "init", "-q", "--object-format="+objectFormat, "--template="+s.hooks, "--separate-git-dir="+gitDir); err != nil {
		return err
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
		return err
	}
	raw, err := s.git(gitTreeListingMaxBytes, "-C", target, "ls-tree", "-r", "-l", "-z", pin)
	if err != nil {
		return err
	}
	s.committedBytes += gitCommittedTreeBytes(raw)
	if s.committedBytes > gitSnapshotCeilingBytes {
		return fmt.Errorf("sealed gitlinks exceed the snapshot byte ceiling")
	}
	if _, err := s.git(64<<10, "-C", target, "checkout", "-q", "--detach", pin, "--"); err != nil {
		return err
	}
	for record := range bytes.SplitSeq(bytes.TrimSuffix(raw, []byte{0}), []byte{0}) {
		if len(record) == 0 {
			continue
		}
		nestedPin, nestedPath, err := reviewGitlinkRecord(record)
		if err != nil {
			return err
		}
		if nestedPin != "" {
			if err := s.checkout(source, target, nestedPath, nestedPin); err != nil {
				return err
			}
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
