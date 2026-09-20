---
spec: review-policy-adoption-is-the-instances
category: fixed
breaking: false
---

`.pose/policy/review.json` is no longer shipped as a byte copy of this repository's
live policy. A fresh `pose install` received this repository's `adopted_at`, a dated
field per governed contract and the overlay profiles this repository had adopted;
because the stamp skips a contract whose key is already present, the instance never
earned its own dates and silently inherited an exemption boundary belonging to
someone else. The distribution now ships the contract shape with no dates and no
adopted overlays, and install stamps the day the instance received the policy and
each contract. An existing instance keeps what it recorded, including under `--force`.
