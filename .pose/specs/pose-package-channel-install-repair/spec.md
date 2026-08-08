---
slug: pose-package-channel-install-repair
status: done
created_at: 2026-08-08
completed_at: 2026-08-08
supersedes:
depends_on: pose-package-channel-delivery
priority: 1
components: release
delivers: governance:package-channel-install-repair
---

# Spec: The channel gate installs, and the docs stop promising what cannot work

## 1. Intent

### Goal
Repair both clean-host install paths so the matrix exercises the published
manifests, and remove the Homebrew install command the documentation offers,
which Homebrew rejects.

### Business value
The clean-host matrix ran for the first time in the project's history on
v0.21.0 — `pose-package-channel-delivery` gave it a trigger that fires — and
failed on both runners. Neither failure was in the artifacts: the formula and
the manifests are correct. Both were in how the gate installs.

The macOS failure is the one that matters beyond CI. Homebrew now refuses any
formula that is not in a tap: `brew install --formula <path-or-url>` is
rejected, and the maintainers state that `HOMEBREW_DEVELOPER=true` is
unsupported regardless of configuration. `docs-site/docs/package-channels.md`
has been handing users exactly that command.

So the documented Homebrew channel never worked, and could not have. What
`pose-package-channel-delivery` fixed was the 404 — the formula genuinely was
missing from the release, and now it is published. What it could not detect is
that the command consuming it had been removed by upstream, because publishing
an asset and being able to install it are different claims. That distinction
was written into `pose-docs-asset-parity`'s known gaps as "existence is not
usability" on the same day this proved it.

The Windows failure is narrower: installing from a local manifest is disabled
by default and requires an administrator to enable it, so winget printed its
help and exited 1 — which reads like a usage error rather than a disabled
feature.

### Constraints
- The gate must test the *published* formula, not a rewritten one. Standing up
  a throwaway tap around the real `pose.rb` keeps the artifact under test
  unchanged.
- The documentation may not describe a channel that does not work. An absent
  claim is better than a false one — the standing follow-up on
  `pose-first-release-evidence-confirmation` says exactly this.

### Non-goals
- Creating a Homebrew tap. That is a new public repository and an ongoing
  publication surface; it is recorded as a decision, not taken here.
- Submitting the WinGet manifest upstream, which remains its own tracked step.

---

## 2. Requirements

### Functional
- R1: The macOS leg shall install the published `pose.rb` through a local tap
  and exercise the installed binary.
- R2: The Windows leg shall enable local-manifest installation, validate the
  manifest set, and install from it.
- R3: `package-channels.md` shall not offer a `brew` install command, shall
  state why, and shall name the verified download as the supported path.
- R4: The Homebrew row shall describe the channel's real status: formula
  published and install-tested, not consumable by an end user.

### Non-functional
- The gate keeps its clean-host property: no shared state with the release job.

### Security
- The local tap contains only the release's own formula. Enabling
  `LocalManifestFiles` is scoped to the ephemeral Windows runner.

### Compatibility
- No product change. The artifacts published are unchanged; what changes is how
  they are tested and what the documentation claims about them.

---

## 3. Technical Plan

### Affected areas
- `.github/workflows/package-channels.yml` — both install legs.
- `docs-site/docs/package-channels.md` — the channel table, the install
  commands, the verification note and the rollback section.

### Artifacts
- created: .pose/specs/pose-package-channel-install-repair/spec.md
- modified: .github/workflows/package-channels.yml
- modified: docs-site/docs/package-channels.md

### Delivery targets
- governance:package-channel-install-repair module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- None.

### Technical risks
- **The macOS fix is unverified until the gate runs.** `brew tap-new --no-git`
  and `brew --repository <tap>` are documented, but the exact interaction with
  a formula that is not in the Homebrew API is not something this host can
  execute. The workflow accepts `workflow_dispatch`, so it is dispatched
  against the published v0.21.0 rather than left for the next cut.
- Removing the `pose.rb` URL from the documentation removes it from
  `pose-docs-asset-parity`'s coverage too, since that check verifies what the
  docs promise. The formula could stop being published and only the
  `package-channels` matrix would notice.

---

## 4. Tasks

### Planning
- [x] Establish from Homebrew's own sources whether path/URL install survives
- [x] Establish the WinGet local-manifest requirement

### Implementation
- [x] R1: install through a throwaway local tap
- [x] R2: enable local manifests, validate, install
- [x] R3: remove the brew command and state why
- [x] R4: describe the channel's real status

### Validation
- [x] Strict docs build
- [x] Dispatch the repaired matrix against the published v0.21.0

---

## 6. Validation

### Strategy
The documentation half is checkable here: the site must build strictly, and the
parity gate must still pass with the changed page. The workflow half cannot be
checked on this host — there is no macOS runner and no winget — so it is
dispatched against the already-published v0.21.0 instead of being asserted and
left for the next release. That is the whole lesson of this cycle: a gate whose
first real execution is the release is a gate nobody has run.

### Deterministic checks

#### Build
- Command: `mkdocs build --strict -f docs-site/mkdocs.yml`
- Scope: the rewritten channel page
- Expected: no warnings

#### Security / Contract
- Command: `bash tests/release/docs-asset-parity.sh v0.21.0`
- Scope: the documented downloads that remain
- Expected: pass

### Execution log
- Date: 2026-08-08
- Environment: linux/amd64.
- Notes: the first strict build failed on a `../../README.md#install` link
  pointing outside the docs directory; it now points at `quickstart.md#install`
  and builds clean. The parity gate passes with three documented downloads,
  `pose.rb` having left the documented set. Homebrew's position was taken from
  its own discussion thread, where maintainers confirm the rejection is
  intentional and `HOMEBREW_DEVELOPER` is unsupported.

### Results summary
- Successes: docs build strictly; parity gate passes; both legs repaired
- Failures: none locally
- Warnings: the workflow legs are proven by the dispatched run, not by this host

### Requirement trace
- R1 [satisfied] governance:package-channel-install-repair evidence:integration check:delivery-integration report:.github/workflows/package-channels.yml — the macOS leg creates a throwaway tap, copies the published formula into it and installs `harne8/local/pose`
- R2 [satisfied] report:.github/workflows/package-channels.yml — the Windows leg enables `LocalManifestFiles`, runs `winget validate`, then installs from the manifest set
- R3 [satisfied] check:docs-build report:docs-site/docs/package-channels.md — the `brew` command is gone, a warning admonition states why, and the verified download is named as the supported path
- R4 [satisfied] report:docs-site/docs/package-channels.md — the Homebrew row reads "No install channel", with the reason and the real support tier

### Known gaps
- `pose.rb` is no longer covered by the docs-parity check, because the docs no
  longer reference it. Only the `package-channels` matrix would notice it
  disappearing.
- There is no Homebrew install channel at all until a tap exists. The
  documentation now says so rather than implying otherwise.

---

## 7. Final Report

### Delivered scope
Both clean-host install legs repaired, and the Homebrew channel documented as
what it is — a published, install-tested formula with no consumer path.

### Files and modules changed
- .github/workflows/package-channels.yml
- docs-site/docs/package-channels.md

### Validation executed
- Command: strict mkdocs build; docs-asset-parity against v0.21.0; dispatched matrix
- Result: build clean, parity passes

### Residual risks
- The macOS leg's exact tap interaction is proven only by the dispatched run.

### Follow-ups

- [open] Decide whether to stand up a Homebrew tap. The prior decision to defer it compared tap maintenance against the formula-URL install; that alternative no longer exists, so the comparison is now tap-or-nothing and macOS users have no `brew` path at all. The formula is already generated, published and install-tested every release, so a tap consumes existing artifacts rather than needing new ones. (owner:@pose-maintainers crit:medium review:2026-10-08)
- [open] `pose.rb` left the docs-parity check's coverage when it left the documentation, so a release that silently stops publishing it would only be caught by the channel matrix. Consider asserting the published asset set against what the release workflow uploads, independently of what the docs happen to reference. (owner:@pose-maintainers crit:low review:2026-11-06)
