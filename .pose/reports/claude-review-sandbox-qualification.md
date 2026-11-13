# Claude review shell sandbox qualification

Verified 2026-11-12 on Linux with Claude Code 2.1.322, Claude Opus 5.5 medium, bubblewrap, socat and jq. Scope: `pose-claude-reviewer-runs-tests-in-sandbox`.

These native **task** runs qualify the selected command in a disposable synthetic project; they are not review attestations. The fixture policy projects the proposed review command as its read task command. Native governed review of the actual implementation is a separate gate.

## Attempts and results

| Native run | Result |
| --- | --- |
| `trn-0dfb1267a24d2e24` | Failed qualification: Go could not initialize its cache because scratch was not explicitly writable. External sentinel redirection was OS-denied. |
| `trn-537e12582cfced71` | Failed qualification: scratch worked and external writes were denied, but `denyWrite:["."]` from inline settings did not deny the project. The synthetic source sentinel changed; no original repository was used. |
| `trn-43c6174032cb0775` | Passed only the synthetic-path qualification after encoding the absolute physical cwd in denyWrite: `go test -v -count=1 ./...` exit 0, `go vet ./...` exit 0; source, external and synthetic shared-metadata sentinels denied by the OS. |
| `trn-a1c0193704f5f055` | Passed the same commands and denials through the native `task --session` continuation option, using `resume.commands.read` with the proposed resumed review command. This is command/session qualification, not a fabricated interrupted governed review. |

Private owner-only records retain the normalized tool calls and redacted transcripts. No session identifier, credential or raw vendor stream is copied here. The first two failures remain in that history.

## Fixture assertions

The test binary first calls `os.WriteFile(filepath.Join(os.TempDir(), "scratch-probe"), ...)` and fails fatally on error. It then attempts writes to `source-sentinel`, `/tmp/pose-sandbox-outside/sentinel` and `/tmp/pose-sandbox-shared-git/sentinel`, failing if any succeeds and printing the actual OS error otherwise. Both qualified executions reported:

```text
OS denied source-sentinel: open source-sentinel: read-only file system
OS denied /tmp/pose-sandbox-outside/sentinel: open /tmp/pose-sandbox-outside/sentinel: read-only file system
OS denied /tmp/pose-sandbox-shared-git/sentinel: open /tmp/pose-sandbox-shared-git/sentinel: read-only file system
--- PASS: TestSandboxBoundary
```

The scratch assertion passed because the test passed; the vendor's narrative cautiously said it saw no explicit successful scratch log. Do not confuse that narrative with the test's fatal-error assertion. The direct Bash command `printf changed > /tmp/pose-sandbox-outside/sentinel` also failed with a read-only-filesystem error in each qualified run, without an unsandboxed retry. The two outside sentinels and, after resetting the failed synthetic source between attempts, the source sentinel retained their expected bytes after the qualified runs.

## Boundary and limitations

The command enables `failIfUnavailable`, disables filesystem-isolation opt-out and nested user namespaces, forbids unsandboxed retries, and supplies absolute paths through jq rather than interpolating paths into JSON. Both wrappers check dependencies before starting Claude. Tests independently verify dependency absence, generated paths/settings and startup failure propagation. No such fake proves OS isolation.

This is the vendor's shell sandbox, including test subprocesses. Claude itself remains a trusted executable with session/auth state outside this shell boundary; file-editing tools and hooks/MCP are not enabled. Claude can use its documented per-user temporary fallback when paths are long; this is not a claim of exclusive full-process confinement to one directory, arbitrary credential-read protection or a newly qualified engine containment backend. The shared-metadata sentinel is synthetic, not a destructive probe against this repository's Git database. Existing dispatch integrity checks cover the sealed worktree; they do not compare the origin repository's refs, tags or object database. Actual shared Git isolation requires an explicit common-dir denial and a real-worktree probe. macOS and Windows have not been qualified here.

Sources: [sandbox boundary and startup refusal](https://code.claude.com/docs/en/sandboxing), [path settings](https://code.claude.com/docs/en/settings-reference), [temporary fallback](https://code.claude.com/docs/en/env-vars#variables).

## First governed execution (not approved)

Native run `rrn-ce731a2bf0bb5e07`, bundle `rvb-60317e86c49d6c9e`, executed and returned exit 0, but failed closed: `adapter capture failed; unredacted output withheld` and runtime cleanup inventory incomplete. No conclusion is usable. The runtime retained about 1.6 GB of Go build cache; the existing residual-runtime cleanup guard is 4096 entries/128 MiB. The historical capture message does not identify its cause, so no specific capture root cause is asserted and the withheld stream is not recovered.

Remediation keeps those guards unchanged: the profile instructs Claude to run `go clean -cache` as its last Bash command INSIDE the mandatory sandbox. Cleanup is verified independently by the engine; failure remains failure. Cleaning this adapter-controlled path in the trusted outer wrapper was rejected because a test could replace it with a symlink to host data. No outer-wrapper cleanup executes. The engine still independently performs bounded owned-resource cleanup and refuses incomplete cleanup. Regressions verify the cleanup instruction exists on fresh/resumed profiles and startup failures propagate unchanged. Review capture failures now reuse the closed-vocabulary diagnostic with an explicit stage, while discarding all unsafe channels; a dispatch regression proves ambiguous session capture remains failed with no output. A new test also verifies the configured session regex against a literal init event. A new governed run must qualify these changes; the failed run and consumed attempt remain recorded.

## Second governed run: substantive feedback, no usable attestation

`rrn-41cc27dbae7fdd47` (bundle `rvb-8faef26f7b9b8061`) reexecuted all eight then-current focused tests successfully (0.091s), ran go vet successfully, confirmed source-write refusal and scratch success, and passed artifact/surface checks. It produced changes-requested feedback with F1 high (Claude automatically allows the original Git common directory), F2 medium (the synthetic Git sentinel did not qualify real shared Git) and F3 low (jq test coverage could silently skip). This was not an attested rejection: the engine refused the run because Claude materialized empty `.cc-writes` directories after the snapshot and cleanup retained about 421 MB of Go build workspaces after optional full-suite timeouts. The full suites had environmental failures, explicitly reported by the reviewer rather than called passing. The original run, feedback and failed operation remain retained.

Remediation: resolve the physical `git rev-parse --path-format=absolute --git-common-dir` before launch and put it alongside physical cwd in denyWrite on fresh/resume. Preparation failure refuses launch. Fail rather than skip the jq-dependent regression, using a disposable Git fixture. For opt-in Claude review only, precreate empty `.claude/.cc-writes` directories at root and declared components inside the disposable checkout, before its integrity snapshot. Do not ignore these directories: the regression proves a new file there changes the snapshot. Cleanup instructions cover only leftover `go-build*` directories directly under GOTMPDIR, inside the sandbox and after background jobs finish. Engine cleanup bounds remain unchanged.

## Actual linked-Git-worktree qualification

Native task `trn-8d24f721bc87ba11` on 2026-11-13 used a real origin `/tmp/pose-sandbox-git-origin` and linked worktree `/tmp/pose-sandbox-git-worktree`, with the corrected profile projected as the task command. This is execution qualification, not a review attestation. Its Go test queried the actual Git common dir (`/tmp/pose-sandbox-git-origin/.git`), then attempted writes to the worktree source sentinel, origin source sentinel and a synthetic sentinel inside that actual Git directory. All failed with read-only-filesystem errors. Non-mutating `test -w` probes for actual common-dir HEAD, refs/heads, refs/tags and objects/pack all reported not writable. Scratch write succeeded under the operation-owned TMPDIR. `go test -v -count=1 ./...` passed (0.005s), `go vet ./...` exited 0, and sandboxed cache/Go-temp cleanup exited 0. The host verified all three sentinels and all refs retained their expected bytes/values after execution. No write was attempted to the main project's refs or objects.

This real worktree probe supersedes the earlier synthetic-directory-only isolation claim. The next governed implementation review must confirm the corrected real origin common-dir boundary, controlled staging and successful cleanup before closeout.
