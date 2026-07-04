#!/usr/bin/env node
/**
 * generate-stickers.mjs — read a folder of per-sticker subfolders and emit
 * one per-sticker TS module per mode (static + animated) under the chosen
 * output folder, plus a barrel index that re-exports them.
 *
 * Expected source layout:
 *
 *   stickers/
 *     coding/
 *       coding.json   — Lottie animation (drives Sticker{Name}Animated)
 *       coding.svg    — static pose (drives Sticker{Name})
 *     vacation/
 *       vacation.json
 *       vacation.svg
 *     …
 *
 *
 * Collision detection — when `--internal` is NOT passed (i.e. consumer
 * run), the generator warns if a generated name collides with one of
 * the pack's built-in stickers. The pack's own build passes `--internal`.
 *
 * `--package <name>` sets the import specifier the emitted modules pull
 * `Sticker` / `StickerAnimated` from (default `@antadesign/stickers`).
 *
 *   node generate-stickers.mjs --input ./my-stickers --output ./src/my-stickers
 */

import fs from "node:fs";
import path from "node:path";

// The pack's built-in sticker names. Consumers who generate a same-named
// sticker get a warning; their generated module will resolve to the same
// import path and shadow ours. Keep this list in sync with the canonical
// names under ./assets/<name>/.
const DEFAULT_BUILT_IN = [
  "angry",
  "ask",
  "balance",
  "butterfly",
  "butterfly-snake",
  "catch",
  "clap",
  "coding",
  "cowboy",
  "dance",
  "detective",
  "disapprove",
  "distracted",
  "dive",
  "eat",
  "facepalm",
  "failed",
  "found",
  "grow",
  "handstand",
  "heart",
  "hello",
  "idea",
  "laugh",
  "love",
  "nap",
  "party",
  "passed",
  "peekaboo",
  "pizza",
  "puzzle",
  "sad",
  "scared",
  "scroll",
  "search",
  "shield",
  "shocked",
  "sleep",
  "stressed",
  "suspicious",
  "thanks",
  "think",
  "think-of-you",
  "thumbs-up",
  "vacation",
  "wait",
  "wink",
  "work",
  "zen",
];

function pascalCase(name) {
  return name
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");
}

function componentName(base) {
  return `Sticker${pascalCase(base)}`;
}

export async function generate({
  input,
  output,
  internal = false,
  packageName = "@antadesign/stickers",
} = {}) {
  if (!input) throw new Error("generate-stickers: --input is required");
  if (!output) throw new Error("generate-stickers: --output is required");

  const inDir = path.resolve(input);
  const outDir = path.resolve(output);
  if (!fs.existsSync(inDir)) {
    throw new Error(`generate-stickers: input folder not found: ${inDir}`);
  }
  fs.mkdirSync(outDir, { recursive: true });

  // Each immediate subfolder of inDir is one sticker. The folder name is
  // the canonical sticker name.
  const subdirs = fs
    .readdirSync(inDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name)
    .sort();

  if (subdirs.length === 0) {
    console.warn(`generate-stickers: no sticker subfolders in ${inDir}`);
  }

  // Clear out previous generated files so renamed/removed stickers don't
  // leave stale modules behind. Keep index.ts (about to be rewritten) and
  // the per-sticker static + animated file names we're about to write.
  // Naming convention: `<name>.ts` is the static (default) sticker;
  // `<name>-animated.ts` is the Lottie-backed animated variant.
  const keep = new Set([
    "index.ts",
    ...subdirs.flatMap((n) => [`${n}.ts`, `${n}-animated.ts`]),
  ]);
  for (const existing of fs.readdirSync(outDir)) {
    if (!keep.has(existing) && /\.ts$/.test(existing)) {
      fs.unlinkSync(path.join(outDir, existing));
    }
  }

  const exports = [];
  const skipped = [];
  const builtInSet = internal ? new Set() : new Set(DEFAULT_BUILT_IN);
  const collisions = [];

  for (const name of subdirs) {
    const subdir = path.join(inDir, name);
    const lottiePath = path.join(subdir, `${name}.json`);
    const svgPath = path.join(subdir, `${name}.svg`);
    const hasLottie = fs.existsSync(lottiePath);
    const hasSvg = fs.existsSync(svgPath) && fs.statSync(svgPath).size > 0;
    if (!hasLottie && !hasSvg) {
      skipped.push(`${name} (no ${name}.json or ${name}.svg)`);
      continue;
    }
    if (builtInSet.has(name)) collisions.push(name);
    const cname = componentName(name);
    const animatedName = `${cname}Animated`;

    // Static module — the default. Emitted whenever a non-empty SVG
    // exists. Imports the `Sticker` runtime wrapper from the package
    // root `@antadesign/anta` (same root specifier the icon codegen
    // uses — no bespoke runtime subpath). Lives at `<name>.ts` so the
    // import path matches the bare sticker name.
    //
    // Two exports: the JSX `Sticker{Name}` component (React/Preact),
    // and the raw `svg` string (vanilla JS / non-JSX consumers, who
    // assign it to an `<a-sticker>` element's `svg` attribute).
    // Bundlers tree-shake the unused one.
    if (hasSvg) {
      const svg = fs.readFileSync(svgPath, "utf8");
      const lines = [
        `/* Auto-generated by ${packageName}. Do not edit by hand. */`,
        "",
        `import { Sticker, type StickerProps } from '${packageName}'`,
        "",
        `export const svg = ${JSON.stringify(svg)}`,
        "",
        `export const ${cname} = (props: StickerProps) => Sticker({ ...props, svg })`,
        "",
      ];
      fs.writeFileSync(path.join(outDir, `${name}.ts`), lines.join("\n"));
    }

    // Animated module — emitted whenever a Lottie JSON exists. Each
    // module lives in `<name>-animated.ts` so importing only the static
    // sticker (`<name>.ts`) never drags the Lottie payload. Bundlers
    // tree-shake per-file when the barrel uses `export { … } from
    // './foo'`. Imports the `StickerAnimated` runtime wrapper from the
    // package root `@antadesign/anta`.
    //
    // The animation is exported as a JSON *string* (not the parsed
    // object). `<a-sticker-animated>` receives it via its `animation`
    // attribute and parses it once on set. Two exports: the JSX
    // `Sticker{Name}Animated` component, and the raw `animationJson`
    // string (vanilla JS / non-JSX consumers who call
    // `el.setAttribute('animation', animationJson)`).
    if (hasLottie) {
      const lottieJson = fs.readFileSync(lottiePath, "utf8");
      const lines = [
        `/* Auto-generated by ${packageName}. Do not edit by hand. */`,
        "",
        `import { StickerAnimated, type StickerAnimatedProps } from '${packageName}'`,
        "",
        `export const animationJson = ${JSON.stringify(lottieJson)}`,
        "",
        `export const ${animatedName} = (props: StickerAnimatedProps) => StickerAnimated({ ...props, animation: animationJson })`,
        "",
      ];
      fs.writeFileSync(
        path.join(outDir, `${name}-animated.ts`),
        lines.join("\n"),
      );
    }

    exports.push({
      base: name,
      componentName: cname,
      hasLottie,
      hasSvg,
      animatedName,
    });
  }

  // Barrel — alphabetical for stable diffs, side-effect free so unused
  // stickers tree-shake. Each export references the correct per-mode
  // file so importing only the static variant doesn't pull the animated
  // module (and vice versa).
  const reExports = [];
  for (const {
    base,
    componentName,
    hasLottie,
    hasSvg,
    animatedName,
  } of exports) {
    if (hasSvg) reExports.push(`export { ${componentName} } from './${base}'`);
    if (hasLottie)
      reExports.push(`export { ${animatedName} } from './${base}-animated'`);
  }
  const indexLines = [
    `/* Auto-generated by ${packageName}. Do not edit by hand. */`,
    "",
    `export type { StickerProps, StickerAnimatedProps } from '${packageName}'`,
    "",
    ...reExports,
    "",
  ];
  fs.writeFileSync(path.join(outDir, "index.ts"), indexLines.join("\n"));

  if (collisions.length > 0) {
    console.warn(
      `generate-stickers: ${collisions.length} sticker name(s) collide with the built-in stickers: ` +
        `${collisions.join(", ")}. ` +
        `Consumers importing from \`@antadesign/stickers\` get the built-ins; your generated barrel exposes yours under its own import path.`,
    );
  }

  console.log(
    `generate-stickers: wrote ${exports.length} sticker${exports.length === 1 ? "" : "s"} to ${outDir}`,
  );
  if (skipped.length) {
    console.warn(
      `generate-stickers: skipped ${skipped.length} (${skipped.join(", ")})`,
    );
  }
  return { stickers: exports };
}

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg.startsWith("--")) {
      const key = arg.slice(2);
      const next = argv[i + 1];
      if (next && !next.startsWith("--")) {
        out[key] = next;
        i++;
      } else {
        out[key] = true;
      }
    }
  }
  return out;
}

const isMain = process.argv[1]?.endsWith("generate-stickers.mjs");
if (isMain) {
  const args = parseArgs(process.argv.slice(2));
  generate({
    input: args.input,
    output: args.output,
    internal: args.internal === true,
    packageName: typeof args.package === "string" ? args.package : undefined,
  }).catch((e) => {
    console.error(e.message);
    process.exit(1);
  });
}
