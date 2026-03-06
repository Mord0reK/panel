import {
  formatBytes,
  formatBytesPerSec,
  formatPercent,
  formatTimestamp,
  formatRelativeTime,
} from './formatters'

// =============================================================================
// formatBytes
// =============================================================================

describe('formatBytes', () => {
  it('zwraca "0 B" dla 0', () => {
    expect(formatBytes(0)).toBe('0 B')
  })

  it('zwraca "0 B" dla wartości ujemnych', () => {
    expect(formatBytes(-100)).toBe('0 B')
  })

  it('zwraca "0 B" dla NaN', () => {
    expect(formatBytes(NaN)).toBe('0 B')
  })

  it('zwraca "0 B" dla Infinity', () => {
    expect(formatBytes(Infinity)).toBe('0 B')
  })

  it('formatuje bajty (< 1 KB)', () => {
    expect(formatBytes(512)).toBe('512 B')
  })

  it('formatuje kilobajty', () => {
    expect(formatBytes(1024)).toBe('1 KB')
  })

  it('formatuje megabajty', () => {
    expect(formatBytes(1024 * 1024)).toBe('1 MB')
  })

  it('formatuje gigabajty', () => {
    expect(formatBytes(1024 * 1024 * 1024)).toBe('1 GB')
  })

  it('formatuje terabajty', () => {
    expect(formatBytes(1024 ** 4)).toBe('1 TB')
  })

  it('zwraca 2 miejsca po przecinku domyślnie', () => {
    expect(formatBytes(1536)).toBe('1.5 KB')
  })

  it('respektuje parametr decimals', () => {
    expect(formatBytes(1536, 0)).toBe('2 KB')
    expect(formatBytes(1536, 1)).toBe('1.5 KB')
    expect(formatBytes(1536, 3)).toBe('1.5 KB')
  })

  it('formatuje realistyczną wartość RAM (16 GB)', () => {
    expect(formatBytes(16 * 1024 ** 3)).toBe('16 GB')
  })
})

// =============================================================================
// formatBytesPerSec
// =============================================================================

describe('formatBytesPerSec', () => {
  it('zwraca "0 B/s" dla 0', () => {
    expect(formatBytesPerSec(0)).toBe('0 B/s')
  })

  it('zwraca "0 B/s" dla wartości ujemnych', () => {
    expect(formatBytesPerSec(-1)).toBe('0 B/s')
  })

  it('zwraca "0 B/s" dla NaN', () => {
    expect(formatBytesPerSec(NaN)).toBe('0 B/s')
  })

  it('zwraca "0 B/s" dla Infinity', () => {
    expect(formatBytesPerSec(Infinity)).toBe('0 B/s')
  })

  it('formatuje B/s', () => {
    expect(formatBytesPerSec(500)).toBe('500 B/s')
  })

  it('formatuje KB/s', () => {
    expect(formatBytesPerSec(1024)).toBe('1 KB/s')
  })

  it('formatuje MB/s', () => {
    expect(formatBytesPerSec(1024 * 1024)).toBe('1 MB/s')
  })

  it('formatuje GB/s', () => {
    expect(formatBytesPerSec(1024 ** 3)).toBe('1 GB/s')
  })

  it('respektuje parametr decimals', () => {
    expect(formatBytesPerSec(1536, 0)).toBe('2 KB/s')
  })
})

// =============================================================================
// formatPercent
// =============================================================================

describe('formatPercent', () => {
  it('formatuje 0%', () => {
    expect(formatPercent(0)).toBe('0.00%')
  })

  it('formatuje 100%', () => {
    expect(formatPercent(100)).toBe('100.00%')
  })

  it('formatuje wartość z ułamkiem', () => {
    expect(formatPercent(12.5)).toBe('12.50%')
  })

  it('zaokrągla do 2 miejsc po przecinku', () => {
    expect(formatPercent(33.333)).toBe('33.33%')
  })

  it('respektuje parametr decimals = 0', () => {
    expect(formatPercent(12.5, 0)).toBe('13%')
  })

  it('respektuje parametr decimals = 1', () => {
    expect(formatPercent(12.567, 1)).toBe('12.6%')
  })

  it('zwraca "0%" dla NaN', () => {
    expect(formatPercent(NaN)).toBe('0%')
  })

  it('zwraca "0%" dla Infinity', () => {
    expect(formatPercent(Infinity)).toBe('0%')
  })
})

// =============================================================================
// formatTimestamp
// =============================================================================

describe('formatTimestamp', () => {
  // Używamy stałego timestampa i sprawdzamy FORMAT przez regex
  // Nie sprawdzamy dokładnych wartości — zależą od lokalnej strefy czasowej

  const fixedTimestamp = 1739967600 // przykładowy timestamp

  const PATTERN_HMS = /^\d{2}:\d{2}:\d{2}$/ // HH:mm:ss
  const PATTERN_HM = /^\d{2}:\d{2}$/ // HH:mm
  const PATTERN_MDHM = /^\d{2}-\d{2} \d{2}:\d{2}$/ // MM-DD HH:mm

  it('bez podanego range zwraca format HH:mm:ss', () => {
    expect(formatTimestamp(fixedTimestamp)).toMatch(PATTERN_HMS)
  })

  it('range "1m" → format HH:mm:ss', () => {
    expect(formatTimestamp(fixedTimestamp, '1m')).toMatch(PATTERN_HMS)
  })

  it('range "5m" → format HH:mm:ss', () => {
    expect(formatTimestamp(fixedTimestamp, '5m')).toMatch(PATTERN_HMS)
  })

  it('range "15m" → format HH:mm:ss', () => {
    expect(formatTimestamp(fixedTimestamp, '15m')).toMatch(PATTERN_HMS)
  })

  it('range "30m" → format HH:mm:ss', () => {
    expect(formatTimestamp(fixedTimestamp, '30m')).toMatch(PATTERN_HMS)
  })

  it('range "1h" → format HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '1h')).toMatch(PATTERN_HM)
  })

  it('range "6h" → format HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '6h')).toMatch(PATTERN_HM)
  })

  it('range "12h" → format HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '12h')).toMatch(PATTERN_HM)
  })

  it('range "24h" → format HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '24h')).toMatch(PATTERN_HM)
  })

  it('range "7d" → format MM-DD HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '7d')).toMatch(PATTERN_MDHM)
  })

  it('range "15d" → format MM-DD HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '15d')).toMatch(PATTERN_MDHM)
  })

  it('range "30d" → format MM-DD HH:mm', () => {
    expect(formatTimestamp(fixedTimestamp, '30d')).toMatch(PATTERN_MDHM)
  })

  it('zwraca spójne wartości dla tego samego wejścia', () => {
    const a = formatTimestamp(fixedTimestamp, '1h')
    const b = formatTimestamp(fixedTimestamp, '1h')
    expect(a).toBe(b)
  })
})

// =============================================================================
// formatRelativeTime
// =============================================================================

describe('formatRelativeTime', () => {
  let dateNowSpy: jest.SpyInstance<number, []>

  beforeEach(() => {
    dateNowSpy = jest
      .spyOn(Date, 'now')
      .mockReturnValue(new Date('2026-02-20T12:00:00.000Z').getTime())
  })

  afterEach(() => {
    dateNowSpy.mockRestore()
  })

  it('zwraca "przed chwilą" dla < 10s temu', () => {
    const date = new Date('2026-02-20T11:59:55.000Z').toISOString()
    expect(formatRelativeTime(date)).toBe('przed chwilą')
  })

  it('zwraca Xs temu dla 10s–59s temu', () => {
    const date = new Date('2026-02-20T11:59:30.000Z').toISOString()
    expect(formatRelativeTime(date)).toBe('30s temu')
  })

  it('zwraca Xm temu dla 1–59 minut temu', () => {
    const date = new Date('2026-02-20T11:45:00.000Z').toISOString()
    expect(formatRelativeTime(date)).toBe('15m temu')
  })

  it('zwraca Xh temu dla 1–23 godzin temu', () => {
    const date = new Date('2026-02-20T09:00:00.000Z').toISOString()
    expect(formatRelativeTime(date)).toBe('3h temu')
  })

  it('zwraca Xd temu dla >= 24 godzin temu', () => {
    const date = new Date('2026-02-18T12:00:00.000Z').toISOString()
    expect(formatRelativeTime(date)).toBe('2d temu')
  })
})
