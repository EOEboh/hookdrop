import { safeParseBody, tryPrettyPrint, tokenizeJson, type JsonTokenType } from '../../lib/json'
import { CopyButton } from '../ui/CopyButton'

// Understated syntax colours — readable, not garish
const TOKEN_CLASS: Record<JsonTokenType, string> = {
  key:         'text-sky-300',
  string:      'text-emerald-300',
  number:      'text-amber-300',
  boolean:     'text-orange-300',
  null:        'text-zinc-500',
  punctuation: 'text-zinc-600',
  other:       'text-zinc-400',
}

function JsonHighlight({ json }: { json: string }) {
  const tokens = tokenizeJson(json)
  return (
    <>
      {tokens.map((token, i) => (
        <span key={i} className={TOKEN_CLASS[token.type]}>
          {token.value}
        </span>
      ))}
    </>
  )
}

export function BodyViewer({ body }: { body: string }) {
  const raw = safeParseBody(body)
  const { pretty, isJson } = tryPrettyPrint(raw)

  if (!raw || raw.trim() === '') {
    return (
      <div className="px-6 py-6 text-center">
        <p className="text-xs text-zinc-600">Empty body</p>
      </div>
    )
  }

  return (
    <div className="relative group">
      {/* Copy button — appears on hover, tucked in corner */}
      <div className="absolute top-3 right-4 opacity-0 group-hover:opacity-100 transition-opacity z-10">
        <CopyButton text={pretty} label="Copy" />
      </div>

      <pre className="px-6 py-4 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all leading-relaxed">
        {isJson
          ? <JsonHighlight json={pretty} />
          : <span className="text-zinc-300">{pretty}</span>
        }
      </pre>
    </div>
  )
}
