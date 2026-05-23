import { useState } from 'react'
import { useBilling } from '../../hooks/useBilling'

interface Props {
  feature: string
  description: string
}

export function UpgradePrompt({ feature, description }: Props) {
  const { currency, startCheckout } = useBilling()
  const [loading, setLoading] = useState(false)

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
    <div className="rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-4 space-y-3">
      <div className="space-y-1">
        <p className="text-sm font-medium text-zinc-200">{feature}</p>
        <p className="text-xs text-zinc-500">{description}</p>
      </div>
      <button
        onClick={handleUpgrade}
        disabled={loading}
        className="w-full py-2 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-xs font-medium transition-colors"
      >
        {loading ? 'Redirecting…' : `Upgrade to Pro — ${price}`}
      </button>
      <p className="text-[11px] text-zinc-600 text-center">14-day free trial · Cancel anytime</p>
    </div>
  )
}