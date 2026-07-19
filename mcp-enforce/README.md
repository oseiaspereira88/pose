# mcp-enforce

Canonical authorization-enforcement layer for Harne8 MCP servers (ADR-004, ADR-021).

A single source of truth for **per-call policy** (OPA-backed, default-deny), **audit**
(allow *and* deny), and the **request-extraction helpers** shared by every MCP server. It
replaces the gate that was duplicated — and already diverging — between `pose-mcp` and
`graphforge/mcp-server`.

Standard library only. The single network dependency is the OPA REST API.

## Two consumption modes (ADR-021)

- **In-process** by Harne8-org Go MCP servers (e.g. `pose-mcp`) — import directly.
- **Embedded in the enforcement sidecar** that fronts foreign/polyglot servers (e.g.
  `graphforge/mcp-server`) — same code, no cross-org coupling.

Using the same package in both modes is what makes enforcement *consistent across servers*,
the property ADR-004 requires.

## Public API

| Symbol | Purpose |
|---|---|
| `PolicyGate` / `NewPolicyGate(PolicyConfig)` | per-call gate |
| `(*PolicyGate).Evaluate(ctx, PolicyInput) (PolicyDecision, error)` | RequirePrincipal → dev allow-all → OPA → default-deny |
| `PolicyConfig` | `OPAURL`, `OPAPath`, `Timeout`, `RequirePrincipal`, `HTTPClient` |
| `PolicyInput` | `Principal`, `ProjectID`, `ProjectIDs`, `Method`, `ToolName` (+ `Scopes`, `RunID`, `ExpiresAt`) |
| `PolicyDecision` / `.Metadata()` | outcome + JSON-RPC error data |
| `DenyDecision(input, reason)` | build a denied decision |
| `Auditor`, `SlogAuditor`, `NopAuditor` | structured allow/deny trail |
| `PrincipalFromHeader`, `HeaderValue`, `ProjectScopeFromArguments`, `ProjectIDFromArguments`, `ConfigFromEnv` | request extraction |
| `Identity`, `MintToken`, `ParseToken`, `IdentityFromHeader`, `(Identity).Apply` | Execution Identity token (ADR-007) |

## Execution Identity (ADR-007)

The gate binds the conductor-issued Execution Identity into authorization. The identity
travels as a compact HMAC-signed token in the `X-MCP-Execution-Identity` header:
`base64url(json-claims).base64url(HMAC-SHA256(secret, payload))` — stdlib only, the V1
"backend leve"; an asymmetric/SPIFFE scheme is the drop-in upgrade. Issuer (conductor) and
verifier (the gate's consumer) share `secret`.

Consumer flow: `id, err := IdentityFromHeader(r.Header, secret)` → `input = id.Apply(input)`
binds `RunID`, `Scopes`, `ExpiresAt` (and `project_id` when absent). The gate then enforces:

- `RequireIdentity` → deny `missing_identity` when no run-bound identity is present.
- **time-box** → deny `identity_expired` when `ExpiresAt` is in the past (a single `Clock`
  governs expiry; injectable for tests/skew). Applied locally before any OPA call.
- `Scopes`/`run_id`/`expires_at` are carried into the OPA input document for scope policies.

## Default-deny contract

`Evaluate` denies on every failure path: marshal error, transport error, OPA HTTP ≠ 200,
decode error, and `result: null` (undefined policy path → violation `policy_path_undefined`).
`RequirePrincipal` denies anonymous callers (`missing_principal`) before any OPA call, even in
dev/allow-all mode.

## OPA wire contract

`Evaluate` POSTs to `<OPAURL>/v1/data/<OPAPath>`:

```json
{ "input": { "principal": "...", "project_id": "...", "method": "tools/call", "tool": "..." } }
```

The four base fields are always present. Aggregate calls add `project_ids`; malformed
project scope is denied locally as `invalid_project_scope`. The Execution Identity scope fields (`scopes`,
`run_id`, `expires_at`) are **omitted when unset** — so shipped Rego policies see an unchanged
document — and are reserved for `mcp-execution-identity-scope-binding` (ADR-007). Golden
contracts: `testdata/opa_input_minimal.json`, `testdata/opa_input_with_scope.json`.

Expected Rego decision document:

```rego
default allow = false
violations contains msg if { ... }
```

GraphForge multi-project isolation is activated per principal through OPA data:

```json
{
  "graphforge": {
    "mcp": {
      "principal_projects": {
        "svc.portal": ["proj.a", "proj.b"]
      }
    }
  }
}
```

Once a principal has an entry, every requested `project_id`/`project_ids` value must be
present in that grant. Aggregate calls are denied atomically with `project_not_granted`.

## Audit event schema

`SlogAuditor` emits one structured line per decision (golden: `testdata/audit_events.jsonl`):

- allow → `Info`  `"<component>: policy allowed"`  `event_type=policy.decided`
- deny  → `Warn`  `"<component>: policy denied"`    `event_type=policy.violation`

Fields: `principal`, `project_id`, optional `project_ids`, `tool`, and `violations` (deny only). No payload content is
ever logged.

## Tests / goldens

```sh
go test ./...
MCP_ENFORCE_UPDATE_GOLDEN=1 go test ./...   # regenerate testdata goldens
```
