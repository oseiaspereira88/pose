# Rule: Knowledge Governance

## When to consult

Consult this guide when creating, updating, reviewing, or removing artifacts under `.pose/knowledge/`.

## TTL and retention

- Set `expires_at` on every artifact when it is created.
- Use a default TTL of 30 days for notes, decision logs, and handoffs.
- Use a TTL of up to 90 days only with a justification recorded in the artifact body.
- Treat an artifact without `expires_at` as non-compliant and block its creation or merge.

## Reusable cross-execution format

- Structure reusable context as a handoff with fixed sections: Context, Current state, Next checks, Risks, and Next owner.
- Keep `source_refs` linked to the relevant spec, workflow, and executed check commands.
- Record `last_reviewed_at` in the body to make effective cross-execution updates traceable.

## Archiving and purging

- Run biweekly triage to list expired artifacts.
- Move expired artifacts to `.pose/knowledge/archive/` when they retain audit value.
- Purge archived artifacts 180 days after expiration unless documented legal or compliance requirements apply.
- Record every archive and purge action in the housekeeping log.

## Sensitive content

- Prohibit secrets, tokens, credentials, private keys, and equivalent material.
- Prohibit personal data and non-anonymized customer data.
- Do not copy restricted incidents or reports in full; keep only a controlled reference.
- Classify frontmatter `sensitivity` as `public-internal` or `restricted`.
- Remove identified sensitive content immediately and open a security follow-up.

## Ownership and review

- Keep `@pose-maintainers` as the primary governance owner.
- Require an owner in every artifact frontmatter.
- Review expiration biweekly and quality monthly.
- Escalate a backlog overdue by more than two cycles to the primary owner.
- Block backlog growth when `list-expired` exceeds the operational limit: zero in strict mode, two in tolerant mode.

## Minimum operational checks

- Run `./pose knowledge-check --strict` biweekly to validate the expired backlog.
- Run `bash .pose/scripts/pose-knowledge-housekeeping.sh list-expired` for detailed triage.
- Run `bash .pose/scripts/pose-knowledge-housekeeping.sh archive-expired --dry-run` before applying changes.
- Execute destructive actions only with an explicit `--apply` flag.
