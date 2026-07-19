# Notifications Specification

## ADDED Requirements

### Requirement: Shipment notification

The system SHALL send a notification when an order transitions to shipped.

#### Scenario: Order ships

- **WHEN** an order transitions to shipped
- **THEN** the user receives a notification within 60 seconds

### Requirement: Duplicate suppression

The system SHALL NOT send a duplicate notification for the same shipment
event.

#### Scenario: Retry of the same event

- **WHEN** the shipment event is redelivered by the queue
- **THEN** no second notification is sent
