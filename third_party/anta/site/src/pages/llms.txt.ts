import type { APIRoute } from 'astro'

const SITE = 'https://anta.design'

const componentModules = import.meta.glob('./components/*.mdx', { eager: true }) as Record<
  string,
  { frontmatter: { title?: string } }
>

const componentLinks = Object.entries(componentModules)
  .filter(([path]) => !path.endsWith('.demo.ts'))
  .sort(([a], [b]) => a.localeCompare(b))
  .map(([path, mod]) => {
    const slug = path.replace('./components/', '').replace('.mdx', '')
    const title = mod.frontmatter?.title ?? slug.charAt(0).toUpperCase() + slug.slice(1)
    return `- [${title}](${SITE}/components/${slug}/)`
  })

const otherLinks = [
  `- [Colors](${SITE}/colors/)`,
  `- [Normalization](${SITE}/normalization/)`,
  `- [Changelog](${SITE}/changelog/)`,
]

const body = `# Anta

> Portable React/Preact UI component library. Works out of the box in React,
> in Preact via compat aliasing, and in custom runtimes via \`configure()\`.
> Published as \`@antadesign/anta\` on npm.

## Components

${componentLinks.join('\n')}

## Other pages

${otherLinks.join('\n')}
`

export const GET: APIRoute = () =>
  new Response(body, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } })
