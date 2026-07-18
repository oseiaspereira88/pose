package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func cmdInitWizard(root string, args []string, stdout, stderr io.Writer) int {
	yes := false
	if len(args) > 1 {
		return usageError(stderr, "Usage: pose init --wizard [--yes]")
	}
	if len(args) == 1 {
		if args[0] != "--yes" {
			return usageError(stderr, "Usage: pose init --wizard [--yes]")
		}
		yes = true
	}
	fmt.Fprintln(stdout, "== POSE init wizard ==")
	if rc := cmdInit(root, stdout, stderr); rc != 0 {
		return rc
	}
	matrixPath := filepath.Join(root, ".pose", "indexes", "validation-matrix.json")
	raw, e := os.ReadFile(matrixPath)
	if e != nil {
		fmt.Fprintln(stderr, "pose init --wizard: validation-matrix.json missing")
		return 1
	}
	var matrix map[string]any
	if e = json.Unmarshal(raw, &matrix); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	mods, e := discoverValidationModules(root)
	if e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	if len(mods) == 0 {
		fmt.Fprintln(stdout, "[INFO] no stack modules detected")
		return 0
	}
	overrides, ok := matrix["moduleOverrides"].(map[string]any)
	if !ok {
		overrides = map[string]any{}
		matrix["moduleOverrides"] = overrides
	}
	reader := bufio.NewReader(os.Stdin)
	for _, m := range mods {
		accept := yes
		if !yes {
			fmt.Fprintf(stdout, "Include %q (%s) in validation matrix? [Y/n] ", m.Rel, m.Stack)
			line, _ := reader.ReadString('\n')
			line = strings.ToLower(strings.TrimSpace(line))
			accept = line != "n" && line != "no" && line != "nao" && line != "não"
		}
		if !accept {
			continue
		}
		if _, exists := overrides[m.Rel]; exists {
			fmt.Fprintf(stdout, "[INFO] already present: %s\n", m.Rel)
			continue
		}
		overrides[m.Rel] = map[string]any{"stack": m.Stack, "mode": "tolerant"}
		fmt.Fprintf(stdout, "[OK] moduleOverrides + %s\n", m.Rel)
	}
	out, _ := json.MarshalIndent(matrix, "", "  ")
	out = append(out, '\n')
	if e = writeAtomic(matrixPath, out, 0o644); e != nil {
		fmt.Fprintln(stderr, e)
		return 1
	}
	_ = cmdIndex(root, nil, io.Discard, stderr)
	fmt.Fprintln(stdout, "[INFO] indexes regenerated. Run: pose validate --tolerant")
	return 0
}
