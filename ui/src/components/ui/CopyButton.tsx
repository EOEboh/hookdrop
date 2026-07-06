import { useState } from 'react'
import { ClipboardIcon, CheckIcon } from './icons'

interface Props {
  text: string
  label?: string
  iconOnly?: boolean
  onCopy?: () => void
}

export function CopyButton({ text, label = 'Copy', iconOnly = false, onCopy }: Props) {
  const [copied, setCopied] = useState(false)

  function copy() {
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true)
      if (onCopy) {
        onCopy()
      }
      setTimeout(() => setCopied(false), 2000)
    })
  }

  if (iconOnly) {
    return (
      <button
        onClick={copy}
        title={copied ? 'Copied!' : label}
        className={`p-1.5 rounded-md transition-all duration-200 ease-(--ease-considered) active:scale-90 ${
          copied
            ? 'text-emerald-400 bg-emerald-500/10 scale-105'
            : 'text-muted hover:text-ink hover:bg-surface-hover'
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
      className={`flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-md transition-all duration-200 ease-(--ease-considered) active:scale-95 ${
        copied
          ? 'text-emerald-400 bg-emerald-500/10'
          : 'text-muted bg-surface hover:bg-surface-hover hover:text-ink'
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
