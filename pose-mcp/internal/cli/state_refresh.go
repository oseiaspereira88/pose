package cli

// Shared refresh engine for `pose state` (spec pose-project-state-artifact)
// and the post-event hook consumer (spec pose-project-state-refresh-contract
// R2/R4/R5/R6): one function renders the artifact, whether the caller wants
// every derived section recomputed (manual/CI refresh) or only the sections
// one event affects (hook-triggered partial refresh).

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// refreshLogEntry is one append-only .pose/state/refresh-log.jsonl record
// (R4): trigger, target, result, duration and the hash of every section
// that actually changed — never section content (Segurança: metadata only).
type refreshLogEntry struct {
	At              string   `json:"at"`
	Trigger         string   `json:"trigger"` // manual | ci | <event kind>
	Target          string   `json:"target,omitempty"`
	DedupKey        string   `json:"dedup_key,omitempty"`
	Result          string   `json:"result"` // ok | failed | skipped
	DurationMS      int64    `json:"duration_ms"`
	ChangedSections []string `json:"changed_sections,omitempty"`
	Error           string   `json:"error,omitempty"`
	Directed        bool     `json:"directed,omitempty"`
}

func stateRefreshLogPath(root string) string {
	return filepath.Join(root, ".pose", "state", "refresh-log.jsonl")
}

// dedupKey fingerprints (kind, target, commit) — R6: the same event
// processed twice (retry, replay) must not repeat refresh work. Empty
// commit still produces a stable key (manual/CI refreshes dedup on
// kind+target only, which is fine since they carry no target).
func dedupKey(kind, target, commit string) string {
	sum := sha256.Sum256([]byte(kind + "\x00" + target + "\x00" + commit))
	return hex.EncodeToString(sum[:])[:16]
}

// recentlyProcessed scans the last `window` refresh-log entries for a
// matching dedup key. Bounded scan: refresh-log is append-only and could
// grow large over a project's lifetime; only recent history matters for
// catching an immediate retry/replay.
func recentlyProcessed(root, key string, window int) bool {
	if key == "" {
		return false
	}
	entries, err := readRefreshLog(stateRefreshLogPath(root))
	if err != nil || len(entries) == 0 {
		return false
	}
	start := 0
	if len(entries) > window {
		start = len(entries) - window
	}
	for _, e := range entries[start:] {
		if e.DedupKey == key && e.Result != "failed" {
			return true
		}
	}
	return false
}

func appendRefreshLog(root string, entry refreshLogEntry) error {
	path := stateRefreshLogPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func readRefreshLog(path string) ([]refreshLogEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []refreshLogEntry
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var entry refreshLogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

// refreshOptions parametrizes one refresh run.
type refreshOptions struct {
	Trigger  string          // manual | ci | <event kind>
	Target   string          // event-specific identity (spec slug, request_id, ...); empty for manual/ci
	Only     map[string]bool // nil/empty = recompute every derived section
	Directed *directedHit    // non-nil when a components_hit result should drive the Arquitetura section
}

type directedHit struct {
	summary string // rendered Arquitetura body when a directed hit is available
}

// runRefresh renders the artifact, recomputing only the requested derived
// sections (or all of them when Only is empty) while always preserving
// curated sections and unrequested derived sections byte-for-byte from the
// existing artifact. Returns (nil, nil) when the artifact does not exist —
// additive no-op (spec Compatibilidade) — except when mustExist is false,
// used by init.
func runRefresh(root string, opts refreshOptions, mustExist bool) (*refreshLogEntry, error) {
	started := time.Now()
	store := pose.Store{Root: root}
	exists := store.HasProjectState()
	if !exists {
		if mustExist {
			return nil, fmt.Errorf("project state not found: %s (run `pose state init` first)", store.StatePath())
		}
	}

	// Dedup (R6) only applies to hook-triggered refreshes — retry/replay of
	// the *same event*. "manual" and "ci" are a human/pipeline explicitly
	// asking for a refresh right now; deduping those against an earlier
	// refresh at the same commit would silently turn a real request into a
	// no-op, which is exactly the "state that lies by omission" this spec
	// exists to prevent.
	dedupable := opts.Trigger != "manual" && opts.Trigger != "ci"
	key := dedupKey(opts.Trigger, opts.Target, gitHeadCommit(root))
	if dedupable && exists && recentlyProcessed(root, key, 200) {
		entry := refreshLogEntry{At: started.UTC().Format(time.RFC3339), Trigger: opts.Trigger,
			Target: opts.Target, DedupKey: key, Result: "skipped", DurationMS: 0}
		_ = appendRefreshLog(root, entry)
		return &entry, nil
	}

	curated := map[string]string{}
	existingDerived := map[string]struct {
		body, status string
	}{}
	if exists {
		raw, err := os.ReadFile(store.StatePath())
		if err != nil {
			return failedRefresh(root, opts, key, started, err)
		}
		fm, body := pose.SplitFrontmatter(string(raw))
		state, perr := pose.ParseProjectState(fm, body)
		if state == nil {
			return failedRefresh(root, opts, key, started, perr)
		}
		for _, sec := range state.Sections {
			if sec.Kind == "curated" {
				curated[sec.Name] = sec.Body
			} else {
				existingDerived[sec.Name] = struct{ body, status string }{sec.Body, sec.Status}
			}
		}
	}

	baseline := gitHeadCommit(root)
	now := time.Now().UTC()
	policy := store.LoadStatePolicy()

	type rendered struct{ name, kind, status, body string }
	var sections []rendered
	historySections := map[string]stateHistorySection{}
	var changed []string
	recomputeAll := len(opts.Only) == 0
	for _, def := range stateSectionOrder {
		if def.curated {
			body := curated[def.name]
			if body == "" {
				body = curatedExecSummaryPlaceholder
			}
			sections = append(sections, rendered{name: def.name, kind: "curated", body: body})
			continue
		}
		recompute := recomputeAll || opts.Only[def.name]
		var body, status string
		if recompute {
			body, status = deriveSection(store, def.name)
			if def.name == "Arquitetura" && opts.Directed != nil {
				body, status = opts.Directed.summary, "unavailable"
				if opts.Directed.summary != "" {
					status = ""
				}
			}
		} else if prior, ok := existingDerived[def.name]; ok {
			body, status = prior.body, prior.status
		} else {
			body, status = deriveSection(store, def.name) // never seen before: compute it once
		}
		if prior, ok := existingDerived[def.name]; !ok || pose.ContentHash12(prior.body) != pose.ContentHash12(body) {
			changed = append(changed, def.name)
		}
		sections = append(sections, rendered{name: def.name, kind: "derived", status: status, body: body})
		historySections[def.name] = stateHistorySection{Hash: pose.ContentHash12(body), Body: body}
	}

	var out strings.Builder
	fmt.Fprintf(&out, "---\nschema_version: %d\ngenerated_at: %s\nbaseline_commit: %s\nstaleness_policy: %s\n",
		pose.ProjectStateSchema, now.Format(time.RFC3339), baseline, pose.FormatStalenessPolicy(policy))
	// Every successful refresh clears refresh_pending, regardless of what
	// triggered it (R5) — a failed hook-triggered refresh sets it via the
	// narrow markRefreshPending edit instead, which never reaches this path.
	fmt.Fprintf(&out, "refresh_pending: \n")
	out.WriteString("---\n\n# Project State\n")
	for _, sec := range sections {
		fmt.Fprintf(&out, "\n## %s\n", sec.name)
		switch sec.kind {
		case "curated":
			fmt.Fprintf(&out, "<!-- state:curated -->\n\n%s\n", sec.body)
		default:
			hash := pose.ContentHash12(sec.body)
			if sec.status != "" {
				fmt.Fprintf(&out, "<!-- state:derived hash:%s status:%s -->\n\n%s\n", hash, sec.status, sec.body)
			} else {
				fmt.Fprintf(&out, "<!-- state:derived hash:%s -->\n\n%s\n", hash, sec.body)
			}
		}
	}

	if err := writeAtomic(store.StatePath(), []byte(out.String()), 0o644); err != nil {
		return failedRefresh(root, opts, key, started, err)
	}
	if err := appendStateHistory(store.StateHistoryPath(), stateHistoryEntry{
		GeneratedAt: now.Format(time.RFC3339), BaselineCommit: baseline, Sections: historySections,
	}); err != nil {
		return failedRefresh(root, opts, key, started, err)
	}

	entry := refreshLogEntry{At: started.UTC().Format(time.RFC3339), Trigger: opts.Trigger, Target: opts.Target,
		DedupKey: key, Result: "ok", DurationMS: time.Since(started).Milliseconds(),
		ChangedSections: changed, Directed: opts.Directed != nil}
	if err := appendRefreshLog(root, entry); err != nil {
		return &entry, err // artifact already written successfully; log append failure is secondary
	}
	return &entry, nil
}

func failedRefresh(root string, opts refreshOptions, key string, started time.Time, cause error) (*refreshLogEntry, error) {
	entry := refreshLogEntry{At: started.UTC().Format(time.RFC3339), Trigger: opts.Trigger, Target: opts.Target,
		DedupKey: key, Result: "failed", DurationMS: time.Since(started).Milliseconds(), Error: cause.Error()}
	_ = appendRefreshLog(root, entry)
	return &entry, cause
}
