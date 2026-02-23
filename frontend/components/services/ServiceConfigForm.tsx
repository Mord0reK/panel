'use client'

import { useState } from 'react'
import { Service } from '@/hooks/useServices'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

interface Props {
  service: Service
  initialConfig: Record<string, string>
  onSave: () => void
}

export function ServiceConfigForm({ service, initialConfig, onSave }: Props) {
  const [config, setConfig] = useState(initialConfig)
  const [isEnabled, setIsEnabled] = useState(service.is_enabled)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)
    setSuccess(false)
    try {
      const res = await fetch(`/api/services/${service.slug}/config`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ config, is_enabled: isEnabled }),
      })
      if (!res.ok) {
        const text = await res.text()
        throw new Error(text || 'Failed to save configuration')
      }
      setSuccess(true)
      onSave()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Błąd zapisu')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      <div className="flex items-center justify-between space-x-2 rounded-lg border p-4">
        <div className="space-y-0.5">
          <Label htmlFor="service-enabled">Włącz usługę</Label>
          <p className="text-sm text-muted-foreground">
            Aktywuje usługę w panelu i sidebarze.
          </p>
        </div>
        <input
          id="service-enabled"
          type="checkbox"
          className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
          checked={isEnabled}
          onChange={(e) => setIsEnabled(e.target.checked)}
        />
      </div>

      <div className="grid gap-4">
        {service.schema.map((field) => (
          <div key={field.name} className="grid gap-2">
            <Label htmlFor={field.name}>{field.label}</Label>
            <Input
              id={field.name}
              type={field.type === 'password' ? 'password' : 'text'}
              placeholder={field.description}
              required={field.required}
              value={config[field.name] || ''}
              onChange={(e) => setConfig({ ...config, [field.name]: e.target.value })}
            />
            {field.description && (
              <p className="text-xs text-muted-foreground">{field.description}</p>
            )}
          </div>
        ))}
      </div>

      {error && <p className="text-sm font-medium text-destructive">{error}</p>}
      {success && <p className="text-sm font-medium text-green-500">Konfiguracja zapisana pomyślnie!</p>}

      <Button type="submit" disabled={loading}>
        {loading ? 'Zapisywanie...' : 'Zapisz zmiany'}
      </Button>
    </form>
  )
}
