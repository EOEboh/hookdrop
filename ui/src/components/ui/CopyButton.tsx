import { useState } from 'react'
import { ClipboardIcon, CheckIcon } from './icons'

interface Props {
  text: string
  label?: string
  iconOnly?: boolean
}

export function CopyButton({ text, label = 'Copy', iconOnly = false }: Props) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  if (iconOnly) {
    return (
      <button
        onClick={copy}
        title={copied ? 'Copied!' : label}
        className={`p-1.5 rounded-md transition-all ${
          copied
            ? 'text-emerald-400 bg-emerald-500/10'
            : 'text-zinc-500 hover:text-zinc-300 hover:bg-zinc-800'
        }`}
      >
        {copied
          ? <CheckIcon className="w-3.5 h-3.5" />
          : <ClipboardIcon className="w-3.5 h-3.5" />
        }
      </button>
    )
  }

  return (
    <button
      onClick={copy}
      className={`flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-md transition-all ${
        copied
          ? 'text-emerald-400 bg-emerald-500/10'
          : 'text-zinc-400 bg-zinc-800 hover:bg-zinc-700 hover:text-zinc-200'
      }`}
    >
      {copied ? (
        <>
          <CheckIcon className="w-3 h-3" />
          Copied
        </>
      ) : (
        <>
          <ClipboardIcon className="w-3 h-3" />
          {label}
        </>
      )}
    </button>
  )
}
