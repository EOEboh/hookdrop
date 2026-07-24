import { useState, type ReactNode } from 'react'
import { api } from '../api/client'
import { usePostHog } from '@posthog/react'
import { Logo, LogoMark } from '../components/ui/Logo'
import { MethodBadge } from '../components/ui/MethodBadge'
import { VerificationBadge } from '../components/ui/VerificationBadge'
import { StatusDot } from '../components/ui/StatusDot'
import {
  ClockIcon,
  CheckCircleIcon,
  ListIcon,
  RefreshCwIcon,
} from '../components/ui/icons'

interface LandingPageProps {
  errorHint?: string | null
}

const FEATURES: { icon: (p: { className?: string }) => ReactNode; title: string; description: string }[] = [
  {
    icon: ClockIcon,
    title: 'Instant capture',
    description: 'The moment a webhook hits your URL, it shows up in the feed. No refresh, no polling.',
  },
  {
    icon: CheckCircleIcon,
    title: 'Signature verification',
    description: "See whether Stripe, Paystack, or GitHub's signature actually checks out, at a glance.",
  },
  {
    icon: ListIcon,
    title: 'Full request detail',
    description: 'Inspect headers, body, and metadata for every request the moment it lands.',
  },
  {
    icon: RefreshCwIcon,
    title: 'Replay anywhere',
    description: 'Replay a captured request to your local server as many times as you need, with no re-triggering from the provider.',
  },
]

const STEPS = [
  {
    title: 'Get your inbox URL',
    description: 'Sign in with just your email, and a unique webhook URL is ready instantly.',
  },
  {
    title: 'Point your provider at it',
    description: "Stripe, Paystack, GitHub, Shopify: anything that sends webhooks.",
  },
  {
    title: 'Watch requests land live',
    description: 'Inspect, verify, and replay every request the moment it arrives.',
  },
]

export function LandingPage({ errorHint }: LandingPageProps) {
  const posthog = usePostHog()

  const [email, setEmail]     = useState('')
  const [sent, setSent]       = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError]     = useState<string | null>(null)

  async function handleSubmit() {
    if (!email) return
    setLoading(true)
    setError(null)
    try {
      await api.requestMagicLink(email)
      posthog?.capture('magic_link_requested')
      setSent(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  function scrollToForm() {
    const el = document.getElementById('email-input')
    el?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    el?.focus()
  }

  return (
    <div className="min-h-screen bg-base text-ink">

      {/* Header */}
      <header className="sticky top-0 z-10 border-b border-border bg-base/80 backdrop-blur">
        <div className="max-w-5xl mx-auto px-6 h-16 flex items-center justify-between">
          <Logo size="sm" />
          <nav className="flex items-center gap-6">
            <a href="#features" className="hidden sm:block text-sm text-muted hover:text-ink transition-colors duration-200 ease-(--ease-considered)">
              Features
            </a>
            <a href="#how-it-works" className="hidden sm:block text-sm text-muted hover:text-ink transition-colors duration-200 ease-(--ease-considered)">
              How it works
            </a>
            <button
              onClick={scrollToForm}
              className="text-sm bg-surface hover:bg-surface-hover border border-border-strong text-ink px-4 py-2 rounded-lg transition-colors duration-200 ease-(--ease-considered)"
            >
              Sign in
            </button>
          </nav>
        </div>
      </header>

      {/* Hero */}
      <section className="relative overflow-hidden">
        {/* Decorative brand glow */}
        <div className="pointer-events-none absolute inset-0 overflow-hidden">
          <div className="absolute -top-32 left-1/2 -translate-x-1/2 w-[36rem] h-[36rem] rounded-full bg-brand/10 blur-3xl" />
          <div className="absolute top-24 left-1/2 translate-x-8 w-[24rem] h-[24rem] rounded-full bg-brand-2/10 blur-3xl" />
        </div>

        <div className="relative max-w-5xl mx-auto px-6 pt-16 pb-20 lg:pt-24 lg:pb-28 grid lg:grid-cols-2 gap-14 items-center">
          <div className="space-y-6 text-center lg:text-left">
            <span className="inline-flex items-center gap-1.5 text-xs font-medium bg-indigo-500/10 text-indigo-300 ring-1 ring-indigo-500/20 px-3 py-1 rounded-full">
              <LogoMark size="sm" className="w-3 h-3.5" /> Now open
            </span>

            <h1 className="text-3xl sm:text-4xl lg:text-5xl font-semibold tracking-tight leading-[1.1]">
              Inspect and replay webhooks in real time
            </h1>

            <p className="text-muted text-base lg:text-lg leading-relaxed max-w-md mx-auto lg:mx-0">
              A drop-in inbox for every webhook. Capture, verify, and replay
              requests instantly. No setup, no password, just your email.
            </p>

            {/* Signup form */}
            <div className="max-w-sm mx-auto lg:mx-0">
              {sent ? (
                <div className="flex items-start gap-3 bg-surface border border-border rounded-lg px-4 py-3.5 text-left">
                  <CheckCircleIcon className="w-5 h-5 text-indigo-400 shrink-0 mt-0.5" />
                  <div className="space-y-0.5">
                    <p className="text-sm font-medium text-ink">Check your email</p>
                    <p className="text-xs text-muted">
                      We sent a login link to <span className="text-ink">{email}</span>. It expires in 15 minutes.
                    </p>
                  </div>
                </div>
              ) : (
                <div className="space-y-3">
                  <div className="flex flex-col sm:flex-row gap-2">
                    <input
                      id="email-input"
                      type="email"
                      value={email}
                      onChange={e => setEmail(e.target.value)}
                      onKeyDown={e => e.key === 'Enter' && handleSubmit()}
                      placeholder="you@example.com"
                      className="flex-1 bg-surface border border-border-strong rounded-lg px-4 py-3 text-sm text-ink placeholder-faint focus:outline-none focus:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)"
                    />
                    <button
                      onClick={handleSubmit}
                      disabled={loading || !email}
                      className="shrink-0 bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium text-sm px-5 py-3 rounded-lg transition-all duration-200 ease-(--ease-considered)"
                    >
                      {loading ? 'Sending…' : 'Get started'}
                    </button>
                  </div>

                  {errorHint === 'invalid_link' && (
                    <p className="text-xs text-amber-400 bg-amber-500/10 px-3 py-2 rounded-lg">
                      That link was invalid or already used. Request a new one above.
                    </p>
                  )}
                  {error && <p className="text-xs text-red-400">{error}</p>}

                  <p className="text-xs text-faint">
                    No password. No credit card. Just your email.
                  </p>
                </div>
              )}
            </div>
          </div>

          {/* Mock live feed panel, built from the real product components */}
          <div className="relative">
            <div className="rounded-xl border border-border bg-surface/60 shadow-[0_0_60px_-20px_rgba(45,212,191,0.35)] overflow-hidden">
              <div className="flex items-center justify-between px-4 py-3 border-b border-border">
                <span className="text-xs font-mono text-muted truncate">hookdrop.app/i/a1b2c3d4</span>
                <StatusDot status="live" />
              </div>
              <div className="divide-y divide-border/60">
                <MockRequestRow method="POST" path="/i/a1b2c3d4" time="just now" verification="verified" provider="Stripe" isNew />
                <MockRequestRow method="POST" path="/i/a1b2c3d4" time="4s ago" verification="verified" provider="Paystack" />
                <MockRequestRow method="GET" path="/i/a1b2c3d4" time="19s ago" verification="unverified" />
                <MockRequestRow method="POST" path="/i/a1b2c3d4" time="41s ago" verification="failed" provider="GitHub" />
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Demo video */}
      <section className="max-w-5xl mx-auto px-6 py-16 lg:py-20">
        <div className="text-center space-y-2 mb-8">
          <h2 className="text-2xl font-semibold tracking-tight">See it in action</h2>
          <p className="text-muted text-sm">A two-minute walkthrough of capturing, inspecting, and replaying a webhook.</p>
        </div>

        <div className="relative aspect-video max-w-3xl mx-auto rounded-xl border border-border-strong bg-surface overflow-hidden">
          <iframe
            src="https://www.loom.com/embed/9f7e293df78049a79ed1f27f0572a7c8"
            title="hookdrop demo"
            allow="fullscreen"
            allowFullScreen
            className="absolute inset-0 w-full h-full"
          />
        </div>
      </section>

      {/* Features */}
      <section id="features" className="max-w-5xl mx-auto px-6 py-16 lg:py-20 border-t border-border">
        <div className="text-center space-y-2 mb-12">
          <h2 className="text-2xl font-semibold tracking-tight">Everything you need to debug webhooks</h2>
          <p className="text-muted text-sm">No tunneling tools, no stale ngrok URLs, no guesswork.</p>
        </div>

        <div className="grid sm:grid-cols-2 gap-6">
          {FEATURES.map(({ icon: Icon, title, description }) => (
            <div key={title} className="rounded-xl border border-border bg-surface/50 p-6 space-y-3">
              <div className="w-9 h-9 rounded-lg bg-indigo-500/10 flex items-center justify-center">
                <Icon className="w-4.5 h-4.5 text-indigo-400" />
              </div>
              <h3 className="text-sm font-semibold text-ink">{title}</h3>
              <p className="text-sm text-muted leading-relaxed">{description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* How it works */}
      <section id="how-it-works" className="max-w-5xl mx-auto px-6 py-16 lg:py-20 border-t border-border">
        <div className="text-center space-y-2 mb-12">
          <h2 className="text-2xl font-semibold tracking-tight">Up and running in under a minute</h2>
        </div>

        <div className="grid sm:grid-cols-3 gap-8">
          {STEPS.map((step, i) => (
            <div key={step.title} className="space-y-3 text-center sm:text-left">
              <span className="inline-flex items-center justify-center w-8 h-8 rounded-full bg-indigo-500/10 text-indigo-400 text-sm font-semibold ring-1 ring-indigo-500/20">
                {i + 1}
              </span>
              <h3 className="text-sm font-semibold text-ink">{step.title}</h3>
              <p className="text-sm text-muted leading-relaxed">{step.description}</p>
            </div>
          ))}
        </div>
      </section>

      {/* CTA */}
      <section className="border-t border-border">
        <div className="max-w-3xl mx-auto px-6 py-16 lg:py-20 text-center space-y-6">
          <h2 className="text-2xl lg:text-3xl font-semibold tracking-tight">Ready to see your webhooks land?</h2>
          <p className="text-muted text-sm">Free to start. Upgrade only when you need permanent endpoints.</p>
          <button
            onClick={scrollToForm}
            className="inline-flex bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] text-white font-medium text-sm px-6 py-3 rounded-lg transition-all duration-200 ease-(--ease-considered)"
          >
            Get started free
          </button>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-border">
        <div className="max-w-5xl mx-auto px-6 py-10 flex flex-col sm:flex-row items-center justify-between gap-4">
          <Logo size="sm" />
          <div className="flex items-center gap-4 text-xs text-faint">
            <span>© {new Date().getFullYear()} hookdrop</span>
            <span>·</span>
            <a href="/privacy" className="hover:text-muted transition-colors duration-200 ease-(--ease-considered)">Privacy</a>
            <span>·</span>
            <a href="/terms" className="hover:text-muted transition-colors duration-200 ease-(--ease-considered)">Terms</a>
            <span>·</span>
            <a href="mailto:support@hookdrop.app" className="hover:text-muted transition-colors duration-200 ease-(--ease-considered)">Support</a>
          </div>
        </div>
      </footer>
    </div>
  )
}

function MockRequestRow({
  method, path, time, verification, provider, isNew = false,
}: {
  method: string
  path: string
  time: string
  verification: 'verified' | 'failed' | 'unverified'
  provider?: string
  isNew?: boolean
}) {
  return (
    <div className={`px-4 py-3 flex items-start gap-3 ${isNew ? 'animate-arrive' : ''}`}>
      <div className="pt-0.5 shrink-0">
        <MethodBadge method={method} size="sm" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs font-mono text-muted truncate leading-tight">{path}</p>
        <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
          <span className="text-[11px] text-faint tabular-nums">{time}</span>
          {verification !== 'unverified' && (
            <>
              <span className="text-border-strong select-none text-[10px]">·</span>
              <VerificationBadge status={verification} provider={provider} />
            </>
          )}
        </div>
      </div>
    </div>
  )
}
