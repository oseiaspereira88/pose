# Rule: Kubernetes

## When to consult

Consult this guide for Kubernetes manifests, Helm or Kustomize, deployment configuration, scaling, and cluster operations.

## Required patterns

- Define `resources.requests` and `resources.limits` for every workload.
- Make liveness and readiness probes reflect actual application behavior.
- Use immutable image versions through fixed tags or digests; never use `latest`.
- Separate secrets from public configuration with Secret and ConfigMap resources.
- Choose rollout strategies that minimize downtime, such as rolling updates.
- Follow project traceability conventions for namespaces, labels, and annotations.

## Blocking anti-patterns

- Deploying without probes or resources, or with unnecessary elevated privileges.
- Using `:latest` for production images.
- Hard-coding secrets in manifests, charts, or values.
- Exposing services externally without minimum security and restriction policies.
- Changing manifests without considering rollout backward compatibility.

## Minimum checks

- Validate YAML and manifest structure without errors.
- Run `kubectl apply --dry-run=client`, or the equivalent, for the changed scope.
- Render Helm or Kustomize templates successfully.
- Verify applicable platform security and contract policies.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
