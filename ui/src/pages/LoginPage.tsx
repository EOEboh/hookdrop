import { useState } from 'react'
import { api } from '../api/client'

interface LoginPageProps {
  errorHint?: string | null
}

export function LoginPage({errorHint}: LoginPageProps) {
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
      setSent(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  if (sent) {
    return (
      <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-6">
        <div className="max-w-sm w-full text-center space-y-4">
          <div className="text-4xl">📬</div>
          <h2 className="text-zinc-100 font-semibold text-lg">Check your email</h2>
          <p className="text-zinc-400 text-sm">
            We sent a login link to <span className="text-zinc-200">{email}</span>.
            It expires in 15 minutes.
          </p>
          <button
            onClick={() => setSent(false)}
            className="text-xs text-zinc-600 hover:text-zinc-400 transition-colors"
          >
            Use a different email
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-zinc-950 flex items-center justify-center p-6">
      <div className="max-w-sm w-full space-y-6">

        <div className="text-center space-y-2">
          <div className="text-3xl">⚡</div>
          <h1 className="text-zinc-100 font-semibold text-xl tracking-tight">hookdrop</h1>
          <p className="text-zinc-500 text-sm">Inspect and replay webhooks in real time</p>
        </div>


        <div className="space-y-3">
          <input
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSubmit()}
            placeholder="you@example.com"
            className="w-full bg-zinc-900 border border-zinc-700 rounded-lg px-4 py-3 text-sm text-zinc-200 placeholder-zinc-600 focus:outline-none focus:border-emerald-500 transition-colors"
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
            className="w-full bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium text-sm py-3 rounded-lg transition-colors"
          >
            {loading ? 'Sending…' : 'Send login link'}
          </button>
        </div>

        {error && (
          <p className="text-xs text-red-400 text-center">{error}</p>
        )}

        <p className="text-xs text-zinc-600 text-center">
          No password. No account setup. Just your email.
        </p>
      </div>
    </div>
  )
}