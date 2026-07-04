import type { BaseProps } from '../general_types'

export interface MenuGroupProps extends BaseProps {
  /** The section heading shown above the grouped items (also the group's
   *  accessible name). */
  label?: string
  /** The grouped `MenuItem`s. */
  children?: React.ReactNode
}

/**
 * MenuGroup — a titled section that organises related `MenuItem`s. Keyboard
 * navigation flattens items across groups, skipping the heading.
 *
 * Add `data-menu-open` to keep the menu open after any item in the group is
 * chosen (it forwards to the element).
 *
 * @example
 * ```tsx
 * <MenuGroup label="Edit">
 *   <MenuItem icon="copy" label="Copy" />
 *   <MenuItem icon="cut" label="Cut" />
 * </MenuGroup>
 * ```
 */
export const MenuGroup = ({ label, className, children, ...rest }: MenuGroupProps) => {
  return (
    <a-menu-group
      role="group"
      aria-label={label}
      class={className}
      {...rest}
    >
      {label != null && <a-menu-group-label aria-hidden="true">{label}</a-menu-group-label>}
      {children}
    </a-menu-group>
  )
}
