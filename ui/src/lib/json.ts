export function tryPrettyPrint(raw: string): { pretty: string; isJson: boolean } {
  try {
    const parsed = JSON.parse(raw)
    return { pretty: JSON.stringify(parsed, null, 2), isJson: true }
  } catch {
    return { pretty: raw, isJson: false }
  }
}

export function safeParseBody(body: string): string {
  try {
    const decoded = atob(body)
    return decoded
  } catch {
    return body
  }
}

// ─── JSON syntax tokenizer ───────────────────────────────────────────────────

export type JsonTokenType = 'key' | 'string' | 'number' | 'boolean' | 'null' | 'punctuation' | 'other'

export interface JsonToken {
  type: JsonTokenType
  value: string
}

/**
 * Tokenizes a pretty-printed JSON string into typed segments for syntax
 * highlighting. Uses 7 capture groups:
 *   1. key    — string immediately followed by ":"  (lookahead, colon not consumed)
 *   2. string — any other quoted string
 *   3. boolean — true | false
 *   4. null
 *   5. number
 *   6. punctuation — { } [ ] , :
 *   7. (fallback other)
 */
export function tokenizeJson(json: string): JsonToken[] {
  const TOKEN_RE =
    /("(?:[^"\\]|\\.)*"(?=\s*:))|("(?:[^"\\]|\\.)*")|(true|false)|(null)|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)|([{}[\],:])/g

  const tokens: JsonToken[] = []
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = TOKEN_RE.exec(json)) !== null) {
    // Preserve whitespace / unmatched text as-is
    if (match.index > lastIndex) {
      tokens.push({ type: 'other', value: json.slice(lastIndex, match.index) })
    }

    const [full, key, str, bool, nul, num, punct] = match

    if      (key   !== undefined) tokens.push({ type: 'key',         value: full })
    else if (str   !== undefined) tokens.push({ type: 'string',      value: full })
    else if (bool  !== undefined) tokens.push({ type: 'boolean',     value: full })
    else if (nul   !== undefined) tokens.push({ type: 'null',        value: full })
    else if (num   !== undefined) tokens.push({ type: 'number',      value: full })
    else if (punct !== undefined) tokens.push({ type: 'punctuation', value: full })
    else                          tokens.push({ type: 'other',       value: full })

    lastIndex = match.index + full.length
  }

  if (lastIndex < json.length) {
    tokens.push({ type: 'other', value: json.slice(lastIndex) })
  }

  return tokens
}
