import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { BillingState } from '../types'

export function useBilling() {
  const [state, setState] = useState<BillingState>({
    subscription: null,
    limits: null,
    is_active: true,
    loading: true,
  })

  const fetchSubscription = useCallback(async () => {
    try {
      const data = await api.getSubscription()
      setState({
        subscription: data.subscription,
        limits: data.limits,
        is_active: data.is_active,
        loading: false,
      })
    } catch {
      setState(s => ({ ...s, loading: false }))
    }
  }, [])

  useEffect(() => { fetchSubscription() }, [fetchSubscription])

  const currency = detectCurrency()

  // For LemonSqueezy (international): redirect flow unchanged
  async function startCheckout(interval: 'month' | 'year') {
    const result = await api.createCheckout(interval, currency)
    window.location.href = result.redirect_url
  }

  // For Paystack (NGN): returns config for react-paystack hook
  function getPaystackConfig(interval: 'month' | 'year', email: string) {
    const planCode = interval === 'year'
      ? import.meta.env.VITE_PAYSTACK_PLAN_PRO_ANNUAL
      : import.meta.env.VITE_PAYSTACK_PLAN_PRO_MONTHLY

      const amount = interval === 'year' ? 3360000 : 350000

    return {
      publicKey:  import.meta.env.VITE_PAYSTACK_PUBLIC_KEY ?? '',
      email,
      amount,
      plan:       planCode,
      label:      'hookdrop Pro',
    }
  }

 async function handlePaystackSuccess(
  reference: string,
  interval: 'month' | 'year',
) {
  await new Promise(resolve => setTimeout(resolve, 1500))

  try {
    await api.verifyPaystackPayment({ reference, plan: 'pro', interval })
    await fetchSubscription()
  } catch (err) {
    console.error('Verify error (non-fatal):', err)
    await fetchSubscription()
  }


  window.location.href = '/?upgraded=true'
}

  async function openPortal() {
    const result = await api.getBillingPortal()
    window.location.href = result.url
  }

  const isPro      = state.subscription?.plan === 'pro' && state.is_active
  const isTrialing = state.subscription?.status === 'trialing'

  return {
    ...state,
    isPro,
    isTrialing,
    currency,
    startCheckout,
    getPaystackConfig,
    handlePaystackSuccess,
    openPortal,
    refetch: fetchSubscription,
  }
}

function detectCurrency(): 'ngn' | 'usd' {
  const stored = localStorage.getItem('hookdrop_currency')
  if (stored === 'ngn' || stored === 'usd') return stored

  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone ?? ''
  const locale = navigator.language ?? ''
  if (tz.includes('Lagos') || tz.includes('Africa') || locale.includes('NG')) {
    return 'ngn'
  }
  return 'usd'
}