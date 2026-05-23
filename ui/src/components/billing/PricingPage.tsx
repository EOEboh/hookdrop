import { useState } from 'react'
import { useBilling } from '../../hooks/useBilling'

const PLANS = {
  usd: {
    monthly: { price: '$7', period: '/mo', total: null },
    annual:  { price: '$5.58', period: '/mo', total: 'Billed $67/yr' },
  },
  ngn: {
    monthly: { price: '₦3,500', period: '/mo', total: null },
    annual:  { price: '₦2,800', period: '/mo', total: 'Billed ₦33,600/yr' },
  },
}

const FREE_FEATURES = [
  '1 named endpoint',
  'Temporary sessions (24h)',
  '500 requests / month',
  '7 day history',
  'Basic replay',
]

const PRO_FEATURES = [
  'Unlimited named endpoints',
  '50,000 requests / month',
  '90 day history',
  'Signature verification',
  'Request filtering + search',
  'Priority support',
  '14-day free trial',
]

export function PricingPage() {
  const { subscription, isPro, isTrialing, currency, startCheckout, openPortal } = useBilling()
  const [interval, setInterval] = useState<'month' | 'year'>('month')
  const [loading, setLoading] = useState(false)
  const [showCurrencyHint, setShowCurrencyHint] = useState(true)

  const prices = PLANS[currency][interval]

  async function handleUpgrade() {
    setLoading(true)
    try {
      await startCheckout(interval)
    } catch {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-3xl mx-auto px-6 py-12 space-y-10">

      {/* Header */}
      <div className="text-center space-y-2">
        <h1 className="text-2xl font-semibold text-zinc-100">Simple pricing</h1>
        <p className="text-zinc-400 text-sm">
          Start free. Upgrade when you need permanent endpoints.
        </p>
      </div>

      {/* Currency toggle */}
      {showCurrencyHint && (
        <div className="flex items-center justify-center gap-3 text-sm">
          <span className="text-zinc-500">Paying from Nigeria?</span>
          <button
            onClick={() => {
              localStorage.setItem('hookdrop_currency', currency === 'ngn' ? 'usd' : 'ngn')
              setShowCurrencyHint(false)
              window.location.reload()
            }}
            className="text-emerald-400 hover:text-emerald-300 transition-colors underline underline-offset-2"
          >
            {currency === 'ngn' ? 'Switch to USD' : 'Pay in NGN instead'}
          </button>
        </div>
      )}

      {/* Billing interval toggle */}
      <div className="flex items-center justify-center">
        <div className="flex items-center bg-zinc-900 rounded-lg p-1 gap-1">
          <button
            onClick={() => setInterval('month')}
            className={`px-4 py-1.5 rounded text-sm font-medium transition-colors ${
              interval === 'month'
                ? 'bg-zinc-700 text-zinc-100'
                : 'text-zinc-500 hover:text-zinc-300'
            }`}
          >
            Monthly
          </button>
          <button
            onClick={() => setInterval('year')}
            className={`px-4 py-1.5 rounded text-sm font-medium transition-colors flex items-center gap-2 ${
              interval === 'year'
                ? 'bg-zinc-700 text-zinc-100'
                : 'text-zinc-500 hover:text-zinc-300'
            }`}
          >
            Annual
            <span className="text-[10px] bg-emerald-600/20 text-emerald-400 px-1.5 py-0.5 rounded font-semibold">
              20% off
            </span>
          </button>
        </div>
      </div>

      {/* Plan cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

        {/* Free */}
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-6 space-y-6">
          <div>
            <h2 className="text-lg font-semibold text-zinc-100">Free</h2>
            <div className="mt-2 flex items-baseline gap-1">
              <span className="text-3xl font-bold text-zinc-100">
                {currency === 'ngn' ? '₦0' : '$0'}
              </span>
              <span className="text-zinc-500 text-sm">/ month</span>
            </div>
            <p className="text-zinc-500 text-xs mt-1">Forever free</p>
          </div>

          <ul className="space-y-2.5">
            {FREE_FEATURES.map(f => (
              <li key={f} className="flex items-start gap-2.5 text-sm text-zinc-400">
                <span className="text-zinc-600 mt-0.5">–</span>
                {f}
              </li>
            ))}
          </ul>

          <div className="pt-2">
            {(!subscription || subscription.plan === 'free') ? (
              <div className="w-full py-2 rounded-lg bg-zinc-800 text-zinc-500 text-sm text-center">
                Current plan
              </div>
            ) : (
              <div className="w-full py-2 rounded-lg border border-zinc-700 text-zinc-400 text-sm text-center">
                Included
              </div>
            )}
          </div>
        </div>

        {/* Pro */}
        <div className="rounded-xl border border-emerald-500/30 bg-zinc-900/50 p-6 space-y-6 relative">
          <div className="absolute -top-3 left-1/2 -translate-x-1/2">
            <span className="bg-emerald-600 text-white text-[11px] font-semibold px-3 py-1 rounded-full">
              Most popular
            </span>
          </div>

          <div>
            <h2 className="text-lg font-semibold text-zinc-100">Pro</h2>
            <div className="mt-2 flex items-baseline gap-1">
              <span className="text-3xl font-bold text-zinc-100">{prices.price}</span>
              <span className="text-zinc-500 text-sm">{prices.period}</span>
            </div>
            {prices.total && (
              <p className="text-zinc-500 text-xs mt-1">{prices.total}</p>
            )}
            {!prices.total && (
              <p className="text-emerald-500 text-xs mt-1">14-day free trial</p>
            )}
          </div>

          <ul className="space-y-2.5">
            {PRO_FEATURES.map(f => (
              <li key={f} className="flex items-start gap-2.5 text-sm text-zinc-300">
                <span className="text-emerald-400 mt-0.5">✓</span>
                {f}
              </li>
            ))}
          </ul>

          <div className="pt-2 space-y-2">
            {isPro ? (
              <>
                <div className="w-full py-2 rounded-lg bg-emerald-600/20 text-emerald-400 text-sm text-center font-medium">
                  {isTrialing ? 'Trial active' : 'Current plan'}
                </div>
                <button
                  onClick={openPortal}
                  className="w-full py-2 rounded-lg border border-zinc-700 text-zinc-400 hover:text-zinc-200 text-sm transition-colors"
                >
                  Manage subscription
                </button>
              </>
            ) : (
              <button
                onClick={handleUpgrade}
                disabled={loading}
                className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
              >
                {loading
                  ? 'Redirecting…'
                  : `Start free trial — ${prices.price}${prices.period} after`
                }
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Footer note */}
     <p className="text-center text-xs text-zinc-600">
  Cancel anytime. No questions asked.{' '}
  Payments processed securely by{' '}
  {currency === 'ngn' ? 'Paystack' : 'Lemon Squeezy'}.
</p>
    </div>
  )
}