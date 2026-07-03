// TypeDoc plugin: restore source-declaration order for union types.
//
// The TypeScript checker canonicalizes union members into an arbitrary
// internal order (roughly first-seen-during-compilation), not the order they
// were written — and `--sort source-order` only orders *declarations*, never
// the members *within* a union. So `tone?: 'neutral' | 'brand' | …` lands in
// api.json scrambled, and two interfaces with the same members come out
// identical regardless of how each was written.
//
// This plugin walks every reflection after resolution, and for any whose type
// is a union it reads the original `UnionTypeNode` off the symbol's
// declaration (the AST preserves written order) and reorders the reflection's
// members to match. That makes the type declaration the single source of truth
// for member order — honored by both the props table and the playground.
import { Converter, ReflectionKind } from 'typedoc'
import ts from 'typescript'

// Normalize a member's text form so the reflection side (typedoc `.toString()`)
// and the AST side (`node.getText()`) compare equal: drop whitespace, peel
// wrapping parens, strip surrounding quotes from string literals.
function norm(s) {
  let t = String(s).trim()
  while (t.startsWith('(') && t.endsWith(')')) t = t.slice(1, -1).trim()
  t = t.replace(/\s+/g, '')
  if (t.length >= 2 && (t[0] === "'" || t[0] === '"') && t[t.length - 1] === t[0]) {
    t = t.slice(1, -1)
  }
  return t
}

export function load(app) {
  app.converter.on(Converter.EVENT_RESOLVE_END, (context) => {
    const project = context.project
    for (const refl of project.getReflectionsByKind(ReflectionKind.All)) {
      const type = refl.type
      if (!type || type.type !== 'union' || !Array.isArray(type.types)) continue

      const symbol = context.getSymbolFromReflection(refl)
      const decls = symbol?.getDeclarations?.() ?? []
      let unionNode
      for (const d of decls) {
        let tn = d.type
        while (tn && ts.isParenthesizedTypeNode(tn)) tn = tn.type
        if (tn && ts.isUnionTypeNode(tn)) { unionNode = tn; break }
      }
      if (!unionNode) continue

      const order = unionNode.types.map((n) => norm(n.getText()))
      const rank = (m) => {
        const i = order.indexOf(norm(m.toString()))
        return i < 0 ? order.length : i
      }
      type.types = type.types
        .map((m, i) => [m, i])
        .sort((a, b) => rank(a[0]) - rank(b[0]) || a[1] - b[1])
        .map(([m]) => m)
    }
  })
}
