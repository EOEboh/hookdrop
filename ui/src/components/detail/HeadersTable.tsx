interface Props {
  headers: Record<string, string>
}

export function HeadersTable({ headers }: Props) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <div className="px-6 py-6 text-center">
        <p className="text-xs text-faint">No headers</p>
      </div>
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <tbody>
          {entries.map(([key, val], i) => (
            <tr
              key={key}
              className={`border-b border-border/50 hover:bg-surface-hover/60 transition-colors duration-200 ease-(--ease-considered) ${
                i % 2 === 1 ? 'bg-surface/40' : ''
              }`}
            >
              <td className="px-6 py-2 font-mono text-muted whitespace-nowrap w-2/5 align-top">
                {key}
              </td>
              <td className="px-6 py-2 font-mono text-ink break-all">
                {val}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
