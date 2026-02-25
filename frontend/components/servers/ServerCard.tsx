'use client'

import Link from 'next/link'
import {
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
  ArrowDownIcon,
  ArrowUpIcon,
} from 'lucide-react'

import { ServerIconDisplay } from '@/components/servers/ServerIconDisplay'
import { formatBitsPerSec } from '@/lib/formatters'
import type { LiveServerSnapshot } from '@/types'

interface ServerCardProps {
  hostname: string
  uuid: string
  snapshot: LiveServerSnapshot | null
  icon?: string
  displayName?: string
}

/**
 * Skala segmentowana po dekadach Mbit/s.
 * Każda dekada (0–1, 1–10, 10–100, 100–1000 Mbit/s) = 25% szerokości paska.
 * Dzięki temu 14 Kbit/s (~0.4%) i 3 Mbit/s (~31%) są czytelnie różne,
 * a wysoki ruch (720 Mbit/s ~92%) nadal wizualnie dominuje.
 */
function netPercent(bytesPerSec: number): number {
  const mbps = (bytesPerSec * 8) / 1_000_000
  if (mbps <= 0) return 0
  if (mbps < 1) return (mbps / 1) * 25
  if (mbps < 10) return 10 + ((mbps - 1) / 9) * 25
  if (mbps < 100) return 25 + ((mbps - 10) / 90) * 25
  return Math.min(100, 75 + ((mbps - 100) / 900) * 25)
}

function cpuColor(v: number) {
  if (v >= 80) return { bar: 'bg-red-400/70', text: 'text-red-400' }
  if (v >= 60) return { bar: 'bg-amber-400/70', text: 'text-amber-400' }
  return { bar: 'bg-blue-400/70', text: 'text-blue-400' }
}

function memColor(v: number) {
  if (v >= 85) return { bar: 'bg-red-400/70', text: 'text-red-400' }
  if (v >= 70) return { bar: 'bg-amber-400/70', text: 'text-amber-400' }
  return { bar: 'bg-emerald-400/70', text: 'text-emerald-400' }
}

function diskColor(v: number) {
  if (v >= 90) return { bar: 'bg-red-400/70', text: 'text-red-400' }
  if (v >= 75) return { bar: 'bg-amber-400/70', text: 'text-amber-400' }
  return { bar: 'bg-amber-400/70', text: 'text-amber-400' }
}

export function ServerCard({ hostname, uuid, snapshot, icon, displayName }: ServerCardProps) {
  const online = snapshot !== null
  const serverName = displayName ?? hostname

  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0

  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  const cpu = cpuColor(snapshot?.cpu ?? 0)
  const mem = memColor(ram)
  const disk = diskColor(snapshot?.disk_used_percent ?? 0)

  return (
    <Link
      href={`/servers/${uuid}/metrics`}
      className="group block rounded-xl border border-zinc-800 bg-zinc-950 p-4 transition-colors hover:border-zinc-700 hover:bg-zinc-900/80"
    >
      {/* Header */}
      <div className="mb-4 flex items-center justify-between">
        <div className="flex min-w-0 items-center gap-2">
          <ServerIconDisplay icon={icon} className="size-5 shrink-0 text-zinc-400" />
          <h3 className="truncate font-bold text-zinc-100">{serverName}</h3>
        </div>
        <div className={`flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${online ? 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20' : 'bg-zinc-800 text-zinc-500'}`}>
          <span className={`size-1.5 rounded-full ${online ? 'bg-emerald-400' : 'bg-zinc-500'}`} />
          {online ? 'Online' : 'Offline'}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {/* CPU */}
        <div>
          <p className="mb-1 flex items-center gap-1 text-[11px] uppercase tracking-wider text-zinc-400">
            <CpuIcon className="size-3" /> CPU
          </p>
          <p className={`font-mono text-1xl font-bold ${cpu.text}`}>
            {snapshot ? snapshot.cpu.toFixed(2) : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${cpu.bar} transition-all`} style={{ width: snapshot ? `${Math.min(snapshot.cpu, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* RAM */}
        <div>
          <p className="mb-1 flex items-center gap-1 text-[11px] uppercase tracking-wider text-zinc-400">
            <MemoryStickIcon className="size-3" /> RAM
          </p>
          <p className={`font-mono text-1xl font-bold ${mem.text}`}>
            {snapshot ? ram.toFixed(1) : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${mem.bar} transition-all`} style={{ width: `${Math.min(ram, 100)}%` }} />
          </div>
        </div>

        {/* Disk */}
        <div>
          <p className="mb-1 flex items-center gap-1 text-[11px] uppercase tracking-wider text-zinc-400">
            <HardDriveIcon className="size-3" /> Dysk
          </p>
          <p className={`font-mono text-1xl font-bold ${disk.text}`}>
            {snapshot ? snapshot.disk_used_percent.toFixed(2) : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${disk.bar} transition-all`} style={{ width: snapshot ? `${Math.min(snapshot.disk_used_percent, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* Network */}
        <div>
          <p className="mb-1 flex items-center gap-1 text-[11px] uppercase tracking-wider text-zinc-400">
            <NetworkIcon className="size-3" /> Sieć
          </p>
          <div className="space-y-1.5">
            <div>
              <div className="flex items-baseline gap-1">
                <ArrowDownIcon className="mb-0.5 size-3 shrink-0 text-cyan-400" />
                <span className="font-mono text-sm font-bold text-cyan-400">{snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 1) : '—'}</span>
              </div>
              <div className="mt-0.5 h-1 w-full overflow-hidden rounded-full bg-zinc-800">
                <div className="h-full rounded-full bg-cyan-400/70 transition-all" style={{ width: `${rxPct}%` }} />
              </div>
            </div>
            <div>
              <div className="flex items-baseline gap-1">
                <ArrowUpIcon className="mb-0.5 size-3 shrink-0 text-cyan-300" />
                <span className="font-mono text-sm font-bold text-cyan-300">{snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 1) : '—'}</span>
              </div>
              <div className="mt-0.5 h-1 w-full overflow-hidden rounded-full bg-zinc-800">
                <div className="h-full rounded-full bg-cyan-300/60 transition-all" style={{ width: `${txPct}%` }} />
              </div>
            </div>
          </div>
        </div>
      </div>
    </Link>
  )
}
