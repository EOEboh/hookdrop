import { createContext, useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { TOUR_STEPS, TOUR_COMPLETED_KEY, type TourStepConfig } from './tourSteps'
import { TourOverlay } from './TourOverlay'
import { TourTooltip } from './TourTooltip'

export interface TourContextValue {
  isActive: boolean
  stepIndex: number
  step: TourStepConfig | null
  totalSteps: number
  targetRect: DOMRect | null
  next: () => void
  back: () => void
  skip: () => void
  start: () => void
}

export const TourContext = createContext<TourContextValue | null>(null)

function resolveTargetRect(target: string | null): DOMRect | null {
  if (!target) return null
  const el = document.querySelector<HTMLElement>(`[data-tour="${target}"]`)
  if (!el) return null
  const rect = el.getBoundingClientRect()
  if (rect.width === 0 && rect.height === 0) return null
  return rect
}

export function TourProvider({ children }: { children: React.ReactNode }) {
  const [isActive, setIsActive] = useState(false)
  const [stepIndex, setStepIndex] = useState(0)
  const [targetRect, setTargetRect] = useState<DOMRect | null>(null)
  const hasAutoStarted = useRef(false)

  const step = isActive ? TOUR_STEPS[stepIndex] ?? null : null
  const totalSteps = TOUR_STEPS.length

  const finish = useCallback(() => {
    localStorage.setItem(TOUR_COMPLETED_KEY, 'true')
    setIsActive(false)
  }, [])

  const skip = useCallback(() => {
    finish()
  }, [finish])

  const next = useCallback(() => {
    setStepIndex(i => {
      if (i >= TOUR_STEPS.length - 1) {
        finish()
        return i
      }
      return i + 1
    })
  }, [finish])

  const back = useCallback(() => {
    setStepIndex(i => Math.max(0, i - 1))
  }, [])

  const start = useCallback(() => {
    setStepIndex(0)
    setIsActive(true)
  }, [])

  // Auto-start once for new users. TourProvider only mounts once the
  // authenticated app shell has passed its loading/session guards, so
  // mount timing already satisfies "only after login + active session".
  useEffect(() => {
    if (hasAutoStarted.current) return
    hasAutoStarted.current = true
    if (localStorage.getItem(TOUR_COMPLETED_KEY) !== 'true') {
      start()
    }
  }, [start])

  // Track the current target element's rect so the overlay/tooltip can
  // follow it (and re-resolve it on scroll/resize/content changes).
  // useLayoutEffect (not useEffect) so the rect updates in the same paint
  // as the step's content — otherwise the old spotlight can flash for a
  // frame while the new step's copy has already rendered.
  useLayoutEffect(() => {
    if (!isActive || !step?.target) {
      setTargetRect(resolveTargetRect(step?.target ?? null))
      return
    }

    const el = document.querySelector<HTMLElement>(`[data-tour="${step.target}"]`)

    const update = () => setTargetRect(resolveTargetRect(step.target))
    update()

    let raf = 0
    const onScrollOrResize = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(update)
    }
    window.addEventListener('resize', onScrollOrResize)
    window.addEventListener('scroll', onScrollOrResize, true)

    const resizeObserver = el ? new ResizeObserver(update) : null
    if (el && resizeObserver) resizeObserver.observe(el)

    return () => {
      cancelAnimationFrame(raf)
      window.removeEventListener('resize', onScrollOrResize)
      window.removeEventListener('scroll', onScrollOrResize, true)
      resizeObserver?.disconnect()
    }
  }, [isActive, step])

  // Keyboard navigation, only while the tour is active.
  useEffect(() => {
    if (!isActive) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.preventDefault()
        skip()
      } else if (e.key === 'ArrowRight') {
        e.preventDefault()
        next()
      } else if (e.key === 'ArrowLeft') {
        e.preventDefault()
        back()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [isActive, next, back, skip])

  const value: TourContextValue = {
    isActive,
    stepIndex,
    step,
    totalSteps,
    targetRect,
    next,
    back,
    skip,
    start,
  }

  return (
    <TourContext.Provider value={value}>
      {children}
      {isActive && step && createPortal(
        <>
          <TourOverlay targetRect={targetRect} />
          <TourTooltip
            step={step}
            stepIndex={stepIndex}
            totalSteps={totalSteps}
            targetRect={targetRect}
            onNext={next}
            onBack={back}
            onSkip={skip}
          />
        </>,
        document.body
      )}
    </TourContext.Provider>
  )
}
