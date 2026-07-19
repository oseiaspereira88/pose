package cli

import (
	"context"
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
		Mode string `json:"mode"`
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
	jsonOut, junitOut, sarifOut := "", "", ""
	autoReport := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--report":
			autoReport = true
		case "--stack", "--module", "--report-task", "--json", "--junit", "--sarif":
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
			}
		default:
			fmt.Fprintf(stderr, cliText(locale, "Error: invalid argument: %s\n", "Erro: argumento inválido: %s\n"), args[i])
			return 2
		}
	}
	for _, out := range []string{jsonOut, junitOut, sarifOut} {
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
			if reason := validationSkipReason(module.Abs, check.When); reason != "" {
				result.Outcome, result.SkipReason = "skipped", reason
				run.Counts.Skipped++
				run.Checks = append(run.Checks, result)
				continue
			}
			executed++
			run.Counts.Executed++
			fmt.Fprintf(stdout, "  -> %s %s\n", check.Program, strings.Join(check.Args, " "))
			capture := &tailBuffer{capacity: 4096}
			cmd := exec.CommandContext(context.Background(), check.Program, check.Args...)
			cmd.Dir = module.Abs
			cmd.Stdout = io.MultiWriter(stdout, capture)
			cmd.Stderr = io.MultiWriter(stderr, capture)
			cmd.Env = os.Environ()
			for key, value := range check.Env {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
			started := time.Now()
			err := cmd.Run()
			result.DurationSeconds = time.Since(started).Seconds()
			result.Output = redactSecrets(capture.String(), check.Env)
			var exitErr *exec.ExitError
			switch {
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
