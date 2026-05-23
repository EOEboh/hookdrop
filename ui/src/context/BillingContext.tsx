import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from 'react'
import { api } from '../api/client'
import type { BillingState, PlanLimits, Subscription } from '../types'

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
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ''
  const locale = navigator.language ?? ''
  if (
    tz.includes('Lagos') ||
    tz.includes('Africa') ||
    locale.includes('NG')
  ) {
    return 'ngn'
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
    limits: null,
    is_active: true,
    loading: true,
  })

  const currency = detectCurrency()

  const fetchSubscription = useCallback(async () => {
    try {
      const data = await api.getSubscription()
      setState({
        subscription: data.subscription,
        limits:       data.limits,
        is_active:    data.is_active,
        loading:      false,
      })
    } catch {
      setState(s => ({ ...s, loading: false }))
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
    await new Promise(resolve => setTimeout(resolve, 1500))

    try {
      await api.verifyPaystackPayment({ reference, plan: 'pro', interval })
    } catch (err) {
      console.error('Verify error (non-fatal):', err)
    }

    // Refetch into the shared context — every subscriber updates
    await fetchSubscription()

    window.location.href = '/?upgraded=true'
  }

  async function openPortal() {
    const result = await api.getBillingPortal()
    window.location.href = result.url
  }

  const isPro =
    state.subscription?.plan === 'pro' && state.is_active
  const isTrialing =
    state.subscription?.status === 'trialing'

  return (
    <BillingContext.Provider
      value={{
        ...state,
        isPro,
        isTrialing,
        currency,
        refetch: fetchSubscription,
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