---
spec: review-verify-retains-completed-scopes
category: fixed
breaking: false
---

`pose review verify` and federated acceptance keep the approval a closed scope was closed with, as `pose review-check` already did. Before, the next validation run after any closeout, or any check added to the validation matrix, made every closed scope read as superseded, so a consumer pinning a newer engine saw all upstream reviews as unapproved until each was resealed and reattested. A newer rejection or request for changes still wins over an older approval, and `review-check` no longer fell back past one. A changed federated manifest, such as revoked trust or a moved source, still stales a closed scope.
