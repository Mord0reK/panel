'use client'

import { usePathname } from 'next/navigation'
import { useEffect, useState, type ReactNode } from 'react'

export default function Template({ children }: { children: ReactNode }) {
  const pathname = usePathname()
  const [mounted, setMounted] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  if (!mounted) {
    return <>{children}</>
  }

  return (
    <div
      key={pathname}
      className="animate-in fade-in slide-in-from-bottom-4 duration-300 ease-out"
    >
      {children}
    </div>
  )
}
