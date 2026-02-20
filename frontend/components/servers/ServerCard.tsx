'use client'

import Link from 'next/link'
import {
  ActivityIcon,
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
} from 'lucide-react'

import { formatBytes, formatBytesPerSec, formatPercent } from '@/lib/formatters'
import type { LiveServerSnapshot } from '@/types'

interface ServerCardProps {
  /** Statyczne dane serwera (hostname, uuid) */
  hostname: string
  uuid: string
  /** Live snapshot z SSE — null = offline / brak danych */
  snapshot: LiveServerSnapshot | null
}

function MetricRow({
  icon: Icon,
  label,
  value,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <span className="flex items-center gap-1.5 text-muted-foreground">
        <Icon className="size-3.5" />
        {label}
      </span>
      <span className="font-mono text-foreground">{value}</span>
    </div>
  )
}

export function ServerCard({ hostname, uuid, snapshot }: ServerCardProps) {
  const online = snapshot !== null
  const cpuPercent = snapshot?.cpu ?? 0

  return (
    <Link
      href={`/servers/${uuid}/metrics`}
      className="group block rounded-lg border border-border bg-card p-4 transition-colors hover:bg-accent/50"
    >
      {/* Header: hostname + status */}
      <div className="mb-3 flex items-center justify-between">
        <h3 className="truncate font-medium text-card-foreground">{hostname}</h3>
        <span
          className={`flex items-center gap-1 text-xs font-medium ${
            online ? 'text-emerald-400' : 'text-muted-foreground'
          }`}
        >
          <span
            className={`inline-block size-2 rounded-full ${
              online ? 'bg-emerald-400 animate-pulse' : 'bg-muted-foreground'
            }`}
          />
          {online ? 'Online' : 'Offline'}
        </span>
      </div>

      {/* CPU progress bar */}
      <div className="mb-3">
        <div className="mb-1 flex items-center justify-between text-xs">
          <span className="flex items-center gap-1 text-muted-foreground">
            <CpuIcon className="size-3" />
            CPU
          </span>
          <span className="font-mono text-foreground">
            {formatPercent(cpuPercent, 1)}
          </span>
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
          <div
            className="h-full rounded-full bg-primary transition-all"
            style={{ width: `${Math.min(cpuPercent, 100)}%` }}
          />
        </div>
      </div>

      {/* Metrics rows */}
      <div className="space-y-1.5">
        <MetricRow
          icon={MemoryStickIcon}
          label="RAM"
          value={snapshot ? formatBytes(snapshot.memory) : '—'}
        />
        <MetricRow
          icon={NetworkIcon}
          label="Net"
          value={
            snapshot
              ? `↓ ${formatBytesPerSec(snapshot.net_rx_bytes_per_sec)} ↑ ${formatBytesPerSec(snapshot.net_tx_bytes_per_sec)}`
              : '—'
          }
        />
        <MetricRow
          icon={HardDriveIcon}
          label="Disk"
          value={
            snapshot
              ? `R ${formatBytesPerSec(snapshot.disk_read_bytes_per_sec)} W ${formatBytesPerSec(snapshot.disk_write_bytes_per_sec)}`
              : '—'
          }
        />
      </div>
    </Link>
  )
}
