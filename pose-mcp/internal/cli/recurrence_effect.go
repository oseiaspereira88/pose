package cli

// Recurrence effectiveness (spec pose-recurrence-effectiveness): escalations
// register an intervention (rule/workflow/spec) with an observation window;
// the engine compares recurrence rate — and cost/duration when recorded —
// before and after, entirely from append-only local history. Aggregation is
// by task/context only; individuals are never ranked. Missing telemetry
// produces partial metrics, never fabricated ones.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const interventionSchema = 1

type intervention struct {
	Schema     int    `json:"schema"`
	At         string `json:"at"` // RFC3339 UTC
	TaskSlug   string `json:"task_slug"`
	Ref        string `json:"ref"` // rule:<name> | workflow:<name> | spec:<slug>
	WindowDays int    `json:"window_days"`
	Rationale  string `json:"rationale"`
	Author     string `json:"author"`
}

var interventionRefRE = regexp.MustCompile(`^(rule|workflow|spec):[a-z0-9][a-z0-9._-]*$`)

func interventionsPath(root string) string {
	return filepath.Join(root, ".pose", "reports", "history", "interventions.jsonl")
}

func loadInterventions(root string) ([]intervention, error) {
	raw, err := os.ReadFile(interventionsPath(root))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []intervention
	for i, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e intervention
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("line %d: %v", i+1, err)
		}
		if e.Schema != interventionSchema || e.TaskSlug == "" || !interventionRefRE.MatchString(e.Ref) || e.WindowDays < 1 {
			return nil, fmt.Errorf("line %d: invalid intervention record", i+1)
		}
		out = append(out, e)
	}
	return out, nil
}

type effectSide struct {
	failures  int
	total     int
	durSum    float64
	durCount  int
	costSum   float64
	costCount int
}

func (s *effectSide) add(r historyRecord) {
	s.total++
	if r.Outcome != "pass" {
		s.failures++
	}
	if r.DurationSeconds != nil {
		s.durSum += *r.DurationSeconds
		s.durCount++
	}
	if r.CostUSD != nil {
		s.costSum += *r.CostUSD
		s.costCount++
	}
}

func cmdRecurrenceEffect(root string, args []string, stdout, stderr io.Writer) int {
	register := false
	task, ref, rationale, author := "", "", "", ""
	windowDays, minSample := 30, 3
	failIneffective := false
	for i := 0; i < len(args); i++ {
		next := func() (string, bool) {
			if i+1 >= len(args) {
				return "", false
			}
			i++
			return args[i], true
		}
		switch args[i] {
		case "--register":
			register = true
		case "--task":
			task, _ = next()
		case "--ref":
			ref, _ = next()
		case "--rationale":
			rationale, _ = next()
		case "--author":
			author, _ = next()
		case "--window-days":
			v, ok := next()
			if !ok {
				return usageError(stderr, "pose recurrence-effect: --window-days requires an integer")
			}
			if _, err := fmt.Sscanf(v, "%d", &windowDays); err != nil || windowDays < 1 {
				return usageError(stderr, "pose recurrence-effect: --window-days requires an integer > 0")
			}
		case "--min-sample":
			v, ok := next()
			if !ok {
				return usageError(stderr, "pose recurrence-effect: --min-sample requires an integer")
			}
			if _, err := fmt.Sscanf(v, "%d", &minSample); err != nil || minSample < 1 {
				return usageError(stderr, "pose recurrence-effect: --min-sample requires an integer > 0")
			}
		case "--fail-ineffective":
			failIneffective = true
		default:
			return usageError(stderr, "Usage: pose recurrence-effect [--register --task <slug> --ref rule:<n>|workflow:<n>|spec:<n> --window-days N --rationale <text> --author @alias] [--min-sample N] [--fail-ineffective]")
		}
	}

	if register {
		if task == "" || rationale == "" || !interventionRefRE.MatchString(ref) || !amendAliasRE.MatchString(author) {
			return usageError(stderr, "pose recurrence-effect --register: requires --task <slug>, --ref rule:|workflow:|spec:<name>, --rationale <text> and --author @alias")
		}
		if ref[:5] == "spec:" {
			slug := strings.TrimPrefix(ref, "spec:")
			if _, err := os.Stat(filepath.Join(root, ".pose", "specs", slug, "spec.md")); err != nil {
				fmt.Fprintf(stderr, "pose recurrence-effect: intervention %s does not resolve to an existing spec\n", ref)
				return 1
			}
		}
		e := intervention{Schema: interventionSchema, At: time.Now().UTC().Format(time.RFC3339),
			TaskSlug: task, Ref: ref, WindowDays: windowDays, Rationale: rationale, Author: author}
		line, _ := json.Marshal(e)
		if err := os.MkdirAll(filepath.Dir(interventionsPath(root)), 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		f, err := os.OpenFile(interventionsPath(root), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		defer f.Close()
		if _, err := f.Write(append(line, '\n')); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintf(stdout, "Intervention registered: %s → %s (window %dd)\n", task, ref, windowDays)
		return 0
	}

	interventions, err := loadInterventions(root)
	if err != nil {
		fmt.Fprintf(stderr, "pose recurrence-effect: %s: %v\n", interventionsPath(root), err)
		return 1
	}
	records, _ := readHistory(root, stderr)
	now := time.Now().UTC()
	fmt.Fprintln(stdout, "# POSE recurrence effectiveness — before/after per intervention (append-only history)")
	if len(interventions) == 0 {
		fmt.Fprintln(stdout, "(no interventions registered — use pose recurrence-effect --register after an escalation)")
	}
	ineffective := 0
	for _, iv := range interventions {
		at, ok := parseHistoryTime(iv.At)
		if !ok {
			continue
		}
		window := time.Duration(iv.WindowDays) * 24 * time.Hour
		var before, after effectSide
		for _, r := range records {
			if r.TaskSlug != iv.TaskSlug {
				continue
			}
			t, ok := parseHistoryTime(r.GeneratedAt)
			if !ok {
				continue
			}
			switch {
			case t.Before(at) && t.After(at.Add(-window)):
				before.add(r)
			case t.After(at) && t.Before(at.Add(window)):
				after.add(r)
			}
		}
		elapsed := now.Sub(at)
		var warnings []string
		if before.total+after.total < minSample {
			warnings = append(warnings, fmt.Sprintf("insufficient sample (%d < %d events)", before.total+after.total, minSample))
		}
		if elapsed < window {
			warnings = append(warnings, fmt.Sprintf("observation window incomplete (%dd of %dd)", int(elapsed.Hours()/24), iv.WindowDays))
		}
		verdict := "INCONCLUSIVE"
		if len(warnings) == 0 {
			if after.failures < before.failures {
				verdict = "EFFECTIVE"
			} else {
				verdict = "INEFFECTIVE"
				ineffective++
			}
		}
		fmt.Fprintf(stdout, "- %s → %s [%s] failures before:%d/%d after:%d/%d window:%dd\n",
			iv.TaskSlug, iv.Ref, verdict, before.failures, before.total, after.failures, after.total, iv.WindowDays)
		if before.durCount > 0 || after.durCount > 0 || before.costCount > 0 || after.costCount > 0 {
			line := "    telemetry:"
			if before.durCount > 0 || after.durCount > 0 {
				line += fmt.Sprintf(" avg_duration_s before:%s after:%s", effectAvg(before.durSum, before.durCount), effectAvg(after.durSum, after.durCount))
			}
			if before.costCount > 0 || after.costCount > 0 {
				line += fmt.Sprintf(" avg_cost_usd before:%s after:%s", effectAvg(before.costSum, before.costCount), effectAvg(after.costSum, after.costCount))
			}
			fmt.Fprintln(stdout, line)
		} else {
			fmt.Fprintln(stdout, "    telemetry: partial (no duration/cost recorded — see pose report --duration-seconds/--cost-usd)")
		}
		sort.Strings(warnings)
		for _, w := range warnings {
			fmt.Fprintf(stdout, "    warning: %s\n", w)
		}
		if verdict == "INEFFECTIVE" {
			fmt.Fprintln(stdout, "    action: reopen or spawn a governed follow-up (recurrence-escalation workflow) — creating the escalation was not the success condition")
		}
	}
	fmt.Fprintf(stdout, "effect.interventions=%d\neffect.ineffective=%d\n", len(interventions), ineffective)
	if failIneffective && ineffective > 0 {
		return 1
	}
	return 0
}

func effectAvg(sum float64, count int) string {
	if count == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f", sum/float64(count))
}
