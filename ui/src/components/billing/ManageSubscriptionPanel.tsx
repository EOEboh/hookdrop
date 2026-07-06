import { useState } from 'react'
import { useBilling } from '../../context/BillingContext'
import { api } from '../../api/client'
import { usePostHog } from '@posthog/react'

export function ManageSubscriptionPanel({
  onClose,
}: {
  onClose: () => void
}) {
  const posthog = usePostHog()
  const { subscription, isTrialing, refetch } = useBilling()
  const [cancelling, setCancelling] = useState(false)
  const [cancelled, setCancelled]   = useState(false)
  const [error, setError]           = useState<string | null>(null)

  const renewalDate = subscription?.current_period_end
    ? new Date(subscription.current_period_end).toLocaleDateString('en-GB', {
        day: 'numeric', month: 'long', year: 'numeric',
      })
    : null

  const trialEndDate = subscription?.trial_end
    ? new Date(subscription.trial_end).toLocaleDateString('en-GB', {
        day: 'numeric', month: 'long', year: 'numeric',
      })
    : null

  async function handleCancel() {
    setCancelling(true)
    setError(null)
    try {
      await api.cancelSubscription()
      posthog?.capture('subscription_cancelled', {      
        provider:  subscription?.provider,
        interval:  subscription?.interval,
        was_trial: isTrialing,
      })
      await refetch()
      setCancelled(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Cancellation failed')
    } finally {
      setCancelling(false)
    }
  }

  if (cancelled) {
    return (
      <div className="space-y-4">
        <div className="rounded-lg bg-surface-hover px-4 py-4 text-center space-y-2">
          <p className="text-sm font-medium text-ink">
            Subscription cancelled
          </p>
          <p className="text-xs text-muted">
            Your Pro access continues until{' '}
            <span className="text-ink">{renewalDate}</span>.
            After that you'll move to the free plan automatically.
          </p>
        </div>
        <button
          onClick={onClose}
          className="w-full py-2 rounded-lg border border-border-strong text-muted text-sm hover:text-ink transition-colors duration-200 ease-(--ease-considered)"
        >
          Close
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-5">

      {/* Current status */}
      <div className="rounded-lg bg-surface-hover/60 px-4 py-3 space-y-2.5">
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">Status</span>
          <span className={`text-xs font-medium ${
            isTrialing ? 'text-amber-400' : 'text-emerald-400'
          }`}>
            {isTrialing ? 'Trial active' : 'Active'}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">Plan</span>
          <span className="text-xs text-ink font-mono">
            hookdrop Pro ·{' '}
            {subscription?.interval === 'year' ? 'Annual' : 'Monthly'}
          </span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-muted">
            {subscription?.cancel_at_period_end
              ? 'Access until'
              : isTrialing
              ? 'Trial ends'
              : 'Next billing date'
            }
          </span>
          <span className="text-xs text-ink">
            {isTrialing ? trialEndDate : renewalDate}
          </span>
        </div>
        {subscription?.cancel_at_period_end && (
          <p className="text-xs text-amber-400 pt-1">
            Cancellation scheduled — access until {renewalDate}
          </p>
        )}
      </div>

      {/* Cancel — only shown if not already cancelled */}
      {!subscription?.cancel_at_period_end && (
        <div className="space-y-2">
          <button
            onClick={handleCancel}
            disabled={cancelling}
            className="w-full py-2 rounded-lg border border-red-500/20 text-red-400 hover:bg-red-500/10 text-sm transition-colors duration-200 ease-(--ease-considered) disabled:opacity-50"
          >
            {cancelling ? 'Cancelling…' : 'Cancel subscription'}
          </button>
          <p className="text-[11px] text-faint text-center">
            You keep Pro access until the end of your current period.
          </p>
        </div>
      )}

      {error && (
        <p className="text-xs text-red-400 bg-red-500/10 px-3 py-2 rounded-lg text-center">
          {error}
        </p>
      )}

      <button
        onClick={onClose}
        className="w-full py-2 rounded-lg border border-border-strong text-muted hover:text-ink text-sm transition-colors duration-200 ease-(--ease-considered)"
      >
        Close
      </button>
    </div>
  )
}
