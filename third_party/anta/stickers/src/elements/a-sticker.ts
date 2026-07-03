import { HTMLElementBase } from '@antadesign/anta/anta_helpers'
import './a-sticker.css'

/**
 * `<a-sticker>` — static sticker carrier.
 *
 * Receives SVG markup as the `svg` attribute. On change, drops it into
 * a shadow-DOM container so the host's light DOM stays untouched.
 * Sizing comes from external CSS reading `--sticker-size` (set by the
 * JSX wrapper, or by the consumer directly).
 *
 * The animated counterpart is `<a-sticker-animated>`.
 */
export class AStickerElement extends HTMLElementBase {
  static observedAttributes = ['svg']
  container: HTMLDivElement

  constructor() {
    super()
    const shadow = this.attachShadow({ mode: 'open' })

    const style = document.createElement('style')
    style.textContent = `
      :host { display: inline-block; }
      div { width: 100%; height: 100%; }
      div > svg { display: block; width: 100%; height: 100%; }
    `

    this.container = document.createElement('div')

    shadow.append(style, this.container)
  }

  attributeChangedCallback() {
    this.container.innerHTML = this.getAttribute('svg') ?? ''
  }
}

export function register_a_sticker() {
  if (typeof customElements === 'undefined') return
  if (!customElements.get('a-sticker')) {
    customElements.define('a-sticker', AStickerElement)
  }
}

// Importing this module registers the element (granular entry point). The
// barrel re-exports it, so importing the barrel registers it too. Idempotent.
register_a_sticker()
