import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react'
import { api } from '../api/client'
import type { BillingState } from '../types'
import { trackEvent } from '../lib/analytics'

interface BillingContextValue extends BillingState {
  isPro: boolean
  isTrialing: boolean
  currency: 'ngn' | 'usd'
  refetch: () => Promise<void>
  startCheckout: (interval: 'month' | 'year') => Promise<void>
  getPaystackConfig: (
    interval: 'month' | 'year',
    email: string,
    userId: string,
  ) => {
    publicKey: string
    email: string
    amount: number
    plan: string
    currency: string
    label: string
  }
  handlePaystackSuccess: (
    reference: string,
    interval: 'month' | 'year',
  ) => Promise<void>
  openPortal: () => Promise<void>
}

const BillingContext = createContext<BillingContextValue | null>(null)

// Timezones for the countries Paystack can actually charge. Derived from
// PaystackCountries in internal/billing/provider.go. Keep the two in step.
//
// Matching on 'Africa/' as a whole was wrong: Paystack serves 14 countries,
// so a visitor in Casablanca or Algiers was being routed to a provider that
// cannot take their money.
const PAYSTACK_TIMEZONES = new Set([
  'Africa/Lagos',        // NG
  'Africa/Accra',        // GH
  'Africa/Johannesburg', // ZA
  'Africa/Nairobi',      // KE
  'Africa/Abidjan',      // CI
  'Africa/Kigali',       // RW
  'Africa/Dar_es_Salaam',// TZ
  'Africa/Cairo',        // EG
  'Africa/Kampala',      // UG
  'Africa/Douala',       // CM
  'Africa/Lusaka',       // ZM
  'Africa/Dakar',        // SN
  'Africa/Addis_Ababa',  // ET
  'Africa/Maputo',       // MZ
])

// Locales whose region is a Paystack country.
const PAYSTACK_LOCALE_REGIONS = new Set([
  'NG', 'GH', 'ZA', 'KE', 'CI', 'RW', 'TZ',
  'EG', 'UG', 'CM', 'ZM', 'SN', 'ET', 'MZ',
])

function detectCurrency(): 'ngn' | 'usd' {
  const stored = localStorage.getItem('hookdrop_currency')
  if (stored === 'ngn' || stored === 'usd') return stored
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ''
    if (PAYSTACK_TIMEZONES.has(tz)) return 'ngn'

    const region = (navigator.language ?? '').split('-')[1]?.toUpperCase()
    if (region && PAYSTACK_LOCALE_REGIONS.has(region)) return 'ngn'
  } catch {
    // Intl not available. Default to usd
  }
  return 'usd'
}

export function BillingProvider({
  children,
}: {
  children: React.ReactNode
}) {
  const [state, setState] = useState<BillingState>({
    subscription: null,
    limits:       null,
    is_active:    false,
    loading:      true,
  })

  // detectCurrency is stable: compute once on mount
  const currency = useMemo(() => detectCurrency(), [])

  const fetchSubscription = useCallback(async () => {
  // No token means logged out: return default free state silently
  const token = localStorage.getItem('hookdrop_token')
  if (!token) {
    setState({
      subscription: null,
      limits:       null,
      is_active:    false,
      loading:      false,
    })
    return
  }

  try {
    const data = await api.getSubscription()
    setState({
      subscription: data.subscription,
      limits:       data.limits,
      is_active:    data.is_active,
      loading:      false,
    })
  } catch {
    // 401 or network error: treat as logged out, don't redirect
    setState({
      subscription: null,
      limits:       null,
      is_active:    false,
      loading:      false,
    })
  }
}, [])

  useEffect(() => {
    fetchSubscription()
  }, [fetchSubscription])

  async function startCheckout(interval: 'month' | 'year') {
    const result = await api.createCheckout(interval, currency)
    window.location.href = result.redirect_url
  }



// Must match the LIVE Paystack plans, which are the source of truth:
//   PLN_xja5qfa3bsw1npb  monthly  350000
//   PLN_pqaarazyxu6np2o  annual  3360000
// The test-mode plans are priced differently (annual is 3350000 there), so
// read the live plans when checking these. Not whatever the test keys show.
const PAYSTACK_PRICES = {
  month: 350000,   // ₦3,500
  year:  3360000,  // ₦33,600
} as const

function getPaystackConfig(
  interval: 'month' | 'year',
  email: string,
  userId: string,
) {
  const isAnnual = interval === 'year'

  const planCode = isAnnual
    ? import.meta.env.VITE_PAYSTACK_PLAN_PRO_ANNUAL
    : import.meta.env.VITE_PAYSTACK_PLAN_PRO_MONTHLY

  const amount = PAYSTACK_PRICES[interval]


  if (!planCode) {
    console.error(
      `[hookdrop] Missing env var: ${
        isAnnual
          ? 'VITE_PAYSTACK_PLAN_PRO_ANNUAL'
          : 'VITE_PAYSTACK_PLAN_PRO_MONTHLY'
      }`
    )
  }

  if (!import.meta.env.VITE_PAYSTACK_PUBLIC_KEY) {
    console.error('[hookdrop] Missing env var: VITE_PAYSTACK_PUBLIC_KEY')
  }

  console.info(`[hookdrop] Paystack config: interval=${interval} plan=${planCode} amount=${amount}`)

  return {
    publicKey: import.meta.env.VITE_PAYSTACK_PUBLIC_KEY ?? '',
    email,
    amount,
    plan:     planCode ?? '',
    currency: 'NGN',
    label:    `hookdrop Pro ${isAnnual ? 'Annual' : 'Monthly'}`,
    // Carried onto the transaction so subscription webhooks can identify the
    // local user. Renewals will not have it. Those resolve by customer code.
    metadata: {
      user_id: userId,
      custom_fields: [],
    },
  }
}

  async function handlePaystackSuccess(
    reference: string,
    interval: 'month' | 'year',
  ) {
    // Small delay: gives Paystack time to finish processing
    await new Promise(resolve => setTimeout(resolve, 1500))

    // Verification is what grants the plan. It is not advisory. If it fails
    // the payment went through but the account was NOT upgraded, and the
    // caller has to say so. Redirecting to ?upgraded=true regardless would
    // leave a charged customer on the free plan with no error anywhere.
    let result
    try {
      result = await api.verifyPaystackPayment({ reference, plan: 'pro', interval })
    } catch (err) {
      console.error('Paystack verification failed:', err)
      trackEvent('checkout_verification_failed', {
        provider: 'paystack',
        interval,
      })
      await fetchSubscription()
      throw new Error(
        `Your payment went through, but we could not activate your subscription. ` +
        `Nothing further will be charged. Please contact support with reference ${reference}.`,
      )
    }

    trackEvent('checkout_completed', {
      provider:  'paystack',
      interval,
      is_trial:  result.is_trial,
    })

    if (result.is_trial) {
      trackEvent('trial_started', {
        provider: 'paystack',
        interval,
      })
    }

    await fetchSubscription()
    window.location.href = '/?upgraded=true'
  }

  async function openPortal() {
    const result = await api.getBillingPortal()
    window.location.href = result.url
  }

  // isPro: plan is pro AND subscription is genuinely active
  // Guard against loading state where is_active defaults to false
  const isPro = !state.loading &&
    state.subscription?.plan === 'pro' &&
    state.is_active

  const isTrialing = state.subscription?.status === 'trialing'

  return (
    <BillingContext.Provider
      value={{
        ...state,
        isPro,
        isTrialing,
        currency,
        refetch:              fetchSubscription,
        startCheckout,
        getPaystackConfig,
        handlePaystackSuccess,
        openPortal,
      }}
    >
      {children}
    </BillingContext.Provider>
  )
}

export function useBilling() {
  const ctx = useContext(BillingContext)
  if (!ctx) throw new Error('useBilling must be used within BillingProvider')
  return ctx
}