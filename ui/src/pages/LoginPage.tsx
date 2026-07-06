import { useState } from 'react'
import { api } from '../api/client'
import { usePostHog } from '@posthog/react'
import { CheckCircleIcon } from '../components/ui/icons'
import { LogoMark } from '../components/ui/Logo'

interface LoginPageProps {
  errorHint?: string | null
}

export function LoginPage({errorHint}: LoginPageProps) {
  const posthog = usePostHog()

  const [email, setEmail]   = useState('')
  const [sent, setSent]     = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError]   = useState<string | null>(null)

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

  if (sent) {
    return (
      <div className="min-h-screen bg-base flex items-center justify-center p-6">
        <div className="max-w-sm w-full text-center space-y-4">
          <div className="w-12 h-12 rounded-xl bg-surface border border-border flex items-center justify-center mx-auto">
            <CheckCircleIcon className="w-5 h-5 text-indigo-400" />
          </div>
          <h2 className="text-ink font-semibold text-lg">Check your email</h2>
          <p className="text-muted text-sm">
            We sent a login link to <span className="text-ink">{email}</span>.
            It expires in 15 minutes.
          </p>
          <button
            onClick={() => setSent(false)}
            className="text-xs text-faint hover:text-muted transition-colors duration-200 ease-(--ease-considered)"
          >
            Use a different email
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-base flex items-center justify-center p-6">
      <div className="max-w-sm w-full space-y-6">

        <div className="text-center space-y-2">
          <LogoMark size="lg" className="mx-auto" />
          <h1 className="text-ink font-semibold text-xl tracking-tight">hookdrop</h1>
          <p className="text-muted text-sm">Inspect and replay webhooks in real time</p>
        </div>


        <div className="space-y-3">
          <input
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSubmit()}
            placeholder="you@example.com"
            className="w-full bg-surface border border-border-strong rounded-lg px-4 py-3 text-sm text-ink placeholder-faint focus:outline-none focus:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)"
            autoFocus
          />
            {errorHint === 'invalid_link' && (
              <p className="text-xs text-amber-400 text-center bg-amber-500/10 px-3 py-2 rounded-lg">
                That link was invalid or already used. Request a new one below.
              </p>
            )}
          <button
            onClick={handleSubmit}
            disabled={loading || !email}
            className="w-full bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium text-sm py-3 rounded-lg transition-all duration-200 ease-(--ease-considered)"
          >
            {loading ? 'Sending…' : 'Send login link'}
          </button>
        </div>

        {error && (
          <p className="text-xs text-red-400 text-center">{error}</p>
        )}

        <p className="text-xs text-faint text-center">
          No password. No account setup. Just your email.
        </p>
      </div>
    </div>
  )
}
