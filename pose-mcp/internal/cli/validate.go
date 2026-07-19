package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type validationWhen struct {
	FileExists    string `json:"fileExists"`
	FileNotExists string `json:"fileNotExists"`
}

type validationCheck struct {
	Name     string            `json:"name"`
	Command  string            `json:"command"`
	Program  string            `json:"program"`
	Args     []string          `json:"args"`
	Env      map[string]string `json:"env"`
	Severity string            `json:"severity"`
	When     validationWhen    `json:"when"`
	// Runtime guardrails (spec pose-validation-runtime-guardrails).
	TimeoutSeconds int    `json:"timeoutSeconds"` // 0 = defaults.timeoutSeconds (600)
	Isolation      string `json:"isolation"`      // "" | "required" — required never runs locally
}

type validationStack struct {
	Checks []validationCheck `json:"checks"`
}

type validationOverride struct {
	Stack                string            `json:"stack"`
	Mode                 string            `json:"mode"`
	Checks               []validationCheck `json:"checks"`
	ReplaceDefaultChecks bool              `json:"replaceDefaultChecks"`
}

type validationMatrix struct {
	Defaults struct {
		Mode           string `json:"mode"`
		TimeoutSeconds int    `json:"timeoutSeconds"` // safe default 600 when 0
		MaxOutputBytes int    `json:"maxOutputBytes"` // safe default 1 MiB when 0
	} `json:"defaults"`
	Stacks          map[string]validationStack    `json:"stacks"`
	ModuleOverrides map[string]validationOverride `json:"moduleOverrides"`
}

func parseValidationMatrix(raw []byte) (validationMatrix, error) {
	var matrix validationMatrix
	err := json.Unmarshal(raw, &matrix)
	return matrix, err
}

func parseStructuredChecks(raw []byte) ([]validationCheck, error) {
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		return nil, err
	}
	var checks []validationCheck
	for _, stack := range matrix.Stacks {
		for _, check := range stack.Checks {
			if check.Program != "" {
				checks = append(checks, check)
			}
		}
	}
	return checks, nil
}

type validationModule struct {
	Rel, Abs, Stack string
}

func confinedRelativePath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return true
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	return !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator))
}

func discoverValidationModules(root string) ([]validationModule, error) {
	ignored := map[string]bool{".git": true, "node_modules": true, "vendor": true, ".venv": true, ".pnpm-store": true, "target": true, "dist": true, "build": true, ".next": true, "coverage": true, ".pose": true}
	byPath := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && ignored[entry.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		stack := ""
		switch entry.Name() {
		case "package.json":
			stack = "node"
		case "go.mod":
			stack = "go"
		case "Cargo.toml":
			stack = "rust"
		case "pom.xml", "build.gradle", "build.gradle.kts":
			stack = "java"
		}
		if stack != "" {
			byPath[filepath.Dir(path)] = stack
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	modules := make([]validationModule, 0, len(byPath))
	for abs, stack := range byPath {
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return nil, err
		}
		modules = append(modules, validationModule{filepath.ToSlash(rel), abs, stack})
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Rel < modules[j].Rel })
	return modules, nil
}

// validationSkipReason returns the deterministic selection reason when a
// check must be skipped ("" = run it). Every skip is recorded with its
// reason in the structured result (spec pose-structured-validation-results).
func validationSkipReason(module string, when validationWhen) string {
	if when.FileExists != "" {
		if !confinedRelativePath(when.FileExists) {
			return "when.fileExists escapes the module: " + when.FileExists
		}
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileExists))); err != nil {
			return "when.fileExists not met: " + when.FileExists
		}
	}
	if when.FileNotExists != "" {
		if !confinedRelativePath(when.FileNotExists) {
			return "when.fileNotExists escapes the module: " + when.FileNotExists
		}
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileNotExists))); err == nil {
			return "when.fileNotExists violated: " + when.FileNotExists + " exists"
		}
	}
	return ""
}

func validateStructuredMatrixPaths(matrix validationMatrix) error {
	validate := func(scope string, checks []validationCheck) error {
		for index, check := range checks {
			for field, path := range map[string]string{"fileExists": check.When.FileExists, "fileNotExists": check.When.FileNotExists} {
				if !confinedRelativePath(path) {
					return fmt.Errorf("%s.checks[%d].when.%s must remain inside its module", scope, index, field)
				}
			}
		}
		return nil
	}
	for name, stack := range matrix.Stacks {
		if err := validate("stacks."+name, stack.Checks); err != nil {
			return err
		}
	}
	for name, override := range matrix.ModuleOverrides {
		if err := validate("moduleOverrides."+name, override.Checks); err != nil {
			return err
		}
	}
	return nil
}

func matrixHasLegacyChecks(matrix validationMatrix) bool {
	for _, stack := range matrix.Stacks {
		for _, check := range stack.Checks {
			if check.Command != "" {
				return true
			}
		}
	}
	for _, override := range matrix.ModuleOverrides {
		for _, check := range override.Checks {
			if check.Command != "" {
				return true
			}
		}
	}
	return false
}

func cmdValidate(root string, args []string, stdout, stderr io.Writer) int {
	locale := cliLocaleValue()
	mode, stackFilter, moduleFilter, reportTask := "", "", "", ""
	jsonOut, junitOut, sarifOut, planOut := "", "", "", ""
	changedFrom, changedTo := "", ""
	explain := false
	autoReport := false
	var isolationChecks []checkResult
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--report":
			autoReport = true
		case "--explain":
			explain = true
		case "--stack", "--module", "--report-task", "--json", "--junit", "--sarif", "--emit-plan", "--changed-from", "--changed-to":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				fmt.Fprintf(stderr, cliText(locale, "Error: %s requires a value.\n", "Erro: %s exige um valor.\n"), args[i])
				return 2
			}
			i++
			switch args[i-1] {
			case "--stack":
				stackFilter = args[i]
			case "--module":
				moduleFilter = filepath.ToSlash(filepath.Clean(args[i]))
			case "--report-task":
				reportTask = args[i]
			case "--json":
				jsonOut = args[i]
			case "--junit":
				junitOut = args[i]
			case "--sarif":
				sarifOut = args[i]
			case "--emit-plan":
				planOut = args[i]
			case "--changed-from":
				changedFrom = args[i]
			case "--changed-to":
				changedTo = args[i]
			}
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	for _, out := range []string{jsonOut, junitOut, sarifOut, planOut} {
		if out != "" && !confinedRelativePath(out) {
			fmt.Fprintln(stderr, cliText(locale, "Error: result output paths must remain inside the project.", "Erro: paths de saída de resultado devem permanecer dentro do projeto."))
			return 2
		}
	}
	if stackFilter != "" && !map[string]bool{"node": true, "go": true, "rust": true, "java": true, "contract": true}[stackFilter] {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid --stack: %s\n", "Erro: --stack inválido: %s\n"), stackFilter)
		return 2
	}
	if moduleFilter == ".." || strings.HasPrefix(moduleFilter, "../") || filepath.IsAbs(moduleFilter) {
		fmt.Fprintln(stderr, cliText(locale, "Error: --module must remain inside the project.", "Erro: --module deve permanecer dentro do projeto."))
		return 2
	}
	matrixPath := filepath.Join(root, ".pose", "indexes", "validation-matrix.json")
	raw, err := os.ReadFile(matrixPath)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: validation matrix not found at %s\n", "Erro: matriz de validação não encontrada em %s\n"), matrixPath)
		return 2
	}
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid validation matrix: %v\n", "Erro: matriz de validação inválida: %v\n"), err)
		return 2
	}
	if err := validateStructuredMatrixPaths(matrix); err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: invalid validation matrix path: %v\n", "Erro: path inválido na matriz de validação: %v\n"), err)
		return 2
	}
	if matrixHasLegacyChecks(matrix) {
		fmt.Fprintln(stderr, cliText(locale, "Error: legacy shell 'command' checks are unsupported; migrate each check to program + args + env.", "Erro: checks shell legados em 'command' não são suportados; migre cada check para program + args + env."))
		return 2
	}
	if mode == "" {
		mode = matrix.Defaults.Mode
		if mode == "" {
			mode = "strict"
		}
	}
	modules, err := discoverValidationModules(root)
	if err != nil {
		fmt.Fprintf(stderr, cliText(locale, "Error: discovering modules: %v\n", "Erro: descobrir módulos: %v\n"), err)
		return 1
	}
	knownModules := map[string]bool{}
	for _, module := range modules {
		knownModules[module.Rel] = true
	}
	for rel, override := range matrix.ModuleOverrides {
		clean := filepath.ToSlash(filepath.Clean(rel))
		if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(rel) {
			fmt.Fprintf(stderr, cliText(locale, "Error: moduleOverrides contains a path outside the project: %s\n", "Erro: moduleOverrides contém path fora do projeto: %s\n"), rel)
			return 2
		}
		if knownModules[clean] {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(clean))
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			stack := override.Stack
			if stack == "" {
				stack = "contract"
			}
			modules = append(modules, validationModule{Rel: clean, Abs: abs, Stack: stack})
			knownModules[clean] = true
		}
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].Rel < modules[j].Rel })
	var scope *scopeSelection
	if changedTo != "" && changedFrom == "" {
		fmt.Fprintln(stderr, cliText(locale, "Error: --changed-to requires --changed-from.", "Erro: --changed-to exige --changed-from."))
		return 2
	}
	if changedFrom != "" {
		sel, err := computeChangedScope(root, changedFrom, changedTo, modules)
		if err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: changed-scope selection: %v\n", "Erro: seleção por escopo alterado: %v\n"), err)
			return 2
		}
		scope = &sel
		if explain {
			fmt.Fprintf(stdout, "[changed-scope] %s: %d changed file(s), %d/%d module(s) selected\n", sel.rangeLabel(), len(sel.Changed), len(sel.Selected), len(modules))
			for _, m := range modules {
				if reason, ok := sel.Selected[m.Rel]; ok {
					fmt.Fprintf(stdout, "  + %s: %s\n", m.Rel, reason)
				} else {
					fmt.Fprintf(stdout, "  - %s: not affected by %s\n", m.Rel, sel.rangeLabel())
				}
			}
		}
	}
	run := validationRunResult{
		SchemaVersion: validationResultSchema,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Mode:          mode,
		StackFilter:   stackFilter,
		ModuleFilter:  moduleFilter,
		Checks:        []checkResult{},
	}
	failures := 0
	optionalFailures := 0
	executed := 0
	for _, module := range modules {
		override := matrix.ModuleOverrides[module.Rel]
		stack := module.Stack
		if override.Stack != "" {
			stack = override.Stack
		}
		if stackFilter != "" && stack != stackFilter || moduleFilter != "" && module.Rel != moduleFilter {
			continue
		}
		checks := append([]validationCheck(nil), matrix.Stacks[stack].Checks...)
		if override.ReplaceDefaultChecks {
			checks = nil
		}
		checks = append(checks, override.Checks...)
		if scope != nil {
			if _, selected := scope.Selected[module.Rel]; !selected {
				// R3: every unselected check keeps a machine-readable reason.
				reason := "changed-scope: module not affected by " + scope.rangeLabel()
				for _, check := range checks {
					if check.Program == "" {
						continue
					}
					severity := check.Severity
					if severity == "" {
						severity = "required"
					}
					run.Counts.Skipped++
					run.Checks = append(run.Checks, checkResult{
						ID: module.Rel + "/" + stack + "/" + check.Name, Module: module.Rel,
						Stack: stack, Name: check.Name, Program: check.Program,
						Args: check.Args, Env: redactedEnv(check.Env), Severity: severity,
						Outcome: "skipped", SkipReason: reason,
					})
				}
				continue
			}
		}
		moduleMode := mode
		if override.Mode != "" {
			moduleMode = override.Mode
		}
		fmt.Fprintf(stdout, "[module] %s (%s, mode=%s)\n", module.Rel, stack, moduleMode)
		for _, check := range checks {
			if check.Program == "" {
				continue
			}
			severity := check.Severity
			if severity == "" {
				severity = "required"
			}
			result := checkResult{
				ID: module.Rel + "/" + stack + "/" + check.Name, Module: module.Rel,
				Stack: stack, Name: check.Name, Program: check.Program,
				Args: check.Args, Env: redactedEnv(check.Env), Severity: severity,
			}
			if check.Isolation == "required" {
				// The local CLI never weakens its boundary: isolated
				// execution is delegated to the Harness via --emit-plan.
				result.Outcome = "skipped"
				result.Isolation = "required"
				result.SkipReason = "requires isolated execution (harness) — include via --emit-plan"
				run.Counts.Skipped++
				run.Checks = append(run.Checks, result)
				isolationChecks = append(isolationChecks, result)
				fmt.Fprintf(stdout, "  -- %s: skipped (%s)\n", check.Name, result.SkipReason)
				continue
			}
			if reason := validationSkipReason(module.Abs, check.When); reason != "" {
				result.Outcome, result.SkipReason = "skipped", reason
				run.Counts.Skipped++
				run.Checks = append(run.Checks, result)
				continue
			}
			executed++
			run.Counts.Executed++
			fmt.Fprintf(stdout, "  -> %s %s\n", check.Program, strings.Join(check.Args, " "))
			timeout := check.TimeoutSeconds
			if timeout <= 0 {
				timeout = matrix.Defaults.TimeoutSeconds
			}
			if timeout <= 0 {
				timeout = 600 // documented safe default
			}
			maxOutput := matrix.Defaults.MaxOutputBytes
			if maxOutput <= 0 {
				maxOutput = 1 << 20 // 1 MiB documented safe default
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
			capture := &tailBuffer{capacity: 4096}
			limiter := &outputLimiter{limit: maxOutput, cancel: cancel}
			cmd := exec.CommandContext(ctx, check.Program, check.Args...)
			setProcessGroup(cmd)
			cmd.Cancel = func() error { return killProcessGroup(cmd) }
			cmd.Dir = module.Abs
			cmd.Stdout = io.MultiWriter(stdout, capture, limiter)
			cmd.Stderr = io.MultiWriter(stderr, capture, limiter)
			cmd.Env = os.Environ()
			for key, value := range check.Env {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
			started := time.Now()
			err := cmd.Run()
			cancel()
			result.DurationSeconds = time.Since(started).Seconds()
			result.Output = redactSecrets(capture.String(), check.Env)
			var exitErr *exec.ExitError
			switch {
			case limiter.exceeded:
				// Explicit guardrail state: the check flooded output and was
				// cancelled (never conflated with a normal check failure).
				result.Outcome, result.LimitState = "error", "output-limit"
				result.Output = fmt.Sprintf("output limit exceeded (%d bytes) — process group terminated", maxOutput)
				run.Counts.Errored++
				if severity == "required" {
					failures++
				} else {
					optionalFailures++
				}
			case ctx.Err() == context.DeadlineExceeded:
				result.Outcome, result.LimitState = "error", "timeout"
				result.Output = fmt.Sprintf("timeout after %ds — process group terminated", timeout)
				run.Counts.Errored++
				if severity == "required" {
					failures++
				} else {
					optionalFailures++
				}
			case err == nil:
				result.Outcome = "pass"
				run.Counts.Passed++
			case errors.As(err, &exitErr):
				code := exitErr.ExitCode()
				result.Outcome, result.ExitCode = "fail", &code
				if severity == "required" {
					failures++
					run.Counts.Failed++
				} else {
					optionalFailures++
					run.Counts.OptionalFailed++
				}
			default:
				// Infrastructure failure: the tool never ran (R3 keeps this
				// distinguishable from a real check failure).
				result.Outcome = "error"
				result.Output = redactSecrets(err.Error(), check.Env)
				run.Counts.Errored++
				if severity == "required" {
					failures++
				} else {
					optionalFailures++
				}
			}
			run.Checks = append(run.Checks, result)
		}
	}
	if executed == 0 {
		fmt.Fprintln(stdout, "No modules/checks matched the matrix and filters.")
	}
	result := "pass"
	if failures > 0 {
		result = "fail"
		fmt.Fprintln(stdout, "Result: FAILURE (required check failed)")
	} else {
		if optionalFailures > 0 {
			result = "partial"
			fmt.Fprintf(stdout, "Warning: %d optional check(s) failed.\n", optionalFailures)
		}
		fmt.Fprintln(stdout, "Result: SUCCESS")
	}
	run.Outcome = result
	for name, writer := range map[string]struct {
		path  string
		write func(string, validationRunResult) error
	}{
		"json":  {jsonOut, writeValidationJSON},
		"junit": {junitOut, writeValidationJUnit},
		"sarif": {sarifOut, writeValidationSARIF},
	} {
		if writer.path == "" {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(writer.path))
		if err := writer.write(target, run); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: writing %s result: %v\n", "Erro: escrevendo resultado %s: %v\n"), name, err)
			return 1
		}
		fmt.Fprintf(stdout, "%s: %s\n", name, target)
	}
	if planOut != "" {
		digest := sha256.Sum256(raw)
		head := ""
		if out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
			head = strings.TrimSpace(string(out))
		}
		plan := executionPlan{
			SchemaVersion: validationResultSchema,
			GeneratedAt:   run.GeneratedAt,
			ProjectID:     filepath.Base(root),
			Spec:          reportTask,
			GitHead:       head,
			MatrixSHA256:  hex.EncodeToString(digest[:]),
			Checks:        isolationChecks,
			Approval:      planApproval{Required: true},
		}
		if plan.Checks == nil {
			plan.Checks = []checkResult{}
		}
		target := filepath.Join(root, filepath.FromSlash(planOut))
		if err := writeExecutionPlan(target, plan); err != nil {
			fmt.Fprintf(stderr, cliText(locale, "Error: writing execution plan: %v\n", "Erro: escrevendo plano de execução: %v\n"), err)
			return 1
		}
		fmt.Fprintf(stdout, "plan: %s (%d isolated check(s); approval required before Harness execution)\n", target, len(plan.Checks))
	}
	if autoReport {
		if reportTask == "" {
			reportTask = "validate-native"
		}
		_ = cmdReport(root, []string{"--task", reportTask, "--outcome", result, "--context", "auto-validate", "--validation-profile", mode}, io.Discard, stderr)
	}
	if failures > 0 {
		return 1
	}
	return 0
}
