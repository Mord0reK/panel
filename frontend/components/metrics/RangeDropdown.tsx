'use client'

import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { MetricRange } from '@/types'

interface RangeDropdownProps {
  value: MetricRange
  onChange: (value: MetricRange) => void
}

const GROUPS: { label: string; items: { value: MetricRange; label: string }[] }[] = [
  {
    label: 'Na żywo',
    items: [{ value: '1m', label: '1 minuta (Na żywo)' }],
  },
  {
    label: 'Minuty',
    items: [
      { value: '5m', label: '5 minut' },
      { value: '15m', label: '15 minut' },
      { value: '30m', label: '30 minut' },
    ],
  },
  {
    label: 'Godziny',
    items: [
      { value: '1h', label: '1 godzina' },
      { value: '6h', label: '6 godzin' },
      { value: '12h', label: '12 godzin' },
      { value: '24h', label: '24 godziny' },
    ],
  },
  {
    label: 'Dni',
    items: [
      { value: '7d', label: '7 dni' },
      { value: '15d', label: '15 dni' },
      { value: '30d', label: '30 dni' },
    ],
  },
]

export function RangeDropdown({ value, onChange }: RangeDropdownProps) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as MetricRange)}>
      <SelectTrigger className="w-44">
        <SelectValue>
          {value === '1m' ? (
            <span className="flex items-center gap-2">
              1 minuta (Na żywo)
              <span className="inline-block size-2 rounded-full bg-red-500 animate-pulse" />
            </span>
          ) : (
            GROUPS.flatMap((g) => g.items).find((i) => i.value === value)?.label
          )}
        </SelectValue>
      </SelectTrigger>
      <SelectContent>
        {GROUPS.map((group) => (
          <SelectGroup key={group.label}>
            <SelectLabel>{group.label}</SelectLabel>
            {group.items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.value === '1m' ? (
                  <span className="flex items-center gap-2">
                    {item.label}
                    <span className="inline-block size-2 rounded-full bg-red-500 animate-pulse" />
                  </span>
                ) : (
                  item.label
                )}
              </SelectItem>
            ))}
          </SelectGroup>
        ))}
      </SelectContent>
    </Select>
  )
}
