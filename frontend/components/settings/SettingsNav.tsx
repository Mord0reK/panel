'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'

import { cn } from '@/lib/utils'

const items = [
  { href: '/settings/services', label: 'Integracje' },
  { href: '/settings/servers', label: 'Serwery' },
]

export function SettingsNav() {
  const pathname = usePathname()

  return (
    <nav className="flex items-center gap-2" aria-label="Nawigacja ustawień">
      {items.map((item) => {
        const isActive = pathname === item.href
        return (
          <Link
            key={item.href}
            href={item.href}
            className={cn(
              'rounded-md border px-3 py-1.5 text-sm transition-colors',
              isActive
                ? 'border-zinc-700 bg-zinc-800 text-zinc-100'
                : 'border-zinc-800 bg-zinc-900/60 text-zinc-400 hover:text-zinc-200'
            )}
          >
            {item.label}
          </Link>
        )
      })}
    </nav>
  )
}
