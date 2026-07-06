import { useLayoutEffect, useRef, useState } from 'react'
import type { TourPlacement, TourStepConfig } from './tourSteps'
import { TourStep } from './TourStep'
import { useReducedMotion } from './useReducedMotion'

interface Props {
  step: TourStepConfig
  stepIndex: number
  totalSteps: number
  targetRect: DOMRect | null
  onNext: () => void
  onBack: () => void
  onSkip: () => void
}

const MARGIN = 14
const VIEWPORT_PADDING = 12

function opposite(placement: TourPlacement): TourPlacement {
  switch (placement) {
    case 'top': return 'bottom'
    case 'bottom': return 'top'
    case 'left': return 'right'
    case 'right': return 'left'
  }
}

function coordsFor(placement: TourPlacement, target: DOMRect, card: { width: number; height: number }) {
  switch (placement) {
    case 'bottom':
      return {
        top: target.bottom + MARGIN,
        left: target.left + target.width / 2 - card.width / 2,
      }
    case 'top':
      return {
        top: target.top - MARGIN - card.height,
        left: target.left + target.width / 2 - card.width / 2,
      }
    case 'right':
      return {
        top: target.top + target.height / 2 - card.height / 2,
        left: target.right + MARGIN,
      }
    case 'left':
      return {
        top: target.top + target.height / 2 - card.height / 2,
        left: target.left - MARGIN - card.width,
      }
  }
}

function fits(pos: { top: number; left: number }, card: { width: number; height: number }) {
  return (
    pos.top >= VIEWPORT_PADDING &&
    pos.left >= VIEWPORT_PADDING &&
    pos.top + card.height <= window.innerHeight - VIEWPORT_PADDING &&
    pos.left + card.width <= window.innerWidth - VIEWPORT_PADDING
  )
}

function clamp(pos: { top: number; left: number }, card: { width: number; height: number }) {
  const maxTop = Math.max(VIEWPORT_PADDING, window.innerHeight - VIEWPORT_PADDING - card.height)
  const maxLeft = Math.max(VIEWPORT_PADDING, window.innerWidth - VIEWPORT_PADDING - card.width)
  return {
    top: Math.min(Math.max(pos.top, VIEWPORT_PADDING), maxTop),
    left: Math.min(Math.max(pos.left, VIEWPORT_PADDING), maxLeft),
  }
}

function computePosition(target: DOMRect, card: { width: number; height: number }, preferred: TourPlacement) {
  const order: TourPlacement[] = [preferred, opposite(preferred), 'bottom', 'top', 'right', 'left']
  for (const placement of order) {
    const pos = coordsFor(placement, target, card)
    if (fits(pos, card)) return pos
  }
  return clamp(coordsFor(preferred, target, card), card)
}

export function TourTooltip({ step, stepIndex, totalSteps, targetRect, onNext, onBack, onSkip }: Props) {
  const cardRef = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null)
  const reducedMotion = useReducedMotion()

  useLayoutEffect(() => {
    if (!targetRect || !cardRef.current) {
      setPos(null)
      return
    }
    const { width, height } = cardRef.current.getBoundingClientRect()
    const next = computePosition(targetRect, { width, height }, step.placement)
    setPos(next)
  }, [targetRect, step])

  // Autofocus the primary action and trap Tab navigation within the card.
  useLayoutEffect(() => {
    const container = cardRef.current
    if (!container) return
    const focusables = container.querySelectorAll<HTMLElement>('button, a[href], [tabindex]:not([tabindex="-1"])')
    focusables[focusables.length - 1]?.focus()

    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Tab') return
      const list = container!.querySelectorAll<HTMLElement>('button, a[href], [tabindex]:not([tabindex="-1"])')
      if (list.length === 0) return
      const first = list[0]
      const last = list[list.length - 1]
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
    container.addEventListener('keydown', onKeyDown)
    return () => container.removeEventListener('keydown', onKeyDown)
  }, [step])

  const animationClass = reducedMotion ? '' : 'motion-safe:animate-arrive'

  const cardClasses = `w-[320px] max-w-[calc(100vw-24px)] bg-surface border border-border-strong rounded-xl shadow-2xl shadow-black/40 p-5 z-[10000] ${animationClass}`

  // Centered mode: welcome step, or fallback when the target can't be found.
  if (!targetRect || !pos) {
    return (
      <div className="fixed inset-0 z-[10000] flex items-center justify-center p-4">
        <div ref={cardRef} className={cardClasses} role="dialog" aria-modal="true">
          <TourStep
            step={step}
            stepIndex={stepIndex}
            totalSteps={totalSteps}
            onNext={onNext}
            onBack={onBack}
            onSkip={onSkip}
          />
        </div>
      </div>
    )
  }

  return (
    <div
      ref={cardRef}
      role="dialog"
      aria-modal="true"
      className={`fixed ${cardClasses}`}
      style={{
        top: pos.top,
        left: pos.left,
        transition: reducedMotion ? 'none' : 'top 220ms var(--ease-considered), left 220ms var(--ease-considered)',
      }}
    >
      <TourStep
        step={step}
        stepIndex={stepIndex}
        totalSteps={totalSteps}
        onNext={onNext}
        onBack={onBack}
        onSkip={onSkip}
      />
    </div>
  )
}
