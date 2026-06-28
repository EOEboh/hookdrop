import React from 'react'
import ReactDOM from 'react-dom/client'
import { PostHogProvider } from '@posthog/react'
import App from './App'
import { AuthProvider } from './context/AuthContext'
import { BillingProvider } from './context/BillingContext'
import './index.css'

const posthogOptions = {
  api_host:          import.meta.env.VITE_POSTHOG_HOST,
  defaults:          '2026-01-30',  // PostHog recommended SDK defaults
  capture_pageview:  false,         // we handle this manually — SPA has no real page loads
  capture_pageleave: true,
  autocapture:       false,         // intentional tracking only — keeps event volume lean
  session_recording: {
    maskAllInputs: true,            // never record passwords, secrets, or API keys
    maskInputOptions: {
      password: true,
      email:    true,
    },
  },
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <PostHogProvider
      apiKey={import.meta.env.VITE_POSTHOG_KEY}
      options={posthogOptions}
    >
      <AuthProvider>
        <BillingProvider>
          <App />
        </BillingProvider>
      </AuthProvider>
    </PostHogProvider>
  </React.StrictMode>
)