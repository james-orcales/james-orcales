import type { APIRoute } from 'astro'
import { renderPropsTable } from '../../lib/llms/props-from-api.ts'
import { parseMdx } from '../../lib/llms/parse-mdx.ts'

const SITE = 'https://anta.design'

const componentModules = import.meta.glob('./components/*.mdx', {
  eager: true,
}) as Record<string, { frontmatter: { title?: string } }>

const rawMdx = import.meta.glob('./components/*.mdx', {
  eager: true,
  as: 'raw',
}) as Record<string, string>

const demoModules = import.meta.glob('./components/*.demo.ts', {
  eager: true,
}) as Record<string, { default: string }>

function extractComponentName(raw: string): string | null {
  const m = raw.match(/<PropsTable\s+component="([^"]+)"/)
  return m ? m[1] : null
}

function extractDemoCode(raw: string, slug: string): string | null {
  const key = `./components/${slug}.demo.ts`
  const mod = demoModules[key]
  if (!mod?.default) return null
  // Strip `export default \`...\`` wrapper, keep the bare TSX string
  return mod.default.replace(/^\s*export\s+default\s+`/, '').replace(/`\s*$/, '').trim()
}

const header = `# Anta

> Portable React/Preact UI component library. Works out of the box in React,
> in Preact via compat aliasing, and in custom runtimes via \`configure()\`.
> Published as \`@antadesign/anta\` on npm. Source: ${SITE}/llms.txt`

const sections: string[] = []

const sortedPaths = Object.keys(componentModules)
  .filter((p) => !p.endsWith('.demo.ts'))
  .sort()

for (const path of sortedPaths) {
  const slug = path.replace('./components/', '').replace('.mdx', '')
  let raw = rawMdx[path] ?? ''

  // Replace <PropsTable component="X" /> with the rendered markdown table
  // before parseMdx runs, so the surrounding Disclosure can wrap it correctly.
  const compName = extractComponentName(raw)
  if (compName) {
    const table = renderPropsTable(compName)
    raw = raw.replace(/<PropsTable\s+component="[^"]+"\s*\/>/, table)
  }

  let body = parseMdx(raw)

  const demo = extractDemoCode(raw, slug)
  if (demo) {
    body += `\n\n### Example\n\n\`\`\`tsx\n${demo}\n\`\`\``
  }

  sections.push(body)
}

const fullBody = [header, ...sections].join('\n\n---\n\n') + '\n'

export const GET: APIRoute = () =>
  new Response(fullBody, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } })
