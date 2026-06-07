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

interface BillingContextValue extends BillingState {
  isPro: boolean
  isTrialing: boolean
  currency: 'ngn' | 'usd'
  refetch: () => Promise<void>
  startCheckout: (interval: 'month' | 'year') => Promise<void>
  getPaystackConfig: (
    interval: 'month' | 'year',
    email: string,
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

function detectCurrency(): 'ngn' | 'usd' {
  const stored = localStorage.getItem('hookdrop_currency')
  if (stored === 'ngn' || stored === 'usd') return stored
  try {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ''
    const locale = navigator.language ?? ''
    if (
      tz.includes('Lagos') ||
      tz.includes('Africa') ||
      locale.startsWith('en-NG')
    ) {
      return 'ngn'
    }
  } catch {
    // Intl not available — default to usd
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

  function getPaystackConfig(interval: 'month' | 'year', email: string) {
    const planCode =
      interval === 'year'
        ? import.meta.env.VITE_PAYSTACK_PLAN_PRO_ANNUAL
        : import.meta.env.VITE_PAYSTACK_PLAN_PRO_MONTHLY

    // Amount satisfies react-paystack validation
    // Paystack overrides this server-side with the plan price
    const amount = interval === 'year' ? 3360000 : 350000

    return {
      publicKey: import.meta.env.VITE_PAYSTACK_PUBLIC_KEY ?? '',
      email,
      amount,
      plan:     planCode,
      currency: 'NGN',
      label:    'hookdrop Pro',
    }
  }

  async function handlePaystackSuccess(
    reference: string,
    interval: 'month' | 'year',
  ) {
    // Small delay: gives Paystack time to finish processing
    await new Promise(resolve => setTimeout(resolve, 1500))

    try {
      await api.verifyPaystackPayment({ reference, plan: 'pro', interval })
    } catch (err) {
      console.error('Verify error (non-fatal):', err)
    }

    // Always refetch: upsert succeeded even if verify had issues
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