import { useState, useEffect } from 'react'
import { useBilling } from '../../context/BillingContext'
import { usePostHog } from '@posthog/react'


interface Props {
  feature: string
  description: string
}

export function UpgradePrompt({ feature, description }: Props) {
  const posthog = usePostHog()
  const { currency, startCheckout } = useBilling()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    posthog?.capture('upgrade_prompt_seen', { feature }) 
  }, [feature])

  const price = currency === 'ngn' ? '₦3,500/mo' : '$7/mo'

  async function handleUpgrade() {
    setLoading(true)
    try {
      await startCheckout('month')
    } catch {
      setLoading(false)
    }
  }

  return (
    <div className="rounded-lg border border-indigo-500/20 bg-indigo-500/5 p-4 space-y-3">
      <div className="space-y-1">
        <p className="text-sm font-medium text-ink">{feature}</p>
        <p className="text-xs text-muted">{description}</p>
      </div>
      <button
        onClick={handleUpgrade}
        disabled={loading}
        className="w-full py-2 rounded-lg bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] disabled:opacity-50 text-white text-xs font-medium transition-all duration-200 ease-(--ease-considered)"
      >
        {loading ? 'Redirecting…' : `Upgrade to Pro — ${price}`}
      </button>
      <p className="text-[11px] text-faint text-center">14-day free trial · Cancel anytime</p>
    </div>
  )
}
