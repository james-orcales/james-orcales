import api from '../../src/api.json'

const byId = new Map<number, any>()
;(function index(n: any) {
  if (typeof n?.id === 'number') byId.set(n.id, n)
  ;(n?.children ?? []).forEach(index)
})(api as any)

function renderType(type: any): string {
  if (!type) return '—'
  switch (type.type) {
    case 'intrinsic':
      return type.name
    case 'literal':
      return typeof type.value === 'string' ? `'${type.value}'` : String(type.value)
    case 'union':
      return type.types
        .map(renderType)
        .filter((t: string) => t !== '—')
        .join(' | ')
    case 'intersection': {
      const members = type.types as any[]
      // `string & {}` — open-string idiom — render explicitly so LLMs know custom values are allowed
      const isOpenString =
        members.length === 2 &&
        members.some((m: any) => m.type === 'intrinsic' && m.name === 'string') &&
        members.some((m: any) => m.type === 'reflection')
      if (isOpenString) return '(string & {})'
      const parts = members.map(renderType).filter((p: string) => p !== '—')
      return parts.join(' & ') || '—'
    }
    case 'reference':
      return type.name || '—'
    case 'array':
      return `${renderType(type.elementType)}[]`
    case 'reflection': {
      const sig = type.declaration?.signatures?.[0]
      if (sig) {
        const params = (sig.parameters ?? []).map((p: any) => p.name).join(', ')
        return `(${params}) => ${renderType(sig.type)}`
      }
      return '—'
    }
    case 'typeOperator':
      if (type.operator === 'keyof' && type.target?.name === 'IconShapes') return 'IconShape'
      return `${type.operator} ${renderType(type.target)}`
    case 'unknown':
      if (typeof type.name === 'string' && /IconShapes?\b/.test(type.name)) return 'IconShape'
      return type.name || '—'
    default:
      return type.name || '—'
  }
}

function readDefault(comment: any): string {
  const tags = comment?.blockTags
  if (!Array.isArray(tags)) return ''
  for (const tag of tags) {
    if (tag.tag !== '@defaultValue' && tag.tag !== '@default') continue
    const raw = (tag.content ?? []).map((p: any) => p.text ?? '').join('').trim()
    if (!raw) continue
    return raw
      .replace(/^```[a-zA-Z]*\s*\n?/, '')
      .replace(/\n?```$/, '')
      .replace(/^`|`$/g, '')
      .trim()
  }
  return ''
}

function renderDescription(comment: any): string {
  if (!comment?.summary) return ''
  return comment.summary.map((part: any) => part.text).join('')
}

const EXCLUDED = new Set(['key', 'ref'])

function isNever(t: any) {
  return t?.type === 'intrinsic' && t.name === 'never'
}

function hasDoc(comment: any) {
  return !!(comment?.summary?.length || comment?.blockTags?.length)
}

function mergeTypes(a: string, b: string): string {
  const parts = [...a.split(' | '), ...b.split(' | ')].map((s) => s.trim()).filter(Boolean)
  return [...new Set(parts)].join(' | ')
}

function collect(decl: any) {
  const out = new Map<string, any>()
  const seen = new Set<any>()

  function addProp(c: any, fromBase: boolean, fromUnion: boolean) {
    if (!c?.name || EXCLUDED.has(c.name)) return
    if (c.kind !== 1024) return
    if (isNever(c.type)) return
    const base = fromBase || !!c.inheritedFrom
    const optional = !!c.flags?.isOptional || fromUnion
    const type = renderType(c.type)
    const existing = out.get(c.name)
    if (!existing) {
      out.set(c.name, { name: c.name, type, optional, comment: c.comment, fromBase: base })
      return
    }
    existing.type = mergeTypes(existing.type, type)
    existing.optional = existing.optional || optional
    if (!hasDoc(existing.comment) && hasDoc(c.comment)) existing.comment = c.comment
    existing.fromBase = existing.fromBase && base
  }

  function visitDecl(node: any, fromBase: boolean, fromUnion: boolean) {
    if (!node || seen.has(node)) return
    seen.add(node)
    if (Array.isArray(node.children))
      node.children.forEach((c: any) => addProp(c, fromBase, fromUnion))
    if (node.type) visitType(node.type, fromBase, fromUnion)
  }

  function visitType(t: any, fromBase: boolean, fromUnion: boolean) {
    if (!t) return
    if (t.type === 'intersection') {
      t.types.forEach((m: any) => visitType(m, fromBase, fromUnion))
      return
    }
    if (t.type === 'union') {
      t.types.forEach((m: any) => visitType(m, fromBase, true))
      return
    }
    if (t.type === 'reference') {
      const target = typeof t.target === 'number' ? byId.get(t.target) : null
      visitDecl(target, fromBase || t.name === 'BaseProps', fromUnion)
      return
    }
    if (t.type === 'reflection') {
      ;(t.declaration?.children ?? []).forEach((c: any) => addProp(c, fromBase, fromUnion))
    }
  }

  visitDecl(decl, false, false)
  return [...out.values()]
}

function escMd(s: string): string {
  return s.replace(/\|/g, '\\|')
}

export function renderPropsTable(componentName: string): string {
  const root = (api as any).children?.find((c: any) => c.name === `${componentName}Props`)
  if (!root) return ''

  const all = collect(root)
  const props = all
    .filter((p) => !p.fromBase)
    .sort((a, b) => {
      const diff = (a.optional ? 1 : 0) - (b.optional ? 1 : 0)
      return diff !== 0 ? diff : a.name.localeCompare(b.name)
    })

  if (!props.length) return ''

  const rows = props.map((p: any) => {
    const name = p.optional ? `${p.name}?` : p.name
    const type = escMd(p.type === '—' ? '—' : p.type)
    const def = escMd(readDefault(p.comment) || '—')
    const desc = escMd(renderDescription(p.comment))
    return `| \`${name}\` | ${type} | ${def} | ${desc} |`
  })

  return ['| Prop | Type | Default | Description |', '|------|------|---------|-------------|', ...rows].join(
    '\n'
  )
}
