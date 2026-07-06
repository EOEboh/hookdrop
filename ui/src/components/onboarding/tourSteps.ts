// Step configs for the first-time onboarding tour.
// `target` is a `data-tour` attribute value; `null` means a centered
// modal not attached to any element (used for the welcome step, and as
// the fallback when a step's target can't be found on screen).
export type TourPlacement = 'top' | 'bottom' | 'left' | 'right'

export interface TourStepConfig {
  id: string
  target: string | null
  copy: string
  placement: TourPlacement
  primaryLabel: string
  /** Hides the step indicator ("N of 5") — spec allows this on the final step. */
  isFinal?: boolean
}

export const TOUR_STEPS: TourStepConfig[] = [
  {
    id: 'welcome',
    target: null,
    copy: "Welcome to hookdrop. Let's get your first webhook flowing in under a minute.",
    placement: 'bottom',
    primaryLabel: "Let's go",
  },
  {
    id: 'inbox-url',
    target: 'inbox-url',
    copy: "This is your unique inbox URL. Copy it and paste it into Stripe, Paystack, GitHub, or any webhook provider's settings.",
    placement: 'right',
    primaryLabel: 'Next',
  },
  {
    id: 'request-feed',
    target: 'request-feed',
    copy: 'The moment a webhook arrives at your URL, it shows up here instantly — no refresh, no polling.',
    placement: 'right',
    primaryLabel: 'Next',
  },
  {
    id: 'detail-panel',
    target: 'detail-panel',
    copy: 'Click any request to see its full headers, body, and whether its signature was verified.',
    placement: 'left',
    primaryLabel: 'Next',
  },
  {
    id: 'replay-panel',
    target: 'replay-panel',
    copy: "Once you've got a real webhook captured, replay it to your local server as many times as you need — no need to trigger it again from the provider.",
    placement: 'left',
    primaryLabel: 'Got it',
    isFinal: true,
  },
]

export const TOUR_COMPLETED_KEY = 'hookdrop_tour_completed'
