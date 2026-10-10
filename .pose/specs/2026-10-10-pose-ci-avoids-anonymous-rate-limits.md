---
slug: pose-ci-avoids-anonymous-rate-limits
status: done
created_at: 2026-10-10
completed_at: 2026-10-10
supersedes:
depends_on:
priority: 1
components: ci, release
task_type: bugfix
surface: minimal
changelog: none
delivers: governance:ci-avoids-anonymous-rate-limits
---

# Spec: CI avoids anonymous third-party rate limits

## 1. Intent

### Goal

Stop CI and the release pipeline from failing on anonymous rate limits of the GitHub API and Docker Hub.

### Business value

On 2026-10-09 and 2026-10-10 three runs failed with no code defect. The v7.3.0 Release (run 38019363282) failed in the install-and-upgrade journey with `curl: (22) ... 403` listing the published releases, because the journey called the GitHub API anonymously and hosted runners share a per-IP limit; a rerun passed. CI on main failed twice in "Delivery images build and start" with `429 Too Many Requests ... unauthenticated pull rate limit` from Docker Hub (runs 37994290035 and its rerun).

### Constraints

The journey still runs without a token locally; images stay pinned by digest; no new secret is required.

## 2. Requirements

### Functional

- R1: The install-and-upgrade journey shall list releases with `GITHUB_TOKEN` or `GH_TOKEN` when set, and anonymously otherwise.
- R2: CI shall pass the workflow token to the journey step.
- R3: CI shall configure the runner's Docker daemon with a Docker Hub pull-through mirror before building delivery images.

## 3. Technical Plan

### Artifacts

- created: .pose/specs/2026-10-10-pose-ci-avoids-anonymous-rate-limits.md
- created: .pose/starts/pose-ci-avoids-anonymous-rate-limits.json
- modified: tests/journeys/install-and-upgrade.sh
- modified: .github/workflows/ci.yml

### Delivery targets

- governance:ci-avoids-anonymous-rate-limits module:. profile:release-governance entrypoint:.github/workflows/ci.yml

## 5. Decisions

### Decision D1
- Date: 2026-10-10
- Context: Docker Hub limits anonymous pulls per IP.
- Options considered: (a) a pull-through mirror (`mirror.gcr.io`) on the runner daemon; (b) `docker login` with a Docker Hub secret.
- Decision: (a).
- Rationale: no secret to provision; every base image is pinned by digest, so the mirror cannot serve different bytes.

## 6. Validation

### Strategy

Locally, `POSE_JOURNEYS=1 go test ./internal/cli -run TestJourneysInstallAndUpgradeFromThePublishedRelease` passes with and without `GITHUB_TOKEN`; `actionlint` accepts the workflow. In CI, run 38021679842 on aa29027d passed the journey (with the workflow token) and the image build (through the mirror). Token-absent behaviour is the local run without `GITHUB_TOKEN`.

### Requirement trace

- R1 [satisfied] governance:ci-avoids-anonymous-rate-limits check:install-and-upgrade-journeys-integration evidence:integration test:TestJourneysInstallAndUpgradeFromThePublishedRelease
- R2 [satisfied] governance:ci-avoids-anonymous-rate-limits evidence:manual <ci.yml journey step env GITHUB_TOKEN; actionlint>
- R3 [satisfied] governance:ci-avoids-anonymous-rate-limits evidence:manual <ci.yml mirror step before "Delivery images build and start"; CI run https://github.com/oseiaspereira88/pose/actions/runs/38021679842 on aa29027d: success, the mirror step printed ["https://mirror.gcr.io/"] and the journey and the image build passed>

## 7. Final Report

### Delivered scope

The journey authenticates when a token exists, and CI pulls Docker Hub images through a mirror.

### Residual risks

- If the mirror is unavailable, the pull falls back to Docker Hub and can still hit the limit.

### Follow-ups
