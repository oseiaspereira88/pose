---
spec: pose-validation-scanner-consolidation
category: added
breaking: false
refs:
---

`pose index`, `pose validate`, `pose install` and `pose init` now classify a repository's modules through one shared stack detector instead of two independently maintained ones, so a stack-detection fix reaches every command that relies on it. Cloudflare Workers (`wrangler.toml`/`.json`/`.jsonc`) is now a recognized stack with a runnable check, and `pose index`'s repo map now picks up Python and .NET modules it previously skipped entirely.
