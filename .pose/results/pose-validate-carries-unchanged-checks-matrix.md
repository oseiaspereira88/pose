# Strict matrix for pose-validate-carries-unchanged-checks, round 15

The [structured result](delivery-validation.json) is the run after the fixes for findings N40 and N41 of review round 14. The engine changed, so `--carry` ran everything. It passed. A first attempt failed only in the install journey, on an unauthenticated GitHub API rate limit (HTTP 403).
