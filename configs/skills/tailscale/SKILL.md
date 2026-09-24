---
name: tailscale
description: >-
  Tailscale tailnet ops — status/inventory via JSON, up/down, exit nodes, ACL notes, and exposing a local port safely (tailnet or internet) plus teardown. Use when the user asks about tailnet, exit node, serve/funnel, device inventory, or reaching a box with no public ports. Triggers: tailscale, tailnet, exit node, serve, funnel, tailnet-only, sem porta pública, inventário de dispositivos.
license: MIT
---

# Tailscale (tailnet-only networking)

Tailnet is the only ingress: no public ports. Everything below is generic —
device names, IPs and domains live on the machine, never in this file.

## Status and inventory (read-only first)

- `tailscale status` — human overview (peers, online/offline, exit-node flags)
- `tailscale status --json` — machine-readable inventory; parse with `jq`:
  `tailscale status --json | jq '.Peer | to_entries[] | {host: .value.HostName, ip: .value.TailscaleIPs, online: .value.Online}'`
- `tailscale ip -4` — this node's tailnet address
- `tailscale ping <peer>` — prove the path before blaming DNS or the app

## Up / down / exit nodes

- `tailscale up` — join (interactive login on first run; subsequent runs reuse state)
- `tailscale down` — leave the tailnet (nothing is reachable afterwards — by design)
- `tailscale exit-node list` — candidate exit nodes
- `tailscale up --exit-node=<node> --exit-node-allow-lan-access` — route via exit node, keep LAN reachable
- `tailscale up --exit-node=` — stop using an exit node

## Exposing a local port (and tearing it down)

Prefer tailnet-only (`serve`); reach for `funnel` (public internet) only with an explicit reason.

1. Tailnet-only: `tailscale serve --bg <port>` — serves localhost on the tailnet
2. Check: `tailscale serve status`
3. Public (explicit only): `tailscale funnel --bg <port>` — bound to the tailnet domain with HTTPS
4. Teardown after use: `tailscale serve reset` (also stops funnel)

## ACL notes

- ACLs live in the admin console (tag-based), not on the node — this skill never edits them
- Common shape: restrict by tag/autogroup, default-deny, then allow what the setup needs
- Debugging order when a peer is unreachable: `status` (online?) → `ping` (path?) → ACL (allowed?) → local firewall (`ufw status`)

## Rules

- Auth keys and OAuth clients are secrets: store in `~/.config/opencode/secrets/` (0700) or the OS keyring — never in the repo, chat logs, or skill files
- Never hardcode IPs, hostnames or domains in scripts — resolve via `status --json` at runtime
- `tailscale up/down` changes connectivity for the whole box — confirm before running on a remote you are connected through
