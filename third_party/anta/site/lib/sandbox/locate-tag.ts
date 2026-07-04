/**
 * locate-tag.ts — find the opening JSX tag of a named element inside
 * a source string. No AST library; uses a small bracket-balanced scan.
 *
 * The Playground's form-↔-code sync only needs to know:
 *   - where the target element's opening tag starts
 *   - where the opening tag ends (`>` or `/>`)
 *   - what the inside of the tag looks like (so we can find / replace
 *     individual `name=…` attributes)
 *
 * That's solvable without a parser as long as we respect strings,
 * template literals, and JSX expression containers (`{…}`). The scan
 * below tracks those.
 */
export interface OpenTagRange {
  /** Index of the `<` character of the opening tag in the source. */
  start: number
  /** Index just past the closing `>` of the opening tag. */
  end: number
  /** Whether the opening tag is self-closing (`<Foo … />`). */
  selfClosing: boolean
  /** Substring of the opening tag, `<Foo … />` or `<Foo …>`. */
  text: string
  /** Inclusive index of the position *after* `<Foo` where attributes begin. */
  attrsStart: number
  /** Index of the `/>` or `>` that closes the opening tag (start of the marker). */
  attrsEnd: number
}

/**
 * Find the first opening tag matching `<componentName` in `source`.
 * Returns `null` if none found (or if scanning fails). The scan
 * respects string/template/expression-container nesting so a `>`
 * inside `value={foo > 1 ? 'a' : 'b'}` doesn't end the tag early.
 */
export function locateOpeningTag(source: string, componentName: string): OpenTagRange | null {
  const opener = `<${componentName}`
  let searchFrom = 0
  while (true) {
    const start = source.indexOf(opener, searchFrom)
    if (start === -1) return null
    // Reject substring matches like `<ProgressBar` when component is `Progress`.
    const afterName = source[start + opener.length]
    if (afterName && /[A-Za-z0-9_$.]/.test(afterName)) {
      searchFrom = start + opener.length
      continue
    }
    const attrsStart = start + opener.length
    const closing = findOpenTagClose(source, attrsStart)
    if (closing == null) return null
    const { end, selfClosing, markerStart } = closing
    return {
      start,
      end,
      selfClosing,
      text: source.slice(start, end),
      attrsStart,
      attrsEnd: markerStart,
    }
  }
}

/**
 * From the position just past `<Foo`, scan forward and find the `>`
 * or `/>` that closes the opening tag. Respects string, template,
 * and `{…}` nesting. Returns `null` if the source ends without a
 * closer (malformed JSX during typing — caller will leave code alone).
 */
export function findOpenTagClose(
  source: string,
  from: number,
): { end: number; selfClosing: boolean; markerStart: number } | null {
  let i = from
  while (i < source.length) {
    const c = source[i]
    if (c === '{') {
      const j = skipBraced(source, i)
      if (j == null) return null
      i = j
      continue
    }
    if (c === '"' || c === "'") {
      const j = skipQuoted(source, i, c)
      if (j == null) return null
      i = j
      continue
    }
    if (c === '`') {
      const j = skipTemplate(source, i)
      if (j == null) return null
      i = j
      continue
    }
    if (c === '/' && source[i + 1] === '>') {
      return { end: i + 2, selfClosing: true, markerStart: i }
    }
    if (c === '>') {
      return { end: i + 1, selfClosing: false, markerStart: i }
    }
    i++
  }
  return null
}

/** Skip a `{…}` block starting at `i` (which must be `{`). Handles nested braces, strings, templates. */
export function skipBraced(source: string, start: number): number | null {
  let i = start + 1
  let depth = 1
  while (i < source.length && depth > 0) {
    const c = source[i]
    if (c === '{') {
      depth++
      i++
    } else if (c === '}') {
      depth--
      i++
    } else if (c === '"' || c === "'") {
      const j = skipQuoted(source, i, c)
      if (j == null) return null
      i = j
    } else if (c === '`') {
      const j = skipTemplate(source, i)
      if (j == null) return null
      i = j
    } else {
      i++
    }
  }
  return depth === 0 ? i : null
}

function skipQuoted(source: string, start: number, quote: string): number | null {
  let i = start + 1
  while (i < source.length) {
    const c = source[i]
    if (c === '\\') {
      i += 2
      continue
    }
    if (c === quote) return i + 1
    i++
  }
  return null
}

function skipTemplate(source: string, start: number): number | null {
  let i = start + 1
  while (i < source.length) {
    const c = source[i]
    if (c === '\\') {
      i += 2
      continue
    }
    if (c === '`') return i + 1
    if (c === '$' && source[i + 1] === '{') {
      const j = skipBraced(source, i + 1)
      if (j == null) return null
      i = j
      continue
    }
    i++
  }
  return null
}

/**
 * Find an attribute by name inside the opening-tag range. Returns the
 * range of the *whole* attribute (e.g. `value={42}` or `tone="info"`)
 * so callers can replace or remove it.
 */
export interface AttributeRange {
  /** Index of the first character of the attribute name. */
  start: number
  /** Index just past the end of the attribute value (or just past the name for boolean attrs). */
  end: number
  /** Original full text of the attribute, e.g. `value={42}`. */
  text: string
  /** Range of the attribute *value* (after `=`), or null for boolean attrs. */
  valueRange: { start: number; end: number; text: string; kind: 'string' | 'expression' } | null
}

export function findAttribute(
  source: string,
  open: OpenTagRange,
  name: string,
): AttributeRange | null {
  let i = open.attrsStart
  while (i < open.attrsEnd) {
    // Skip whitespace.
    while (i < open.attrsEnd && /\s/.test(source[i])) i++
    if (i >= open.attrsEnd) return null
    // Attribute name (or spread / other unsupported form — we skip it).
    if (source[i] === '{') {
      const j = skipBraced(source, i)
      if (j == null) return null
      i = j
      continue
    }
    const nameStart = i
    while (i < open.attrsEnd && /[A-Za-z0-9_$\-]/.test(source[i])) i++
    const attrName = source.slice(nameStart, i)
    // Boolean shorthand (no `=`): the attribute is just the name.
    if (i >= open.attrsEnd || source[i] !== '=') {
      if (attrName === name) {
        return { start: nameStart, end: i, text: source.slice(nameStart, i), valueRange: null }
      }
      continue
    }
    // Has `=`. Skip the `=`.
    i++
    // Value: `"…"`, `'…'`, or `{…}`.
    if (source[i] === '"' || source[i] === "'") {
      const valStart = i
      const valEnd = skipQuoted(source, i, source[i])
      if (valEnd == null) return null
      i = valEnd
      if (attrName === name) {
        return {
          start: nameStart,
          end: i,
          text: source.slice(nameStart, i),
          valueRange: {
            start: valStart,
            end: i,
            text: source.slice(valStart, i),
            kind: 'string',
          },
        }
      }
      continue
    }
    if (source[i] === '{') {
      const valStart = i
      const valEnd = skipBraced(source, i)
      if (valEnd == null) return null
      i = valEnd
      if (attrName === name) {
        return {
          start: nameStart,
          end: i,
          text: source.slice(nameStart, i),
          valueRange: {
            start: valStart,
            end: i,
            text: source.slice(valStart, i),
            kind: 'expression',
          },
        }
      }
      continue
    }
    // Unrecognised form (e.g. `name=value` without quotes — invalid JSX);
    // bail rather than guess.
    return null
  }
  return null
}

/**
 * Enumerate the attribute names present on the opening tag in source
 * order. Skips spread attributes (`{...rest}`), unquoted forms we
 * don't understand, and obviously malformed cases.
 */
export function listAttributes(source: string, open: OpenTagRange): string[] {
  const names: string[] = []
  let i = open.attrsStart
  while (i < open.attrsEnd) {
    const c = source[i]
    if (c === ' ' || c === '\t' || c === '\n' || c === '\r') { i++; continue }
    if (c === '{') {
      const j = skipBraced(source, i)
      if (j == null) break
      i = j
      continue
    }
    // Identifier characters per JSX attribute names.
    let nameEnd = i
    while (nameEnd < open.attrsEnd && /[A-Za-z0-9_$\-:]/.test(source[nameEnd])) nameEnd++
    if (nameEnd === i) { i++; continue }
    names.push(source.slice(i, nameEnd))
    i = nameEnd
    while (i < open.attrsEnd && (source[i] === ' ' || source[i] === '\t')) i++
    if (source[i] === '=') {
      i++
      while (i < open.attrsEnd && (source[i] === ' ' || source[i] === '\t')) i++
      const v = source[i]
      if (v === '"' || v === "'") {
        const j = skipQuoted(source, i, v)
        if (j == null) break
        i = j
      } else if (v === '{') {
        const j = skipBraced(source, i)
        if (j == null) break
        i = j
      }
    }
  }
  return names
}
