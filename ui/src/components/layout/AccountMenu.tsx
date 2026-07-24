import { useEffect, useRef, useState } from 'react'
import { usePostHog } from '@posthog/react'
import { ChevronDownIcon, SettingsIcon } from '../ui/icons'

interface Props {
  isPro: boolean
  onLogout: () => void
}

/**
 * Compact account menu for the sidebar header. Replaces the row of inline
 * Upgrade / API tokens / Status / Log out links, which overflowed the fixed
 * 320px rail. A single trigger opens a small right-aligned dropdown; the
 * trigger tints teal when the account is on Pro so plan state stays glanceable.
 */
export function AccountMenu({ isPro, onLogout }: Props) {
  const posthog = usePostHog()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)

  // Close on outside click and Escape while open.
  useEffect(() => {
    if (!open) return
    function onPointerDown(e: MouseEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('mousedown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  function handleLogout() {
    posthog?.capture('user_logged_out')
    setOpen(false)
    onLogout()
  }

  const itemClass =
    'flex items-center justify-between gap-2 px-3 py-2 text-xs text-muted hover:text-ink hover:bg-surface-hover transition-colors duration-200 ease-(--ease-considered)'

  return (
    <div ref={rootRef} className="relative">
      <button
        onClick={() => setOpen(o => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label="Account menu"
        className={`flex items-center gap-0.5 p-1.5 rounded-md border transition-colors duration-200 ease-(--ease-considered) ${
          isPro
            ? 'text-indigo-400 border-indigo-400/40 hover:border-indigo-400/70'
            : 'text-faint border-border-strong hover:text-muted hover:border-faint'
        }`}
      >
        <SettingsIcon className="w-4 h-4" />
        <ChevronDownIcon className={`w-3 h-3 transition-transform duration-200 ease-(--ease-considered) ${open ? 'rotate-180' : ''}`} />
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 mt-1 w-44 bg-surface border border-border rounded-lg shadow-lg py-1 z-50 animate-fade-in"
        >
          <a
            href="/settings/billing"
            role="menuitem"
            className={`${itemClass} ${isPro ? 'text-indigo-400 hover:text-indigo-300' : ''}`}
          >
            {isPro ? '⚡ Pro' : 'Upgrade'}
          </a>
          <a href="/settings/tokens" role="menuitem" className={itemClass}>
            API tokens
          </a>
          <a
            href="https://status.hookdrop.app"
            target="_blank"
            rel="noopener noreferrer"
            role="menuitem"
            className={itemClass}
          >
            Status
          </a>
          <div className="my-1 border-t border-border" />
          <button role="menuitem" onClick={handleLogout} className={`${itemClass} w-full text-left`}>
            Log out
          </button>
        </div>
      )}
    </div>
  )
}
