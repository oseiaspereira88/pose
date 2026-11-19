# Bind retroactive delivery attribution to signed Git ranges

## Status

Accepted. The maintainer authorized the capability and required a trusted maintainer signature on 2026-11-18.

## Context

The strict delivery gate reports 19 undeclared paths in published change sets. Their specs and review subjects are sealed. Commits that carry several spec trailers and later intervening commits account for the observed paths.

## Decision

Store a versioned attribution manifest in `.pose/policy/artifact-attributions.json` with a detached SSHSIG in `.pose/policy/artifact-attributions.json.sig`. Reuse SSH public keys registered to principals in the project's action policy. Require the named signer to be a human principal expressly assigned the maintainer role, and verify the signature using the dedicated `pose-retroactive-attribution` namespace. The signed bytes include the project ID, signer, baseline revision, reason, spec, complete attributed range, Git witness commit and exact observed action and path for each record. Anchor the baseline in the repository's raw history.

Apply a record exclusively to the corresponding undeclared finding in that exact range and expose the verified signer and reason. Keep the existing observed changes, claims and sealed subject provenance. Bind manifest and signature bytes through the existing policy-tree index input. Repositories without attribution records keep the same index read path. A newer range requires its own record, and ordinary artifact, review and closeout gates continue to run.

## Consequences

- Use an already registered maintainer key for the production manifest. A repository without one requires a separately authorized key enrollment.
- Archived review inputs and Git commit identities retain their original meaning.
- Implicit shared-trailer inference cannot account for every historical finding and would change future attribution.
- Changing the historical Artifacts sections would invalidate sealed subject provenance.
