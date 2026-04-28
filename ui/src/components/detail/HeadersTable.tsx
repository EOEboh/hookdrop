interface Props {
  headers: Record<string, string>
}

export function HeadersTable({ headers }: Props) {
  const entries = Object.entries(headers)

  if (entries.length === 0) {
    return <p className="text-xs text-zinc-600 px-6 py-4">No headers</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs font-mono">
        <tbody>
          {entries.map(([key, val]) => (
            <tr key={key} className="border-b border-zinc-800/50 hover:bg-zinc-800/20">
              <td className="px-6 py-2 text-zinc-400 whitespace-nowrap w-1/3">{key}</td>
              <td className="px-6 py-2 text-zinc-300 break-all">{val}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}