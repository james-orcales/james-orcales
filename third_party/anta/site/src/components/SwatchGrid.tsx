import { useEffect, useRef, useState } from 'preact/hooks'
import s from './Swatches.module.css'

type Tone = 'neutral' | 'brand' | 'info' | 'success' | 'critical' | 'warning'
type Kind = 'bg' | 'text' | 'border'

// Normalize a computed `rgb()` / `rgba()` color string to hex (8-digit when
// the token carries alpha), so the live label reads like the hex authored in
// tokens.css.
function toHex(color: string): string {
  const n = color.match(/[\d.]+/g)?.map(Number)
  if (!n || n.length < 3) return color
  const h = (x: number) => Math.round(x).toString(16).padStart(2, '0')
  let out = `#${h(n[0])}${h(n[1])}${h(n[2])}`
  if (n.length >= 4 && n[3] < 1) out += h(n[3] * 255)
  return out
}

// Token NAMES only — values are never hardcoded. Backgrounds use a numeric
// elevation scale `bg-1 … bg-5` (1 = deepest / recessed, 5 = most raised);
// `bg-1` is neutral-only, so tinted bg rows lead with the shared `bg-1`, then
// `bg-2-${tone}` … `bg-5-${tone}`. Text / border follow `prefix-1…5(-tone)`.
const sfx = (tone: Tone) => (tone === 'neutral' ? '' : `-${tone}`)
function tokenNames(kind: Kind, tone: Tone): string[] {
  if (kind === 'bg') {
    return tone === 'neutral'
      ? ['bg-1', 'bg-2', 'bg-3', 'bg-4', 'bg-5']
      : ['bg-1', `bg-2-${tone}`, `bg-3-${tone}`, `bg-4-${tone}`, `bg-5-${tone}`]
  }
  return [1, 2, 3, 4, 5].map((n) => `${kind}-${n}${sfx(tone)}`)
}

function Swatch({ kind, token }: { kind: Kind; token: string }) {
  // Read the live computed color off the rendered preview (which resolves
  // `var(--token)` within its themed `.light`/`.dark` row) and show it as hex
  // — so the label always tracks tokens.css, no hardcoded values.
  const previewRef = useRef<HTMLDivElement>(null)
  const [value, setValue] = useState('')
  useEffect(() => {
    const el = previewRef.current
    if (!el) return
    const cs = getComputedStyle(el)
    setValue(toHex(kind === 'bg' ? cs.backgroundColor : cs.color))
  }, [kind, token])
  return (
    <div class={s.swatch}>
      {kind === 'bg' && (
        <div ref={previewRef} class={s.bgPreview} style={{ background: `var(--${token})` }} />
      )}
      {kind === 'text' && (
        <div ref={previewRef} class={s.textPreview} style={{ color: `var(--${token})` }}>Aa</div>
      )}
      {kind === 'border' && (
        <div ref={previewRef} class={s.borderPreview} style={{ color: `var(--${token})` }}>
          <div class={s.borderCorner} />
        </div>
      )}
      <span class={`${s.tokenName} copyable`}>{token}</span>
      <span class={s.hex}>{value}</span>
    </div>
  )
}

// One swatch grid — a single "theme example". The themed `.light`/`.dark`
// ancestor (rendered by the static Swatches.astro shell) supplies the per-mode
// token values; each Swatch reads them live for its hex label.
export default function SwatchGrid({ kind, tone }: { kind: Kind; tone: Tone }) {
  return (
    <div class={s.swatchGrid}>
      {tokenNames(kind, tone).map((name) => (
        <Swatch key={name} kind={kind} token={name} />
      ))}
    </div>
  )
}
