import { useEffect, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

/**
 * Renders children into document.body via a React portal, escaping any
 * ancestor stacking contexts (transforms, overflow, sticky/fixed z-index).
 * This is the standard approach used by Radix UI, Headless UI, MUI, etc.
 */
export function Portal({ children }: { children: ReactNode }) {
  const el = useRef(document.createElement('div'))

  useEffect(() => {
    const container = el.current
    document.body.appendChild(container)
    return () => { document.body.removeChild(container) }
  }, [])

  return createPortal(children, el.current)
}
