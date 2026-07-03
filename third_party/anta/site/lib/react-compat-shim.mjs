// React → preact/compat shim with a useEffectEvent polyfill.
//
// `interface-kit` (a React 19 dev tool) imports `useEffectEvent` from "react".
// That hook is a React 19.2+/experimental API that preact/compat 10.29 does not
// export, so a bare `react` → `preact/compat` alias throws
// "useEffectEvent is not a function" at module-eval time.
//
// This module re-exports everything preact/compat provides, then fills the one
// gap. We alias `react` to this file in astro.config.mjs (overriding the alias
// that @astrojs/preact's preset installs) so the whole island graph — anta's
// JSX wrappers and interface-kit alike — resolves React through here.
import * as compat from 'preact/compat';
import { useRef, useLayoutEffect, useCallback } from 'preact/compat';

export * from 'preact/compat';
export { default } from 'preact/compat';

// useEffectEvent: returns a stable function identity that always calls the
// latest closure. Matches the documented React semantics closely enough for
// interface-kit (which uses it for stable event handlers inside effects).
export const useEffectEvent =
  compat.useEffectEvent ||
  function useEffectEvent(fn) {
    const ref = useRef(fn);
    useLayoutEffect(() => {
      ref.current = fn;
    });
    return useCallback((...args) => ref.current?.(...args), []);
  };
