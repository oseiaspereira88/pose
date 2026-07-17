# Security Policy

## Supported versions

POSE is pre-1.0. Only the latest released version receives security fixes.

## Reporting a vulnerability

**Do not open a public issue for security reports.**

Email **oseiaspereira88@gmail.com** with:

- a description of the vulnerability and its impact;
- reproduction steps or a proof of concept;
- the POSE version / commit hash affected.

You will receive an acknowledgment within **5 business days**. We ask for up to
**90 days** of coordinated disclosure before any public write-up; we will credit
you in the release notes unless you prefer otherwise.

## Scope notes

- The POSE engine is designed to run **offline**: gates make no network calls.
  Any gate observed performing network I/O is itself a reportable issue.
- The MCP server (`pose-mcp`) is read-heavy by design; any write path reachable
  through an MCP tool without explicit configuration is a reportable issue.
- Secrets do not belong in `.pose/` artifacts. A template or workflow that
  encourages placing credentials in versioned files is a reportable issue.
