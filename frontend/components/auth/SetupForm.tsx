'use client'

import { useState, type FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'

export default function SetupForm() {
  const router = useRouter()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)

    if (username.length < 3) {
      setError('Nazwa użytkownika musi mieć minimum 3 znaki.')
      return
    }
    if (password.length < 8) {
      setError('Hasło musi mieć minimum 8 znaków.')
      return
    }

    setLoading(true)

    try {
      await api.setup({ username, password })
      router.push('/dashboard')
      router.refresh()
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 403) {
          setError('Konto administracyjne już istnieje.')
        } else if (err.status === 400) {
          setError('Nazwa użytkownika min. 3 znaki, hasło min. 8 znaków.')
        } else {
          setError('Błąd podczas konfiguracji. Spróbuj ponownie.')
        }
      } else {
        setError('Nie można połączyć się z serwerem.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-card-foreground mb-2">
        Pierwsze uruchomienie
      </h2>
      <p className="text-sm text-muted-foreground mb-6">
        Utwórz konto administratora, aby rozpocząć korzystanie z panelu.
      </p>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="username">Nazwa użytkownika</Label>
          <Input
            id="username"
            type="text"
            autoComplete="username"
            required
            minLength={3}
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin"
            disabled={loading}
            data-testid="setup-username"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Hasło</Label>
          <Input
            id="password"
            type="password"
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="minimum 8 znaków"
            disabled={loading}
            data-testid="setup-password"
          />
        </div>
        {error && (
          <p
            className="text-sm text-destructive"
            role="alert"
            data-testid="setup-error"
          >
            {error}
          </p>
        )}
        <Button
          type="submit"
          className="w-full"
          disabled={loading}
          data-testid="setup-submit"
        >
          {loading ? 'Konfigurowanie…' : 'Utwórz konto'}
        </Button>
      </form>
    </div>
  )
}
