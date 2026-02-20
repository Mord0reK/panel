'use client'

import { useState, type FormEvent } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api, ApiError } from '@/lib/api'

export default function LoginForm() {
  const router = useRouter()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      await api.login({ username, password })
      router.push('/dashboard')
      router.refresh()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.status === 401 ? 'Nieprawidłowa nazwa użytkownika lub hasło.' : 'Błąd logowania. Spróbuj ponownie.')
      } else {
        setError('Nie można połączyć się z serwerem.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card p-6 shadow-sm">
      <h2 className="text-lg font-semibold text-card-foreground mb-6">Logowanie</h2>
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="username">Nazwa użytkownika</Label>
          <Input
            id="username"
            type="text"
            autoComplete="username"
            required
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="admin"
            disabled={loading}
            data-testid="login-username"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="password">Hasło</Label>
          <Input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            placeholder="••••••••"
            disabled={loading}
            data-testid="login-password"
          />
        </div>
        {error && (
          <p
            className="text-sm text-destructive"
            role="alert"
            data-testid="login-error"
          >
            {error}
          </p>
        )}
        <Button
          type="submit"
          className="w-full"
          disabled={loading}
          data-testid="login-submit"
        >
          {loading ? 'Logowanie…' : 'Zaloguj się'}
        </Button>
      </form>
    </div>
  )
}
