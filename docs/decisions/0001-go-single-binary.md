# ADR 0001: Go Single Binary

## Status

Accepted

## Decision

Qratum uses Go and ships as a single `qrt` binary.

## Reason

The product needs local hooks, a daemon, CLI commands, and future server
components without a Python runtime.

## Consequences

Milestone A has no pip, venv, or Python runtime dependency.
