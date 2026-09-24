---
name: linux-performance-tuning
description: >-
  Audit and tune Linux performance for Ubuntu Server 24.04+ and CachyOS without mixing OS profiles; covers zram, sysctl, fstrim, journald, swap planning, schedulers, governors, services, benchmarks, and rollback. Use when running `envctl run performance`, reviewing server performance, or deciding whether a Linux tweak is safe. Triggers: performance, Ubuntu server, CachyOS tuning, zram, swap, sysctl, fstrim, journald, I/O scheduler, governor, benchmark, lentidão, performance Linux.
license: MIT
---

# Linux Performance Tuning

## Quando usar

Use after `envctl run performance`, before changing a server's performance
settings, or when `envctl doctor` reports the read-only `Performance` section.
The envctl profiles are deliberately OS-specific:

- **Ubuntu Server 24.04+**: `envctl run performance --dry-run` followed by
  `envctl run performance`; installs the zram generator and applies the managed
  Ubuntu sysctl drop-in.
- **CachyOS**: `envctl run performance --dry-run` followed by
  `envctl run performance`; ensures `zram-generator` is present but does not
  replace the existing zram, scheduler, governor, or GPU profile.

Never run the Ubuntu profile on CachyOS or the CachyOS profile on Ubuntu. The
CLI rejects unsupported exact OS identities.

## Inspect first

```bash
envctl run performance --dry-run
cat /proc/swaps
zramctl
systemctl is-enabled fstrim.timer
systemctl is-active fstrim.timer
sysctl vm.swappiness vm.vfs_cache_pressure
journalctl --disk-usage
for f in /sys/block/*/queue/scheduler; do printf '%s: ' "$f"; cat "$f"; done
for f in /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor; do printf '%s: ' "$f"; cat "$f"; done
```

`envctl doctor` reports swap, zram, governors, schedulers, journald, `fstrim.timer`,
and selected services as informational evidence. A missing optional component
is not a health error.

## What the command may change

`run performance` is opt-in and outside `run all`:

- Ubuntu 24.04+ may install `systemd-zram-generator`, load the module, reload
  systemd units, and start the generated `dev-zram0.swap` unit when the device
  is absent.
- Ubuntu 24.04+ may write `/etc/sysctl.d/90-envctl-performance.conf` and apply
  that file with `sysctl -p`.
- CachyOS may install `zram-generator` only when it is missing and may start
  the generated `dev-zram0.swap` unit when the device is absent.
- Package checks, sysctl file content, and backups are idempotent.

The command never creates or removes a swapfile, changes an existing swapfile,
changes zram size/algorithm/priority, changes journald limits, changes CPU
governors or EPP, changes an I/O scheduler, starts/stops unrelated services,
edits kernel parameters, disables mitigations, or runs benchmarks. The only
service lifecycle it may trigger is the zram generator described above.

## Ubuntu server sysctl policy

The managed values are a conservative starting point, not universal tuning:

- `vm.swappiness=10`
- `vm.vfs_cache_pressure=50`
- `net.core.somaxconn=65535`
- `net.ipv4.tcp_max_syn_backlog=4096`
- `fs.file-max=2097152`

Review them against the workload. Database, storage, and high-concurrency
network services may need different values. Do not add `overcommit_memory`,
TCP congestion-control, scheduler, or hugepage settings without a benchmark.

## CachyOS policy

CachyOS already has a tuned zram/scheduler/governor profile on the managed
workstation. The performance command must be a no-op when `zram-generator` and
`/dev/zram0` are present. If zram is missing, install the package and start the
existing generator; do not create `/dev/zram1` or a second swap device.

## Swapfile planning

Swapfile creation is intentionally deferred. Before implementing it, record:

1. filesystem type and compression/NOCOW behavior;
2. free space and expected write amplification;
3. required size and whether hibernation is needed;
4. encrypted-swap policy;
5. rollback command and service dependency order.

Btrfs, XFS, ext4, encrypted swap, and hibernation require different setup.
zram is not a substitute for a hibernation swapfile.

## Safe verification and rollback

After an Ubuntu apply:

```bash
sudo sysctl -p /etc/sysctl.d/90-envctl-performance.conf
sudo cat /etc/sysctl.d/90-envctl-performance.conf
envctl doctor
```

A repeated `envctl run performance` must report the managed file as already up
to date. To roll back, restore the timestamped backup printed by envctl, then
apply that restored file with `sysctl -p`; do not delete the backup before the
replacement is verified.

## Benchmark rule

Run benchmarks only in staging or during an approved maintenance window. Use a
representative workload, record the baseline, change one variable at a time,
and keep the rollback artifact. A synthetic CPU score is not evidence that a
server workload improved.
