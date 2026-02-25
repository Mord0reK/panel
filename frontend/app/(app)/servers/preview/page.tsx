'use client'

import Link from 'next/link'
import {
  CpuIcon,
  HardDriveIcon,
  MemoryStickIcon,
  NetworkIcon,
  ArrowDownIcon,
  ArrowUpIcon,
  ActivityIcon,
  ServerIcon,
} from 'lucide-react'

import { formatBitsPerSec, formatPercent, formatBytes, formatBytesPerSec } from '@/lib/formatters'
import type { LiveServerSnapshot } from '@/types'

// ---------------------------------------------------------------------------
// Mock data — serwer podczas speedtest (~720 Mbit/s download, ~56 Mbit/s upload)
// ---------------------------------------------------------------------------
const MOCK: LiveServerSnapshot = {
  uuid: 'mock-server-1',
  hostname: 'web-prod-01',
  cpu: 23.4,
  memory: 12_884_901_888,       // ~12 GB
  mem_percent: 75.2,
  memory_total: 17_179_869_184, // 16 GB
  disk_used_percent: 68.3,
  disk_read_bytes_per_sec: 2_097_152,   // 2 MB/s
  disk_write_bytes_per_sec: 524_288,    // 512 KB/s
  net_rx_bytes_per_sec: 94_371_840,     // ~720 Mbit/s
  net_tx_bytes_per_sec: 7_340_032,      // ~56 Mbit/s
}

// ---------------------------------------------------------------------------
// Narzędzia wspólne
// ---------------------------------------------------------------------------

/** Skala segmentowana po dekadach Mbit/s (spójna z ServerCard). */
function netPercent(bytesPerSec: number): number {
  const mbps = (bytesPerSec * 8) / 1_000_000
  if (mbps <= 0) return 0
  if (mbps < 1) return (mbps / 1) * 25
  if (mbps < 10) return 25 + ((mbps - 1) / 9) * 25
  if (mbps < 100) return 50 + ((mbps - 10) / 90) * 25
  return Math.min(100, 75 + ((mbps - 100) / 900) * 25)
}

function cpuColor(v: number) {
  if (v >= 80) return 'bg-red-400'
  if (v >= 60) return 'bg-amber-400'
  return 'bg-violet-400'
}

function memColor(v: number) {
  if (v >= 85) return 'bg-red-400'
  if (v >= 70) return 'bg-amber-400'
  return 'bg-emerald-400'
}

function diskColor(v: number) {
  if (v >= 90) return 'bg-red-400'
  if (v >= 75) return 'bg-amber-400'
  return 'bg-amber-400'
}

// ===========================================================================
// WARIANT 1 — "Sleek Minimal" (ewolucja obecnego, naprawiona sieć)
// ===========================================================================
function V1Card({ hostname, snapshot }: { hostname: string; snapshot: LiveServerSnapshot | null }) {
  const online = snapshot !== null
  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0

  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-4 transition-colors hover:border-zinc-700 hover:bg-zinc-900/80 cursor-pointer">
      {/* Header */}
      <div className="mb-4 flex items-center gap-3">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-zinc-800">
          <ServerIcon className="size-4 text-zinc-400" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate font-semibold text-zinc-100">{hostname}</p>
        </div>
        <span className={`flex items-center gap-1.5 text-xs font-medium ${online ? 'text-emerald-400' : 'text-zinc-500'}`}>
          <span className={`size-1.5 rounded-full ${online ? 'bg-emerald-400 animate-pulse' : 'bg-zinc-600'}`} />
          {online ? 'Online' : 'Offline'}
        </span>
      </div>

      {/* Tiles 2×2 */}
      <div className="grid grid-cols-2 gap-2">
        {/* CPU */}
        <div className="rounded-lg bg-zinc-900 p-2.5">
          <div className="mb-2 flex items-center justify-between">
            <span className="flex items-center gap-1 text-[11px] text-zinc-500"><CpuIcon className="size-3" /> CPU</span>
            <span className="font-mono text-xs text-zinc-200">{snapshot ? formatPercent(snapshot.cpu, 1) : '—'}</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${cpuColor(snapshot?.cpu ?? 0)} transition-all`} style={{ width: snapshot ? `${Math.min(snapshot.cpu, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* RAM */}
        <div className="rounded-lg bg-zinc-900 p-2.5">
          <div className="mb-2 flex items-center justify-between">
            <span className="flex items-center gap-1 text-[11px] text-zinc-500"><MemoryStickIcon className="size-3" /> RAM</span>
            <span className="font-mono text-xs text-zinc-200">{snapshot ? formatPercent(ram, 0) : '—'}</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${memColor(ram)} transition-all`} style={{ width: `${Math.min(ram, 100)}%` }} />
          </div>
        </div>

        {/* Disk */}
        <div className="rounded-lg bg-zinc-900 p-2.5">
          <div className="mb-2 flex items-center justify-between">
            <span className="flex items-center gap-1 text-[11px] text-zinc-500"><HardDriveIcon className="size-3" /> Dysk</span>
            <span className="font-mono text-xs text-zinc-200">{snapshot ? formatPercent(snapshot.disk_used_percent, 1) : '—'}</span>
          </div>
          <div className="h-1 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className={`h-full rounded-full ${diskColor(snapshot?.disk_used_percent ?? 0)} transition-all`} style={{ width: snapshot ? `${Math.min(snapshot.disk_used_percent, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* Network — dwa oddzielne paski, skala log */}
        <div className="rounded-lg bg-zinc-900 p-2.5">
          <div className="mb-1.5 flex items-center gap-1 text-[11px] text-zinc-500">
            <NetworkIcon className="size-3" /> Sieć
          </div>
          <div className="space-y-1">
            <div className="flex items-center gap-1.5">
              <ArrowDownIcon className="size-2.5 shrink-0 text-cyan-400" />
              <div className="flex-1 h-1 overflow-hidden rounded-full bg-zinc-800">
                <div className="h-full rounded-full bg-cyan-400 transition-all" style={{ width: `${rxPct}%` }} />
              </div>
              <span className="w-16 text-right font-mono text-[10px] text-zinc-400">{snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 0) : '—'}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <ArrowUpIcon className="size-2.5 shrink-0 text-cyan-300" />
              <div className="flex-1 h-1 overflow-hidden rounded-full bg-zinc-800">
                <div className="h-full rounded-full bg-cyan-300 opacity-70 transition-all" style={{ width: `${txPct}%` }} />
              </div>
              <span className="w-16 text-right font-mono text-[10px] text-zinc-400">{snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 0) : '—'}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// ===========================================================================
// WARIANT 2 — "Neon Accent" (kolorowe lewe bordery, duże liczby)
// ===========================================================================
function V2Card({ hostname, snapshot }: { hostname: string; snapshot: LiveServerSnapshot | null }) {
  const online = snapshot !== null
  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0
  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  const metrics = [
    {
      label: 'CPU',
      value: snapshot ? formatPercent(snapshot.cpu, 1) : '—',
      pct: snapshot?.cpu ?? 0,
      bar: 'bg-violet-400',
      border: 'border-l-violet-500',
      icon: <CpuIcon className="size-3.5" />,
    },
    {
      label: 'RAM',
      value: snapshot ? formatPercent(ram, 1) : '—',
      pct: ram,
      bar: 'bg-emerald-400',
      border: 'border-l-emerald-500',
      icon: <MemoryStickIcon className="size-3.5" />,
    },
    {
      label: 'Dysk',
      value: snapshot ? formatPercent(snapshot.disk_used_percent, 1) : '—',
      pct: snapshot?.disk_used_percent ?? 0,
      bar: 'bg-amber-400',
      border: 'border-l-amber-500',
      icon: <HardDriveIcon className="size-3.5" />,
    },
  ]

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 overflow-hidden cursor-pointer hover:border-zinc-700 transition-colors">
      {/* Header with gradient */}
      <div className="flex items-center justify-between border-b border-zinc-800 bg-zinc-900/50 px-4 py-3">
        <div className="flex items-center gap-2.5">
          <div className={`size-2 rounded-full ${online ? 'bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.8)]' : 'bg-zinc-600'}`} />
          <span className="font-semibold text-zinc-100">{hostname}</span>
        </div>
        <span className={`text-xs ${online ? 'text-emerald-400' : 'text-zinc-500'}`}>
          {online ? 'Online' : 'Offline'}
        </span>
      </div>

      <div className="p-3 space-y-2">
        {/* Metric rows */}
        {metrics.map((m) => (
          <div key={m.label} className={`flex items-center gap-3 rounded-md border-l-2 ${m.border} bg-zinc-900/60 px-3 py-2`}>
            <span className="text-zinc-500 shrink-0">{m.icon}</span>
            <span className="w-8 text-xs text-zinc-500">{m.label}</span>
            <div className="flex-1 h-1.5 overflow-hidden rounded-full bg-zinc-800">
              <div className={`h-full rounded-full ${m.bar} transition-all`} style={{ width: `${Math.min(m.pct, 100)}%` }} />
            </div>
            <span className="w-14 text-right font-mono text-xs text-zinc-200">{m.value}</span>
          </div>
        ))}

        {/* Network oddzielnie */}
        <div className="rounded-md border-l-2 border-l-cyan-500 bg-zinc-900/60 px-3 py-2 space-y-1.5">
          <div className="flex items-center gap-1.5 text-[11px] text-zinc-500 mb-1">
            <NetworkIcon className="size-3" /> Sieć <span className="text-zinc-700">(log₁₀ · 1 Gbps ref)</span>
          </div>
          <div className="flex items-center gap-3">
            <ArrowDownIcon className="size-3 text-cyan-400 shrink-0" />
            <div className="flex-1 h-1.5 overflow-hidden rounded-full bg-zinc-800">
              <div className="h-full rounded-full bg-cyan-400 transition-all" style={{ width: `${rxPct}%` }} />
            </div>
            <span className="w-20 text-right font-mono text-xs text-zinc-200">{snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 0) : '—'}</span>
          </div>
          <div className="flex items-center gap-3">
            <ArrowUpIcon className="size-3 text-cyan-300 shrink-0" />
            <div className="flex-1 h-1.5 overflow-hidden rounded-full bg-zinc-800">
              <div className="h-full rounded-full bg-cyan-300 opacity-80 transition-all" style={{ width: `${txPct}%` }} />
            </div>
            <span className="w-20 text-right font-mono text-xs text-zinc-200">{snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 0) : '—'}</span>
          </div>
        </div>
      </div>
    </div>
  )
}

// ===========================================================================
// WARIANT 3 — "Big Numbers" (duże liczby + cienkie paski pod spodem)
// ===========================================================================
function V3Card({ hostname, snapshot }: { hostname: string; snapshot: LiveServerSnapshot | null }) {
  const online = snapshot !== null
  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0
  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-4 cursor-pointer hover:border-zinc-700 transition-colors">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between">
        <h3 className="font-bold text-zinc-100">{hostname}</h3>
        <div className={`flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${online ? 'bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20' : 'bg-zinc-800 text-zinc-500'}`}>
          <span className={`size-1.5 rounded-full ${online ? 'bg-emerald-400' : 'bg-zinc-500'}`} />
          {online ? 'Online' : 'Offline'}
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        {/* CPU */}
        <div>
          <p className="flex items-center gap-1 mb-1 text-[11px] uppercase tracking-wider text-zinc-600">
            <CpuIcon className="size-3" /> CPU
          </p>
          <p className="font-mono text-2xl font-bold text-violet-400">
            {snapshot ? `${snapshot.cpu.toFixed(1)}` : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-0.5 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className="h-full rounded-full bg-violet-400/70 transition-all" style={{ width: snapshot ? `${Math.min(snapshot.cpu, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* RAM */}
        <div>
          <p className="flex items-center gap-1 mb-1 text-[11px] uppercase tracking-wider text-zinc-600">
            <MemoryStickIcon className="size-3" /> RAM
          </p>
          <p className="font-mono text-2xl font-bold text-emerald-400">
            {snapshot ? `${ram.toFixed(0)}` : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-0.5 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className="h-full rounded-full bg-emerald-400/70 transition-all" style={{ width: `${Math.min(ram, 100)}%` }} />
          </div>
        </div>

        {/* Disk */}
        <div>
          <p className="flex items-center gap-1 mb-1 text-[11px] uppercase tracking-wider text-zinc-600">
            <HardDriveIcon className="size-3" /> Dysk
          </p>
          <p className="font-mono text-2xl font-bold text-amber-400">
            {snapshot ? `${(snapshot.disk_used_percent).toFixed(1)}` : '—'}
            <span className="text-base text-zinc-500">%</span>
          </p>
          <div className="mt-1.5 h-0.5 w-full overflow-hidden rounded-full bg-zinc-800">
            <div className="h-full rounded-full bg-amber-400/70 transition-all" style={{ width: snapshot ? `${Math.min(snapshot.disk_used_percent, 100)}%` : '0%' }} />
          </div>
        </div>

        {/* Network */}
        <div>
          <p className="flex items-center gap-1 mb-1 text-[11px] uppercase tracking-wider text-zinc-600">
            <NetworkIcon className="size-3" /> Sieć
          </p>
          <div className="space-y-1">
            <div className="flex items-baseline gap-1">
              <ArrowDownIcon className="size-3 text-cyan-400 shrink-0 mb-0.5" />
              <span className="font-mono text-sm font-bold text-cyan-400">{snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 0) : '—'}</span>
            </div>
            <div className="mt-0.5 h-0.5 w-full overflow-hidden rounded-full bg-zinc-800">
              <div className="h-full rounded-full bg-cyan-400/70 transition-all" style={{ width: `${rxPct}%` }} />
            </div>
            <div className="flex items-baseline gap-1">
              <ArrowUpIcon className="size-3 text-cyan-300 shrink-0 mb-0.5" />
              <span className="font-mono text-sm font-bold text-cyan-300">{snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 0) : '—'}</span>
            </div>
            <div className="mt-0.5 h-0.5 w-full overflow-hidden rounded-full bg-zinc-800">
              <div className="h-full rounded-full bg-cyan-300/60 transition-all" style={{ width: `${txPct}%` }} />
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// ===========================================================================
// WARIANT 4 — "Status Panel" (pasek kolorystyczny na górze jako gauge)
// ===========================================================================
function GaugeBar({ pct, color }: { pct: number; color: string }) {
  const segments = 20
  const filled = Math.round((Math.min(pct, 100) / 100) * segments)
  return (
    <div className="flex gap-px">
      {Array.from({ length: segments }).map((_, i) => (
        <div
          key={i}
          className={`h-1.5 flex-1 rounded-sm transition-all ${i < filled ? color : 'bg-zinc-800'}`}
        />
      ))}
    </div>
  )
}

function V4Card({ hostname, snapshot }: { hostname: string; snapshot: LiveServerSnapshot | null }) {
  const online = snapshot !== null
  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0
  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 overflow-hidden cursor-pointer hover:border-zinc-700 transition-colors">
      {/* Accent top bar */}
      <div className={`h-0.5 w-full ${online ? 'bg-linear-to-r from-violet-500 via-cyan-500 to-emerald-500' : 'bg-zinc-800'}`} />

      <div className="p-4">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ActivityIcon className={`size-4 ${online ? 'text-emerald-400' : 'text-zinc-600'}`} />
            <span className="font-semibold text-zinc-100">{hostname}</span>
          </div>
          <span className={`text-xs font-mono ${online ? 'text-emerald-400' : 'text-zinc-600'}`}>
            {online ? '● ONLINE' : '○ OFFLINE'}
          </span>
        </div>

        <div className="space-y-3">
          {/* CPU */}
          <div>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="flex items-center gap-1 text-zinc-500"><CpuIcon className="size-3" /> CPU</span>
              <span className="font-mono text-violet-300">{snapshot ? formatPercent(snapshot.cpu, 1) : '—'}</span>
            </div>
            <GaugeBar pct={snapshot?.cpu ?? 0} color="bg-violet-400" />
          </div>

          {/* RAM */}
          <div>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="flex items-center gap-1 text-zinc-500"><MemoryStickIcon className="size-3" /> RAM</span>
              <span className="font-mono text-emerald-300">
                {snapshot ? `${formatBytes(snapshot.memory, 0)} / ${formatBytes(snapshot.memory_total, 0)}` : '—'}
              </span>
            </div>
            <GaugeBar pct={ram} color="bg-emerald-400" />
          </div>

          {/* Disk */}
          <div>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="flex items-center gap-1 text-zinc-500"><HardDriveIcon className="size-3" /> Dysk</span>
              <span className="font-mono text-amber-300">{snapshot ? formatPercent(snapshot.disk_used_percent, 1) : '—'}</span>
            </div>
            <GaugeBar pct={snapshot?.disk_used_percent ?? 0} color="bg-amber-400" />
          </div>

          {/* Network RX */}
          <div>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="flex items-center gap-1 text-zinc-500"><ArrowDownIcon className="size-3" /> RX</span>
              <span className="font-mono text-cyan-300">{snapshot ? formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 0) : '—'}</span>
            </div>
            <GaugeBar pct={rxPct} color="bg-cyan-400" />
          </div>

          {/* Network TX */}
          <div>
            <div className="mb-1 flex items-center justify-between text-xs">
              <span className="flex items-center gap-1 text-zinc-500"><ArrowUpIcon className="size-3" /> TX</span>
              <span className="font-mono text-cyan-400">{snapshot ? formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 0) : '—'}</span>
            </div>
            <GaugeBar pct={txPct} color="bg-cyan-300" />
          </div>
        </div>
      </div>
    </div>
  )
}

// ===========================================================================
// WARIANT 5 — "Terminal Mono" (CLI-inspired, monospace wszystko)
// ===========================================================================
function V5Card({ hostname, snapshot }: { hostname: string; snapshot: LiveServerSnapshot | null }) {
  const online = snapshot !== null
  const ram = snapshot && snapshot.memory_total > 0
    ? (snapshot.memory / snapshot.memory_total) * 100
    : snapshot?.mem_percent ?? 0
  const rxPct = snapshot ? netPercent(snapshot.net_rx_bytes_per_sec) : 0
  const txPct = snapshot ? netPercent(snapshot.net_tx_bytes_per_sec) : 0

  function AsciiBar({ pct, color }: { pct: number; color: string }) {
    const filled = Math.round((Math.min(pct, 100) / 100) * 16)
    return (
      <span className="font-mono text-[11px]">
        <span className="text-zinc-700">[</span>
        <span className={color}>{'█'.repeat(filled)}</span>
        <span className="text-zinc-800">{'░'.repeat(16 - filled)}</span>
        <span className="text-zinc-700">]</span>
      </span>
    )
  }

  const rows: Array<{ key: string; bar: number; barColor: string; value: string }> = snapshot
    ? [
        { key: 'cpu     ', bar: snapshot.cpu, barColor: 'text-violet-400', value: formatPercent(snapshot.cpu, 1) },
        { key: 'mem     ', bar: ram, barColor: 'text-emerald-400', value: `${formatBytes(snapshot.memory, 0)}/${formatBytes(snapshot.memory_total, 0)}` },
        { key: 'disk    ', bar: snapshot.disk_used_percent, barColor: 'text-amber-400', value: formatPercent(snapshot.disk_used_percent, 1) },
        { key: 'net ↓ rx', bar: rxPct, barColor: 'text-cyan-400', value: formatBitsPerSec(snapshot.net_rx_bytes_per_sec, 0) },
        { key: 'net ↑ tx', bar: txPct, barColor: 'text-cyan-300', value: formatBitsPerSec(snapshot.net_tx_bytes_per_sec, 0) },
      ]
    : []

  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-950 p-4 font-mono cursor-pointer hover:border-zinc-700 transition-colors">
      {/* Header line */}
      <div className="mb-3 border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-zinc-600">$</span>
          <span className="text-zinc-300">status </span>
          <span className="text-violet-400 font-bold">{hostname}</span>
          <span className={`ml-auto text-xs ${online ? 'text-emerald-400' : 'text-red-400'}`}>
            [{online ? 'UP' : 'DOWN'}]
          </span>
        </div>
      </div>

      {/* Rows */}
      {online && rows.length > 0 ? (
        <div className="space-y-2">
          {rows.map((r) => (
            <div key={r.key} className="flex items-center gap-2 text-[11px]">
              <span className="w-16 shrink-0 text-zinc-600">{r.key}</span>
              <AsciiBar pct={r.bar} color={r.barColor} />
              <span className={`${r.barColor} ml-auto`}>{r.value}</span>
            </div>
          ))}
          <div className="mt-2 pt-2 border-t border-zinc-900 text-[10px] text-zinc-700">
            net scale: log₁₀ · ref=1Gbps
          </div>
        </div>
      ) : (
        <p className="text-sm text-red-400">connection lost</p>
      )}
    </div>
  )
}

// ===========================================================================
// PAGE
// ===========================================================================
export default function ServerCardPreviewPage() {
  const variants = [
    { id: 1, name: 'Sleek Minimal', desc: 'Naprawiony net bars (log scale), 2×2 siatka, server icon, czyste tło', Component: V1Card },
    { id: 2, name: 'Neon Accent', desc: 'Kolorowe lewe bordery per metryka, RX/TX osobno z glow status', Component: V2Card },
    { id: 3, name: 'Big Numbers', desc: 'Duże cyfry w kolorach metryk, cienkie paski jako podkreślenie', Component: V3Card },
    { id: 4, name: 'Status Panel', desc: 'Segmented gauge bary (20 segmentów), gradient top-bar, RX i TX osobno', Component: V4Card },
    { id: 5, name: 'Terminal Mono', desc: 'CLI-inspired, ASCII progress bary monospace, styl terminala', Component: V5Card },
  ]

  return (
    <div className="min-h-screen bg-zinc-950 p-8">
      <div className="mx-auto max-w-5xl">
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-zinc-100">ServerCard — propozycje redesignu</h1>
          <p className="mt-1 text-sm text-zinc-500">
            Mock server: <span className="font-mono text-zinc-400">web-prod-01</span> podczas speedtestu (~720 Mbit/s ↓ / ~56 Mbit/s ↑).
            Sieć wizualizowana skalą logarytmiczną (log₁₀, ref = 1 Gbps) — pełny download widoczny jako ~98% paska.
          </p>
          <div className="mt-2 flex flex-wrap gap-3 text-xs text-zinc-600">
            <span>CPU: <span className="text-violet-400">23.4%</span></span>
            <span>RAM: <span className="text-emerald-400">12 GB / 16 GB (75%)</span></span>
            <span>Dysk: <span className="text-amber-400">68.3%</span></span>
            <span>↓ RX: <span className="text-cyan-400">~720 Mbit/s</span></span>
            <span>↑ TX: <span className="text-cyan-300">~56 Mbit/s</span></span>
          </div>
        </div>

        {/* Comparison grid */}
        <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
          {variants.map(({ id, name, desc, Component }) => (
            <div key={id}>
              <div className="mb-2 flex items-baseline gap-2">
                <span className="text-xs font-bold text-zinc-600">#{id}</span>
                <span className="font-semibold text-zinc-300">{name}</span>
              </div>
              <p className="mb-3 text-[11px] text-zinc-600 leading-relaxed">{desc}</p>
              <Component hostname="web-prod-01" snapshot={MOCK} />
            </div>
          ))}
        </div>

        {/* Offline state demo for variant 1 */}
        <div className="mt-12">
          <h2 className="mb-4 text-sm font-semibold uppercase tracking-widest text-zinc-600">Stan Offline (wszystkie warianty)</h2>
          <div className="grid grid-cols-1 gap-8 sm:grid-cols-2 lg:grid-cols-3">
            {variants.map(({ id, name, Component }) => (
              <div key={id}>
                <p className="mb-3 text-[11px] text-zinc-700">#{id} {name} — offline</p>
                <Component hostname="web-prod-01" snapshot={null} />
              </div>
            ))}
          </div>
        </div>

        <div className="mt-12 rounded-lg border border-zinc-800 bg-zinc-900/50 p-4 text-sm text-zinc-500">
          <p className="font-semibold text-zinc-400 mb-2">Naprawione problemy vs obecny kard:</p>
          <ul className="space-y-1 text-xs list-disc list-inside">
            <li>Sieć: poprzedni kod używał <code className="text-zinc-300">* 0.30</code> — przy 1 Gbps pasek wynosił maks 30%. Teraz: skala log₁₀ względem 1 Gbps → 720 Mbit/s = ~98% paska</li>
            <li>RX i TX wyświetlane jako dwa oddzielne paski zamiast podzielonego lustrzanego baru</li>
            <li>Każdy wariant pokazuje wartości bezwzględne sieci (Mbit/s) obok paska</li>
            <li>Kolory dynamiczne: zielony/żółty/czerwony w zależności od obciążenia CPU i RAM</li>
          </ul>
        </div>
      </div>
    </div>
  )
}
