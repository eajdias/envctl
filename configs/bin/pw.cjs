#!/usr/bin/env node
/**
 * pw — hang-safe playwright-cli runner for Windows agent shells.
 *
 * WHY THIS EXISTS
 * `playwright-cli open`/`attach` spawns a long-lived daemon via
 * `spawn(process.execPath, args, { detached: true, ... })`. On Windows the
 * daemon inherits the parent Job Object, so any shell that waits on the
 * whole job tree (opencode bash tool, CI runners) never returns — even
 * though the snapshot output is correct (microsoft/playwright#41530,
 * opencode#24731, both closed as not planned, no upstream fix).
 *
 * This wrapper uses the ONLY pattern verified to return on Windows
 * (empirical matrix, 2026-09-17): spawn DETACHED + unref, then poll
 * `playwright-cli list` for session appearance. Synchronous waits
 * (execFileSync / spawn attached + waitForExit) HANG even though the
 * browser work completes. Linux/macOS are unaffected (detached+unref
 * detaches for real there) but the wrapper works identically everywhere.
 *
 * USAGE
 *   pw [--timeout <seconds>] <playwright-cli args...>   (default 120s)
 *   PW_TIMEOUT env overrides the default. The timeout only bounds how long
 *   we poll for the session — it never kills a healthy session.
 *   Exit code mirrors playwright-cli (124 on poll timeout, after reaping).
 *
 * EXAMPLES
 *   pw open https://example.com --browser=chromium
 *   pw --timeout 180 open https://example.com --headed
 *   pw snapshot --depth 4
 *   pw close-all
 */
'use strict';

const { spawn, execFileSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const POLL_MS = 1000;

function parseArgs(argv) {
  let timeoutSec = 120;
  if (process.env.PW_TIMEOUT) {
    const v = parseInt(process.env.PW_TIMEOUT, 10);
    if (Number.isFinite(v) && v > 0) timeoutSec = v;
  }
  const forward = [];
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--timeout' && i + 1 < argv.length) {
      const v = parseInt(argv[i + 1], 10);
      if (Number.isFinite(v) && v > 0) timeoutSec = v;
      i++;
    } else {
      forward.push(argv[i]);
    }
  }
  return { timeoutSec, forward };
}

function resolveCliJs() {
  // Resolve the real node entry behind the launcher shim, so every spawn
  // below runs shell-free: argv array straight to CreateProcess, no DEP0190
  // warning, no shell-injection surface. Layouts covered:
  //   - npm global: <prefix>\playwright-cli.cmd + <prefix>\node_modules\@playwright\cli\...
  //   - volta: <...>\Volta\bin\playwright-cli.cmd + <...>\Volta\tools\image\packages\@playwright\cli\...
  const candidates = [];
  try {
    const lines = execFileSync('where.exe playwright-cli', { encoding: 'utf8' }).split(/\r?\n/);
    for (const line of lines) {
      const w = line.trim();
      if (!w) continue;
      candidates.push(path.join(path.dirname(w), 'node_modules', '@playwright', 'cli', 'playwright-cli.js'));
      const m = w.match(/^(.*[\\/][Vv]olta)[\\/]bin[\\/]?/);
      if (m) {
        candidates.push(path.join(m[1], 'tools', 'image', 'packages', '@playwright', 'cli',
          'node_modules', '@playwright', 'cli', 'playwright-cli.js'));
      }
    }
  } catch {}
  if (process.env.LOCALAPPDATA) {
    candidates.push(path.join(process.env.LOCALAPPDATA, 'Volta', 'tools', 'image', 'packages',
      '@playwright', 'cli', 'node_modules', '@playwright', 'cli', 'playwright-cli.js'));
  }
  for (const c of candidates) {
    try {
      if (fs.existsSync(c)) return c;
    } catch {}
  }
  return null;
}

function cliSync(args) {
  // Short-lived commands (list, close-all, --version): no daemon spawn
  // involved, safe to run synchronously.
  const cliJs = process.platform === 'win32' ? resolveCliJs() : null;
  if (cliJs) {
    return execFileSync(process.execPath, [cliJs, ...args], { encoding: 'utf8', timeout: 60000 });
  }
  return execFileSync('playwright-cli', args, { encoding: 'utf8', timeout: 60000 });
}

function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}

async function main() {
  const { timeoutSec, forward } = parseArgs(process.argv.slice(2));
  if (!forward.length) {
    process.stderr.write('Usage: pw [--timeout <seconds>] <playwright-cli args...>\n');
    process.exitCode = 2;
    return;
  }
  const verb = forward[0];

  // Commands that never spawn a daemon: run them straight through.
  const passthrough = new Set([
    'list', 'close', 'close-all', 'kill-all', 'delete-data',
    'tab-list', '--version', '-V', '--help', '-h', 'install-browser',
  ]);
  if (passthrough.has(verb) || verb.startsWith('-')) {
    try {
      const out = cliSync(forward);
      if (out) process.stdout.write(out);
      return;
    } catch (err) {
      if (err.stdout) process.stdout.write(err.stdout);
      if (err.stderr) process.stderr.write(err.stderr);
      process.exitCode = typeof err.status === 'number' ? err.status : 1;
      return;
    }
  }

  // Daemon-spawning path (open/attach/...): detached + unref, then poll.
  // Shell-free (see resolveCliJs): argv array straight to CreateProcess.
  const cliJs = process.platform === 'win32' ? resolveCliJs() : null;
  const child = cliJs
    ? spawn(process.execPath, [cliJs, ...forward], { detached: true, stdio: 'ignore', shell: false })
    : spawn('playwright-cli', forward, { detached: true, stdio: 'ignore', shell: process.platform === 'win32' });
  child.unref();

  const deadline = Date.now() + timeoutSec * 1000;
  let lastList = '';
  // eslint-disable-next-line no-constant-condition
  while (true) {
    await sleep(POLL_MS);
    let list = '';
    try {
      list = cliSync(['list']);
    } catch {
      list = '';
    }
    lastList = list;
    // A new/updated session line means the daemon finished its work.
    // `open` prints nothing itself under detached spawn, so the list is
    // our completion signal. Snapshot files also appear, but list is
    // authoritative (it reflects daemon state, not just file writes).
    if (/status:\s*open/i.test(list)) break;
    if (Date.now() >= deadline) {
      process.stderr.write(
        `pw: timed out after ${timeoutSec}s waiting for session (last list: ${JSON.stringify(list.trim().slice(0, 200))}). ` +
          `Run 'pw kill-all' to reap, or use the chrome-devtools MCP instead.\n`,
      );
      try {
        cliSync(['kill-all']);
      } catch {}
      process.exitCode = 124;
      return;
    }
  }

  // Fetch the fresh snapshot for the caller (open writes one per command).
  process.stdout.write(lastList);
}

main().catch((err) => {
  process.stderr.write(`pw: ${err && err.message ? err.message : err}\n`);
  process.exitCode = 1;
});
