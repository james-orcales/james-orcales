/**
 * generate-social-images.mjs — rasterize the brand SVGs into the PNG
 * assets that browsers / iOS / link-preview crawlers expect:
 *
 *   site/public/apple-touch-icon.png  (180×180 — iOS home-screen icon,
 *                                      from apple-touch-icon.svg —
 *                                      square brand-bg + mark; iOS
 *                                      applies its own rounded-square
 *                                      mask on the home screen)
 *   site/public/og-image.png          (1920×1080 — Open Graph + Twitter
 *                                      card, from og-image.svg)
 *
 * Run after editing either source SVG (or as a one-off when these assets
 * are missing):
 *   node site/scripts/generate-social-images.mjs
 */
import { readFileSync } from 'node:fs'
import sharp from 'sharp'

const APPLE_SVG_PATH = new URL('../public/apple-touch-icon.svg', import.meta.url)
const OG_SVG_PATH = new URL('../public/og-image.svg', import.meta.url)
const APPLE_TOUCH = new URL('../public/apple-touch-icon.png', import.meta.url)
const OG_IMAGE = new URL('../public/og-image.png', import.meta.url)

// 1. Apple touch icon — 180×180 raster of the square logo. iOS applies
//    its own rounded-square mask, so the SVG is square (no rx) and we
//    just hand the square pixels off.
await sharp(readFileSync(APPLE_SVG_PATH))
  .resize(180, 180)
  .png()
  .toFile(APPLE_TOUCH.pathname)

// 2. Open Graph image — render the designed og-image.svg at its
//    native 1920×1080 dimensions. The SVG already includes the brand
//    field, the centred mark, and the "Antadesign" wordmark; nothing
//    to composite. `density: 384` (4× the default 96 DPI) gives the
//    SVG renderer enough headroom to keep the gaussian-blur drop
//    shadows on the centred mark crisp at the output resolution.
await sharp(readFileSync(OG_SVG_PATH), { density: 384 })
  .resize(1920, 1080)
  .png()
  .toFile(OG_IMAGE.pathname)

console.log(`wrote ${APPLE_TOUCH.pathname} (180×180)`)
console.log(`wrote ${OG_IMAGE.pathname} (1920×1080)`)
