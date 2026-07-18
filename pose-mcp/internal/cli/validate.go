package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

func discoverValidationModules(root string) ([]validationModule, error) {
	ignored := map[string]bool{".git": true, "node_modules": true, "vendor": true, "target": true, "dist": true, "build": true, ".pose": true}
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

func validationCheckEnabled(module string, when validationWhen) bool {
	if when.FileExists != "" {
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileExists))); err != nil {
			return false
		}
	}
	if when.FileNotExists != "" {
		if _, err := os.Stat(filepath.Join(module, filepath.FromSlash(when.FileNotExists))); err == nil {
			return false
		}
	}
	return true
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
	mode, stackFilter, moduleFilter, reportTask := "", "", "", ""
	autoReport := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			mode = "strict"
		case "--tolerant":
			mode = "tolerant"
		case "--report":
			autoReport = true
		case "--stack", "--module", "--report-task":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				fmt.Fprintf(stderr, "Erro: %s exige um valor.\n", args[i])
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
			}
		default:
			fmt.Fprintf(stderr, "Erro: argumento inválido: %s\n", args[i])
			return 2
		}
	}
	if stackFilter != "" && !map[string]bool{"node": true, "go": true, "rust": true, "java": true, "contract": true}[stackFilter] {
		fmt.Fprintf(stderr, "Erro: --stack inválido: %s\n", stackFilter)
		return 2
	}
	if moduleFilter == ".." || strings.HasPrefix(moduleFilter, "../") || filepath.IsAbs(moduleFilter) {
		fmt.Fprintln(stderr, "Erro: --module deve permanecer dentro do projeto.")
		return 2
	}
	matrixPath := filepath.Join(root, ".pose", "indexes", "validation-matrix.json")
	raw, err := os.ReadFile(matrixPath)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: matriz de validação não encontrada em %s\n", matrixPath)
		return 2
	}
	matrix, err := parseValidationMatrix(raw)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: matriz de validação inválida: %v\n", err)
		return 2
	}
	if matrixHasLegacyChecks(matrix) {
		return delegate("pose-validate.sh", args, stdout, stderr)
	}
	if mode == "" {
		mode = matrix.Defaults.Mode
		if mode == "" {
			mode = "strict"
		}
	}
	modules, err := discoverValidationModules(root)
	if err != nil {
		fmt.Fprintf(stderr, "Erro: descobrir módulos: %v\n", err)
		return 1
	}
	failures := 0
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
		if override.Mode != "" && mode == matrix.Defaults.Mode {
			moduleMode = override.Mode
		}
		fmt.Fprintf(stdout, "[module] %s (%s, mode=%s)\n", module.Rel, stack, moduleMode)
		for _, check := range checks {
			if check.Program == "" || !validationCheckEnabled(module.Abs, check.When) {
				continue
			}
			executed++
			fmt.Fprintf(stdout, "  -> %s %s\n", check.Program, strings.Join(check.Args, " "))
			cmd := exec.CommandContext(context.Background(), check.Program, check.Args...)
			cmd.Dir = module.Abs
			cmd.Stdout, cmd.Stderr = stdout, stderr
			cmd.Env = os.Environ()
			for key, value := range check.Env {
				cmd.Env = append(cmd.Env, key+"="+value)
			}
			if err := cmd.Run(); err != nil && (check.Severity == "required" || check.Severity == "") {
				failures++
			}
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
		fmt.Fprintln(stdout, "Result: SUCCESS")
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
