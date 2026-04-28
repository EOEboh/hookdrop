import { safeParseBody, tryPrettyPrint } from '../../lib/json'
import { CopyButton } from '../ui/CopyButton'

export function BodyViewer({ body }: { body: string }) {
  const raw = safeParseBody(body)
  const { pretty, isJson } = tryPrettyPrint(raw)

  if (!raw || raw.trim() === '') {
    return <p className="text-xs text-zinc-600 px-6 py-4">Empty body</p>
  }

  return (
    <div className="relative">
      <div className="absolute top-3 right-4">
        <CopyButton text={pretty} label="Copy body" />
      </div>
      <pre className="px-6 py-4 text-xs font-mono text-zinc-300 overflow-x-auto whitespace-pre-wrap break-all leading-relaxed">
        <code className={isJson ? 'text-emerald-300' : 'text-zinc-300'}>
          {pretty}
        </code>
      </pre>
    </div>
  )
}