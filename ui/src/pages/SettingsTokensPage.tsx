import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { CopyButton } from '../components/ui/CopyButton'
import { Spinner } from '../components/ui/Spinner'
import type { APIToken, CreatedAPIToken } from '../types'

function formatDate(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', {
    day: 'numeric', month: 'short', year: 'numeric',
  })
}

export function SettingsTokensPage() {
  const [tokens, setTokens]     = useState<APIToken[] | null>(null)
  const [error, setError]       = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [newToken, setNewToken] = useState<CreatedAPIToken | null>(null)
  const [name, setName]         = useState('')
  const [expiresDays, setExpiresDays] = useState<number>(0)
  const [busy, setBusy]         = useState(false)

  async function refresh() {
    try {
      setTokens(await api.listTokens())
    } catch {
      setError('Failed to load tokens')
    }
  }

  useEffect(() => { refresh() }, [])

  async function handleCreate() {
    if (!name.trim()) return
    setBusy(true)
    setError(null)
    try {
      const created = await api.createToken({
        name: name.trim(),
        ...(expiresDays > 0 ? { expires_in_days: expiresDays } : {}),
      })
      setNewToken(created)
      setCreating(false)
      setName('')
      setExpiresDays(0)
      await refresh()
    } catch {
      setError('Failed to create token')
    } finally {
      setBusy(false)
    }
  }

  async function handleRevoke(id: string) {
    setError(null)
    try {
      await api.revokeToken(id)
      await refresh()
    } catch {
      setError('Failed to revoke token')
    }
  }

  async function handleRevokeAll() {
    if (!window.confirm('Revoke ALL API tokens? Every CLI session using them will stop working.')) return
    setError(null)
    try {
      await api.revokeAllTokens()
      setNewToken(null)
      await refresh()
    } catch {
      setError('Failed to revoke tokens')
    }
  }

  const activeTokens  = tokens?.filter(t => !t.revoked_at) ?? []
  const revokedTokens = tokens?.filter(t => t.revoked_at) ?? []

  return (
    <div className="max-w-3xl mx-auto px-6 py-10 space-y-8">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold">API tokens</h1>
          <p className="text-sm text-muted mt-1">
            Long-lived tokens for the hookdrop CLI. Run <code className="text-xs bg-surface px-1.5 py-0.5 rounded">hookdrop login</code> to
            create one automatically, or create one here and paste it.
          </p>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {activeTokens.length > 0 && (
            <button
              onClick={handleRevokeAll}
              className="text-xs text-red-400 hover:text-red-300 px-3 py-1.5 rounded-md border border-red-500/30 hover:bg-red-500/10 transition-colors duration-200 ease-(--ease-considered)"
            >
              Revoke all
            </button>
          )}
          <button
            onClick={() => { setCreating(true); setNewToken(null) }}
            className="text-xs text-white bg-indigo-600 hover:bg-indigo-500 px-3 py-1.5 rounded-md transition-colors duration-200 ease-(--ease-considered)"
          >
            Create token
          </button>
        </div>
      </div>

      {error && <p className="text-sm text-red-400">{error}</p>}

      {/* One-time token reveal */}
      {newToken && (
        <div className="border border-emerald-500/30 bg-emerald-500/5 rounded-lg p-4 space-y-2">
          <p className="text-sm text-emerald-400 font-medium">
            Token created — copy it now, you won't see it again
          </p>
          <div className="flex items-center gap-2">
            <code className="text-xs bg-surface px-2 py-1.5 rounded font-mono break-all flex-1">
              {newToken.token}
            </code>
            <CopyButton text={newToken.token} />
          </div>
        </div>
      )}

      {/* Create form */}
      {creating && (
        <div className="border border-border rounded-lg p-4 space-y-3">
          <div className="flex flex-col gap-1.5">
            <label className="text-xs text-muted">Name</label>
            <input
              autoFocus
              value={name}
              onChange={e => setName(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleCreate() }}
              placeholder="e.g. CLI on my laptop"
              className="bg-surface border border-border rounded-md px-3 py-2 text-sm outline-none focus:border-indigo-500"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-xs text-muted">Expiry</label>
            <select
              value={expiresDays}
              onChange={e => setExpiresDays(Number(e.target.value))}
              className="bg-surface border border-border rounded-md px-3 py-2 text-sm outline-none focus:border-indigo-500 w-48"
            >
              <option value={0}>Never expires</option>
              <option value={30}>30 days</option>
              <option value={90}>90 days</option>
              <option value={365}>1 year</option>
            </select>
          </div>
          <div className="flex items-center gap-2 pt-1">
            <button
              onClick={handleCreate}
              disabled={busy || !name.trim()}
              className="text-xs text-white bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 px-3 py-1.5 rounded-md transition-colors duration-200 ease-(--ease-considered)"
            >
              {busy ? 'Creating…' : 'Create'}
            </button>
            <button
              onClick={() => setCreating(false)}
              className="text-xs text-muted hover:text-ink px-3 py-1.5 transition-colors duration-200 ease-(--ease-considered)"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Token list */}
      {tokens === null ? (
        <div className="flex justify-center py-10"><Spinner size={5} /></div>
      ) : activeTokens.length === 0 && !creating ? (
        <p className="text-sm text-muted py-6 text-center border border-dashed border-border rounded-lg">
          No active tokens yet.
        </p>
      ) : (
        <div className="border border-border rounded-lg divide-y divide-border">
          {activeTokens.map(t => (
            <div key={t.id} className="flex items-center justify-between gap-4 px-4 py-3">
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{t.name}</p>
                <p className="text-xs text-faint font-mono mt-0.5">{t.token_prefix}…</p>
              </div>
              <div className="flex items-center gap-6 text-xs text-muted shrink-0">
                <span title="Created">{formatDate(t.created_at)}</span>
                <span title="Last used">
                  {t.last_used_at ? `used ${formatDate(t.last_used_at)}` : 'never used'}
                </span>
                <span title="Expires">
                  {t.expires_at ? `expires ${formatDate(t.expires_at)}` : 'no expiry'}
                </span>
                <button
                  onClick={() => handleRevoke(t.id)}
                  className="text-red-400 hover:text-red-300 transition-colors duration-200 ease-(--ease-considered)"
                >
                  Revoke
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {revokedTokens.length > 0 && (
        <details className="text-sm text-muted">
          <summary className="cursor-pointer text-xs">Revoked tokens ({revokedTokens.length})</summary>
          <div className="mt-2 border border-border rounded-lg divide-y divide-border opacity-60">
            {revokedTokens.map(t => (
              <div key={t.id} className="flex items-center justify-between gap-4 px-4 py-2.5">
                <div className="min-w-0">
                  <p className="text-sm truncate line-through">{t.name}</p>
                  <p className="text-xs text-faint font-mono mt-0.5">{t.token_prefix}…</p>
                </div>
                <span className="text-xs shrink-0">revoked {formatDate(t.revoked_at)}</span>
              </div>
            ))}
          </div>
        </details>
      )}
    </div>
  )
}
