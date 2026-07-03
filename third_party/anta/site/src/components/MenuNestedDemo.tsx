import { useState, useEffect } from 'preact/hooks'
import { Button, Checkbox, Menu, MenuItem, MenuSeparator } from '@antadesign/anta'

const COLUMNS = ['Name', 'Status', 'Owner', 'Created', 'Modified']

/**
 * Interactive demo for the Menu docs "Submenus" section: a menu nested three
 * submenus deep (View → Columns → Visible columns) whose deepest flyout is a
 * list of Anta `Checkbox`es with a tri-state "Select all" toggle and a per-row
 * "Only" tertiary button. Hydrated as an island so the checkbox state is live.
 *
 * Every interactive row carries `data-menu-open`, so ticking a box (or hitting
 * "Only") never dismisses the menu — it just updates state. Closing stays on the
 * usual paths (outside-click, Esc, ←, picking a real MenuItem).
 */
export default function MenuNestedDemo() {
  useEffect(() => {
    import('@antadesign/anta/elements')
  }, [])

  const [on, setOn] = useState<Record<string, boolean>>(() =>
    Object.fromEntries(COLUMNS.map((c) => [c, true])),
  )
  const allOn = COLUMNS.every((c) => on[c])
  const someOn = COLUMNS.some((c) => on[c])
  // Tri-state value for the "Select all" Checkbox — Anta takes 'indeterminate'
  // directly, so no imperative ref poking like a native <input> needs.
  const allValue = allOn ? true : someOn ? 'indeterminate' : false

  // Functional updates throughout so a handler never reads stale render state.
  const toggle = (c: string) => setOn((s) => ({ ...s, [c]: !s[c] }))
  const only = (c: string) => setOn(() => Object.fromEntries(COLUMNS.map((k) => [k, k === c])))
  const toggleAll = () =>
    setOn((s) => {
      const next = !COLUMNS.every((c) => s[c])
      return Object.fromEntries(COLUMNS.map((c) => [c, next]))
    })

  return (
    <>
      <Button>More</Button>
      <Menu>
        <MenuItem icon="edit" label="Rename" />
        <MenuItem label="Move to" submenu>
          <Menu>
            <MenuItem icon="folder-open" label="Projects" />
            <MenuItem icon="folder-open" label="Archive" />
          </Menu>
        </MenuItem>
        <MenuItem label="View" submenu>
          <Menu>
            <MenuItem label="Columns" submenu>
              <Menu>
                <MenuItem label="Visible columns" submenu>
                  <Menu>
                    <div class="menu-check-row" data-menu-open>
                      <Checkbox
                        className="menu-check"
                        label="Select all"
                        checked={allValue}
                        onStateChange={toggleAll}
                      />
                    </div>
                    <MenuSeparator />
                    {COLUMNS.map((c) => (
                      <div class="menu-check-row" data-menu-open key={c}>
                        <Checkbox
                          className="menu-check"
                          label={c}
                          checked={on[c]}
                          onStateChange={() => toggle(c)}
                        />
                        <Button priority="tertiary" size="small" onClick={() => only(c)}>
                          Only
                        </Button>
                      </div>
                    ))}
                  </Menu>
                </MenuItem>
              </Menu>
            </MenuItem>
          </Menu>
        </MenuItem>
      </Menu>
    </>
  )
}
