# Rule: Infrastructure Docker

## When to consult

Consult this guide for Dockerfiles, Containerfiles, Docker Compose definitions, container image builds, and runtime container specifications.

## Required patterns

- Use multi-stage builds to produce lean runtime images without build toolchains and development dependencies.
- Enforce non-root execution by creating and switching to a dedicated `USER` with minimal UID/GID privileges.
- Pin base images to specific immutable tags or digests rather than `:latest`.
- Optimize layer caching: copy dependency manifests (`package.json`, `go.mod`, `Cargo.toml`, `requirements.txt`) before copying full source trees.
- Define explicit container healthchecks (`HEALTHCHECK`) and configure sensible termination signals (`STOPSIGNAL`).
- Use `.dockerignore` to exclude local artifacts, `.git`, `node_modules`, test caches, and secret files from the build context.

## Blocking anti-patterns

- Running containers as `root` in production images without documented security justification.
- Passing secrets, tokens, or credentials via `ARG` or `ENV` instructions that persist in image layers.
- Using unpinned or mutable `:latest` base image tags in production Dockerfiles.
- Installing package manager caches without cleanup in the same `RUN` layer (e.g. missing `rm -rf /var/lib/apt/lists/*`).
- Mounting the host Docker daemon socket (`/var/run/docker.sock`) into unprivileged application containers.

## Minimum checks

- Run `hadolint Dockerfile` without error-level violations.
- Run `docker build --check .` or container vulnerability scanning (Trivy / Grype).
- Verify that runtime images successfully build and start without unexpected permission failures.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
