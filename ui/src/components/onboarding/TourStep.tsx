import type { TourStepConfig } from './tourSteps'

interface Props {
  step: TourStepConfig
  stepIndex: number
  totalSteps: number
  onNext: () => void
  onBack: () => void
  onSkip: () => void
}

/** Pure content for a tour card: copy, step indicator, and actions.
 *  Positioning/animation/focus-trap live in TourTooltip. */
export function TourStep({ step, stepIndex, totalSteps, onNext, onBack, onSkip }: Props) {
  const isFirst = stepIndex === 0
  const showIndicator = !step.isFinal

  return (
    <div className="space-y-4">
      <p className="text-sm text-ink leading-relaxed">{step.copy}</p>

      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          {!isFirst && (
            <button
              onClick={onBack}
              aria-label="Previous step"
              className="flex items-center justify-center w-8 h-8 rounded-lg text-muted hover:text-ink hover:bg-surface-hover transition-colors duration-200 ease-(--ease-considered)"
            >
              ←
            </button>
          )}
          {showIndicator && (
            <span className="text-[11px] text-faint tabular-nums">
              {stepIndex + 1} of {totalSteps}
            </span>
          )}
        </div>

        <div className="flex items-center gap-4">
          <button
            onClick={onSkip}
            className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
          >
            Skip tour
          </button>
          <button
            onClick={onNext}
            className="px-4 py-1.5 bg-indigo-500 hover:bg-indigo-400 active:bg-indigo-600 active:scale-[0.98] text-white text-sm font-medium rounded-lg transition-all duration-200 ease-(--ease-considered)"
          >
            {step.primaryLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
