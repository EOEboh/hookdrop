import { useState, useEffect } from 'react'
import { usePaystackPayment } from 'react-paystack'
import { useBilling } from '../../context/BillingContext'
import { useAuth } from '../../context/AuthContext'
import { ManageSubscriptionPanel } from './ManageSubscriptionPanel'
import { Spinner } from '../ui/Spinner'
import { usePostHog } from '@posthog/react'


const PLANS: Record<string, Record<string, {
  price: string; period: string; total: string | null
}>> = {
  usd: {
    month: { price: '$7',     period: '/mo', total: null },
    year:  { price: '$5.58',  period: '/mo', total: 'Billed $67/yr' },
  },
  ngn: {
    month: { price: '₦3,500', period: '/mo', total: null },
    year:  { price: '₦2,800', period: '/mo', total: 'Billed ₦33,600/yr' },
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

const PRICE_DISPLAY = {
  month: { amount: '₦3,500', suffix: '/mo after trial' },
  year:  { amount: '₦33,600', suffix: '/yr after trial' },
} as const

function PaystackButton({
  interval,
  email,
  loading,
  setLoading,
  onSuccess,
}: {
  interval: 'month' | 'year'
  email: string
  loading: boolean
  setLoading: (v: boolean) => void
  onSuccess: (ref: string, interval: 'month' | 'year') => void
}) {
  const posthog = usePostHog() 
  const { getPaystackConfig } = useBilling()
  const config = getPaystackConfig(interval, email)
  const initializePayment = usePaystackPayment(config)

  const display = PRICE_DISPLAY[interval]

  function handleClick() {
    setLoading(true)
     posthog?.capture('checkout_started', {              
      provider: 'paystack',
      interval,
      currency: 'ngn',
    })
    initializePayment({
      onSuccess: (response: {
        reference: string
        trxref: string
        [key: string]: unknown
      }) => {
        onSuccess(response.reference ?? response.trxref, interval)
      },
      onClose: () => {
        setLoading(false)
      },
    })
  }

  return (
    <button
      onClick={handleClick}
      disabled={loading || !email || !config.publicKey || !config.plan}
      className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
    >
      {loading
        ? 'Processing…'
        : `Start free trial: ${display.amount} ${display.suffix}`
      }
    </button>
  )
}

export function PricingPage() {
  const {
    subscription, isPro, isTrialing, loading,
    currency, startCheckout,
    handlePaystackSuccess,
  } = useBilling()
  const { user } = useAuth()
  const posthog = usePostHog()

  useEffect(() => {
    posthog?.capture('pricing_page_viewed', {           
      currency,
      already_pro: isPro,
    })
  }, []) 

  const [interval, setInterval]           = useState<'month' | 'year'>('month')
  const [payLoading, setPayLoading]        = useState(false)
  const [showManagePanel, setShowManagePanel] = useState(false)

  const prices = PLANS[currency]?.[interval] ?? PLANS['usd']['month']

  // ── Loading guard — prevents flash of free view for Pro users
  if (loading) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center">
        <Spinner size={5} />
      </div>
    )
  }

  async function handleLSCheckout() {
    setPayLoading(true)
      posthog?.capture('checkout_started', {              
      provider: 'lemonsqueezy',
      interval,
      currency: 'usd',
    })
    try {
      await startCheckout(interval)
    } catch {
      setPayLoading(false)
    }
  }

  async function handlePaystackSuccess_(
    reference: string,
    iv: 'month' | 'year',
  ) {
    await handlePaystackSuccess(reference, iv)
    setPayLoading(false)
  }

  // ── Renewal date label — aware of cancellation and trial states
  function renewalLabel(): string {
    if (!subscription?.current_period_end) return 'monthly'
    const date = new Date(subscription.current_period_end).toLocaleDateString(
      'en-GB',
      { day: 'numeric', month: 'long', year: 'numeric' },
    )
    if (subscription.cancel_at_period_end) return `Access until ${date}`
    return `Renews ${date}`
  }

  function trialLabel(): string {
    if (!subscription?.trial_end) return 'Trial ends soon'
    return `Trial ends ${new Date(subscription.trial_end).toLocaleDateString(
      'en-GB',
      { day: 'numeric', month: 'long', year: 'numeric' },
    )}`
  }

  // ── Pro view
  if (isPro) {
    return (
      <div className="max-w-lg mx-auto px-6 py-16 space-y-8">
        <div className="rounded-xl border border-emerald-500/30 bg-zinc-900/50 p-8 space-y-6">

          <div className="flex items-start justify-between">
            <div>
              <div className="mb-2">
                <span className="text-[11px] font-semibold bg-emerald-600 text-white px-2.5 py-0.5 rounded-full">
                  {isTrialing ? 'Trial active' : 'Pro plan'}
                </span>
              </div>
              <h1 className="text-xl font-semibold text-zinc-100">
                hookdrop Pro
              </h1>
              <p className="text-zinc-400 text-sm mt-1">
                {isTrialing ? trialLabel() : renewalLabel()}
              </p>
            </div>
            <span className="text-3xl">⚡</span>
          </div>

          <div className="space-y-2.5 pt-4 border-t border-zinc-800">
            {PRO_FEATURES.map(f => (
              <div
                key={f}
                className="flex items-center gap-2.5 text-sm text-zinc-300"
              >
                <span className="text-emerald-400">✓</span>
                {f}
              </div>
            ))}
          </div>

          {/* Management section: toggles between button and panel */}
          <div className="pt-4 border-t border-zinc-800">
            {showManagePanel ? (
              <ManageSubscriptionPanel
                onClose={() => setShowManagePanel(false)}
              />
            ) : (
              <div className="space-y-2.5">
                <button
                  onClick={() => setShowManagePanel(true)}
                  className="w-full py-2.5 rounded-lg border border-zinc-700 hover:border-zinc-500 text-zinc-300 hover:text-zinc-100 text-sm font-medium transition-colors"
                >
                  Manage subscription
                </button>
                <p className="text-xs text-zinc-600 text-center">
                  Cancel anytime! Access continues until end of billing period.
                </p>
              </div>
            )}
          </div>
        </div>

        <div className="text-center">
          <a
            href="/"
            className="text-sm text-zinc-500 hover:text-zinc-300 transition-colors"
          >
            ← Back to hookdrop
          </a>
        </div>
      </div>
    )
  }

  // ── Free view
  return (
    <div className="max-w-3xl mx-auto px-6 py-12 space-y-10">

      <div className="text-center space-y-2">
        <h1 className="text-2xl font-semibold text-zinc-100">Simple pricing</h1>
        <p className="text-zinc-400 text-sm">
          Start free. Upgrade when you need permanent endpoints.
        </p>
      </div>

      <div className="flex items-center justify-center gap-3 text-sm">
        <span className="text-zinc-500">
          {currency === 'ngn' ? 'Paying in NGN' : 'Paying from Nigeria?'}
        </span>
        <button
          onClick={() => {
            localStorage.setItem(
              'hookdrop_currency',
              currency === 'ngn' ? 'usd' : 'ngn',
            )
            window.location.reload()
          }}
          className="text-emerald-400 hover:text-emerald-300 transition-colors underline underline-offset-2"
        >
          {currency === 'ngn' ? 'Switch to USD' : 'Pay in NGN instead'}
        </button>
      </div>

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

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">

        {/* Free card */}
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
              <li
                key={f}
                className="flex items-start gap-2.5 text-sm text-zinc-400"
              >
                <span className="text-zinc-600 mt-0.5">–</span>
                {f}
              </li>
            ))}
          </ul>
          <div className="pt-2">
            <div className="w-full py-2 rounded-lg bg-zinc-800 text-zinc-500 text-sm text-center">
              Current plan
            </div>
          </div>
        </div>

        {/* Pro card */}
        <div className="rounded-xl border border-emerald-500/30 bg-zinc-900/50 p-6 space-y-6 relative">
          <div className="absolute -top-3 left-1/2 -translate-x-1/2">
            <span className="bg-emerald-600 text-white text-[11px] font-semibold px-3 py-1 rounded-full">
              Most popular
            </span>
          </div>
          <div>
            <h2 className="text-lg font-semibold text-zinc-100">Pro</h2>
            <div className="mt-2 flex items-baseline gap-1">
              <span className="text-3xl font-bold text-zinc-100">
                {prices.price}
              </span>
              <span className="text-zinc-500 text-sm">{prices.period}</span>
            </div>
            {prices.total
              ? <p className="text-zinc-500 text-xs mt-1">{prices.total}</p>
              : <p className="text-emerald-500 text-xs mt-1">14-day free trial</p>
            }
          </div>
          <ul className="space-y-2.5">
            {PRO_FEATURES.map(f => (
              <li
                key={f}
                className="flex items-start gap-2.5 text-sm text-zinc-300"
              >
                <span className="text-emerald-400 mt-0.5">✓</span>
                {f}
              </li>
            ))}
          </ul>
          <div className="pt-2">
            {currency === 'ngn' ? (
              <PaystackButton
                interval={interval}
                email={user?.email ?? ''}
                loading={payLoading}
                setLoading={setPayLoading}
                onSuccess={handlePaystackSuccess_}
              />
            ) : (
              <button
                onClick={handleLSCheckout}
                disabled={payLoading}
                className="w-full py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 text-white text-sm font-medium transition-colors"
              >
                {payLoading
                  ? 'Redirecting…'
                  : `Start free trial: ${prices.price}${prices.period} after`
                }
              </button>
            )}
          </div>
        </div>
      </div>

      <p className="text-center text-xs text-zinc-600">
        Cancel anytime. No questions asked.{' '}
        Payments processed securely by{' '}
        {currency === 'ngn' ? 'Paystack' : 'Lemon Squeezy'}.
      </p>
    </div>
  )
}