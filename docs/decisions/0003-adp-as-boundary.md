# ADR 0003: ADP As Boundary

## Status

Accepted

## Decision

ADP is an import/export boundary, not Qratum's internal model.

## Reason

Qratum needs a product-native model while still exporting interoperable
trajectory data.

## Consequences

QratumSession remains the source of truth. ADP strict export is deterministic
and excludes Qratum-only fields.
