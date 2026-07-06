import { useReducedMotion } from './useReducedMotion'

interface Props {
  /** null → no specific element to highlight (welcome step, or fallback
   *  when the current step's target can't be found on screen). */
  targetRect: DOMRect | null
}

const SPOTLIGHT_PADDING = 8
const OVERLAY_COLOR = 'rgba(9, 9, 14, 0.72)'

/**
 * Click-blocking "frame": four fixed rectangles that tile the viewport
 * around the padded target rect, leaving a literal gap over the target
 * itself. Unlike a single full-screen blocker + z-index bump on the
 * target, this never depends on the target's ancestors *not* creating
 * their own stacking context (e.g. the sidebar's `sticky` positioning) —
 * the real target element is simply never covered by anything, so it
 * stays natively interactive with zero DOM/class mutation.
 */
function ClickBlockerFrame({ rect }: { rect: DOMRect }) {
  const top = rect.top - SPOTLIGHT_PADDING
  const left = rect.left - SPOTLIGHT_PADDING
  const right = rect.right + SPOTLIGHT_PADDING
  const bottom = rect.bottom + SPOTLIGHT_PADDING

  const shared: React.CSSProperties = { position: 'fixed', pointerEvents: 'auto' }

  return (
    <>
      <div className="z-[9998]" aria-hidden="true" style={{ ...shared, top: 0, left: 0, width: '100%', height: Math.max(0, top) }} />
      <div className="z-[9998]" aria-hidden="true" style={{ ...shared, top: bottom, left: 0, width: '100%', bottom: 0 }} />
      <div className="z-[9998]" aria-hidden="true" style={{ ...shared, top, left: 0, width: Math.max(0, left), height: Math.max(0, bottom - top) }} />
      <div className="z-[9998]" aria-hidden="true" style={{ ...shared, top, left: right, right: 0, height: Math.max(0, bottom - top) }} />
    </>
  )
}

export function TourOverlay({ targetRect }: Props) {
  const reducedMotion = useReducedMotion()
  const transition = reducedMotion ? 'none' : 'top 220ms var(--ease-considered), left 220ms var(--ease-considered), width 220ms var(--ease-considered), height 220ms var(--ease-considered)'

  return (
    <>
      {targetRect ? (
        <ClickBlockerFrame rect={targetRect} />
      ) : (
        // Centered mode: no target to carve a hole for, so a single
        // full-screen tinted blocker doubles as the modal backdrop.
        <div className="fixed inset-0 z-[9998]" aria-hidden="true" style={{ background: OVERLAY_COLOR }} />
      )}

      {targetRect && (
        <div
          className="fixed pointer-events-none z-[9999] rounded-xl"
          aria-hidden="true"
          style={{
            top: targetRect.top - SPOTLIGHT_PADDING,
            left: targetRect.left - SPOTLIGHT_PADDING,
            width: targetRect.width + SPOTLIGHT_PADDING * 2,
            height: targetRect.height + SPOTLIGHT_PADDING * 2,
            transition,
            boxShadow: `0 0 0 9999px ${OVERLAY_COLOR}, 0 0 24px 6px rgba(99, 102, 241, 0.35), 0 0 0 2px #818cf8`,
          }}
        />
      )}
    </>
  )
}
