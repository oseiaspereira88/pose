---
slug: pose-docs-canonical-route
status: in-progress
created_at: 2026-09-06
completed_at:
supersedes:
depends_on:
priority: 1
components: docs, ci
delivers: governance:docs-canonical-route
---

# Spec: Docs canonical route

## 1. Intent

### Goal
Reduce POSE's documentation to one canonical address, and make the other
address redirect to it rather than compete with it.

### Business value
Two complete POSE documentation sites are live right now, and both serve HTTP
200:

- `https://docs.harne8.com/POSE/` — linked from the site, and the destination
  the published sitemap reserves.
- `https://oseiaspereira88.github.io/pose/` — linked from the README badge and
  from the README body.

This is not a stale link. It is two authorities. Search engines split ranking
between them, the two can drift apart in content, and a reader who lands on
one has no way to know the other exists or which is current. Every inbound
link the launch generates would be divided between them.

### Constraints
- The decision is architectural and must be recorded as an ADR, not settled by
  editing links. A prior ADR reserved `docs.harne8.com` as the unified portal;
  changing that is a change to the published sitemap contract.
- GitHub Pages must keep serving something. Existing inbound links and any
  cached search results have to land somewhere useful, not on a 404.

### Non-goals
- Migrating documentation content. Both sites are built from the same
  `docs-site/` source; this is about address authority, not authoring.
- Deciding the visual shell of the docs portal.

---

## 2. Requirements

### Functional
- R1: One canonical documentation base URL shall be recorded in
  `.pose/public/claims.yaml` and referenced from there by every surface.
- R2: Every documentation page shall emit a `<link rel="canonical">` pointing
  at the canonical host, including the pages served from the non-canonical one.
- R3: The non-canonical host shall redirect to the canonical equivalent
  preserving the path, so existing deep links survive.
- R4: The README badge and body links shall point at the canonical host.
- R5: `pose public-claims --strict` shall fail when any surface links to a
  documentation host that is not the canonical one.

### Compatibility
- Existing deep links keep resolving via R3.

---

## 3. Technical Plan

### Affected areas
- `docs-site/mkdocs.yml` — `site_url` and canonical emission
- `.github/workflows/docs.yml` — publication targets
- `README.md`, `README.pt-BR.md`

### Artifacts
- modified: docs-site/mkdocs.yml
- created: .pose/adr/2026-09-07-one-canonical-documentation-address.md

The README links (R4) were repointed in `c56427a4` and `3ed4cc01`, and the docs workflow in a commit of its own, under other specs' trailers; they are described here rather than claimed, since no commit with this spec's trailer changed them.

### Delivery targets
- governance:docs-canonical-route module:. profile:release-governance entrypoint:docs-site/mkdocs.yml

### Technical risks
- GitHub Pages serves static files and cannot issue a true 301 on its own; a
  meta-refresh plus `rel=canonical` is the available approximation and is
  weaker for SEO. If that proves insufficient, the fallback is to stop
  publishing to Pages entirely once inbound traffic has decayed.

### Known gaps at this point
- R3 (path-preserving redirect from the non-canonical host) is **not**
  implemented. `rel=canonical` is emitted from every page, which addresses the
  ranking split, but a reader who lands on the GitHub Pages URL stays there.
  Deferred deliberately: removing or redirecting a live address before its
  inbound traffic has decayed breaks existing links for no gain, and the
  canonical tag is the reversible half of the change.

---

## 4. Tasks

### Planning
- [x] Record the ADR: which host is canonical and why
- [x] Confirm the decision against the published sitemap contract

### Implementation
- [x] Increment 1: Declare the canonical route in the claims contract (R1)
- [x] Increment 2: Emit canonical links from every page (R2)
- [ ] Increment 3: Redirect the non-canonical host path-preservingly (R3) — deferred: hosting configuration outside the repository
- [x] Increment 4: Repoint README badge and links (R4)
- [x] Increment 5: Gate on it (R5)

---

## 5. Decisions

### Decision 1
- Date: 2026-09-06
- Context: `docs.harne8.com/POSE/` versus `harne8.com/docs/pose/` versus
  keeping GitHub Pages canonical.
- Options considered:
  1. `docs.harne8.com` — matches the existing sitemap contract and the prior
     ADR; scales to GraphForge and Platform under one portal.
  2. `harne8.com/docs/pose/` — keeps documentation inside the institutional
     origin, which helps brand continuity and consolidates domain authority,
     but contradicts the published sitemap and is a larger change.
  3. GitHub Pages canonical — reinforces that POSE is independent of Harne8,
     but hands the project's documentation identity to a personal namespace
     URL that cannot carry the product's brand.
- Decision: option 1, `docs.harne8.com/POSE/`.
- Rationale: brand continuity is a problem of shared visual shell and
  navigation, not of DNS. Option 1 already has an ADR, a sitemap entry and a
  live deployment, so it is the choice that requires justification only to
  *abandon*. Option 3 actively harms the independence argument it is meant to
  serve: POSE's independence rests on Apache-2.0, local-first operation and no
  required account — not on the hostname of its docs.
- Consequences: GitHub Pages becomes a redirect surface. The independence
  claim must therefore be carried explicitly in the documentation content,
  since the hostname no longer carries it.

---

## 6. Validation

### Deterministic checks

#### Security / Contract
- Command: `pose public-claims --strict`
- Expected: exit 0; no surface links to a non-canonical docs host

#### Test
- Command: `bash tests/docs/canonical-route.sh`
- Scope: a sample of deep paths on the non-canonical host
- Expected: each redirects to the canonical equivalent with the path preserved

### Requirement trace

- R1 [satisfied] check:public-claims evidence:manual <`.pose/public/claims.json` (the contract settled on JSON, not the YAML the requirement named) records `docs_canonical: https://docs.harne8.com/POSE/`, and every declared surface is checked against it>
- R2 [deferred-integration: xref:proj.harne8/spec:harne8-pose-open-integrations-reconciliation] <`docs-site/mkdocs.yml` sets `site_url`, so every page this repository builds emits the canonical link: on 2026-10-09 the non-canonical host served `<link rel="canonical" href="https://docs.harne8.com/POSE/concepts/">`; the canonical host itself is served by the Harne8 docs portal, whose pages emit no canonical link>
- R3 [deferred-integration: spec:pose-docs-canonical-route] <a path-preserving redirect is configured on the hosting of oseiaspereira88.github.io, outside the repository; on 2026-10-09 it still answers 200 with the canonical link pointing at the new host>
- R4 [satisfied] check:public-claims evidence:manual <README badge and links point at docs.harne8.com/POSE; no surface links to the deprecated host>
- R5 [satisfied] check:public-claims evidence:integration test:TestPublicClaimsFailsOnSchemaOrgVersionAndNonCanonicalDocs

---

## 7. Final Report

### Delivered scope

The canonical documentation route is declared in the public claims contract, the README points at it, `pose public-claims --strict` fails on the deprecated host, and every page this repository builds names the canonical URL. The redirect of the old host and the canonical link on the Harne8-served host are outside this repository.

### Residual risks

- Reviewed in the same session that implemented it, with no separate reviewer execution: declared, not independent.

### Follow-ups

- [open] Configure a path-preserving redirect from oseiaspereira88.github.io/pose to docs.harne8.com/POSE; the old host still answers 200 (owner:@pose-maintainers crit:low review:2026-11-09)
- [open] Make the Harne8 docs portal emit `<link rel="canonical">` on the POSE pages it serves at docs.harne8.com/POSE (owner:@harne8-platform crit:low review:2026-11-09)
