---
spec: pose-ossf-security-baseline
category: security
breaking: false
refs:
---

Pull requests now run CodeQL static analysis, known-vulnerability scanning,
full-history secret detection and dependency review; OpenSSF Scorecard
measures the pipeline weekly. Workflow permissions and action pinning are
enforced by a contract test with owned, expiring exceptions, and the release
pipeline refuses to publish while an unwaived critical finding exists.
