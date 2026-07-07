import type { Endpoint } from '../../types'
import { CopyButton } from '../ui/CopyButton'
import { SecretManager } from './SecretManager'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export function EndpointDetail({ endpoint }: { endpoint: Endpoint }) {
  const url = `${BASE_URL}/i/${endpoint.slug}`

  return (
    <div className="flex flex-col h-full overflow-y-auto">
      <div className="px-6 py-5 border-b border-border">
        <p className="text-sm font-medium text-ink">{endpoint.name}</p>
        {endpoint.description && (
          <p className="text-xs text-muted mt-1">{endpoint.description}</p>
        )}
        <div className="mt-3 flex items-center gap-2">
          <code className="flex-1 text-xs font-mono text-indigo-400 bg-surface border border-border rounded-lg px-3 py-2 truncate">
            {url}
          </code>
          <CopyButton text={url} />
        </div>
      </div>

      <div className="px-6 py-2.5 bg-surface/30 border-y border-border/60 flex items-center">
        <span className="text-[11px] font-semibold text-muted uppercase tracking-widest">
          Verification
        </span>
      </div>
      <div className="px-6 py-4">
        <SecretManager endpointId={endpoint.id} />
      </div>
    </div>
  )
}
