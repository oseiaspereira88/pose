---
slug: pose-composition-contract
status: done
created_at: 2026-08-09
completed_at: 2026-08-09
supersedes:
depends_on: pose-container-build-gate
priority: 1
components: pose-mcp
delivers: governance:composition-contract
---

# Spec: The composition consumes a contract instead of describing us from memory

## 1. Intent

### Goal
Publish, from this repository, a machine-readable declaration of how its images
are built and configured — and check it — so the `harne8` composition stops
restating that knowledge independently.

### Business value
`pose-container-build-gate` proved the images build and start here. It did not
close the gap it opened: how they are *composed* is written down twice, in two
repositories, with nothing reconciling the copies.

The duplicated surface is larger than the build contexts that motivated the
follow-up. `docker-compose.prod.yaml` independently restates:

- both build contexts and Dockerfile paths;
- the internal port (`8790:8799`, matching an `ENV`/`EXPOSE` declared here);
- a read-only volume mount at `/harne8-projects`;
- **eight environment variable names** the service is configured through.

Each is a fact this repository owns and the composition repeats. A rename here
does not fail anything there — the container simply starts with a default and
nobody is told.

Verified rather than assumed, and the verification is the argument. Auditing
those eight names for drift, two appeared to be set by production and never read
by any code: `POSE_MCP_REQUIRE_PRINCIPAL` and `POSE_MCP_REQUIRE_IDENTITY`, both
authorization controls. They turned out to be read — through
`mcpenforce.ConfigFromEnv("POSE_MCP_", …)`, which builds each name by
concatenating a prefix, so the literals exist nowhere in the source.

There was no defect. But it took reading the enforcement library to establish
that, and **no static analysis of this repository can enumerate its own
configuration contract**. If the owning repository cannot list the valid names,
a consumer in another repository certainly cannot, and a typo in the compose
file is indistinguishable from an intentionally unset option.

This is the shape `pose-machinery-distribution-contract` addressed for rules,
workflows and templates: content the engine owns and instances consumed by
copying, until the engine started declaring what it ships and how. Compose
configuration is the same relationship seen from outside the process boundary.

### Constraints
- The contract is published here and consumed there. This repository cannot
  edit the monorepo, so the deliverable is a declaration plus a check that
  proves the declaration matches this side; adopting it is the monorepo's step.
- The declaration must be derived from the code where possible. A hand-written
  list of environment variables would drift exactly like the compose file does,
  and would be the same defect with an extra file.

### Non-goals
- Publishing images to a registry, which would replace build-from-source with a
  different distribution contract and its own signing consequences.
- Describing the monorepo's own services. The contract covers what this
  repository owns: its images, their configuration surface and their ports.

---

## 2. Requirements

### Functional
- R1: This repository shall publish a versioned, machine-readable composition
  contract declaring, per image: build context, Dockerfile path, exposed port
  and the environment variables it reads. Volume *paths* are excluded: the
  container reads `HARNE8_PROJECTS_DIR`, which this repository owns, but where a
  consumer mounts it is the consumer's decision — declaring `/harne8-projects`
  here would restate a monorepo fact and reintroduce the duplication.
- R2: The environment-variable list shall be derived from the code, including
  the prefix-concatenated names that no literal search finds, so the contract
  cannot claim a surface the binary does not have.
- R3: A deterministic check shall fail when the contract disagrees with this
  repository — a declared variable no longer read, a port that no longer
  matches the Dockerfile, a build context that does not build.
- R4: The contract shall carry a schema version, so a consumer can detect a
  breaking change rather than silently misreading it.

### Non-functional
- The check runs offline against repository files; resolving the environment
  surface must not require starting a container.

### Security
- The contract declares variable *names*, never values. Several are secrets
  (`POSE_MCP_TOKEN`, `POSE_MCP_IDENTITY_SECRET`, `POSE_MCP_ADMIN_TOKEN`) and
  must be documented as required-and-secret without any example value that
  could be mistaken for a default.

### Compatibility
- Additive. The monorepo keeps working unchanged until it chooses to consume
  the contract; nothing here depends on that adoption.

---

## 3. Technical Plan

### Affected areas
- `composition-contract.json` — the published artifact, at the repository root
  beside `compatibility.json`.
- `mcp-enforce/extract.go` — exports `ConfigEnvSuffixes`, the list
  `ConfigFromEnv` itself reads.
- `mcp-enforce/config_env_test.go` — keeps that list from becoming inert.
- `pose-mcp/internal/version/composition_contract_test.go` — derives and checks.

### Artifacts
- created: .pose/specs/pose-composition-contract/spec.md
- created: composition-contract.json
- created: mcp-enforce/config_env_test.go
- created: pose-mcp/internal/version/composition_contract_test.go
- modified: mcp-enforce/extract.go

### Delivery targets
- governance:composition-contract module:pose-mcp profile:release-governance entrypoint:pose-mcp/cmd/pose/main.go

### API/contract changes
- A new published contract file. Once the monorepo consumes it, changing its
  shape becomes a breaking change for that consumer — which is what R4's schema
  version exists to make legible.

### Technical risks
- **The environment surface is genuinely hard to enumerate.** `ConfigFromEnv`
  builds names from a prefix, so deriving them requires the enforcement library
  to declare its own keys rather than a test grepping for them. If that
  declaration is added by hand, this spec reproduces the defect it is closing,
  one layer down. This is the requirement to defend under time pressure.
- **A contract nobody consumes is worse than none.** It would look like the
  problem was solved while both sides still drift, and the drift would now be
  invisible behind an artifact that claims to prevent it. Publishing must be
  paired with the monorepo actually reading it, which is outside this
  repository's control and belongs in the follow-up rather than in the claim.
- The port appears in three places today (`ENV`, `EXPOSE`, the compose
  mapping). A contract that declares it makes four unless the Dockerfile becomes
  the single source the check reads from.

---

## 4. Tasks

### Planning
- [x] Decide where the environment surface is declared so it is derived, not restated
- [x] Confirm `compatibility.json` is the right precedent for the artifact's shape and location

### Implementation
- [x] R1: publish the contract with build, port and environment facts
- [x] R2: derive the environment list from the enforcement library's own keys
- [x] R3: check the contract against the repository
- [x] R4: schema version

### Validation
- [x] Prove the check fails when a declared variable stops being read
- [x] Prove it fails on a port that no longer matches
- [x] Prove the derived list cannot go inert or silently grow
- [x] Run the mandatory checks

---

## 6. Validation

### Strategy
The failure that matters is the silent one: a rename here that leaves the
composition setting a name nobody reads. So the check must be proven by removing
a variable from the code and confirming the contract is reported as stale —
not by confirming that a correct contract passes, which any inert file does.

The environment surface deserves its own proof. If the derivation is honest, it
must produce the prefix-concatenated names — the ones that took reading the
enforcement library to find — without any test grepping for literals. A
derivation that only finds `os.Getenv("LITERAL")` would pass on this repository
today while missing exactly the class that made this spec necessary.

### Deterministic checks

#### Test
- Command: a Go test in `pose-mcp/internal/...` covering contract-versus-repository agreement
- Scope: environment names, port, build contexts
- Expected: pass, with injected-drift cases failing

#### Build
- Command: `bash tests/release/container-build.sh`
- Scope: the build contexts the contract declares are the ones that build
- Expected: pass

### Execution log
- Date: 2026-08-09
- Environment: linux/amd64.
- The derivation was made honest before the contract was written.
  `ConfigFromEnv` now builds its reads from an exported `ConfigEnvSuffixes`, and
  two tests keep that list from being decorative: one sets each declared suffix
  and requires the resulting config to change; the other parses `extract.go` and
  compares the declared list against the `prefix + "X"` reads found there. Both
  were proven by injection — a declared `GHOST_OPTION` is reported as declared
  but unread, and a `SECRET_BACKDOOR` read added without declaring it is
  reported as undeclared.
- The derivation immediately found more than the manual audit that motivated
  this spec had: `HARNE8_PROJECT_ID_PREFIX` and `GF_SIDECAR_EXCHANGE_LOG` are
  part of the surface and appeared in no earlier hand-made list, including mine.
- The contract check was proven on the failure that matters. Renaming
  `REQUIRE_PRINCIPAL` to `REQUIRE_CALLER` in the source — the silent rename this
  spec exists for — reports the contract as disagreeing with the repository, as
  does changing the Dockerfile's `EXPOSE`.
- Final surface: 21 variables for `pose-mcp`, 9 for the sidecar, ports 8799 and
  8770 read from the Dockerfiles.

### Results summary
- Successes: the contract is published, derived rather than restated, and both
  its own drift and the derivation's decay are caught
- Failures: none
- Warnings: publication is not consumption — the monorepo still restates these
  facts until it adopts the artifact

### Requirement trace
- R1 [satisfied] governance:composition-contract evidence:integration check:delivery-integration report:composition-contract.json — build context, Dockerfile path, port and environment surface are published per image, schema-versioned
- R2 [satisfied] test:pose-mcp/internal/version/composition_contract_test.go — the prefixed keys come from `mcpenforce.ConfigEnvSuffixes` rather than a literal search, which is why the published surface includes names no `grep` of this repository would find
- R3 [satisfied] check:composition-contract — a renamed variable and a changed `EXPOSE` both report the contract as disagreeing, naming the regeneration command
- R4 [satisfied] report:composition-contract.json — `schema_version: 1`, so a consumer can detect a shape change instead of misreading it

### Known gaps
- **Publication is not consumption.** The monorepo still restates every one of
  these facts; until its compose derives from this file, both sides can still
  drift and the contract is documentation with a test attached. That is the
  first follow-up and it is outside this repository.
- Volumes are not declared. The container reads `HARNE8_PROJECTS_DIR`, which is
  ours, but where the consumer mounts it is the consumer's decision — so the
  contract names the variable and not the path.
- The literal derivation matches `os.Getenv("PREFIX_...")`. A read constructed
  some other way, in a package that does not use `ConfigFromEnv`, would be
  missed the same way the prefixed keys originally were.

---

## 7. Final Report

### Delivered scope
`composition-contract.json` publishes, per image, the build context, Dockerfile,
port and full environment surface — derived from the source, including the
prefixed keys that exist nowhere as literals — and a check fails when the file
and the repository disagree.

### Files and modules changed
- composition-contract.json
- mcp-enforce/extract.go, mcp-enforce/config_env_test.go
- pose-mcp/internal/version/composition_contract_test.go

### Validation executed
- Command: the contract check plus injected drift; the suffix-list tests plus
  injected decay
- Result: all four injections reported; clean tree passes

### Residual risks
- The contract is only half the loop. Nothing changes for the composition until
  the monorepo consumes it.

### Follow-ups

- [open] Publishing the contract does not make the composition consume it. Until `docker-compose.prod.yaml` derives its build contexts, ports and environment names from the published artifact, both sides still restate the same facts and the contract is documentation. Coordinate the monorepo-side change, and decide whether a check there should fail when its compose disagrees with the contract this repository ships. (owner:@pose-maintainers crit:medium review:2026-10-09)
- [open] The internal port is declared in the Dockerfile's `ENV` and `EXPOSE`, in the compose mapping, and would be declared a fourth time by the contract. Decide which one is authoritative and have the others derive from it, rather than adding a copy that agrees today. (owner:@pose-maintainers crit:low review:2026-11-13)
