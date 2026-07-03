interface Props {
  headers: Record<string, string>
}

export function HeadersTable({ headers }: Props) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return (
      <div className="px-6 py-6 text-center">
        <p className="text-xs text-zinc-600">No headers</p>
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
              className={`border-b border-zinc-800/50 hover:bg-zinc-900/50 transition-colors ${
                i % 2 === 1 ? 'bg-zinc-900/20' : ''
              }`}
            >
              <td className="px-6 py-2 font-mono text-zinc-500 whitespace-nowrap w-2/5 align-top">
                {key}
              </td>
              <td className="px-6 py-2 font-mono text-zinc-300 break-all">
                {val}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
