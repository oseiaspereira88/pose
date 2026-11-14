# ADR: Governance association links are read with a CommonMark parser

## Status
Accepted

## Context

Spec `pose-closeout-review-policy-by-cost-benefit` (R3) makes a roadmap's
closeout review depend on its ADR associations: `adrs:` on the roadmap,
`roadmaps:` on the ADR, or a local Markdown link either way. Under
`when-justified` a roadmap with more than one canonical spec and no
association is exempt, so a real link the reader misses grants an exemption
the policy does not allow.

The links were read by a hand-written extractor: two regular expressions for
inline and reference links, then ad hoc decoding. Review rounds 12 to 16 of
that spec each found valid CommonMark in which a real association vanished
without a diagnostic and an exemption followed:

- a filter for code and comments truncated fence delimiters, ignored indented
  code and let a comment opener inside code swallow a later link (N44, N46);
- a destination with balanced parentheses was cut short (N47);
- percent escapes, character references and backslash escapes were not
  decoded, or decoded in the wrong order (N48, N52);
- a reference definition spanning two lines, inside a block quote, or with an
  escaped bracket in its label was not seen (N50);
- an escaped `>` inside an angle-bracket destination ended it early (N51).

Each fix opened the next variant. The extractor approximates a grammar that
is defined by a specification, and every approximation is a place where an
exemption can be manufactured.

## Decision

Read the structure of an association-bearing document (a roadmap body, an
ADR body) with a CommonMark parser, `github.com/yuin/goldmark`, and take the
association links from its syntax tree: every link and image node, with
reference definitions resolved, block quotes and multi-line labels handled,
and code spans, code blocks and comments excluded as CommonMark excludes
them.

Decode each destination in POSE, in CommonMark precedence: one left-to-right
pass in which a backslash escape makes the next ASCII punctuation literal and
an unescaped `&` starts a character reference; then cut the fragment and the
query; then decode percent escapes. goldmark's own HTML renderer applies
backslash escapes before character references, which turns `\&num;` into `#`;
the association reader does not use the renderer and does not inherit that
order.

Fail closed where the parser cannot speak for the rendered document: raw HTML
(inline or block) carrying an `href` or `src` attribute leaves the answer
unknown, as does a destination whose percent escapes do not decode. A
document the parser reads normally is not otherwise second-guessed: a link
shown inside code is not an association, and a destination with balanced
parentheses is read whole.

## Alternatives rejected

- **Keep fixing the extractor.** Five rounds showed the variants do not
  converge; the extractor would have to become a CommonMark parser to stop.
- **Fail closed on any escape or entity character.** It stops the silent
  drops but makes ordinary prose and examples require review, and still
  misses structural forms such as reference definitions in block quotes.
- **A C CommonMark library through cgo.** The engine is a static, pure Go
  distribution; cgo would change how it is built and released.

## Consequences

- The engine gains one dependency, `github.com/yuin/goldmark` v1.8.4: MIT
  licensed (compatible with the distribution's Apache-2.0), pure Go, no
  transitive module dependencies, maintained, and clean under `govulncheck`
  when adopted. It is used only to parse; nothing is rendered with it.
- The association reader follows CommonMark, not GitHub Flavored Markdown
  extensions: an extension that creates links from text (autolinked bare
  paths) is not an association, as a bare path in prose never was.
- Follow-up F6 of the closeout review policy spec (links shown as code
  counting as associations) is resolved by this decision.
- A future change of the parser version is a change of the association
  contract and is reviewed as one.
