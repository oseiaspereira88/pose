package cli

// pose stacks: read-only, offline catalog inspection (spec
// pose-stack-catalog-expansion R1/R2). Reports, per directory, every
// matched profile with its manager, prerequisite availability and
// confidence (medium when multiple managers' markers conflict) — never
// mutates the matrix, never executes a project file.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func cmdStacks(root string, args []string, stdout, stderr io.Writer) int {
	target := root
	jsonOut := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 >= len(args) {
				return usageError(stderr, "pose stacks: --path requires a value")
			}
			i++
			clean := filepath.ToSlash(filepath.Clean(args[i]))
			if !confinedRelativePath(clean) {
				fmt.Fprintln(stderr, "pose stacks: --path must remain inside the project")
				return 2
			}
			target = filepath.Join(root, filepath.FromSlash(clean))
		case "--json":
			jsonOut = true
		default:
			return usageError(stderr, "Usage: pose stacks [--path dir] [--json]")
		}
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		fmt.Fprintf(stderr, "pose stacks: %v\n", err)
		return 2
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	detections := detectStackProfiles(names)
	if jsonOut {
		if err := json.NewEncoder(stdout).Encode(map[string]any{"path": target, "detections": detections}); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if len(detections) == 0 {
		fmt.Fprintln(stdout, "(no stack markers found in this directory)")
		return 0
	}
	byStack := map[string][]stackDetection{}
	var stacks []string
	for _, d := range detections {
		if _, ok := byStack[d.Profile.Stack]; !ok {
			stacks = append(stacks, d.Profile.Stack)
		}
		byStack[d.Profile.Stack] = append(byStack[d.Profile.Stack], d)
	}
	sort.Strings(stacks)
	for _, s := range stacks {
		fmt.Fprintf(stdout, "# %s\n", s)
		for _, d := range byStack[s] {
			manager := d.Profile.Manager
			if manager == "" {
				manager = d.Profile.ID
			}
			marker := "winner"
			if !d.Winner {
				marker = "shadowed"
			}
			prereq := "found"
			if !d.PrerequisiteFound {
				prereq = "MISSING"
			}
			fmt.Fprintf(stdout, "  - %s (%s): marker=%s confidence=%s prerequisite=%s(%s)\n",
				manager, marker, d.Profile.Marker, d.Confidence, d.Profile.Prerequisite, prereq)
		}
		fmt.Fprintf(stdout, "  override: %s\n", byStack[s][0].OverrideHint)
	}
	return 0
}
