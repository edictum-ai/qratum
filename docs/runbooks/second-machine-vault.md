# Qratum vault second-machine runbook

Each machine is its own capture sensor.

## On every machine

1. Install the global Claude Code SessionEnd hook:

   ```sh
   qrt hook install
   ```

2. Backfill the local Claude transcript history:

   ```sh
   qrt vault backfill
   ```

3. Check preservation health:

   ```sh
   qrt vault doctor
   ```

## How two machines merge

Vault blobs are content-addressed by sha256, so two machines can merge
`~/.qratum/` without duplicate blob growth.

- `raw/blobs/sha256/**` dedups cleanly by digest
- `raw/refs/raw_<digest12>.json` is one ref per digest; copying missing refs is enough
- `events/` and `state/vault.json` are local operational history, not the source of truth for raw preservation

Practical merge rule: copy the missing files from one machine's `~/.qratum/`
into the other's, then rerun `qrt vault doctor`.

## Scope limit in v1

Cloud-only sessions are not captured in vault v1.

If a session starts and ends on vendor-managed infrastructure and never writes a
local transcript, the local SessionEnd hook will not see it. Each machine still
needs its own `qrt hook install` and `qrt vault backfill`.
