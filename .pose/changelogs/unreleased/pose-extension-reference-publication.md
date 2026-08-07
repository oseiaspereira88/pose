---
spec: pose-extension-reference-publication
category: changed
breaking: true
refs:
---

The Kubernetes review rule is no longer embedded in every instance. It ships as
`pose-rule-kubernetes`, the project's first signed reference extension, and is
installed with `pose extension install` by repositories that actually deploy to
a cluster. Fresh installs no longer receive `.pose/rules/kubernetes.md`;
existing instances keep their copy, but it stops being refreshed by `pose
upgrade` because the engine no longer ships it.

This also gives the extension chain its first real artifact. The release
workflow signs the extension with cosign and publishes it beside the engine's
own assets, and the independent verification job now fetches it as a consumer,
checks the signature before executing anything, installs it, and requires a
tampered copy to be rejected. Until now every test of that path substituted a
fake for cosign.
