import type { ReactNode } from 'react'
import {
  ActivityIcon,
  BanIcon,
  GlobeIcon,
  RadarIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  ShieldOffIcon,
  SparklesIcon,
  UsersIcon,
  WandSparklesIcon,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { AdGuardHomeDashboardResponse, AdGuardTopItem } from '@/types'

interface AdGuardHomeDashboardProps {
  serviceName: string
  dashboard: AdGuardHomeDashboardResponse
}

function formatCount(value: number): string {
  return new Intl.NumberFormat('pl-PL').format(value)
}

function formatMilliseconds(value: number): string {
  if (!Number.isFinite(value) || value <= 0) {
    return '—'
  }

  return `${value.toFixed(2)} ms`
}

function formatRatio(numerator: number, denominator: number): string {
  if (
    !Number.isFinite(numerator) ||
    !Number.isFinite(denominator) ||
    denominator <= 0
  ) {
    return '0.00%'
  }

  return `${((numerator / denominator) * 100).toFixed(2)}%`
}

function SeriesBars({
  title,
  values,
  accentClassName,
  testId,
}: {
  title: string
  values: number[]
  accentClassName: string
  testId: string
}) {
  const maxValue = Math.max(...values, 1)

  return (
    <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-medium text-zinc-200">{title}</h3>
        <span className="text-xs text-zinc-500">{values.length} pkt</span>
      </div>
      <div
        data-testid={testId}
        className="flex h-24 items-end gap-0.5 rounded-lg bg-zinc-950/60 px-2 py-2"
      >
        {values.length > 0 ? (
          values.map((value, index) => {
            const height = `${Math.max((value / maxValue) * 100, value > 0 ? 12 : 4)}%`
            return (
              <div
                key={`${title}-${index}`}
                className={cn('flex-1 rounded-sm opacity-80', accentClassName)}
                style={{ height }}
              />
            )
          })
        ) : (
          <div className="flex w-full items-center justify-center text-xs text-zinc-600">
            Brak danych
          </div>
        )}
      </div>
    </section>
  )
}

function RankingList({
  title,
  icon,
  items,
  testId,
}: {
  title: string
  icon: ReactNode
  items: AdGuardTopItem[]
  testId: string
}) {
  return (
    <section className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
      <div className="mb-3 flex items-center gap-2">
        <span className="text-zinc-400">{icon}</span>
        <h3 className="text-sm font-medium text-zinc-200">{title}</h3>
      </div>
      <div className="space-y-1" data-testid={testId}>
        {items.length > 0 ? (
          items.slice(0, 5).map((item, index) => (
            <div
              key={`${title}-${item.name}-${index}`}
              className="flex items-center justify-between gap-3 rounded-lg px-2 py-1.5 hover:bg-zinc-800/50"
            >
              <div className="flex min-w-0 items-center gap-2">
                <span className="w-4 shrink-0 text-right text-xs text-zinc-600">
                  {index + 1}.
                </span>
                <p className="truncate text-sm text-zinc-300">{item.name}</p>
              </div>
              <span className="shrink-0 text-sm tabular-nums text-zinc-400">
                {formatCount(item.count)}
              </span>
            </div>
          ))
        ) : (
          <div className="py-4 text-center text-xs text-zinc-600">
            Brak danych
          </div>
        )}
      </div>
    </section>
  )
}

export function AdGuardHomeDashboard({
  serviceName,
  dashboard,
}: AdGuardHomeDashboardProps) {
  const blockedRatio = formatRatio(
    dashboard.stats.num_blocked_filtering,
    dashboard.stats.num_dns_queries
  )

  const hasUpdate =
    dashboard.version.update_available && dashboard.version.latest_version

  return (
    <div
      data-testid="service-dashboard-adguardhome"
      className="space-y-4"
    >
      {/* Status bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge
            className={cn(
              'gap-1.5 border',
              dashboard.status.protection_enabled
                ? 'border-emerald-500/30 bg-emerald-500/10 text-emerald-300'
                : 'border-red-500/25 bg-red-500/10 text-red-300'
            )}
          >
            {dashboard.status.protection_enabled ? (
              <ShieldCheckIcon className="size-3" />
            ) : (
              <ShieldOffIcon className="size-3" />
            )}
            {dashboard.status.protection_enabled
              ? 'Aktywna ochrona'
              : 'Ochrona wyłączona'}
          </Badge>
          <Badge
            className={cn(
              'border',
              dashboard.status.running
                ? 'border-cyan-500/30 bg-cyan-500/10 text-cyan-300'
                : 'border-zinc-700 bg-zinc-800 text-zinc-400'
            )}
          >
            {dashboard.status.running ? 'DNS aktywny' : 'DNS zatrzymany'}
          </Badge>
        </div>
        <div
          data-testid="adguard-version-badge"
          className="flex flex-col items-end gap-0.5"
        >
          <div className="flex items-center gap-2 text-xs">
            <span className="text-zinc-400">{serviceName}</span>
            {hasUpdate ? (
              <Badge className="border border-amber-500/30 bg-amber-500/10 text-amber-300">
                Dostępna aktualizacja
              </Badge>
            ) : (
              <span className="text-zinc-600">Aktualna wersja</span>
            )}
          </div>
          {dashboard.version.current_version && (
            <p className="text-xs text-zinc-600">
              {hasUpdate
                ? `${dashboard.version.current_version} → ${dashboard.version.latest_version}`
                : `Instancja działa na wersji ${dashboard.version.current_version}`}
            </p>
          )}
        </div>
      </div>

      {/* Kluczowe statystyki */}
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs text-zinc-500">Zapytania DNS</p>
            <ActivityIcon className="size-3.5 text-cyan-400" />
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatCount(dashboard.stats.num_dns_queries)}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs text-zinc-500">Zablokowane</p>
            <BanIcon className="size-3.5 text-emerald-400" />
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatCount(dashboard.stats.num_blocked_filtering)}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs text-zinc-500">Skuteczność</p>
            <ShieldAlertIcon className="size-3.5 text-emerald-400" />
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {blockedRatio}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center justify-between">
            <p className="text-xs text-zinc-500">Śr. czas odpowiedzi</p>
            <RadarIcon className="size-3.5 text-cyan-400" />
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatMilliseconds(dashboard.stats.avg_processing_time)}
          </p>
        </div>
      </section>

      {/* Wykresy */}
      <section className="grid gap-3 lg:grid-cols-2">
        <SeriesBars
          title="Ruch DNS"
          values={dashboard.stats.dns_queries}
          accentClassName="bg-cyan-500"
          testId="adguard-series-dns"
        />
        <SeriesBars
          title="Blokowanie"
          values={dashboard.stats.blocked_filtering}
          accentClassName="bg-emerald-500"
          testId="adguard-series-blocked"
        />
        <SeriesBars
          title="Safe Browsing"
          values={dashboard.stats.replaced_safebrowsing}
          accentClassName="bg-amber-500"
          testId="adguard-series-safe-browsing"
        />
        <SeriesBars
          title="Kontrola rodzicielska"
          values={dashboard.stats.replaced_parental}
          accentClassName="bg-fuchsia-500"
          testId="adguard-series-parental"
        />
      </section>

      {/* Podmiany ochronne */}
      <section className="grid gap-3 sm:grid-cols-3">
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center gap-2">
            <WandSparklesIcon className="size-3.5 text-emerald-400" />
            <p className="text-xs text-zinc-500">Safe Search</p>
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatCount(dashboard.stats.num_replaced_safesearch)}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center gap-2">
            <SparklesIcon className="size-3.5 text-amber-400" />
            <p className="text-xs text-zinc-500">Safe Browsing</p>
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatCount(dashboard.stats.num_replaced_safebrowsing)}
          </p>
        </div>
        <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
          <div className="mb-2 flex items-center gap-2">
            <ShieldAlertIcon className="size-3.5 text-fuchsia-400" />
            <p className="text-xs text-zinc-500">Kontrola rodzicielska</p>
          </div>
          <p className="text-2xl font-semibold tabular-nums text-zinc-100">
            {formatCount(dashboard.stats.num_replaced_parental)}
          </p>
        </div>
      </section>

      {/* Rankingi */}
      <section className="grid gap-3 xl:grid-cols-3">
        <RankingList
          title="Najaktywniejsi klienci"
          icon={<UsersIcon className="size-3.5" />}
          items={dashboard.stats.top_clients}
          testId="adguard-top-clients"
        />
        <RankingList
          title="Najczęściej pytane domeny"
          icon={<GlobeIcon className="size-3.5" />}
          items={dashboard.stats.top_queried_domains}
          testId="adguard-top-domains"
        />
        <RankingList
          title="Najczęściej blokowane"
          icon={<BanIcon className="size-3.5" />}
          items={dashboard.stats.top_blocked_domains}
          testId="adguard-top-blocked-domains"
        />
      </section>
    </div>
  )
}