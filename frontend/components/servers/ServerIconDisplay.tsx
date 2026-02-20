'use client'

import Image from 'next/image'
import {
  ServerIcon,
  DatabaseIcon,
  CloudIcon,
  BoxIcon,
  MonitorIcon,
  HardDriveIcon,
  NetworkIcon,
  ShieldIcon,
  GlobeIcon,
  CpuIcon,
} from 'lucide-react'
import { cn } from '@/lib/utils'

const LUCIDE_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  server: ServerIcon,
  database: DatabaseIcon,
  cloud: CloudIcon,
  box: BoxIcon,
  monitor: MonitorIcon,
  'hard-drive': HardDriveIcon,
  network: NetworkIcon,
  shield: ShieldIcon,
  globe: GlobeIcon,
  cpu: CpuIcon,
}

interface ServerIconProps {
  icon?: string
  className?: string
}

/**
 * Renders a server icon based on the icon string stored in the DB.
 * Format: "lucide:<name>" for Lucide icons, "custom:<filename>" for /public/icons/ files.
 * Falls back to the generic ServerIcon when icon is empty or unrecognised.
 */
export function ServerIconDisplay({ icon, className }: ServerIconProps) {
  if (!icon) {
    return <ServerIcon className={cn('size-4', className)} />
  }

  if (icon.startsWith('lucide:')) {
    const name = icon.slice('lucide:'.length)
    const Icon = LUCIDE_MAP[name] ?? ServerIcon
    return <Icon className={cn('size-4', className)} />
  }

  if (icon.startsWith('custom:')) {
    const filename = icon.slice('custom:'.length)
    return (
      <Image
        src={`/icons/${filename}`}
        alt={filename}
        width={16}
        height={16}
        className={cn('size-4 object-contain', className)}
        unoptimized
      />
    )
  }

  return <ServerIcon className={cn('size-4', className)} />
}
