import { useEffect, useState } from 'react'

const STORAGE_KEY = 'hookdrop_onboarded'

/**
 * A small first-time nudge pointing at the copy button, dismissed permanently
 * once the user copies the inbox URL (or manually dismisses it).
 */
export function OnboardingHint({ onDismiss }: { onDismiss?: () => void }) {
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const seen = localStorage.getItem(STORAGE_KEY)
    if (!seen) setVisible(true)
  }, [])

  function dismiss() {
    localStorage.setItem(STORAGE_KEY, 'true')
    setVisible(false)
    onDismiss?.()
  }

  if (!visible) return null

  return (
    <div className="relative flex items-center gap-2 pl-1 animate-fade-in">
      <svg
        viewBox="0 0 24 16"
        className="w-4 h-3.5 text-indigo-400 shrink-0 -mt-3"
        fill="none"
      >
        <path
          d="M2 2c4 0 14 0 18 9"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          fill="none"
        />
        <path d="M15 9l5 2-2 5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" fill="none" />
      </svg>
      <p className="text-[11px] text-indigo-300 leading-snug">
        Paste this into Stripe or Paystack to get started
      </p>
      <button
        onClick={dismiss}
        className="ml-auto text-[11px] text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered) shrink-0"
      >
        Got it
      </button>
    </div>
  )
}

export function markOnboardingSeen() {
  localStorage.setItem(STORAGE_KEY, 'true')
}
