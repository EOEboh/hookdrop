import { useEffect, useState } from 'react'
import { api } from '../../api/client'
import type { WebhookSecret } from '../../types'
import { usePostHog } from '@posthog/react'

const PROVIDERS = [
  { value: 'stripe',   label: 'Stripe',   hint: 'Your webhook signing secret (whsec_...)' },
  { value: 'paystack', label: 'Paystack', hint: 'Your Paystack secret key (sk_live_... or sk_test_...)' },
  { value: 'github',   label: 'GitHub',   hint: 'Your webhook secret from GitHub settings' },
  { value: 'generic',  label: 'Generic',  hint: 'HMAC-SHA256 secret for any other provider' },
]

export function SecretManager({ endpointId }: { endpointId: string }) {
  const posthog = usePostHog()

  const [secrets, setSecrets]     = useState<WebhookSecret[]>([])
  const [provider, setProvider]   = useState('stripe')
  const [secret, setSecret]       = useState('')
  const [loading, setLoading]     = useState(false)
  const [saving, setSaving]       = useState(false)
  const [error, setError]         = useState<string | null>(null)
  const [showInput, setShowInput] = useState(false)

  useEffect(() => {
    setLoading(true)
    api.getSecrets(endpointId)
      .then(data => setSecrets(data ?? []))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false))
  }, [endpointId])

  async function handleSave() {
    if (!secret.trim()) return
    setSaving(true)
    setError(null)
    try {
      const ws = await api.saveSecret(endpointId, { provider, secret: secret.trim() })
       posthog?.capture('webhook_secret_added', {        
        provider,
      })
      setSecrets(prev => {
        // Replace if same provider already exists
        const filtered = prev.filter(s => s.provider !== provider)
        return [ws, ...filtered]
      })
      setSecret('')
      setShowInput(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save secret')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: string) {
    try {
      await api.deleteSecret(endpointId, id)
      setSecrets(prev => prev.filter(s => s.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete secret')
    }
  }

  const selectedProvider = PROVIDERS.find(p => p.value === provider)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xs font-semibold text-muted uppercase tracking-wider">
          Signature verification
        </h3>
        {!showInput && (
          <button
            onClick={() => setShowInput(true)}
            className="text-xs text-indigo-400 hover:text-indigo-300 transition-colors duration-200 ease-(--ease-considered)"
          >
            + Add secret
          </button>
        )}
      </div>

      {/* Existing secrets */}
      {!loading && secrets.length > 0 && (
        <div className="space-y-2">
          {secrets.map(s => (
            <div
              key={s.id}
              className="flex items-center justify-between px-3 py-2 bg-surface rounded-lg"
            >
              <div className="flex items-center gap-2">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0" />
                <span className="text-xs font-medium text-ink capitalize">{s.provider}</span>
                <span className="text-xs text-faint">secret configured</span>
              </div>
              <button
                onClick={() => handleDelete(s.id)}
                className="text-xs text-faint hover:text-red-400 transition-colors duration-200 ease-(--ease-considered)"
              >
                Remove
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add secret form */}
      {showInput && (
        <div className="space-y-3 p-3 bg-surface rounded-lg border border-border animate-fade-in">

          {/* Provider picker */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted">Provider</label>
            <div className="grid grid-cols-2 gap-1.5">
              {PROVIDERS.map(p => (
                <button
                  key={p.value}
                  onClick={() => setProvider(p.value)}
                  className={`px-3 py-1.5 rounded text-xs font-medium transition-colors duration-200 ease-(--ease-considered) ${
                    provider === p.value
                      ? 'bg-indigo-500 text-white'
                      : 'bg-surface-hover text-muted hover:text-ink'
                  }`}
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          {/* Secret input */}
          <div className="space-y-1.5">
            <label className="text-xs text-muted">Secret</label>
            <input
              type="password"
              value={secret}
              onChange={e => setSecret(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && handleSave()}
              placeholder={selectedProvider?.hint}
              className="w-full bg-base border border-border-strong rounded-lg px-3 py-2 text-sm font-mono text-ink placeholder-faint focus:outline-none focus:border-indigo-500 transition-colors duration-200 ease-(--ease-considered)"
            />
            <p className="text-xs text-faint">{selectedProvider?.hint}</p>
          </div>

          {error && (
            <p className="text-xs text-red-400">{error}</p>
          )}

          <div className="flex gap-2">
            <button
              onClick={() => { setShowInput(false); setSecret(''); setError(null) }}
              className="flex-1 px-3 py-1.5 rounded border border-border-strong text-xs text-muted hover:text-ink transition-colors duration-200 ease-(--ease-considered)"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={saving || !secret.trim()}
              className="flex-1 px-3 py-1.5 rounded bg-indigo-500 hover:bg-indigo-400 active:scale-[0.98] disabled:opacity-50 text-white text-xs font-medium transition-all duration-200 ease-(--ease-considered)"
            >
              {saving ? 'Saving…' : 'Save secret'}
            </button>
          </div>
        </div>
      )}

      {!loading && secrets.length === 0 && !showInput && (
        <p className="text-xs text-faint px-1">
          Add a secret to verify incoming webhook signatures automatically.
        </p>
      )}
    </div>
  )
}
