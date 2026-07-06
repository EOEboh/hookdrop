import { useId, type SVGProps } from 'react'

type LogoSize = 'sm' | 'md' | 'lg'

const GLYPH_SIZE: Record<LogoSize, string> = {
  sm: 'w-4 h-5',
  md: 'w-6 h-7',
  lg: 'w-9 h-11',
}

const WORDMARK_SIZE: Record<LogoSize, string> = {
  sm: 'text-sm',
  md: 'text-base',
  lg: 'text-xl',
}

/**
 * Custom hookdrop mark: a fishhook whose eye is shaped like a water drop —
 * the "eye" of a real hook (the loop line is tied through) doubles as the
 * "drop" of hookdrop, with the shank curving into the hook's bend and barb
 * below it. Drawn as a single monoline stroke (no fill), so it reads as a
 * clean line-art mark rather than a filled icon.
 */
function HookDropGlyph({ gradientId, ...props }: SVGProps<SVGSVGElement> & { gradientId: string }) {
  return (
    <svg viewBox="0 0 24 30" fill="none" {...props}>
      <defs>
        <linearGradient id={gradientId} x1="4" y1="0" x2="20" y2="30" gradientUnits="userSpaceOnUse">
          <stop offset="0" stopColor="var(--color-brand)" />
          <stop offset="1" stopColor="var(--color-brand-2)" />
        </linearGradient>
      </defs>
      {/* drop-shaped eye */}
      <path
        d="M12 2.2 C15.8 6.6 17.3 9.4 17.3 11.6 C17.3 14.6 15 16.8 12 16.8 C9 16.8 6.7 14.6 6.7 11.6 C6.7 9.4 8.2 6.6 12 2.2 Z"
        stroke={`url(#${gradientId})`}
        strokeWidth="2"
      />
      {/* shank, bend and barb */}
      <path
        d="M12 15 C12 19 12 22 12 24.5 C12 27.3 14.3 28.8 16.7 27.5 C18.5 26.5 18.8 24.3 17.5 23.2"
        stroke={`url(#${gradientId})`}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

interface LogoMarkProps {
  size?: LogoSize
  className?: string
}

/** Icon-only mark. */
export function LogoMark({ size = 'md', className = '' }: LogoMarkProps) {
  const gradientId = useId()
  return <HookDropGlyph gradientId={gradientId} className={`shrink-0 ${GLYPH_SIZE[size]} ${className}`} />
}

interface LogoProps {
  size?: LogoSize
  className?: string
}

/** Full lockup: icon mark + "hookdrop" wordmark. */
export function Logo({ size = 'md', className = '' }: LogoProps) {
  return (
    <span className={`inline-flex items-center gap-2 ${className}`}>
      <LogoMark size={size} />
      <span className={`font-semibold text-ink tracking-tight ${WORDMARK_SIZE[size]}`}>
        hookdrop
      </span>
    </span>
  )
}
