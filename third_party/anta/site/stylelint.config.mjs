/** @type {import('stylelint').Config} */
export default {
  extends: ['stylelint-config-standard'],
  // Lint the `<style>` blocks inside .astro components (where the data-*/aria-*
  // comment bug lived), not just standalone .css. postcss-html parses the
  // embedded styles out of the component.
  overrides: [{ files: ['**/*.astro'], customSyntax: 'postcss-html' }],
  ignoreFiles: ['dist/**', 'public/**', 'src/generated/**', 'src/api.json'],
  rules: {
    // The point of this config is to catch broken / dropped CSS (parse errors,
    // unknown selectors from leaked comment text, empty rules) — not to enforce
    // a house style on the existing hand-written CSS. So the correctness rules
    // below stay on (stylelint-config-standard defaults) and the purely
    // stylistic ones that would flag swathes of current CSS are turned off.

    // --- stylistic, off (would require reformatting the whole site) ---
    'custom-property-pattern': null,
    'keyframes-name-pattern': null,
    'selector-class-pattern': null,
    'color-function-notation': null,
    'alpha-value-notation': null,
    'hue-degree-notation': null,
    'lightness-notation': null,
    'color-hex-length': null,
    'shorthand-property-no-redundant-values': null,
    'declaration-block-no-redundant-longhand-properties': null,
    'value-keyword-case': null,
    'number-max-precision': null,
    'comment-empty-line-before': null,
    'rule-empty-line-before': null,
    'declaration-empty-line-before': null,
    'custom-property-empty-line-before': null,
    'at-rule-empty-line-before': null,
    'no-descending-specificity': null,
    'media-feature-range-notation': null,
    'selector-not-notation': null,
    'declaration-block-single-line-max-declarations': null,
    'no-duplicate-selectors': null,
    // The site intentionally ships -webkit- prefixes (mask, user-select,
    // backdrop-filter) for Safari; these are deliberate, not mistakes.
    'property-no-vendor-prefix': null,
    // Modern @font-face descriptors stylelint doesn't know yet
    // (font-optical-sizing, text-rendering).
    'at-rule-descriptor-no-unknown': null,
    // postcss-html parses .astro inline `style="--x: y"` attributes as CSS,
    // which trips this rule with false positives; the bug class we care about
    // lives in <style> blocks, not inline attributes.
    'no-invalid-position-declaration': null,

    // --- allow modern CSS the site relies on ---
    // Newer pseudo-elements (e.g. ::details-content) shouldn't be flagged.
    'selector-pseudo-element-no-unknown': [true, { ignorePseudoElements: ['details-content'] }],
    'property-no-unknown': [true, { ignoreProperties: ['content-visibility'] }],
    // color-mix(in oklch, …), oklch(from …) relative-color syntax, etc.
    'function-no-unknown': null,
    // Astro/CSS-modules `:global()` / `:local()` aren't standard pseudo-classes;
    // the default rule throws on them, so turn it off (our target bug class is
    // caught by selector-type-no-unknown + parse errors regardless).
    'selector-pseudo-class-no-unknown': null,
  },
}
