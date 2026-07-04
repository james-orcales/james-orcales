/**
 * copy-esbuild-wasm.mjs — copy node_modules/esbuild-wasm/esbuild.wasm into
 * site/public/ so the Playground's bundler can fetch it at runtime
 * via `/esbuild.wasm`. Runs as part of the `docs` pre-step.
 */
import { copyFileSync, existsSync, mkdirSync } from 'node:fs'
import { dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const src = fileURLToPath(new URL('../node_modules/esbuild-wasm/esbuild.wasm', import.meta.url))
const dest = fileURLToPath(new URL('../public/esbuild.wasm', import.meta.url))

if (!existsSync(src)) {
  console.error(`esbuild-wasm not found at ${src} — run pnpm install`)
  process.exit(1)
}

mkdirSync(dirname(dest), { recursive: true })
copyFileSync(src, dest)
console.log(`copied esbuild.wasm → ${dest}`)
