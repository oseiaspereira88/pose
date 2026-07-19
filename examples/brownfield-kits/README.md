# Brownfield reference kits

Three small, real, checked-in repositories showing how an existing project
adopts POSE without a governance rewrite. Each kit is executable, not
illustrative: `pose-mcp/internal/cli/brownfield_kits_test.go` runs every
staged command below against the exact `fixture/` tree in this directory
on every `go test ./...` — if a kit's guide drifts from what the CLI
actually does, the test fails.

| Kit | Starting point | Path |
|---|---|---|
| [Direct adoption](direct-adoption/) | No spec system at all | `direct-adoption/fixture/` |
| [Spec Kit import](spec-kit-import/) | Existing GitHub Spec Kit feature | `spec-kit-import/fixture/` |
| [OpenSpec import](openspec-import/) | Existing OpenSpec change | `openspec-import/fixture/` |

## Shared shape

Every kit follows the same three stages:

1. **Visibility** — run something read-only or `--dry-run` first. Nothing
   is written; you see exactly what POSE found and what it would do.
2. **Adoption** — apply it. Pre-existing files are never modified, only
   new paths are added (`.pose/`, `.agents/`, `AGENTS.md`, `POSE.md`, and
   for imports, `.pose/specs/<slug>/`).
3. **Blocking gate** — promote from tolerant/preview to a strict gate that
   actually fails the build/CI on a real problem.

## Rollback

POSE never modifies a pre-existing tracked file during adoption or import
(every kit's test asserts this with a byte-for-byte comparison plus a
`git status --porcelain` check). Rollback is therefore always a plain git
operation — before committing the adoption, `git clean -fdx` (or just
don't commit) removes everything POSE added and leaves the original
repository exactly as it was. After committing, `git revert` the adoption
commit(s) the same as any other change.

## Mapping loss

Import (`pose import spec-kit` / `pose import openspec`) never claims a
perfect translation. Every gap the importer notices — a missing `plan.md`,
a missing `design.md`, an unmapped source section — becomes a curation
warning printed at import time, a note under the generated spec's
`## 8. Import Provenance` section, and an `[open]` follow-up you can list
with `pose followups --open`. The generated spec still clears the DoR
readiness gate (`pose lint-spec <slug> --ready-check`) immediately —
readiness is a structural floor (the required sections aren't empty), not
a claim that curation is done. Read the follow-up before promoting the
spec past `draft`.
