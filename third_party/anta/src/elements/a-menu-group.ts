import { HTMLElementBase } from '../anta_helpers'
import './a-menu-group.css'

/**
 * `<a-menu-group>` — a titled section grouping related menu items.
 * No JS / no shadow DOM; styled by `a-menu-group.css`. The group's heading
 * is a non-focusable `<a-menu-group-label>` child rendered by the
 * `MenuGroup` JSX wrapper; `role="group"` + `aria-label` are added there too.
 * Keyboard navigation in `<a-menu>` flattens items across group boundaries,
 * skipping the heading.
 */
export class AMenuGroupElement extends HTMLElementBase {}

export function register_a_menu_group() {
  if (typeof customElements === 'undefined') return
  if (!customElements.get('a-menu-group')) {
    customElements.define('a-menu-group', AMenuGroupElement)
  }
}

register_a_menu_group()
