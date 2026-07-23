package cli

// Post-event hook registry (spec pose-project-state-refresh-contract R1):
// a single, generic mechanism other consumers can register against too
// (the sister spec pose-capability-assessment-triggers reuses it for
// reassessment triggers) — no daemon, no filesystem watcher, just a
// synchronous call from the exact point in the CLI where each event
// already, deterministically happens.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/harne8/pose-mcp/internal/pose"
)

// HookEvent is the typed payload every registered consumer receives.
type HookEvent struct {
	Kind   string // spec_closeout | spec_amend | evidence_reconciled | assessment_snapshot | release_cut
	Target string // event-specific identity: spec slug, request_id, ...
	Commit string // best-effort; "" when the event has no associated commit
	At     time.Time
}

// HookConsumer processes one event. A returned error never blocks the
// operation that emitted the event — EmitHook always recovers and swallows
// it (R5: best-effort by default); the state-refresh consumer converts a
// failure into a visible refresh_pending marker instead.
type HookConsumer func(root string, ev HookEvent) error

var hookRegistry = map[string][]HookConsumer{}

// RegisterHook subscribes fn to every event of the given kind. Intended to
// be called from an init() in the consumer's own file — the registry is
// process-global and populated once at startup, mirroring how every other
// POSE mechanism in this binary is wired (no runtime plugin loading).
func RegisterHook(kind string, fn HookConsumer) {
	hookRegistry[kind] = append(hookRegistry[kind], fn)
}

// EmitHook invokes every consumer registered for ev.Kind and returns the
// first error any of them reported — nil in the (default, R5) best-effort
// case, since stateRefreshConsumer itself only returns non-nil when strict
// mode is configured. Callers that want R5's "modo estrito... a
// operação-gatilho falha com mensagem nominal" MUST check this return
// value and fail their own command; callers that don't care (there are
// none left in this codebase, but future consumers could be genuinely
// fire-and-forget) may discard it. A panicking consumer is recovered and
// reported as an error like any other — it never crashes the caller.
func EmitHook(root string, ev HookEvent) error {
	var first error
	for _, fn := range hookRegistry[ev.Kind] {
		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("hook consumer panicked: %v", r)
				}
			}()
			return fn(root, ev)
		}()
		if err != nil && first == nil {
			first = err
		}
	}
	return first
}

// eventSections maps each event kind to the derived sections it affects
// (R2). An event kind with no entry here defaults to a full refresh —
// never silently skips sections just because the map wasn't updated for a
// newly introduced section (Riscos técnicos in the spec).
var eventSections = map[string][]string{
	"spec_closeout":       {"Specs & Roadmaps", "Follow-ups", "Validação & Evidência"},
	"spec_amend":          {"Specs & Roadmaps", "Decisões & Conhecimento"},
	"evidence_reconciled": {"Validação & Evidência"},
	"assessment_snapshot": {"Capabilities"},
	"release_cut":         {"Decisões & Conhecimento", "Validação & Evidência"},
}

func init() {
	consumer := stateRefreshConsumer
	for kind := range eventSections {
		RegisterHook(kind, consumer)
	}
	// release_cut has no producer inside pose-mcp today — pose-mcp does not
	// cut releases itself (that lives in Conductor, a separate service).
	// The kind is still registered so nothing needs to change here once
	// that integration exists; see spec Gaps conhecidos.
}

// stateRefreshConsumer is the "state-refresh" consumer (R2): recomputes
// only the sections the event affects, attempts a components_hit-directed
// Arquitetura refresh when the event carries a commit and GraphForge is
// configured, and records the outcome. Best-effort by default; strict mode
// (policy strict_refresh) returns the error so the caller can propagate it.
func stateRefreshConsumer(root string, ev HookEvent) error {
	store := pose.Store{Root: root}
	if !store.HasProjectState() {
		return nil // additive: no artifact, nothing to keep in sync
	}
	policy := store.LoadStatePolicy()

	only := map[string]bool{}
	for _, name := range sectionsForEvent(ev.Kind) {
		only[name] = true
	}

	opts := refreshOptions{Trigger: ev.Kind, Target: ev.Target, Only: only}
	if ev.Commit != "" {
		if hit := tryDirectedHit(root, store, ev.Commit); hit != nil {
			only["Arquitetura"] = true
			opts.Directed = hit
		}
	}

	_, err := runRefresh(root, opts, true)
	if err == nil {
		return nil
	}
	if policy.StrictRefresh {
		return fmt.Errorf("state-refresh (strict): %w", err)
	}
	if markErr := markRefreshPending(root, ev.Kind); markErr != nil {
		return fmt.Errorf("state-refresh failed (%v) and could not mark refresh_pending (%v)", err, markErr)
	}
	return nil
}

func sectionsForEvent(kind string) []string {
	if sections, ok := eventSections[kind]; ok {
		return sections
	}
	names := make([]string, 0, len(stateSectionOrder))
	for _, def := range stateSectionOrder {
		if !def.curated {
			names = append(names, def.name)
		}
	}
	return names
}

var statePendingLineRE = regexp.MustCompile(`(?m)^refresh_pending:.*$`)

// markRefreshPending performs a narrow, frontmatter-only edit of the
// existing (still valid) artifact when a hook-triggered refresh fails
// (R5) — it never touches section bodies, so a failed refresh cannot
// corrupt otherwise-good content.
func markRefreshPending(root, eventKind string) error {
	store := pose.Store{Root: root}
	raw, err := os.ReadFile(store.StatePath())
	if err != nil {
		return err
	}
	line := "refresh_pending: " + eventKind
	content := string(raw)
	if statePendingLineRE.MatchString(content) {
		content = statePendingLineRE.ReplaceAllString(content, line)
	} else {
		content = strings.Replace(content, "---\n", "---\n"+line+"\n", 1)
	}
	// Deliberately os.WriteFile, not writeAtomic: this in-place edit only
	// ever runs when a refresh has just failed, on a file that already
	// exists — it must succeed independently of whether the containing
	// directory can accept new entries (writeAtomic's temp+rename needs
	// that; a same-inode truncate+write does not). The narrow blast radius
	// (one frontmatter line) makes the small torn-write risk on process
	// crash an acceptable trade for not compounding an existing failure.
	return os.WriteFile(store.StatePath(), []byte(content), 0o644)
}

// tryDirectedHit calls components_hit (from the artifact's last known
// baseline to the event's commit) when GraphForge is configured; returns
// nil when unconfigured or on any failure — directed refresh is a bonus,
// never a requirement (R3: undirected refresh always still happens).
func tryDirectedHit(root string, store pose.Store, toCommit string) *directedHit {
	caller := resolveComponentsHitCaller(root)
	if caller == nil {
		return nil
	}
	state, err := store.ProjectState(context.Background(), "")
	if err != nil || state.BaselineCommit == "" || state.BaselineCommit == toCommit {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, ok, err := caller.ComponentsHit(ctx, state.BaselineCommit, toCommit)
	if err != nil || !ok || result == nil {
		return nil
	}
	return &directedHit{summary: renderDirectedHitSummary(result)}
}

func renderDirectedHitSummary(result *graphForgeHitResult) string {
	if len(result.Hits) == 0 {
		return "- hit dirigido: nenhum componente atingido pelo evento mais recente."
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("- hit dirigido (components_hit): %d componente(s) atingido(s).", len(result.Hits)))
	limit := result.Hits
	if len(limit) > 10 {
		limit = limit[:10]
	}
	for _, hit := range limit {
		id, _ := hit["component_id"].(string)
		level, _ := hit["level"].(string)
		if id != "" {
			lines = append(lines, fmt.Sprintf("  - component:%s [%s]", id, level))
		}
	}
	if len(result.Hits) > 10 {
		lines = append(lines, fmt.Sprintf("  - ... e mais %d", len(result.Hits)-10))
	}
	return strings.Join(lines, "\n")
}

// graphForgeHitResult is the subset of graphforge.components-hit.v1 this
// consumer reads — deliberately partial, not a full client library.
type graphForgeHitResult struct {
	Hits          []map[string]any `json:"hits"`
	UnmappedFiles []string         `json:"unmapped_files"`
}

// componentsHitCaller abstracts the GraphForge components_hit MCP tool
// call. ok=false means "not configured/unavailable", never an error — R3's
// "zero acoplamento de build" contract: the default is the null
// implementation, swapped only when POSE_GRAPHFORGE_MCP_URL is set.
type componentsHitCaller interface {
	ComponentsHit(ctx context.Context, fromCommit, toCommit string) (result *graphForgeHitResult, ok bool, err error)
}

// resolveComponentsHitCaller returns nil (never a null struct — a plain
// nil interface is cheaper to check at the one call site) when GraphForge
// is not configured for this project.
func resolveComponentsHitCaller(root string) componentsHitCaller {
	url := strings.TrimSpace(os.Getenv("POSE_GRAPHFORGE_MCP_URL"))
	if url == "" {
		return nil
	}
	projectID := strings.TrimSpace(os.Getenv("POSE_GRAPHFORGE_PROJECT_ID"))
	if projectID == "" {
		projectID = filepath.Base(root)
	}
	return httpComponentsHitCaller{url: url, projectID: projectID}
}

type httpComponentsHitCaller struct {
	url, projectID string
}

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type jsonRPCToolCallResult struct {
	Result *struct {
		StructuredContent json.RawMessage `json:"structuredContent"`
		IsError           bool            `json:"isError"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ComponentsHit calls the GraphForge MCP server's components_hit tool over
// Streamable HTTP (JSON-RPC 2.0 tools/call) — the same protocol GraphForge
// documents for every other consumer, no bespoke wire format.
func (c httpComponentsHitCaller) ComponentsHit(ctx context.Context, fromCommit, toCommit string) (*graphForgeHitResult, bool, error) {
	body, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0", ID: 1, Method: "tools/call",
		Params: map[string]any{
			"name": "components_hit",
			"arguments": map[string]any{
				"project_id": c.projectID, "from_commit": fromCommit, "to_commit": toCommit, "max_depth": 2,
			},
		},
	})
	if err != nil {
		return nil, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, nil // unreachable GraphForge degrades to "unavailable", not an error
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, nil
	}
	var rpc jsonRPCToolCallResult
	if err := json.NewDecoder(resp.Body).Decode(&rpc); err != nil {
		return nil, false, nil
	}
	if rpc.Error != nil || rpc.Result == nil || rpc.Result.IsError {
		return nil, false, nil
	}
	var result graphForgeHitResult
	if err := json.Unmarshal(rpc.Result.StructuredContent, &result); err != nil {
		return nil, false, nil
	}
	return &result, true, nil
}
