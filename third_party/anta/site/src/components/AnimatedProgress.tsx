import { useState, useEffect } from 'preact/hooks'
import { Progress } from '@antadesign/anta'

/**
 * Small live demo for the Progress docs page: a `<Progress>` whose value
 * climbs on a timer and loops. Hydrated as an island (`client:visible`)
 * so the animation runs in the browser.
 */
export default function AnimatedProgress({ speed = 1, tone }: { speed?: number; tone?: string }) {
  const [v, setV] = useState(0)
  useEffect(() => {
    const id = setInterval(() => setV((x) => (x + speed) % 101), 50)
    return () => clearInterval(id)
  }, [speed])
  return <Progress value={v} tone={tone} label="Animated" />
}
