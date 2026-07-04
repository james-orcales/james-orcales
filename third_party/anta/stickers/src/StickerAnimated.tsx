import type { StickerA11y, StickerProps } from "./Sticker";

/** Public props for animated stickers — extends `StickerProps` with
 *  `paused`. The static `Sticker` variant doesn't accept `paused`. */
export interface StickerAnimatedProps extends StickerProps {
  /** `true` freezes the animation at the current frame; a number freezes
   *  at that time in seconds; omit (or `false`) to play. */
  paused?: boolean | number;
}

/** Internal — the bound Lottie animation is supplied by the generated
 *  per-sticker module as a JSON string. The element parses it once on
 *  attribute set. */
interface StickerAnimatedInternalProps extends StickerAnimatedProps {
  animation: string;
}

/**
 * Internal renderer for animated stickers. Generated
 * `Sticker{Name}Animated` exports under `@antadesign/stickers`
 * call this with their bound Lottie JSON string. The
 * `<a-sticker-animated>` element receives it as an attribute, parses
 * it once, and drives a shadow-DOM `<svg>` via `lottie-web`.
 */
export const StickerAnimated = ({
  animation,
  size,
  paused,
  label,
  ...rest
}: StickerAnimatedInternalProps) => {
  const a11y: StickerA11y =
    label != null
      ? { role: "img", "aria-label": label }
      : { "aria-hidden": "true" };
  // `paused` semantics:
  //   undefined / false → omit attribute (plays)
  //   true              → empty string (freeze at current frame)
  //   number            → that value as string (seconds; freeze at time)
  const pausedAttr =
    paused === true
      ? ""
      : typeof paused === "number"
        ? String(paused)
        : undefined;
  const style =
    size != null
      ? ({
          ...rest.style,
          ["--sticker-size" as string]: `${size}px`,
        } as React.CSSProperties)
      : rest.style;
  return (
    <a-sticker-animated
      animation={animation}
      paused={pausedAttr}
      {...a11y}
      {...rest}
      style={style}
    />
  );
};
