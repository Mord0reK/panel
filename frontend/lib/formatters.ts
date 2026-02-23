import type { MetricRange } from '@/types'

/**
 * Formatuje bajty na czytelną jednostkę (B, KB, MB, GB, TB).
 * @param bytes  - wartość w bajtach
 * @param decimals - liczba miejsc po przecinku (domyślnie 2)
 */
export function formatBytes(bytes: number, decimals = 2): string {
  if (!isFinite(bytes) || bytes < 0) return '0 B'
  if (bytes === 0) return '0 B'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))
  const index = Math.min(i, sizes.length - 1)

  return `${parseFloat((bytes / Math.pow(k, index)).toFixed(dm))} ${sizes[index]}`
}

/**
 * Formatuje bajty na sekundę (B/s, KB/s, MB/s, GB/s).
 * @param bytesPerSec - wartość w bajtach/s
 * @param decimals    - liczba miejsc po przecinku (domyślnie 2)
 */
export function formatBytesPerSec(bytesPerSec: number, decimals = 2): string {
  if (!isFinite(bytesPerSec) || bytesPerSec < 0) return '0 B/s'
  if (bytesPerSec === 0) return '0 B/s'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s', 'TB/s']

  const i = Math.floor(Math.log(bytesPerSec) / Math.log(k))
  const index = Math.min(i, sizes.length - 1)

  return `${parseFloat((bytesPerSec / Math.pow(k, index)).toFixed(dm))} ${sizes[index]}`
}

/**
 * Formatuje bajty/s na megabity/s (Mbit/s).
 * @param bytesPerSec - wartość w bajtach/s
 * @param decimals    - liczba miejsc po przecinku (domyślnie 2)
 */
export function formatBitsPerSec(bytesPerSec: number, decimals = 2): string {
  if (!isFinite(bytesPerSec) || bytesPerSec < 0) return '0 Mbit/s'
  if (bytesPerSec === 0) return '0 Mbit/s'

  const mbits = (bytesPerSec * 8) / 1_000_000

  if (mbits < 1) {
    const kbits = (bytesPerSec * 8) / 1000
    return `${parseFloat(kbits.toFixed(decimals))} Kbit/s`
  }

  return `${parseFloat(mbits.toFixed(decimals))} Mbit/s`
}

/**
 * Formatuje wartość procentową z zadaną liczbą miejsc po przecinku.
 * Zwraca np. "12.50%".
 */
export function formatPercent(value: number, decimals = 2): string {
  if (!isFinite(value)) return '0%'
  return `${value.toFixed(decimals)}%`
}

/**
 * Formatuje Unix timestamp (sekundy) na czytelny string.
 * Format zależy od zakresu metryk:
 * - `1m`              → HH:mm:ss
 * - `5m`–`30m`        → HH:mm:ss
 * - `1h`–`24h`        → HH:mm
 * - `7d`–`30d`        → MM-DD HH:mm
 * - undefined / inny  → HH:mm:ss
 */
export function formatTimestamp(
  unixSeconds: number,
  range?: MetricRange,
): string {
  const date = new Date(unixSeconds * 1000)

  const pad = (n: number) => String(n).padStart(2, '0')
  const hms = `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  const hm = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  const mdhm = `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${hm}`

  if (!range) return hms

  switch (range) {
    case '1m':
    case '5m':
    case '15m':
    case '30m':
      return hms
    case '1h':
    case '6h':
    case '12h':
    case '24h':
      return hm
    case '7d':
    case '15d':
    case '30d':
      return mdhm
    default:
      return hms
  }
}

/**
 * Formatuje względny czas (np. "2 minuty temu") na podstawie daty ISO 8601.
 */
export function formatRelativeTime(isoDate: string): string {
  const date = new Date(isoDate)
  const now = Date.now()
  const diffMs = now - date.getTime()
  const diffSec = Math.floor(diffMs / 1000)

  if (diffSec < 10) return 'przed chwilą'
  if (diffSec < 60) return `${diffSec}s temu`

  const diffMin = Math.floor(diffSec / 60)
  if (diffMin < 60) return `${diffMin}m temu`

  const diffH = Math.floor(diffMin / 60)
  if (diffH < 24) return `${diffH}h temu`

  const diffD = Math.floor(diffH / 24)
  return `${diffD}d temu`
}
