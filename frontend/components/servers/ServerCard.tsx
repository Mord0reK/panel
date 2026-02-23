'use client'

import Link from 'next/link'
import {
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
} from 'lucide-react'

import { formatBytesPerSec, formatBitsPerSec, formatPercent, formatBytes } from '@/lib/formatters'
import type { LiveServerSnapshot } from '@/types'

interface ServerCardProps {
  hostname: string
  uuid: string
  snapshot: LiveServerSnapshot | null
}

function MetricTile({
  icon: Icon,
  label,
  value,
  percent,
  colorClass,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  value: string
  percent?: number
  colorClass: string
}) {
  const percentValue = percent !== undefined ? Math.min(percent, 100) : 0

  return (
    <div className="flex flex-col rounded-md border border-zinc-800 bg-zinc-900/30 p-2">
      <div className="mb-1 flex items-center justify-between">
        <span className="flex items-center gap-1.5 text-xs text-zinc-400">
          <Icon className="size-3.5" />
          {label}
        </span>
        <span className="font-mono text-xs text-zinc-200">{value}</span>
      </div>
      {percent !== undefined && (
        <div className="mt-auto h-1.5 w-full overflow-hidden rounded-full bg-zinc-800">
          <div
            className={`h-full rounded-full ${colorClass} transition-all`}
            style={{ width: `${percentValue}%` }}
          />
        </div>
      )}
    </div>
  )
}

function NetworkTile({
  icon: Icon,
  label,
  rxValue,
  txValue,
  rxPercent,
  txPercent,
  colorClass,
}: {
  icon: React.ComponentType<{ className?: string }>
  label: string
  rxValue: string
  txValue: string
  rxPercent?: number
  txPercent?: number
  colorClass: string
}) {
  const rxBar = rxPercent ?? 0
  const txBar = txPercent ?? 0

  return (
    <div className="flex flex-col rounded-md border border-zinc-800 bg-zinc-900/30 p-2">
      <div className="mb-1 flex items-center justify-between">
        <span className="flex items-center gap-1.5 text-xs text-zinc-400">
          <Icon className="size-3.5" />
          {label}
        </span>
      </div>
      <div className="mt-auto flex items-center justify-center gap-1">
        <span className="text-xs text-zinc-500">{rxValue}</span>
        <div className="relative h-1.5 w-16 overflow-hidden rounded-full bg-zinc-800">
          <div
            className={`absolute right-1/2 top-0 h-full ${colorClass} transition-all`}
            style={{ width: `${rxBar}%`, right: '50%', left: 'auto' }}
          />
          <div
            className={`absolute left-1/2 top-0 h-full ${colorClass} opacity-70 transition-all`}
            style={{ width: `${txBar}%`, left: '50%', right: 'auto' }}
          />
        </div>
        <span className="text-xs text-zinc-500">{txValue}</span>
      </div>
    </div>
  )
}

export function ServerCard({ hostname, uuid, snapshot }: ServerCardProps) {
  const online = snapshot !== null

  const ramPercent = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0

  const maxNet = Math.max(snapshot?.net_rx_bytes_per_sec ?? 0, snapshot?.net_tx_bytes_per_sec ?? 0, 1)
  const rxPercent = snapshot ? (snapshot.net_rx_bytes_per_sec / maxNet) * 100 : 0
  const txPercent = snapshot ? (snapshot.net_tx_bytes_per_sec / maxNet) * 100 : 0

  return (
    <Link
      href={`/servers/${uuid}/metrics`}
      className="group block rounded-lg border border-border bg-card p-4 transition-colors hover:bg-accent/50"
    >
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

      <div className="grid grid-cols-2 gap-2">
        <MetricTile
          icon={CpuIcon}
          label="CPU"
          value={snapshot ? formatPercent(snapshot.cpu, 1) : '—'}
          percent={snapshot?.cpu}
          colorClass="bg-violet-400"
        />
        <MetricTile
          icon={MemoryStickIcon}
          label="RAM"
          value={
            snapshot
              ? `${formatBytes(snapshot.memory)} / ${formatBytes(snapshot.memory_total)}`
              : '—'
          }
          percent={ramPercent}
          colorClass="bg-emerald-400"
        />
        <MetricTile
          icon={HardDriveIcon}
          label="Disk"
          value={
            snapshot
              ? `${formatPercent(snapshot.disk_used_percent, 1)}`
              : '—'
          }
          percent={snapshot?.disk_used_percent}
          colorClass="bg-amber-400"
        />
        <NetworkTile
          icon={NetworkIcon}
          label="Net"
          rxValue={snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec) : '—'}
          txValue={snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec) : '—'}
          rxPercent={rxPercent}
          txPercent={txPercent}
          colorClass="bg-cyan-400"
        />
      </div>
    </Link>
  )
}
