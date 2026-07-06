import { useTour } from './useTour'

/** Persistent re-trigger for the onboarding tour, placed near "Log out". */
export function TourHelpButton() {
  const { start } = useTour()

  return (
    <button
      onClick={start}
      title="Show onboarding tour"
      aria-label="Show onboarding tour"
      className="flex items-center justify-center w-5 h-5 rounded-full border border-border-strong text-[11px] font-semibold text-faint hover:text-indigo-400 hover:border-indigo-400/50 transition-colors duration-200 ease-(--ease-considered)"
    >
      ?
    </button>
  )
}
