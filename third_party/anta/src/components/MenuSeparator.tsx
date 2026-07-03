import type { BaseProps } from '../general_types'

export interface MenuSeparatorProps extends BaseProps {}

/**
 * MenuSeparator — a thin divider between groups of `MenuItem`s.
 *
 * @example
 * ```tsx
 * <MenuItem label="Edit" />
 * <MenuSeparator />
 * <MenuItem tone="critical" label="Delete" />
 * ```
 */
export const MenuSeparator = ({ className, ...rest }: MenuSeparatorProps) => {
  return <a-menu-separator role="separator" class={className} {...rest} />
}
