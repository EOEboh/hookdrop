export function tryPrettyPrint(raw: string): { pretty: string; isJson: boolean } {
  try {
    const parsed = JSON.parse(raw)
    return { pretty: JSON.stringify(parsed, null, 2), isJson: true }
  } catch {
    return { pretty: raw, isJson: false }
  }
}

export function safeParseBody(body: string): string {
  // Go sends body as base64 for binary, raw string for text
  try {
    const decoded = atob(body)
    return decoded
  } catch {
    return body
  }
}