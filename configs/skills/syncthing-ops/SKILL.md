---
name: syncthing-ops
description: >-
  Syncthing sync ops via REST API — folders/devices status, adding folders, rescan, conflict resolution; never hand-edit config.xml. Use when the user asks about sync state, adding a synced folder, stale files, conflicts, or API-driven config. Triggers: syncthing, sincronizar pasta, sync, conflito de sync, config.xml, rescan, staggered, pasta nova.
license: MIT
---

# Syncthing Operations (API-driven)

## Golden rule

Never hand-edit `config.xml` — drive everything through the GUI/REST API.
A hand edit races the running daemon and gets overwritten or rejected.

## Auth and endpoints (local per machine)

- API key and GUI credentials live on the machine (`~/.config/opencode/secrets/` 0700 or the service's own config) — never in the repo
- Default local endpoint: `http://127.0.0.1:8384/rest/...` with header `X-API-Key: <key>`
- Prove access first: `curl -s -H "X-API-Key: $KEY" http://127.0.0.1:8384/rest/system/status | jq '{version, myID}'`

## Read-only diagnostics first

- `.../rest/system/status` — version, own ID
- `.../rest/system/connections` — peer sessions, rates, addresses
- `.../rest/stats/folder` — per-folder last scan/sync state
- `.../rest/db/status?folder=<id>` — need/scanning state for one folder
- GUI Events endpoint (`.../rest/events`) for "what just changed" instead of guessing

## Adding a folder (the safe order)

1. Decide the folder ID and label before touching the API
2. Create via `POST .../rest/config/folders` with the full folder object (path must exist on every participating device; paths differ per device — resolve per-device, never hardcode one path for all)
3. Share to devices in the same call (`devices: [{deviceID}]`), or follow with `POST .../rest/config/devices` for a new device first
4. Trigger: `POST .../rest/db/scan?folder=<id>` and watch `.../rest/db/status?folder=<id>` until `state: idle`
5. Back up the effective config first (`GET .../rest/config` → file) so a bad write is one PUT away from rollback

## Conflicts and safety

- Sync is not backup: versioning (staggered/trashcan) only versions changes arriving from *other* devices — pair with a real backup (see backup tooling) for anything irreplaceable
- Conflict files (`*.sync-conflict-*`) mean two writers won at once: diff, keep one, delete the other — never bulk-delete by pattern without listing first
- Never put a `.git` working copy inside a synced folder — lockfiles and index churn corrupt both sides
- Never sync live database files or emulator save-states across builds — copy quiesced snapshots instead

## Rules

- Small reads before writes: `GET` the object, edit, `PUT/POST` it back
- Restart via the API (`POST .../rest/system/restart`) rather than killing the process
- Report folder ID + device ID short-prefix in every summary so the next step is unambiguous
