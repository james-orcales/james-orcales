/**
 * Transforms raw MDX component page content to clean Markdown.
 *
 * Call-site contract: replace <PropsTable component="X" /> with the rendered
 * markdown table BEFORE calling this function. parseMdx then unwraps the
 * surrounding <Disclosure title="Component props"> into a ### Props section.
 */
export function parseMdx(raw: string): string {
  let s = raw

  // Protect fenced code blocks from all subsequent regex passes by replacing
  // them with NUL-delimited placeholders. They are restored at the end.
  const codeBlocks: string[] = []
  s = s.replace(/```[\s\S]*?```/g, (match) => {
    codeBlocks.push(match)
    return `\x00CODE${codeBlocks.length - 1}\x00`
  })

  // Strip frontmatter
  s = s.replace(/^---\n[\s\S]*?\n---\n/, '')

  // Strip import lines
  s = s.replace(/^import .+\n/gm, '')

  // Strip JSX comment blocks {/* ... */}
  s = s.replace(/\{\/\*[\s\S]*?\*\/\}/g, '')

  // Strip Playground Disclosure blocks entirely (including their Playground child)
  s = s.replace(/<Disclosure\s+title="Playground"[^>]*>[\s\S]*?<\/Disclosure>/g, '')

  // Strip any remaining standalone <Playground .../> or <Playground ...>...</Playground>
  s = s.replace(/<Playground[\s\S]*?(?:\/>|<\/Playground>)/g, '')

  // Component tokens Disclosure → ### Component tokens section
  s = s.replace(
    /<Disclosure\s+title="Component tokens"[^>]*>([\s\S]*?)<\/Disclosure>/g,
    (_, inner) => `### Component tokens\n\n${inner.trim()}`,
  )

  // Component props Disclosure → ### Props section
  // (caller has already replaced <PropsTable .../> with the markdown table)
  s = s.replace(
    /<Disclosure\s+title="Component props"[^>]*>([\s\S]*?)<\/Disclosure>/g,
    (_, inner) => `### Props\n\n${inner.trim()}`,
  )

  // All other Disclosure blocks → unwrap (strip tags, keep inner content)
  s = s.replace(/<Disclosure[^>]*>([\s\S]*?)<\/Disclosure>/g, (_, inner) => inner.trim())

  // Strip Preview blocks entirely — visual renders have no text value;
  // the companion code blocks are siblings in the MDX, not children.
  s = s.replace(/<Preview[^>]*>[\s\S]*?<\/Preview>/g, '')

  // Strip Columns and Col wrapper tags (keep children / text content)
  s = s.replace(/^<\/?(?:Columns|Col)(?:\s[^>]*)?>[ \t]*\n?/gm, '')

  // Strip remaining self-closing PascalCase JSX tags (e.g. <AnimatedProgress ... />)
  s = s.replace(/^<[A-Z][A-Za-z]*(?:\s[^>]*)?\/>[ \t]*\n?/gm, '')

  // Restore code blocks, stripping the `folded` expressive-code directive
  s = s.replace(/\x00CODE(\d+)\x00/g, (_, idx) => {
    let block = codeBlocks[parseInt(idx, 10)]
    block = block.replace(/^(```\w+)\s+folded\b/m, '$1')
    return block
  })

  // Collapse 3+ consecutive blank lines to 2
  s = s.replace(/\n{3,}/g, '\n\n')

  return s.trim()
}
