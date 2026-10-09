---
spec: pose-mcp-review-attest-signed-only
category: added
breaking: false
refs:
---

New MCP tool `pose_review_attest` records a review attestation only inside a trusted issuer's envelope, the same proof `pose review attest --envelope` verifies. Its preview completes a draft for a sealed bundle, binding the confirmation to the exact conclusions when it names a confirming person, and returns the bytes to sign; a call that applies without an envelope is refused under any identity assurance. A trusted channel such as the Harne8 issuer can now deliver a person's confirmation of review conclusions over MCP.
