/**
 * remark-unwrap-jsx-paragraph.mjs — undo MDX's "wrap multi-line JSX
 * children in a paragraph" behavior.
 *
 * When you write a JSX element in MDX with its children spread across
 * multiple lines, e.g.
 *
 *   <Title level={3}>
 *     <Icon shape="info" /> Mix of icon and <span>tinted</span> text
 *   </Title>
 *
 * MDX treats the children as block-level markdown and wraps them in a
 * `<p>`. That makes the resulting HTML invalid inside inline-context
 * parents (a `<p>` inside a heading), and it injects unwanted block
 * layout. This plugin walks the mdast tree, finds JSX elements whose
 * children consist of a single `paragraph` node (the wrapper MDX
 * inserted), and replaces those children with the paragraph's own
 * children.
 *
 * Applies to both flow JSX (block-level) and text JSX (inline) so it
 * works regardless of how the JSX itself was authored.
 */
export default function remarkUnwrapJsxParagraph() {
  return (tree) => walk(tree)
}

function walk(node) {
  if (!node || !Array.isArray(node.children)) return
  if (
    (node.type === 'mdxJsxFlowElement' || node.type === 'mdxJsxTextElement') &&
    node.children.length === 1 &&
    node.children[0]?.type === 'paragraph'
  ) {
    node.children = node.children[0].children
  }
  for (const child of node.children) walk(child)
}
