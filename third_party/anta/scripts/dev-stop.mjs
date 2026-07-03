#!/usr/bin/env node
/**
 * dev-stop.mjs — stop *exactly* the `pnpm run dev` process tree this repo
 * started, and nothing else.
 *
 * `pnpm run dev` records its own shell PID in `.dev.pid` (that shell is the
 * parent of both the `nodemon` rebuild-watcher and the `astro dev` server).
 * Here we read that PID, walk `ps` to collect the whole descendant tree, and
 * terminate it — SIGTERM first, then SIGKILL any stragglers. Unrelated
 * `astro` / `nodemon` processes elsewhere on the machine are left alone (the
 * opposite of a blanket `pkill -f astro`).
 *
 * Safe to run when nothing is up: a missing or stale pidfile is just cleaned
 * up. Targets macOS / Linux (`ps` BSD/POSIX flags), matching the repo's dev
 * platforms.
 */
import { readFileSync, existsSync, unlinkSync } from 'node:fs'
import { execFileSync } from 'node:child_process'

const PIDFILE = '.dev.pid'
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
const alive = (pid) => {
  try { process.kill(pid, 0); return true } catch { return false }
}

if (!existsSync(PIDFILE)) {
  console.log('No .dev.pid found — nothing to stop. (Is `pnpm run dev` running?)')
  process.exit(0)
}

const root = Number.parseInt(readFileSync(PIDFILE, 'utf8').trim(), 10)
if (!Number.isInteger(root)) {
  console.error(`${PIDFILE} did not contain a valid PID — removing it.`)
  unlinkSync(PIDFILE)
  process.exit(1)
}

if (!alive(root)) {
  console.log(`Recorded dev process (pid ${root}) is not running — clearing stale ${PIDFILE}.`)
  unlinkSync(PIDFILE)
  process.exit(0)
}

// Guard against PID reuse: the recorded root is the shell that ran the dev
// script, so its command line contains the script body (nodemon / anta-site).
// If a stale pidfile now points at some unrelated recycled PID, refuse.
const commandOf = (pid) => {
  try {
    return execFileSync('ps', ['-o', 'command=', '-p', String(pid)], { encoding: 'utf8' }).trim()
  } catch { return '' }
}
const rootCmd = commandOf(root)
if (!/nodemon|anta-site|astro|pnpm/.test(rootCmd)) {
  console.log(`pid ${root} doesn't look like the dev server (\`${rootCmd || 'unknown'}\`) — refusing to kill it. Clearing stale ${PIDFILE}.`)
  unlinkSync(PIDFILE)
  process.exit(0)
}

// Snapshot every process and index children by parent, then collect the
// root's subtree in post-order (children before parents) so we never orphan
// a child by killing its parent first.
const childrenByParent = new Map()
for (const line of execFileSync('ps', ['-A', '-o', 'pid=,ppid='], { encoding: 'utf8' }).split('\n')) {
  const m = line.trim().match(/^(\d+)\s+(\d+)$/)
  if (!m) continue
  const pid = Number(m[1]), ppid = Number(m[2])
  if (!childrenByParent.has(ppid)) childrenByParent.set(ppid, [])
  childrenByParent.get(ppid).push(pid)
}

const tree = []
;(function walk(pid) {
  for (const child of childrenByParent.get(pid) ?? []) walk(child)
  tree.push(pid)
})(root)

const signal = (sig) => {
  for (const pid of tree) {
    try { process.kill(pid, sig) } catch { /* already gone */ }
  }
}

console.log(`Stopping dev tree rooted at pid ${root} (${tree.length} process${tree.length === 1 ? '' : 'es'})…`)
signal('SIGTERM')

// Wait up to ~3s for a graceful exit, then SIGKILL whatever's left.
const deadline = Date.now() + 3000
while (Date.now() < deadline && tree.some(alive)) await sleep(100)
const survivors = tree.filter(alive)
if (survivors.length) {
  console.log(`Force-killing ${survivors.length} straggler${survivors.length === 1 ? '' : 's'}…`)
  signal('SIGKILL')
}

if (existsSync(PIDFILE)) unlinkSync(PIDFILE)
console.log('Dev stopped.')
