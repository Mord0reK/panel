// =============================================================================
// AUTH
// =============================================================================

export interface AuthStatusResponse {
  setup_required: boolean
  authenticated: boolean
}

export interface AuthTokenResponse {
  token: string
}

// =============================================================================
// SERVERS
// =============================================================================

export interface Server {
  uuid: string
  hostname: string
  display_name?: string // empty or missing — use hostname as fallback
  icon?: string // empty or missing; prefix: "lucide:" or "custom:"
  status?: 'active' | 'rejected' // defaults to "active" when absent
  approved: boolean
  online?: boolean // computed by backend (last_seen < 30s), may be absent in old responses
  cpu_model: string
  cpu_cores: number
  cpu_threads: number
  memory_total: number
  platform: string
  kernel: string
  architecture: string
  last_seen: string // ISO 8601
  created_at: string // ISO 8601
}

export interface Container {
  id: number
  agent_uuid: string
  container_id: string
  name: string
  image: string
  project: string
  service: string
  state: string // e.g. "running", "exited", "paused", "unknown"
  health: string
  status: string
  first_seen: string // ISO 8601
  last_seen: string // ISO 8601
}

export interface ServerDetailResponse {
  server: Server
  containers: Container[]
}

// =============================================================================
// SERVER ICONS
// =============================================================================

export interface CustomIcon {
  name: string // filename e.g. "proxmox.svg"
  url: string // public path e.g. "/icons/proxmox.svg"
}

// =============================================================================
// SERVICES INTEGRATIONS
// =============================================================================

export type ServiceAuthType = 'token' | 'basic_auth'

export interface ServiceDefinition {
  key: string
  display_name: string
  icon: string
  enabled: boolean
  requires_base_url: boolean
  auth_type: ServiceAuthType
  fixed_base_url?: string
  endpoints: string[]
}

// =============================================================================
// COMMANDS
// =============================================================================

export type ContainerAction =
  | 'start'
  | 'stop'
  | 'restart'
  | 'update'
  | 'check-updates'

export interface CommandRequest {
  action: ContainerAction | string
  target?: string
}

// =============================================================================
// METRICS — LIVE (RAM buffer, range=1m)
// =============================================================================

/** Surowy punkt hosta z RAM buffera (range=1m) — snake_case */
export interface RawHostMetricPoint {
  timestamp: number
  cpu: number
  mem_used: number
  mem_percent: number
  disk_used: number
  disk_read_bytes_per_sec: number
  disk_write_bytes_per_sec: number
  net_rx_bytes_per_sec: number
  net_tx_bytes_per_sec: number
  disk_used_percent: number
}

/** Surowy punkt kontenera z RAM buffera (range=1m) */
export interface RawContainerMetricPoint {
  timestamp: number
  cpu: number
  mem_used: number
  disk_used: number
  net_rx_bytes: number
  net_tx_bytes: number
}

// =============================================================================
// METRICS — HISTORYCZNE (DB, range > 1m)
// =============================================================================

/** Zagregowany punkt hosta z DB (range > 1m) */
export interface AggregatedHostMetricPoint {
  timestamp: number
  cpu_avg: number
  cpu_min: number
  cpu_max: number
  mem_used_avg: number
  mem_used_min: number
  mem_used_max: number
  disk_read_bytes_per_sec_avg: number
  disk_write_bytes_per_sec_avg: number
  disk_write_bytes_per_sec_min: number
  disk_write_bytes_per_sec_max: number
  net_rx_bytes_per_sec_avg: number
  net_rx_bytes_per_sec_min: number
  net_rx_bytes_per_sec_max: number
  net_tx_bytes_per_sec_avg: number
  net_tx_bytes_per_sec_min: number
  net_tx_bytes_per_sec_max: number
  disk_used_percent_avg: number
  disk_used_percent_min: number
  disk_used_percent_max: number
}

/** Zagregowany punkt kontenera z DB (range > 1m) */
export interface AggregatedContainerMetricPoint {
  timestamp: number
  cpu_avg: number
  cpu_min: number
  cpu_max: number
  mem_avg: number
  mem_min: number
  mem_max: number
  net_rx_avg: number
  net_rx_min: number
  net_rx_max: number
  net_tx_avg: number
  net_tx_min: number
  net_tx_max: number
}

// =============================================================================
// METRICS — RESPONSE TYPES
// =============================================================================

export type HostMetricPoint = RawHostMetricPoint | AggregatedHostMetricPoint
export type ContainerMetricPoint =
  | RawContainerMetricPoint
  | AggregatedContainerMetricPoint

export interface ContainerHistory {
  container_id: string
  name: string
  image: string
  project: string
  service: string
  points: RawContainerMetricPoint[] | AggregatedContainerMetricPoint[]
}

/** Response z GET /api/metrics/history/servers/[uuid] */
export interface ServerHistoryResponse {
  host: {
    points: RawHostMetricPoint[] | AggregatedHostMetricPoint[]
  }
  containers: ContainerHistory[]
}

/** Response z GET /api/metrics/history/servers/[uuid]/containers/[id] */
export interface ContainerHistoryResponse {
  points: RawContainerMetricPoint[] | AggregatedContainerMetricPoint[]
}

// =============================================================================
// SSE — LIVE ALL (/api/metrics/live/all)
// =============================================================================

export interface LiveServerSnapshot {
  uuid: string
  hostname: string
  cpu: number
  memory: number // mem_used w bajtach
  mem_percent: number
  memory_total: number
  disk_used: number
  disk_used_percent: number
  disk_read_bytes_per_sec: number
  disk_write_bytes_per_sec: number
  net_rx_bytes_per_sec: number
  net_tx_bytes_per_sec: number
}

export interface LiveAllEvent {
  servers: LiveServerSnapshot[]
}

// =============================================================================
// SSE — LIVE SERVER (/api/metrics/live/servers/[uuid])
// UWAGA: pole `host` zwraca PascalCase (Go struct serializowany bezpośrednio)
// =============================================================================

/** Host z /live/servers/[uuid] — PascalCase z backendu */
export interface LiveServerHostRaw {
  Timestamp: number
  CPU: number
  MemUsed: number
  MemPercent: number
  DiskReadBytesPerSec: number
  DiskWriteBytesPerSec: number
  NetRxBytesPerSec: number
  NetTxBytesPerSec: number
  DiskUsed: number
  DiskUsedPercent: number
}

/** Kontener z /live/servers/[uuid] — PascalCase z backendu */
export interface LiveServerContainerRaw {
  ContainerID: string
  Timestamp: number
  CPU: number
  MemUsed: number
  MemPercent: number
  DiskUsed: number
  DiskPercent: number
  NetRx: number
  NetTx: number
  State: string
  Health: string
  Status: string
  Project: string
}

export interface LiveServerEvent {
  server_uuid: string
  timestamp: number
  host: LiveServerHostRaw
  containers: LiveServerContainerRaw[]
}

/** Znormalizowany host po mapowaniu PascalCase → snake_case */
export interface LiveServerHost {
  timestamp: number
  cpu: number
  mem_used: number
  mem_percent: number
  disk_read_bytes_per_sec: number
  disk_write_bytes_per_sec: number
  net_rx_bytes_per_sec: number
  net_tx_bytes_per_sec: number
  disk_used: number
  disk_used_percent: number
}

/** Znormalizowany kontener po mapowaniu PascalCase → snake_case */
export interface LiveServerContainer {
  container_id: string
  timestamp: number
  cpu: number
  mem_used: number
  mem_percent: number
  disk_used: number
  disk_percent: number
  net_rx: number
  net_tx: number
  state: string
  health: string
  status: string
  project: string
}

// =============================================================================
// HELPERS
// =============================================================================

/** Dostępne zakresy metryk */
export type MetricRange =
  | '1m'
  | '5m'
  | '15m'
  | '30m'
  | '1h'
  | '6h'
  | '12h'
  | '24h'
  | '7d'
  | '15d'
  | '30d'

export type MetricStat = 'min' | 'avg' | 'max'

/** Type guard — czy punkt jest surowy (range=1m) */
export function isRawHostPoint(
  point: HostMetricPoint
): point is RawHostMetricPoint {
  return 'cpu' in point && !('cpu_avg' in point)
}

export function isRawContainerPoint(
  point: ContainerMetricPoint
): point is RawContainerMetricPoint {
  return 'cpu' in point && !('cpu_avg' in point)
}

/** Normalizuje PascalCase hosta z SSE /live/servers na snake_case */
export function normalizeLiveHost(raw: LiveServerHostRaw): LiveServerHost {
  return {
    timestamp: raw.Timestamp,
    cpu: raw.CPU,
    mem_used: raw.MemUsed,
    mem_percent: raw.MemPercent,
    disk_read_bytes_per_sec: raw.DiskReadBytesPerSec,
    disk_write_bytes_per_sec: raw.DiskWriteBytesPerSec,
    net_rx_bytes_per_sec: raw.NetRxBytesPerSec,
    net_tx_bytes_per_sec: raw.NetTxBytesPerSec,
    disk_used: raw.DiskUsed ?? 0,
    disk_used_percent: raw.DiskUsedPercent ?? 0,
  }
}

/** Normalizuje PascalCase kontenera z SSE /live/servers na snake_case */
export function normalizeLiveContainer(
  raw: LiveServerContainerRaw
): LiveServerContainer {
  return {
    container_id: raw.ContainerID,
    timestamp: raw.Timestamp,
    cpu: raw.CPU,
    mem_used: raw.MemUsed,
    mem_percent: raw.MemPercent,
    disk_used: raw.DiskUsed,
    disk_percent: raw.DiskPercent,
    net_rx: raw.NetRx,
    net_tx: raw.NetTx,
    state: raw.State ?? '',
    health: raw.Health ?? '',
    status: raw.Status ?? '',
    project: raw.Project ?? '',
  }
}
