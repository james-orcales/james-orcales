/**
 * rehype-table-wrap.mjs — wraps every <table> the rehype tree finds inside
 * a <div class="table-wrap">. Combined with the `.table-wrap` CSS rule in
 * site/src/styles/base.css, that lets a wide table scroll inside its own
 * container instead of overflowing the page (which would force the entire
 * sidebar+content layout to scroll horizontally).
 *
 * Idempotent: skips tables that are already inside a div.table-wrap.
 */
export default function rehypeTableWrap() {
  return (tree) => walk(tree)
}

function walk(node) {
  if (!node || !Array.isArray(node.children)) return
  for (let i = 0; i < node.children.length; i++) {
    const child = node.children[i]
    if (child?.type === 'element' && child.tagName === 'table') {
      // Skip if already wrapped (parent is a div.table-wrap).
      if (
        node.type === 'element' &&
        node.tagName === 'div' &&
        Array.isArray(node.properties?.className) &&
        node.properties.className.includes('table-wrap')
      ) continue
      node.children[i] = {
        type: 'element',
        tagName: 'div',
        properties: { className: ['table-wrap'] },
        children: [child],
      }
      // Don't descend into the new wrapper — the table's children aren't
      // candidates for further wrapping.
      continue
    }
    walk(child)
  }
}
