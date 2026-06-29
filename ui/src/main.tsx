import React from 'react'
import ReactDOM from 'react-dom/client'
import { PostHogProvider } from '@posthog/react'
import App from './App'
import { AuthProvider } from './context/AuthContext'
import { BillingProvider } from './context/BillingContext'
import './index.css'

const posthogOptions = {
  api_host:          import.meta.env.VITE_POSTHOG_HOST,
  capture_pageview:  false,
  capture_pageleave: true,
  autocapture:       false,
  session_recording: {
    maskAllInputs: true,
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