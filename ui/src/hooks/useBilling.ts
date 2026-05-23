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

  const fetch = useCallback(async () => {
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

  useEffect(() => { fetch() }, [fetch])

  // Detect region for provider routing
  const currency = detectCurrency()

  async function startCheckout(interval: 'month' | 'year') {
    const result = await api.createCheckout(interval, currency)
    window.location.href = result.redirect_url
  }

  async function openPortal() {
    const result = await api.getBillingPortal()
    window.location.href = result.url
  }

  const isPro = state.subscription?.plan === 'pro' && state.is_active
  const isTrialing = state.subscription?.status === 'trialing'

  return { ...state, isPro, isTrialing, currency, startCheckout, openPortal, refetch: fetch }
}

function detectCurrency(): 'ngn' | 'usd' {
  // Check stored preference first
  const stored = localStorage.getItem('hookdrop_currency')
  if (stored === 'ngn' || stored === 'usd') return stored

  // Browser locale as signal
  const locale = navigator.language || ''
  if (locale.includes('NG') || Intl.DateTimeFormat().resolvedOptions().timeZone?.includes('Lagos')) {
    return 'ngn'
  }
  return 'usd'
}