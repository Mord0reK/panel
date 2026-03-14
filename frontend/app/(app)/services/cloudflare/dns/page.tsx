'use client'

import Link from 'next/link'
import {
  ArrowLeftIcon,
  ArrowUpDownIcon,
  CloudIcon,
  RefreshCwIcon,
  Settings2Icon,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { CloudflareDNSRecord } from '@/types'

type SortKey = 'name' | 'type'
type SortDir = 'asc' | 'desc'

const DNS_TYPES = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA']

function formatTTL(ttl: number): string {
  if (ttl === 1) return 'Auto'
  if (ttl < 60) return `${ttl}s`
  if (ttl < 3600) return `${Math.round(ttl / 60)}m`
  return `${Math.round(ttl / 3600)}h`
}

function formatDate(iso: string): string {
  if (!iso) return '—'
  try {
    return new Intl.DateTimeFormat('pl-PL', {
      dateStyle: 'short',
      timeStyle: 'short',
    }).format(new Date(iso))
  } catch {
    return iso
  }
}

function RecordTypeBadge({ type }: { type: string }) {
  const colors: Record<string, string> = {
    A: 'bg-blue-500/15 text-blue-400 border-blue-500/30',
    AAAA: 'bg-indigo-500/15 text-indigo-400 border-indigo-500/30',
    CNAME: 'bg-violet-500/15 text-violet-400 border-violet-500/30',
    MX: 'bg-amber-500/15 text-amber-400 border-amber-500/30',
    TXT: 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30',
    NS: 'bg-zinc-500/15 text-zinc-400 border-zinc-500/30',
    SRV: 'bg-rose-500/15 text-rose-400 border-rose-500/30',
    CAA: 'bg-orange-500/15 text-orange-400 border-orange-500/30',
  }
  const color = colors[type] ?? 'bg-zinc-500/15 text-zinc-400 border-zinc-500/30'
  return (
    <span
      className={cn(
        'inline-flex min-w-[3rem] items-center justify-center rounded border px-1.5 py-0.5 font-mono text-xs font-medium',
        color
      )}
    >
      {type}
    </span>
  )
}

export default function CloudflareDNSPage() {
  const [records, setRecords] = useState<CloudflareDNSRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [sortKey, setSortKey] = useState<SortKey>('name')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const loadRecords = useCallback(async (options?: { refresh?: boolean }) => {
    if (options?.refresh) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }
    setError(null)
    try {
      const data = await api.getCloudflareDNSRecords()
      setRecords(data ?? [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Nie udało się pobrać rekordów DNS.')
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  useEffect(() => {
    loadRecords()
  }, [loadRecords])

  const presentTypes = useMemo(
    () => Array.from(new Set(records.map((r) => r.type))).sort(),
    [records]
  )

  const filtered = useMemo(() => {
    const base = typeFilter === 'all' ? records : records.filter((r) => r.type === typeFilter)
    return [...base].sort((a, b) => {
      const valA = sortKey === 'name' ? a.name : a.type
      const valB = sortKey === 'name' ? b.name : b.type
      const cmp = valA.localeCompare(valB)
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [records, typeFilter, sortKey, sortDir])

  function toggleSort(key: SortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-3">
          <Button asChild variant="ghost" size="icon" className="size-8 shrink-0">
            <Link href="/services/cloudflare">
              <ArrowLeftIcon className="size-4" />
            </Link>
          </Button>
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">Rekordy DNS</h1>
            <p className="text-xs text-zinc-500">Cloudflare — zarządzanie strefą DNS</p>
          </div>
        </div>
        <div className="flex items-center gap-2 self-start sm:self-auto">
          <Button
            variant="outline"
            size="sm"
            disabled={loading || refreshing}
            onClick={() => loadRecords({ refresh: true })}
            className="gap-1.5"
          >
            <RefreshCwIcon className={cn('size-3.5', refreshing && 'animate-spin')} />
            Odśwież
          </Button>
          <Button asChild variant="ghost" size="sm" className="gap-1.5">
            <Link href="/settings/services">
              <Settings2Icon className="size-3.5" />
              Ustawienia
            </Link>
          </Button>
        </div>
      </div>

      {/* Loading state */}
      {loading && (
        <div className="animate-pulse space-y-3 rounded-xl border border-zinc-800 bg-zinc-900/60 p-5">
          <div className="h-4 w-24 rounded-full bg-zinc-800" />
          <div className="h-8 w-full rounded-lg bg-zinc-800" />
          <div className="h-8 w-full rounded-lg bg-zinc-800" />
          <div className="h-8 w-full rounded-lg bg-zinc-800" />
        </div>
      )}

      {/* Error state */}
      {!loading && error && (
        <div className="rounded-xl border border-red-900/70 bg-red-950/40 p-5">
          <p className="text-sm font-medium text-red-300">Błąd ładowania rekordów DNS</p>
          <p className="mt-1 text-sm text-red-400/80">{error}</p>
          <div className="mt-4 flex gap-2">
            <Button size="sm" variant="outline" onClick={() => loadRecords()}>
              Spróbuj ponownie
            </Button>
            <Button asChild size="sm" variant="ghost">
              <Link href="/settings/services">Sprawdź konfigurację</Link>
            </Button>
          </div>
        </div>
      )}

      {/* Content */}
      {!loading && !error && (
        <>
          {/* Type filter */}
          <div className="flex flex-wrap items-center gap-1.5">
            <button
              onClick={() => setTypeFilter('all')}
              className={cn(
                'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
                typeFilter === 'all'
                  ? 'border-zinc-600 bg-zinc-700 text-zinc-100'
                  : 'border-zinc-800 bg-transparent text-zinc-400 hover:border-zinc-700 hover:text-zinc-300'
              )}
            >
              Wszystkie
              <span className="ml-1.5 text-zinc-500">{records.length}</span>
            </button>
            {presentTypes.map((type) => {
              const count = records.filter((r) => r.type === type).length
              return (
                <button
                  key={type}
                  onClick={() => setTypeFilter(type === typeFilter ? 'all' : type)}
                  className={cn(
                    'rounded-full border px-3 py-1 text-xs font-medium transition-colors',
                    typeFilter === type
                      ? 'border-zinc-600 bg-zinc-700 text-zinc-100'
                      : 'border-zinc-800 bg-transparent text-zinc-400 hover:border-zinc-700 hover:text-zinc-300'
                  )}
                >
                  {type}
                  <span className="ml-1.5 text-zinc-500">{count}</span>
                </button>
              )
            })}
          </div>

          {/* Records count */}
          <p className="text-xs text-zinc-500">
            {typeFilter === 'all'
              ? `${records.length} rekordów`
              : `${filtered.length} z ${records.length} rekordów (filtr: ${typeFilter})`}
          </p>

          {/* Table */}
          {filtered.length === 0 ? (
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/60 p-8 text-center text-sm text-zinc-400">
              Brak rekordów{typeFilter !== 'all' ? ` typu ${typeFilter}` : ''}.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-zinc-800">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-zinc-800 bg-zinc-900/80">
                    <th className="px-4 py-3 text-left">
                      <button
                        onClick={() => toggleSort('type')}
                        className="flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-zinc-400 hover:text-zinc-200"
                      >
                        Typ
                        <ArrowUpDownIcon className={cn('size-3', sortKey === 'type' && 'text-zinc-300')} />
                      </button>
                    </th>
                    <th className="px-4 py-3 text-left">
                      <button
                        onClick={() => toggleSort('name')}
                        className="flex items-center gap-1 text-xs font-medium uppercase tracking-wider text-zinc-400 hover:text-zinc-200"
                      >
                        Nazwa
                        <ArrowUpDownIcon className={cn('size-3', sortKey === 'name' && 'text-zinc-300')} />
                      </button>
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-zinc-400">
                      Wartość
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-zinc-400">
                      TTL
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-zinc-400">
                      Proxy
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-zinc-400">
                      Zmodyfikowano
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/60">
                  {filtered.map((record) => (
                    <tr
                      key={record.id}
                      className="bg-zinc-900/40 transition-colors hover:bg-zinc-900/80"
                    >
                      <td className="px-4 py-3">
                        <RecordTypeBadge type={record.type} />
                      </td>
                      <td className="max-w-[200px] truncate px-4 py-3 font-mono text-xs text-zinc-200">
                        {record.name}
                      </td>
                      <td className="max-w-[240px] truncate px-4 py-3 font-mono text-xs text-zinc-400">
                        {record.content}
                      </td>
                      <td className="px-4 py-3 text-xs text-zinc-400">
                        {formatTTL(record.ttl)}
                      </td>
                      <td className="px-4 py-3">
                        {record.proxied ? (
                          <span title="Proxied przez Cloudflare">
                            <CloudIcon className="size-4 text-orange-400" />
                          </span>
                        ) : (
                          <span className="text-xs text-zinc-600">—</span>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-4 py-3 text-xs text-zinc-500">
                        {formatDate(record.modified_on)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  )
}
