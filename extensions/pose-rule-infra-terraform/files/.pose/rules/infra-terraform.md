# Rule: Infrastructure Terraform

## When to consult

Consult this guide for Terraform modules, OpenTofu configurations, cloud resource definitions (AWS, GCP, Azure), state management, and provider setups.

## Required patterns

- Lock all provider and required Terraform versions explicitly in `versions.tf` / `required_providers` blocks.
- Pin state files to secure remote backends (S3 with DynamoDB locking, GCS, Terraform Cloud) with server-side encryption enabled.
- Follow least-privilege principles on IAM roles, policies, and security groups.
- Modularize infrastructure components with clear input variables, descriptions, and explicit output contracts.
- Use variable validation rules and sensitive flags (`sensitive = true`) on secrets.

## Blocking anti-patterns

- Committing plain-text credentials, access keys, or API tokens into `.tf` files or git repositories.
- Using wildcard `*` permissions on IAM actions or resource ARNs without explicit documented justification.
- Maintaining unencrypted local `terraform.tfstate` files in production or shared branches.
- Modifying cloud resources manually (drift) without reconciling terraform state.
- Defining open ingress rules (`0.0.0.0/0`) on sensitive database or administrative ports (SSH, RDP, Postgres, MySQL).

## Minimum checks

- Run `terraform fmt -check` across the infrastructure directory.
- Run `terraform validate` in initialized modules.
- Run security linters like `tflint` or `checkov` / `tfsec` without high or critical severity violations.

## Precedence in multi-domain conflicts

- Apply the most restrictive security, contract, and operational rule when domain rules conflict.
- Prefer verifiable check evidence and explicit risk mitigation when speed conflicts with control.
- Record the precedence decision and objective rationale in the review.

## Recurrence traceability

> Also apply: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
