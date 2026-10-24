---
spec: pose-action-waiver-and-supersession-authority
category: security
breaking: true
---

Waiver and supersession now require operation-specific authority and proof under verified policy; requester withdrawal remains distinct from waiver. MCP retirement always needs proof, and arbitrary actor invalidation is refused. New lifecycle events reverify recorded proof on read. Schema-1 answer signatures cannot authorize retirement; unsigned legacy exemptions cannot satisfy verified policy. Review compatibility before publication or adoption; cross-journal supersession recovery remains a separate capability.
