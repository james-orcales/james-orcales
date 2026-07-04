## 0.2.0 — June 9, 2026

The first stable minor release, consolidating the `0.1.1-dev.7` and `0.1.1-dev.8` prereleases plus the final pre-stable batch. Highlights below; the full per-version detail is on the [Dev releases](/changelog/dev/) page.

- **New components:** `Button` (`<a-button>`), `Tooltip` (`<a-tooltip>`), and `Tag` (`<a-tag>`) — plus new `rotate-ccw` and `tag` icon shapes.
- **Stickers extracted** into the separate `@antadesign/stickers` package, taking the `lottie-web` dependency with them — apps that don't use stickers no longer install it.
- **Granular element registration:** `import '@antadesign/anta/elements/a-{name}'` registers (and loads CSS for) just that element; the `elements` barrel still registers everything.
- **New runtime dependency:** [`es-toolkit`](https://es-toolkit.dev) (`1.47.0`), tree-shakeable named imports — currently just `debounce`, used by the Tooltip element.
- **Breaking:** background tokens renamed to a numeric elevation scale (`--bg-section`/`--bg-base`/`--bg-pane`/`--bg-block`/`--bg-spot` → `--bg-1 … --bg-5`); `<Button>`'s `iconButton` prop removed and `leadingIcon` renamed to `icon`; `<Text>`'s default `priority` is now `secondary` (was `primary`).
- **Polish:** dark theme palette comprehensively retuned; extensive `<Button>` tuning (sizes, tinted fills, hairline edges, label weight); touch-device hover no longer sticks after a tap.
