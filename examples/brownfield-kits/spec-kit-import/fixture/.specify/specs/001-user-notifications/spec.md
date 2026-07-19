# Feature Specification: User Notifications

**Created**: 2026-06-01
**Input**: User description: "Notify users when their order ships"

## User Scenarios & Testing

### User Story 1 - Shipment notice
A user with an order that just shipped receives a notification within one
minute of the shipment event.

## Requirements

- FR-001: System MUST send a notification when an order transitions to shipped.
- FR-002: System MUST record delivery status for every notification attempt.
- FR-003: System MUST NOT send a duplicate notification for the same shipment event.

## Success Criteria

- SC-001: 99% of shipment notifications are delivered within 60 seconds.
