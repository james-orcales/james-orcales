import { HTMLElementBase } from '../anta_helpers'
import './a-menu-separator.css'

/**
 * `<a-menu-separator>` — a thin divider between groups of menu items.
 * No JS / no shadow DOM; styled entirely by `a-menu-separator.css`. The
 * trivial class exists only so importing this module registers the tag and
 * pulls its CSS along the granular entry point. `role="separator"` is added
 * by the `MenuSeparator` JSX wrapper. The hairline uses `--border-5` — the
 * most subtle border token, already mode-adaptive, so no `.dark` override.
 */
export class AMenuSeparatorElement extends HTMLElementBase {}

export function register_a_menu_separator() {
  if (typeof customElements === 'undefined') return
  if (!customElements.get('a-menu-separator')) {
    customElements.define('a-menu-separator', AMenuSeparatorElement)
  }
}

register_a_menu_separator()
