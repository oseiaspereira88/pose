package pose

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const cliTimeout = 30 * time.Second

// Suggest wraps `pose suggest <type> [--domain d] [--path p] --json`. The
// CLI stays the deterministic source of truth for the canonical trail
// (workflow + skill + rules + validation) — ADR-003: adapter, not fork.
func (s Store) Suggest(ctx context.Context, taskType, domain, relPath string) (any, error) {
	if err := ValidateName(taskType); err != nil {
		return nil, fmt.Errorf("pose_suggest: invalid task_type: %w", err)
	}
	args := []string{"suggest", taskType}
	if domain != "" {
		if err := ValidateName(domain); err != nil {
			return nil, fmt.Errorf("pose_suggest: invalid domain: %w", err)
		}
		args = append(args, "--domain", domain)
	}
	if relPath != "" {
		if filepath.IsAbs(relPath) || strings.Contains(relPath, "..") {
			return nil, fmt.Errorf("pose_suggest: invalid path %q", relPath)
		}
		args = append(args, "--path", relPath)
	}
	args = append(args, "--json")

	raw, err := s.runCLI(ctx, args)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("pose suggest: unexpected non-JSON output")
	}
	return out, nil
}

// Followups wraps `pose followups --open|--all --json` — the live backlog
// of spec follow-ups with lexical near-duplicate candidates (always exit 0).
func (s Store) Followups(ctx context.Context, all bool) (any, error) {
	scope := "--open"
	if all {
		scope = "--all"
	}
	raw, err := s.runFollowups(ctx, scope)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("pose followups: unexpected non-JSON output")
	}
	return out, nil
}

func (s Store) runFollowups(ctx context.Context, scope string) ([]byte, error) {
	return s.runCLI(ctx, []string{"followups", scope, "--json"})
}

// GateResult is the outcome of a deterministic POSE gate evaluated in
// read-only mode: the exit code is the verdict, the output is the evidence.
type GateResult struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Passed   bool   `json:"passed"`
	Output   string `json:"output"`
}

// Check evaluates `pose check` (structural integrity gate). A failing gate
// is a legitimate result (Passed=false), not a tool error.
func (s Store) Check(ctx context.Context, strict bool) (*GateResult, error) {
	return s.runGate(ctx, []string{"check", modeFlag(strict)})
}

// LintSpec evaluates `pose lint-spec <slug>|--all` (spec content +
// lifecycle gate). Empty slug evaluates every spec.
func (s Store) LintSpec(ctx context.Context, slug string, strict bool) (*GateResult, error) {
	target := "--all"
	if slug != "" {
		if err := ValidateSlug(slug); err != nil {
			return nil, err
		}
		target = slug
	}
	return s.runGate(ctx, []string{"lint-spec", target, modeFlag(strict)})
}

func modeFlag(strict bool) string {
	if strict {
		return "--strict"
	}
	return "--tolerant"
}

func (s Store) runGate(ctx context.Context, args []string) (*GateResult, error) {
	out, exitCode, err := s.runCLIExit(ctx, args)
	if err != nil {
		return nil, err
	}
	return &GateResult{
		Command:  "pose " + strings.Join(args, " "),
		ExitCode: exitCode,
		Passed:   exitCode == 0,
		Output:   strings.TrimSpace(string(out)),
	}, nil
}

// runCLI executes a JSON-emitting, side-effect-free pose command; any
// non-zero exit is an error (these commands always succeed structurally).
func (s Store) runCLI(ctx context.Context, args []string) ([]byte, error) {
	executable, err := s.nativeExecutablePath()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = s.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("pose %s failed: %s", args[0], msg)
	}
	return stdout.Bytes(), nil
}

// runCLIExit executes a gate command capturing stdout+stderr together; a
// non-zero exit is returned as a verdict, not an error. Errors are reserved
// for execution failures (native executable missing, timeout).
func (s Store) runCLIExit(ctx context.Context, args []string) ([]byte, int, error) {
	executable, err := s.nativeExecutablePath()
	if err != nil {
		return nil, -1, err
	}
	ctx, cancel := context.WithTimeout(ctx, cliTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = s.Root
	var combined bytes.Buffer
	cmd.Stdout, cmd.Stderr = &combined, &combined
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return combined.Bytes(), exitErr.ExitCode(), nil
		}
		return nil, -1, fmt.Errorf("pose %s: %v", args[0], err)
	}
	return combined.Bytes(), 0, nil
}

func (s Store) nativeExecutablePath() (string, error) {
	if configured := os.Getenv("POSE_EXECUTABLE"); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("pose: POSE_EXECUTABLE must be absolute")
		}
		return configured, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("pose: resolving native executable: %w", err)
	}
	if strings.HasSuffix(executable, ".test") {
		return "", fmt.Errorf("pose: native executable unavailable in test process")
	}
	return executable, nil
}
