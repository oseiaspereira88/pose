---
spec: pose-rule-extension-locale-parity
category: added
breaking: false
refs:
---

`pose extension install` now supports a `--locale` flag (auto-detected from the target when omitted, same as `pose install`), so an extension package can ship localized content alongside its base files. `pose-rule-backend-go` and `pose-rule-frontend-react` now ship a pt-BR variant, so a pt-BR instance installing either gets locale-consistent content instead of English-only text.
