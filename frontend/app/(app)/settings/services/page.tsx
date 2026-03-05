'use client'

import { useCallback, useEffect, useMemo, useState } from 'react'
import Image from 'next/image'

import { SettingsNav } from '@/components/settings/SettingsNav'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { ServiceDefinition } from '@/types'

interface ServiceDraft {
  enabled: boolean
  baseUrl: string
  token: string
  username: string
  password: string
}

interface ServiceActionState {
  saving: boolean
  testing: boolean
  message?: string
  messageType?: 'success' | 'error'
}

const FALLBACK_SERVICE_ICON = '/icons/hetzner-h.svg'

function buildDraft(service: ServiceDefinition): ServiceDraft {
  return {
    enabled: service.enabled,
    baseUrl: service.requires_base_url ? '' : (service.fixed_base_url ?? ''),
    token: '',
    username: '',
    password: '',
  }
}

export default function SettingsServicesPage() {
  const [services, setServices] = useState<ServiceDefinition[]>([])
  const [drafts, setDrafts] = useState<Record<string, ServiceDraft>>({})
  const [originalDrafts, setOriginalDrafts] = useState<
    Record<string, ServiceDraft>
  >({})
  const [testingServices, setTestingServices] = useState<Set<string>>(new Set())
  const [globalSaving, setGlobalSaving] = useState(false)
  const [globalMessage, setGlobalMessage] = useState<{
    text: string
    type: 'success' | 'error'
  } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const hasChanges = useMemo(() => {
    for (const key of Object.keys(drafts)) {
      const draft = drafts[key]
      const original = originalDrafts[key]
      if (!original) continue
      if (
        draft.enabled !== original.enabled ||
        draft.baseUrl !== original.baseUrl ||
        (draft.token && draft.token !== original.token) ||
        (draft.username && draft.username !== original.username) ||
        (draft.password && draft.password !== original.password)
      ) {
        return true
      }
    }
    return false
  }, [drafts, originalDrafts])

  const loadServices = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const payload = await api.getServices()
      setServices(payload)
      const nextDrafts: Record<string, ServiceDraft> = {}
      await Promise.all(
        payload.map(async (service) => {
          if (!service.enabled) {
            nextDrafts[service.key] = buildDraft(service)
            return
          }
          try {
            const config = await api.getServiceConfig(service.key)
            nextDrafts[service.key] = {
              enabled: config.enabled,
              baseUrl: config.base_url ?? '',
              username: config.username ?? '',
              password: config.password ?? '',
              token: config.token ?? '',
            }
          } catch {
            nextDrafts[service.key] = buildDraft(service)
          }
        })
      )
      setDrafts(nextDrafts)
      setOriginalDrafts(nextDrafts)
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Nie udało się pobrać usług'
      )
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadServices()
  }, [loadServices])

  const sortedServices = useMemo(
    () =>
      [...services].sort((a, b) =>
        a.display_name.localeCompare(b.display_name, 'pl')
      ),
    [services]
  )

  function updateDraft(serviceKey: string, patch: Partial<ServiceDraft>) {
    setDrafts((prev) => ({
      ...prev,
      [serviceKey]: {
        ...(prev[serviceKey] ?? {
          enabled: false,
          baseUrl: '',
          token: '',
          username: '',
          password: '',
        }),
        ...patch,
      },
    }))
  }

  function buildConfigPayload(service: ServiceDefinition, draft: ServiceDraft) {
    const payload: {
      enabled: boolean
      base_url?: string
      token?: string
      username?: string
      password?: string
    } = {
      enabled: draft.enabled,
    }

    if (service.requires_base_url && draft.baseUrl.trim() !== '') {
      payload.base_url = draft.baseUrl.trim()
    }

    if (service.auth_type === 'token' && draft.token.trim() !== '') {
      payload.token = draft.token.trim()
    }

    if (service.auth_type === 'basic_auth') {
      if (draft.username.trim() !== '') {
        payload.username = draft.username.trim()
      }
      if (draft.password.trim() !== '') {
        payload.password = draft.password
      }
    }

    return payload
  }

  async function saveAll() {
    setGlobalSaving(true)
    setGlobalMessage(null)
    const changedServices = services.filter((s) => {
      const draft = drafts[s.key]
      const original = originalDrafts[s.key]
      if (!original || !draft) return false
      return (
        draft.enabled !== original.enabled ||
        draft.baseUrl !== original.baseUrl ||
        (draft.token && draft.token !== original.token) ||
        (draft.username && draft.username !== original.username) ||
        (draft.password && draft.password !== original.password)
      )
    })
    const results = await Promise.allSettled(
      changedServices.map((service) =>
        api.saveServiceConfig(
          service.key,
          buildConfigPayload(service, drafts[service.key])
        )
      )
    )
    const failed = results.filter((r) => r.status === 'rejected')
    setGlobalSaving(false)
    if (failed.length === 0) {
      setGlobalMessage({
        text: 'Wszystkie zmiany zostały zapisane.',
        type: 'success',
      })
      setOriginalDrafts({ ...drafts })
    } else {
      setGlobalMessage({
        text: `Nie udało się zapisać ${failed.length} usług.`,
        type: 'error',
      })
    }
  }

  async function handleTest(service: ServiceDefinition, draft: ServiceDraft) {
    setTestingServices((prev) => new Set(prev).add(service.key))
    try {
      await api.testServiceConfig(
        service.key,
        buildConfigPayload(service, draft)
      )
      setGlobalMessage({
        text: `${service.display_name}: Test połączenia zakończony powodzeniem.`,
        type: 'success',
      })
    } catch (err) {
      setGlobalMessage({
        text: `${service.display_name}: ${err instanceof Error ? err.message : 'Nie udało się przetestować połączenia.'}`,
        type: 'error',
      })
    } finally {
      setTestingServices((prev) => {
        const next = new Set(prev)
        next.delete(service.key)
        return next
      })
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold">Integracje usług</h1>
        <Button
          onClick={saveAll}
          disabled={!hasChanges || globalSaving}
          className="mt-2"
        >
          {globalSaving ? 'Zapisywanie…' : 'Zastosuj zmiany'}
        </Button>
        {globalMessage && (
          <p
            className={cn(
              'mt-2 text-sm',
              globalMessage.type === 'success'
                ? 'text-emerald-400'
                : 'text-red-400'
            )}
          >
            {globalMessage.text}
          </p>
        )}
        <p className="text-sm text-muted-foreground">
          Włączaj usługi i uzupełniaj dane dostępu na podstawie metadanych
          integracji.
        </p>
      </div>

      <SettingsNav />

      {loading && (
        <div className="text-sm text-zinc-400">Ładowanie integracji…</div>
      )}

      {error && (
        <div className="rounded-md border border-red-900 bg-red-950/50 p-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {!loading && !error && sortedServices.length === 0 && (
        <div className="rounded-md border border-zinc-800 bg-zinc-900/60 p-4 text-sm text-zinc-400">
          Brak zarejestrowanych integracji.
        </div>
      )}

      <div className="space-y-4">
        {sortedServices.map((service) => {
          const draft = drafts[service.key] ?? buildDraft(service)
          const isTesting = testingServices.has(service.key)
          const isExpanded = draft.enabled
          return (
            <section
              key={service.key}
              data-testid={`service-card-${service.key}`}
              className={cn(
                'space-y-4 rounded-lg border p-4 transition-colors',
                isExpanded
                  ? 'border-emerald-700/60 bg-zinc-900/80'
                  : 'border-zinc-800 bg-zinc-900/55'
              )}
            >
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  <div className="flex size-10 shrink-0 items-center justify-center overflow-hidden rounded-md border border-zinc-800 bg-zinc-950/80">
                    <Image
                      src={service.icon || FALLBACK_SERVICE_ICON}
                      alt={`${service.display_name} icon`}
                      width={28}
                      height={28}
                      className="size-7 object-contain"
                    />
                  </div>
                  <div>
                    <h2 className="text-base font-medium text-zinc-100">
                      {service.display_name}
                    </h2>
                    <p className="text-xs text-zinc-400">{service.key}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    role="switch"
                    aria-checked={draft.enabled}
                    onClick={() => {
                      updateDraft(service.key, { enabled: !draft.enabled })
                    }}
                    className={cn(
                      'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
                      draft.enabled ? 'bg-emerald-500' : 'bg-zinc-700'
                    )}
                    data-testid={`service-enabled-${service.key}`}
                  >
                    <span
                      className={cn(
                        'inline-block size-5 transform rounded-full bg-white transition-transform',
                        draft.enabled ? 'translate-x-5' : 'translate-x-1'
                      )}
                    />
                  </button>
                  <Label
                    className={cn(
                      'text-sm',
                      draft.enabled ? 'text-emerald-400' : 'text-zinc-300'
                    )}
                  >
                    {draft.enabled ? 'Włączona' : 'Wyłączona'}
                  </Label>
                </div>
              </div>

              {isExpanded ? (
                <>
                  <div className="grid gap-4 md:grid-cols-2">
                    {service.requires_base_url ? (
                      <div className="space-y-2 md:col-span-2">
                        <Label htmlFor={`base-url-${service.key}`}>
                          URL usługi
                        </Label>
                        <Input
                          id={`base-url-${service.key}`}
                          value={draft.baseUrl}
                          onChange={(event) => {
                            updateDraft(service.key, {
                              baseUrl: event.target.value,
                            })
                          }}
                          placeholder="https://twoja-usluga.local"
                          data-testid={`service-base-url-${service.key}`}
                        />
                      </div>
                    ) : (
                      <div className="space-y-2 md:col-span-2">
                        <Label>Stały endpoint API</Label>
                        <Input
                          value={service.fixed_base_url ?? '—'}
                          readOnly
                          disabled
                          data-testid={`service-fixed-url-${service.key}`}
                        />
                      </div>
                    )}

                    {service.auth_type === 'token' && (
                      <div className="space-y-2 md:col-span-2">
                        <Label htmlFor={`token-${service.key}`}>Token</Label>
                        <Input
                          id={`token-${service.key}`}
                          type="password"
                          autoComplete="off"
                          value={draft.token}
                          onChange={(event) => {
                            updateDraft(service.key, {
                              token: event.target.value,
                            })
                          }}
                          placeholder={
                            draft.token.startsWith('••') ? '' : 'Wklej token'
                          }
                          data-testid={`service-token-${service.key}`}
                        />
                      </div>
                    )}

                    {service.auth_type === 'basic_auth' && (
                      <>
                        <div className="space-y-2">
                          <Label htmlFor={`username-${service.key}`}>
                            Nazwa użytkownika
                          </Label>
                          <Input
                            id={`username-${service.key}`}
                            value={draft.username}
                            onChange={(event) => {
                              updateDraft(service.key, {
                                username: event.target.value,
                              })
                            }}
                            placeholder="admin"
                            data-testid={`service-username-${service.key}`}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor={`password-${service.key}`}>
                            Hasło
                          </Label>
                          <Input
                            id={`password-${service.key}`}
                            type="password"
                            autoComplete="off"
                            value={draft.password}
                            onChange={(event) => {
                              updateDraft(service.key, {
                                password: event.target.value,
                              })
                            }}
                            placeholder={
                              draft.password.startsWith('••') ? '' : '••••••••'
                            }
                            data-testid={`service-password-${service.key}`}
                          />
                        </div>
                      </>
                    )}
                  </div>

                  <div className="flex items-center gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={isTesting || globalSaving}
                      onClick={() => handleTest(service, draft)}
                      data-testid={`service-test-${service.key}`}
                    >
                      {isTesting ? 'Testowanie…' : 'Testuj połączenie'}
                    </Button>
                  </div>
                </>
              ) : (
                <div className="space-y-2">
                  <p className="text-xs text-zinc-500">
                    Usługa jest wyłączona. Włącz ją, aby rozwinąć konfigurację.
                  </p>
                </div>
              )}
            </section>
          )
        })}
      </div>
    </div>
  )
}
