import type {
  AuthStatusResponse,
  CommandRequest,
  ContainerAction,
  ContainerHistoryResponse,
  CustomIcon,
  MetricRange,
  Server,
  ServerDetailResponse,
  ServerHistoryResponse,
} from '@/types'

/**
 * Błąd HTTP z kodem statusu.
 * Rzucany przez apiFetch gdy response.ok === false.
 */
export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)

  if (!res.ok) {
    const text = await res.text().catch(() => 'Unknown error')
    throw new ApiError(res.status, text)
  }

  return res.json() as Promise<T>
}

export const api = {
  // -------------------------------------------------------------------------
  // Auth
  // -------------------------------------------------------------------------

  login(body: { username: string; password: string }) {
    return apiFetch<{ success: boolean }>('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  },

  setup(body: { username: string; password: string }) {
    return apiFetch<{ success: boolean }>('/api/auth/setup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  },

  authStatus() {
    return apiFetch<AuthStatusResponse>('/api/auth/status')
  },

  // -------------------------------------------------------------------------
  // Servers
  // -------------------------------------------------------------------------

  getServers() {
    return apiFetch<Server[]>('/api/servers')
  },

  getServer(uuid: string) {
    return apiFetch<ServerDetailResponse>(`/api/servers/${uuid}`)
  },

  approveServer(uuid: string) {
    return apiFetch<{ success: boolean }>(`/api/servers/${uuid}/approve`, {
      method: 'PUT',
    })
  },

  patchServer(uuid: string, body: { display_name?: string; icon?: string; status?: 'active' | 'rejected' }) {
    return apiFetch<Server>(`/api/servers/${uuid}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  },

  getIcons() {
    return apiFetch<CustomIcon[]>('/api/icons')
  },

  deleteServer(uuid: string) {
    return apiFetch<{ success: boolean }>(`/api/servers/${uuid}`, {
      method: 'DELETE',
    })
  },

  serverCommand(uuid: string, body: CommandRequest) {
    return apiFetch<unknown>(`/api/servers/${uuid}/command`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  },

  containerCommand(uuid: string, containerId: string, action: ContainerAction) {
    return apiFetch<unknown>(
      `/api/servers/${uuid}/containers/${containerId}/command`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      },
    )
  },

  deleteContainer(uuid: string, containerId: string, password: string) {
    return apiFetch<{ success: boolean }>(
      `/api/servers/${uuid}/containers/${containerId}`,
      {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      },
    )
  },

  // -------------------------------------------------------------------------
  // Metrics — history
  // -------------------------------------------------------------------------

  getMetricsHistory(uuid: string, range: MetricRange) {
    return apiFetch<ServerHistoryResponse>(
      `/api/metrics/history/servers/${uuid}?range=${range}`,
    )
  },

  getContainerHistory(uuid: string, containerId: string, range: MetricRange) {
    return apiFetch<ContainerHistoryResponse>(
      `/api/metrics/history/servers/${uuid}/containers/${containerId}?range=${range}`,
    )
  },
}
