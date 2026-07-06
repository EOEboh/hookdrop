import { safeParseBody, tryPrettyPrint, tokenizeJson, type JsonTokenType } from '../../lib/json'
import { CopyButton } from '../ui/CopyButton'

// Understated syntax colours tuned to the warm indigo/violet palette —
// distinct roles, not a generic green-on-black terminal.
const TOKEN_CLASS: Record<JsonTokenType, string> = {
  key:         'text-indigo-300',
  string:      'text-emerald-300',
  number:      'text-amber-300',
  boolean:     'text-sky-300',
  null:        'text-faint',
  punctuation: 'text-muted',
  other:       'text-muted',
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
        <p className="text-xs text-faint">Empty body</p>
      </div>
    )
  }

  return (
    <div className="relative group">
      {/* Copy button — appears on hover, tucked in corner */}
      <div className="absolute top-3 right-4 opacity-0 group-hover:opacity-100 transition-opacity duration-200 ease-(--ease-considered) z-10">
        <CopyButton text={pretty} label="Copy" />
      </div>

      <pre className="px-6 py-4 text-xs font-mono overflow-x-auto whitespace-pre-wrap break-all leading-relaxed">
        {isJson
          ? <JsonHighlight json={pretty} />
          : <span className="text-ink">{pretty}</span>
        }
      </pre>
    </div>
  )
}
