# ADR: One canonical documentation address for POSE

## Status
Accepted (2026-09-07) — spec `pose-docs-canonical-route`, roadmap
`community-launch`

## Context

Two complete POSE documentation sites are live right now, and both serve
HTTP 200:

- `https://docs.harne8.com/POSE/` — linked from the Harne8 site, and the
  destination the published sitemap reserves for the unified docs portal.
- `https://oseiaspereira88.github.io/pose/` — built and deployed by this
  repository's own `.github/workflows/docs.yml`, and linked from the README
  badge and the README body.

This is not a stale link that someone forgot to update. It is two authorities
serving the same content from different addresses. Search engines split
ranking between them, the two can drift apart the moment either is edited
independently, and a reader who lands on one has no way to know the other
exists or which is current. Every inbound link the Community Launch generates
would be divided between them.

The question had to be settled before traffic was sent anywhere, and it is
architectural rather than editorial: a prior ADR reserved `docs.harne8.com` as
the unified portal for POSE, GraphForge and Platform, and the published
sitemap encodes that. Changing it is a change to a published contract, not a
link edit.

## Decision

**`https://docs.harne8.com/POSE/` is the canonical documentation address.**

1. `docs-site/mkdocs.yml` declares `site_url` as the canonical address, so
   mkdocs-material emits `<link rel="canonical">` pointing there from every
   page — including the pages served by GitHub Pages.
2. The README badge and body links point at the canonical address.
3. GitHub Pages keeps serving the build. It becomes a mirror that declares
   itself non-canonical rather than a second authority.
4. `pose public-claims --strict` fails when any declared surface links to the
   non-canonical host, so the split cannot silently reappear.

## Options considered

**`harne8.com/docs/pose/`** — keeps documentation inside the institutional
origin, which helps brand continuity and consolidates domain authority onto a
single hostname. Rejected because it contradicts the published sitemap and is
a materially larger change, and because the problem it solves — brand
continuity — is a problem of shared visual shell and navigation, not of DNS.
A docs portal with the same header, identity, product switcher and search
reads as the same product regardless of hostname.

**Keeping GitHub Pages canonical** — reinforces that POSE is independent of
Harne8, which is a real concern for an open-source project whose maintainer
also sells a platform. Rejected because it actively harms the independence it
is meant to serve: it hands the project's documentation identity to a personal
namespace URL that cannot carry the product's brand, and it makes the docs
look like a side artifact of a personal account rather than a product surface.

POSE's independence does not rest on a hostname. It rests on Apache-2.0, a
public repository, local-first operation, no required account, no mandatory
hosted service, and functioning with no part of the Harne8 platform present.
Those properties are unaffected by where the documentation is served.

**`docs.harne8.com`** — chosen. It already has an ADR, a sitemap entry and a
live deployment, so it is the option that would require justification to
*abandon* rather than to adopt. It also scales: `docs.harne8.com/POSE/`,
`/graphforge/` and `/platform/` under one portal is the structure the sitemap
already anticipates.

## Consequences

**Positive**

- One address accumulates ranking, inbound links and reader familiarity.
- The choice is enforced by a gate rather than by memory.
- The docs portal can grow to cover the other products without a second
  migration.

**Negative / accepted trade-offs**

- GitHub Pages serves static files and cannot issue a true 301 on its own. A
  `rel=canonical` is weaker than a redirect for consolidating ranking, and a
  reader who lands on the Pages URL stays there. If that proves insufficient,
  the fallback is to stop publishing to Pages once inbound traffic to it has
  decayed — deliberately deferred rather than done now, because removing a
  live address before its traffic decays breaks existing links for no gain.
- POSE's documentation now resolves under a Harne8 hostname, which is exactly
  the association the independence concern worries about. That association has
  to be carried explicitly in the documentation content — the license, the
  local-first operation and the absence of a required account — since the
  hostname no longer carries it implicitly.

**Neutral**

- No documentation content moves. Both sites already build from the same
  `docs-site/` source; this decides address authority, not authoring.
