package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type followup struct {
	Spec        string `json:"spec"`
	Disposition string `json:"raw_disposition"`
	Text        string `json:"text"`
}

var followupBullet = regexp.MustCompile(`^\s*-\s+(.*\S)\s*$`)
var followupDisposition = regexp.MustCompile(`^\[\s*([a-z-]+)(?:\s*:\s*[^\]]+)?\s*\]\s*(.*)$`)

func cmdFollowups(root string, args []string, stdout, stderr io.Writer) int {
	all, jsonOut := false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--open":
		case "--all":
			all = true
		case "--json":
			jsonOut = true
		case "--similarity":
			if i+1 >= len(args) {
				fmt.Fprintln(stderr, "Erro: --similarity exige inteiro 0..100.")
				return 2
			}
			i++
			value, err := strconv.Atoi(args[i])
			if err != nil || value < 0 || value > 100 {
				fmt.Fprintln(stderr, "Erro: --similarity exige inteiro 0..100.")
				return 2
			}
		default:
			fmt.Fprintf(stderr, "Erro: opção desconhecida: %s\n", arg)
			return 2
		}
	}
	entries := []followup{}
	paths, _ := filepath.Glob(filepath.Join(root, ".pose", "specs", "*", "spec.md"))
	sort.Strings(paths)
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		inFinal, inFollowups := false, false
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "## ") {
				inFinal = strings.Contains(strings.ToLower(line), "final report")
				inFollowups = false
				continue
			}
			if inFinal && strings.HasPrefix(line, "### ") {
				inFollowups = strings.Contains(strings.ToLower(line), "follow-up")
				continue
			}
			if !inFollowups {
				continue
			}
			m := followupBullet.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			text, disposition := m[1], ""
			if d := followupDisposition.FindStringSubmatch(text); d != nil {
				disposition, text = d[1], d[2]
			}
			if text == "" {
				continue
			}
			if all || disposition == "" || disposition == "open" {
				entries = append(entries, followup{filepath.Base(filepath.Dir(path)), disposition, text})
			}
		}
	}
	if jsonOut {
		_ = json.NewEncoder(stdout).Encode(map[string]any{"items": entries, "total": len(entries)})
		return 0
	}
	for _, entry := range entries {
		disp := entry.Disposition
		if disp == "" {
			disp = "open"
		}
		fmt.Fprintf(stdout, "- %s [%s]\n    %s\n", entry.Spec, disp, entry.Text)
	}
	return 0
}
