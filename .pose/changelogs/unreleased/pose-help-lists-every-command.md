---
spec: pose-help-lists-every-command
category: fixed
breaking: false
---
`pose help` now lists every command the binary dispatches, in English and pt-BR: `setup`, `context`, `identity`, `issuer`, `adopt`, `migrate`, `specs`, `spec-format`, `start`, `action`, `assess`, `docs-check`, `docs-init`, `docs-review`, `docs-sync` and `release-package-manifests` were missing (and `public-claims` in pt-BR). A test reads the dispatch from the source and fails when a command is not advertised, and the manual's command reference now documents `action`, `issuer`, `migrate` and `release-package-manifests`.
