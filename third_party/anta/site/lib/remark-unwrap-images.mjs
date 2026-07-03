/**
 * remark-unwrap-images.mjs — drop the `<p>` that markdown puts around
 * an image-only paragraph.
 *
 * `![alt](src)` on its own line parses to `paragraph > image`, which
 * forces the image into block flow inside a `<p>` even though the
 * author clearly meant a standalone image. This plugin walks the
 * mdast tree, finds paragraphs whose only meaningful child is an
 * image (ignoring whitespace-only text siblings), and replaces the
 * paragraph with the image itself.
 *
 * Equivalent in intent to the npm `remark-unwrap-images` package,
 * kept local so we don't add a dependency for a 20-line transform.
 */
export default function remarkUnwrapImages() {
  return (tree) => walk(tree)
}

function walk(node) {
  if (!node || !Array.isArray(node.children)) return
  for (let i = 0; i < node.children.length; i++) {
    const child = node.children[i]
    if (child?.type === 'paragraph' && hasOnlyImage(child.children)) {
      const image = child.children.find((c) => c.type === 'image')
      node.children[i] = image
      continue
    }
    walk(child)
  }
}

function hasOnlyImage(children) {
  let sawImage = false
  for (const c of children) {
    if (c.type === 'image') {
      if (sawImage) return false
      sawImage = true
    } else if (c.type === 'text' && /^\s*$/.test(c.value)) {
      // whitespace-only text — fine
    } else {
      return false
    }
  }
  return sawImage
}
