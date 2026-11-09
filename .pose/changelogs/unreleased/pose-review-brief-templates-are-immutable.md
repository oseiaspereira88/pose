---
spec: pose-review-brief-templates-are-immutable
category: fixed
breaking: false
---
The text a review brief template version renders is now frozen by a golden digest per version: changing a sentence of a published version fails the build with a message saying that a change takes a new version, and a new version cannot be published without its digest. A run records and signs the brief it was sent, so a text that changed under one version name left its runs impossible to re-render.
