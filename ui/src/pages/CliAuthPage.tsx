import { useState } from 'react'
import { useAuth } from '../context/AuthContext'
import { api } from '../api/client'
import { Logo } from '../components/ui/Logo'

const CLI_AUTH_KEY = 'hookdrop_cli_auth'

interface CliAuthParams {
  port: number
  state: string
}

// Reads port+state from the URL, falling back to sessionStorage (set before
// a login round-trip through the magic-link flow).
export function getCliAuthParams(): CliAuthParams | null {
  const qs = new URLSearchParams(window.location.search)
  let port = Number(qs.get('port'))
  let state = qs.get('state') ?? ''

  if (!port || !state) {
    try {
      const saved = JSON.parse(sessionStorage.getItem(CLI_AUTH_KEY) ?? 'null')
      if (saved) { port = Number(saved.port); state = String(saved.state) }
    } catch { /* ignore */ }
  }

  if (!Number.isInteger(port) || port < 1 || port > 65535 || !state) return null
  return { port, state }
}

export function stashCliAuthParams(): void {
  const qs = new URLSearchParams(window.location.search)
  const port = qs.get('port')
  const state = qs.get('state')
  if (port && state) {
    sessionStorage.setItem(CLI_AUTH_KEY, JSON.stringify({ port, state }))
  }
}

export function hasPendingCliAuth(): boolean {
  return sessionStorage.getItem(CLI_AUTH_KEY) !== null
}

export function CliAuthPage() {
  const { user } = useAuth()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const params = getCliAuthParams()

  if (!params) {
    return (
      <Shell>
        <p className="text-sm text-red-400">
          Invalid CLI authorization link. Run <code className="text-xs bg-surface px-1.5 py-0.5 rounded">hookdrop login</code> again.
        </p>
      </Shell>
    )
  }

  async function handleAuthorize() {
    if (!params) return
    setBusy(true)
    setError(null)
    try {
      const hostname = window.navigator.platform || 'device'
      const created = await api.createToken({
        name: `CLI (browser login, ${hostname})`,
      })
      sessionStorage.removeItem(CLI_AUTH_KEY)
      // Top-level navigation to the loopback listener the CLI runs —
      // navigation (unlike fetch) is exempt from mixed-content blocking.
      window.location.href =
        `http://127.0.0.1:${params.port}/callback` +
        `?token=${encodeURIComponent(created.token)}&state=${encodeURIComponent(params.state)}`
    } catch {
      setError('Failed to create a token. Try again, or create one manually under Settings → API tokens.')
      setBusy(false)
    }
  }

  function handleCancel() {
    sessionStorage.removeItem(CLI_AUTH_KEY)
    window.location.href = '/'
  }

  return (
    <Shell>
      <div className="space-y-5 text-center">
        <h1 className="text-lg font-semibold">Authorize hookdrop CLI</h1>
        <p className="text-sm text-muted">
          The hookdrop CLI on your machine is asking for an API token for{' '}
          <span className="text-ink font-medium">{user?.email}</span>. The token lets it
          stream and forward your webhooks until you revoke it.
        </p>
        {error && <p className="text-sm text-red-400">{error}</p>}
        <div className="flex items-center justify-center gap-3">
          <button
            onClick={handleAuthorize}
            disabled={busy}
            className="text-sm text-white bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 px-5 py-2 rounded-md transition-colors duration-200 ease-(--ease-considered)"
          >
            {busy ? 'Authorizing…' : 'Authorize'}
          </button>
          <button
            onClick={handleCancel}
            className="text-sm text-muted hover:text-ink px-4 py-2 transition-colors duration-200 ease-(--ease-considered)"
          >
            Cancel
          </button>
        </div>
        <p className="text-xs text-faint">
          Only continue if you just ran <code className="bg-surface px-1 py-0.5 rounded">hookdrop login</code> in your terminal.
        </p>
      </div>
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-base text-ink flex flex-col items-center justify-center px-6">
      <div className="w-full max-w-md space-y-8">
        <div className="flex justify-center"><Logo size="sm" /></div>
        <div className="border border-border rounded-lg p-6 bg-surface/50">
          {children}
        </div>
      </div>
    </div>
  )
}
