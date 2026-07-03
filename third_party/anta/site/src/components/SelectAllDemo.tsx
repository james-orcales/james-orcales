import { useState } from 'preact/hooks'
import { Checkbox } from '@antadesign/anta'

/**
 * Live indeterminate demo for the Checkbox docs page (mirrors MUI's parent /
 * child example). A parent checkbox whose state is **derived** from two
 * children — `checked` when both are on, `indeterminate` when they differ.
 * Every box is controlled (`checked` from app state, `onStateChange` writes
 * back from `detail.next`), the pattern the props-driven playground can't
 * express. Hydrated as an island.
 */
export default function SelectAllDemo() {
  const [on, setOn] = useState<[boolean, boolean]>([true, false])

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
      <Checkbox
        checked={on[0] && on[1] ? true : on[0] || on[1] ? 'indeterminate' : false}
        onStateChange={(_e, { next }) => setOn([next === true, next === true])}
      >
        Parent
      </Checkbox>
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '12px',
          marginInlineStart: '24px',
        }}
      >
        <Checkbox
          checked={on[0]}
          onStateChange={(_e, { next }) => setOn([next === true, on[1]])}
        >
          Child 1
        </Checkbox>
        <Checkbox
          checked={on[1]}
          onStateChange={(_e, { next }) => setOn([on[0], next === true])}
        >
          Child 2
        </Checkbox>
      </div>
    </div>
  )
}
