import { TerminalIcon, ListIcon, MousePointerIcon } from '../ui/icons'

export type MobileTab = 'endpoints' | 'requests' | 'detail'

interface Props {
  active: MobileTab
  onChange: (tab: MobileTab) => void
  hasSelection: boolean
}

const TABS: { key: MobileTab; label: string; Icon: typeof TerminalIcon }[] = [
  { key: 'endpoints', label: 'Endpoints', Icon: TerminalIcon },
  { key: 'requests',  label: 'Requests',  Icon: ListIcon },
  { key: 'detail',    label: 'Detail',    Icon: MousePointerIcon },
]

// Fixed bottom nav shown only below the lg breakpoint — desktop two-panel
// layout is completely untouched.
export function MobileTabBar({ active, onChange, hasSelection }: Props) {
  return (
    <nav className="lg:hidden fixed bottom-0 inset-x-0 z-40 flex border-t border-border bg-base/95 backdrop-blur-sm pb-[env(safe-area-inset-bottom)]">
      {TABS.map(({ key, label, Icon }) => {
        const disabled = key === 'detail' && !hasSelection
        const isActive = active === key
        return (
          <button
            key={key}
            onClick={() => !disabled && onChange(key)}
            disabled={disabled}
            className={`flex-1 min-h-[52px] flex flex-col items-center justify-center gap-1 transition-colors duration-200 ease-(--ease-considered) ${
              disabled
                ? 'text-faint/50'
                : isActive
                ? 'text-indigo-400'
                : 'text-muted hover:text-ink'
            }`}
          >
            <Icon className="w-5 h-5" />
            <span className="text-[10px] font-medium">{label}</span>
          </button>
        )
      })}
    </nav>
  )
}
