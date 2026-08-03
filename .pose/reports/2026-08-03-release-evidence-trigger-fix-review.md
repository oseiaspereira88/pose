# Review: release evidence trigger fix

Approved by `agent:independent-release-trigger-review-1`. The fix removes
producer-only files from publication evidence, adds no credential, limits the
workflow_run trigger to successful version-tag release runs and checks out the
exact tag. Workflow security and live v0.16.1 verification are the exit gates.
