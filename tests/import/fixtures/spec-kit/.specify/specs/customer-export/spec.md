# Feature Specification: Customer export

Input: Operators need to export customer records for audit.

## User Scenarios & Testing

An operator requests an export and receives a signed CSV within the retention
window.

## Requirements

- FR-001: The system MUST export customer records as CSV.
- FR-002: The system MUST reject an export request without an audit reason.

## Success Criteria

Export completes within 30 seconds for 100k records.

## Assumptions

Audit retention is already configured.

## Out Of Scope

Streaming exports.
