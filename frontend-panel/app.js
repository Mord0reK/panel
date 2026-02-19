/* global Chart */

// ═══════════════════════════════════════════════
// CONFIG
// ═══════════════════════════════════════════════
const Config = {
	BASE: 'http://localhost:8080',
};

// ═══════════════════════════════════════════════
// UTILS — formatters
// ═══════════════════════════════════════════════
function fmtBytes(bytes) {
  const v = bytes || 0;
  if (v < 1) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.min(4, Math.floor(Math.log(v) / Math.log(1024)));
  return `${(v / Math.pow(1024, i)).toFixed(2)} ${units[i]}`;
}

function fmtBytesShort(bytes) {
  const v = bytes || 0;
  if (v < 1) return '0';
  const units = ['', 'K', 'M', 'G', 'T'];
  const i = Math.min(4, Math.floor(Math.log(v) / Math.log(1024)));
  return `${(v / Math.pow(1024, i)).toFixed(i > 0 ? 2 : 0)}${units[i]}`;
}

function fmtPercent(v) {
  return `${(v || 0).toFixed(2)}%`;
}

function fmtTS(ts) {
  return new Date((ts || 0) * 1000).toLocaleTimeString('pl-PL', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function relTime(isoStr) {
  const diff = (Date.now() - new Date(isoStr).getTime()) / 1000;
  if (diff < 5)   return 'teraz';
  if (diff < 60)  return `${Math.floor(diff)}s temu`;
  if (diff < 3600) return `${Math.floor(diff / 60)}min temu`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h temu`;
  return `${Math.floor(diff / 86400)}d temu`;
}

// ═══════════════════════════════════════════════
// TOAST
// ═══════════════════════════════════════════════
const Toast = {
  show(msg, type = 'info', duration = 3500) {
    const container = document.getElementById('toast-container');
    const el = document.createElement('div');
    el.className = `toast ${type}`;

    const iconMap = { success: '✓', error: '✕', info: 'ℹ' };
    el.innerHTML = `<span>${iconMap[type] || 'ℹ'}</span><span>${msg}</span>`;
    container.appendChild(el);

    setTimeout(() => {
      el.classList.add('removing');
      setTimeout(() => el.remove(), 200);
    }, duration);
  },
  success: (m) => Toast.show(m, 'success'),
  error:   (m) => Toast.show(m, 'error'),
  info:    (m) => Toast.show(m, 'info'),
};

// ═══════════════════════════════════════════════
// AUTH
// ═══════════════════════════════════════════════
const Auth = {
  KEY: 'panel_token',
  getToken()    { return localStorage.getItem(this.KEY); },
  setToken(t)   { localStorage.setItem(this.KEY, t); },
  clearToken()  { localStorage.removeItem(this.KEY); },
  isLoggedIn()  { return !!this.getToken(); },
};

// ═══════════════════════════════════════════════
// API helper
// ═══════════════════════════════════════════════
const API = {
  async req(method, path, body = null) {
    const token = Auth.getToken();
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = `Bearer ${token}`;
    const opts = { method, headers };
    if (body !== null) opts.body = JSON.stringify(body);

    let res;
    try {
      res = await fetch(Config.BASE + path, opts);
    } catch (e) {
      throw new Error(`Brak połączenia z backendem (${Config.BASE})`);
    }

    // 401 → wyloguj
    if (res.status === 401) {
      Auth.clearToken();
      App.goToLogin();
      throw new Error('Sesja wygasła — zaloguj się ponownie');
    }

    let data;
    try { data = await res.json(); } catch { data = {}; }
    if (!res.ok) throw new Error(data.error || data.message || `HTTP ${res.status}`);
    return data;
  },

  get:  (path)       => API.req('GET',    path),
  post: (path, body) => API.req('POST',   path, body),
  put:  (path)       => API.req('PUT',    path),
  del:  (path)       => API.req('DELETE', path),
};

// ═══════════════════════════════════════════════
// CHART FACTORY — shared Chart.js defaults
// ═══════════════════════════════════════════════
const ChartFactory = {
  COLORS: {
    accent:  '#4f6ef7',
    blue:    '#38bdf8',
    green:   '#22c55e',
    red:     '#ef4444',
    yellow:  '#f59e0b',
    purple:  '#a78bfa',
  },

  defaultOpts(yFormatter) {
    return {
      responsive: true,
      maintainAspectRatio: false,
      animation: { duration: 0 },
      interaction: { intersect: false, mode: 'index' },
      plugins: {
        legend: { display: false },
        tooltip: {
          backgroundColor: '#1a1d2e',
          borderColor: '#2a2d45',
          borderWidth: 1,
          titleColor: '#8b8fad',
          bodyColor: '#e2e4f0',
          callbacks: {
            label: (ctx) => `  ${ctx.dataset.label}: ${yFormatter ? yFormatter(ctx.parsed.y) : ctx.parsed.y}`,
          },
        },
      },
      scales: {
        x: {
          ticks: { color: '#565a80', maxTicksLimit: 6, font: { size: 10 } },
          grid:  { color: 'rgba(42,45,69,0.6)' },
          border: { color: '#2a2d45' },
        },
        y: {
          ticks: {
            color: '#565a80',
            font: { size: 10 },
            callback: yFormatter || undefined,
          },
          grid:  { color: 'rgba(42,45,69,0.6)' },
          border: { color: '#2a2d45' },
        },
      },
    };
  },

  makeGradient(ctx, color) {
    const gradient = ctx.createLinearGradient(0, 0, 0, 160);
    gradient.addColorStop(0, color + '40');
    gradient.addColorStop(1, color + '00');
    return gradient;
  },

  create(canvasId, datasets, yFormatter) {
    const canvas = document.getElementById(canvasId);
    if (!canvas) return null;
    const ctx = canvas.getContext('2d');

    const ds = datasets.map(d => ({
      ...d,
      backgroundColor: d.fill !== false ? this.makeGradient(ctx, d.borderColor) : 'transparent',
      borderWidth: d.borderWidth || 1.5,
      pointRadius: d.pointRadius !== undefined ? d.pointRadius : 0,
      pointHoverRadius: d.pointHoverRadius !== undefined ? d.pointHoverRadius : 3,
      fill: d.fill !== false,
      tension: d.tension !== undefined ? d.tension : 0.3,
      borderDash: d.borderDash || undefined,
    }));

    return new Chart(ctx, {
      type: 'line',
      data: { labels: [], datasets: ds },
      options: this.defaultOpts(yFormatter),
    });
  },

  destroy(chart) {
    if (chart) { chart.destroy(); }
    return null;
  },
};

// ═══════════════════════════════════════════════
// SSE — live metrics streaming
// ═══════════════════════════════════════════════
const SSE = {
  _ctrl: null,

  disconnect() {
    if (this._ctrl) { this._ctrl.abort(); this._ctrl = null; }
  },

  connect(uuid, onData, onStatus) {
    this.disconnect();
    const ctrl = new AbortController();
    this._ctrl = ctrl;
    const url = `${Config.BASE}/api/metrics/live/servers/${uuid}`;

    onStatus && onStatus('connecting');

    (async () => {
      try {
        const token = Auth.getToken();
        const res = await fetch(url, {
          signal: ctrl.signal,
          headers: {
            'Authorization': `Bearer ${token}`,
            'Accept': 'text/event-stream',
          },
        });

        if (!res.ok) { onStatus && onStatus('error'); return; }
        onStatus && onStatus('connected');

        const reader = res.body.getReader();
        const dec = new TextDecoder();
        let buf = '';

        while (true) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += dec.decode(value, { stream: true });
          const lines = buf.split('\n');
          buf = lines.pop();

          for (const line of lines) {
            if (line.startsWith('data:')) {
              try {
                const json = JSON.parse(line.slice(5).trim());
                onData && onData(json);
              } catch { /* skip malformed event */ }
            }
          }
        }

        onStatus && onStatus('disconnected');
      } catch (e) {
        if (e.name !== 'AbortError') {
          console.warn('SSE error:', e.message);
          onStatus && onStatus('error');
        }
      }
    })();
  },
};

// ═══════════════════════════════════════════════
// LIVE METRICS module
// ═══════════════════════════════════════════════
const LiveMetrics = {
  MAX_POINTS: 60,
  charts: {},
  buffers: {},
  _serverInfo: null,
  _serverUUID: null,

  _initBuffers() {
    this.buffers = {
      labels: [],
      cpu: [],
      ram: [],
      diskRead: [],
      diskWrite: [],
      netRx: [],
      netTx: [],
    };
  },

  _destroyCharts() {
    ['cpu', 'ram', 'disk', 'net'].forEach(k => {
      this.charts[k] = ChartFactory.destroy(this.charts[k]);
    });
  },

  _loadHistoryData(points) {
    if (!points || points.length === 0) return;
    
    // Pobierz ostatnie 60 punktów (1 minuta przy rozdzielczości 1s)
    const recentPoints = points.slice(-this.MAX_POINTS);
    
    recentPoints.forEach(p => {
      this.buffers.labels.push(fmtTS(p.timestamp));
      this.buffers.cpu.push(p.cpu_avg || p.cpu || 0);
      this.buffers.ram.push(p.mem_used_avg || p.mem_used || 0);
      this.buffers.diskRead.push(p.disk_read_bytes_per_sec_avg || p.disk_read_bytes_per_sec || 0);
      this.buffers.diskWrite.push(p.disk_write_bytes_per_sec_avg || p.disk_write_bytes_per_sec || 0);
      this.buffers.netRx.push(p.net_rx_bytes_per_sec_avg || p.net_rx_bytes_per_sec || 0);
      this.buffers.netTx.push(p.net_tx_bytes_per_sec_avg || p.net_tx_bytes_per_sec || 0);
    });
    
    // Zaktualizuj wykresy historycznymi danymi
    const setData = (chart, labels, ...dataSets) => {
      if (!chart) return;
      chart.data.labels = labels;
      dataSets.forEach((ds, i) => { if (chart.data.datasets[i]) chart.data.datasets[i].data = ds; });
      chart.update('none');
    };
    
    setData(this.charts.cpu,  this.buffers.labels, this.buffers.cpu);
    setData(this.charts.ram,  this.buffers.labels, this.buffers.ram);
    setData(this.charts.disk, this.buffers.labels, this.buffers.diskRead, this.buffers.diskWrite);
    setData(this.charts.net,  this.buffers.labels, this.buffers.netRx, this.buffers.netTx);
    
    // Zaktualizuj karty metryk ostatnimi wartościami
    const lastPoint = recentPoints[recentPoints.length - 1];
    if (lastPoint) {
      this._updateCards({
        cpu: lastPoint.cpu_avg || lastPoint.cpu || 0,
        mem_used: lastPoint.mem_used_avg || lastPoint.mem_used || 0,
        mem_percent: lastPoint.mem_percent || 0,
        disk_read_bytes_per_sec: lastPoint.disk_read_bytes_per_sec_avg || lastPoint.disk_read_bytes_per_sec || 0,
        disk_write_bytes_per_sec: lastPoint.disk_write_bytes_per_sec_avg || lastPoint.disk_write_bytes_per_sec || 0,
        net_rx_bytes_per_sec: lastPoint.net_rx_bytes_per_sec_avg || lastPoint.net_rx_bytes_per_sec || 0,
        net_tx_bytes_per_sec: lastPoint.net_tx_bytes_per_sec_avg || lastPoint.net_tx_bytes_per_sec || 0,
      });
    }
  },

  async init(serverInfo, uuid) {
    this._serverInfo = serverInfo;
    this._serverUUID = uuid;
    this._initBuffers();
    this._destroyCharts();

    const C = ChartFactory.COLORS;

    this.charts.cpu = ChartFactory.create('chart-live-cpu',
      [{ label: 'CPU %', borderColor: C.accent }],
      v => `${v.toFixed(2)}%`
    );
    this.charts.ram = ChartFactory.create('chart-live-ram',
      [{ label: 'RAM', borderColor: C.blue }],
      v => fmtBytes(v)
    );
    this.charts.disk = ChartFactory.create('chart-live-disk',
      [{ label: 'Odczyt', borderColor: C.green }, { label: 'Zapis', borderColor: C.yellow }],
      v => `${fmtBytesShort(v)}/s`
    );
    this.charts.net = ChartFactory.create('chart-live-net',
      [{ label: 'RX', borderColor: C.purple }, { label: 'TX', borderColor: C.red }],
      v => `${fmtBytesShort(v)}/s`
    );
    
    // Pobierz dane historyczne z ostatniej minuty
    if (uuid) {
      try {
        const data = await API.get(`/api/metrics/history/servers/${uuid}?range=1m`);
        if (data.host && data.host.points) {
          this._loadHistoryData(data.host.points);
        }
      } catch (e) {
        console.warn('Nie udało się pobrać danych historycznych:', e.message);
      }
    }
  },

  // Normalizes PascalCase fields from SSE host payload to snake_case
  _normalizeHost(raw) {
    if (!raw) return null;
    // SSE live/server returns PascalCase keys: CPU, MemUsed, DiskReadBytesPerSec, etc.
    // History and live/all use snake_case. Handle both.
    if ('CPU' in raw) {
      return {
        timestamp:                raw.Timestamp,
        cpu:                      raw.CPU,
        mem_used:                 raw.MemUsed,
        mem_percent:              raw.MemPercent,
        disk_read_bytes_per_sec:  raw.DiskReadBytesPerSec,
        disk_write_bytes_per_sec: raw.DiskWriteBytesPerSec,
        net_rx_bytes_per_sec:     raw.NetRxBytesPerSec,
        net_tx_bytes_per_sec:     raw.NetTxBytesPerSec,
      };
    }
    return raw;
  },

  // Normalizes PascalCase fields from SSE container payload to snake_case
  _normalizeContainer(raw) {
    if (!raw) return null;
    if ('CPU' in raw) {
      return {
        timestamp:   raw.Timestamp,
        cpu:         raw.CPU,
        mem_used:    raw.MemUsed,
        mem_percent: raw.MemPercent,
        disk_used:   raw.DiskUsed,
        net_rx:      raw.NetRx,
        net_tx:      raw.NetTx,
      };
    }
    return raw;
  },

  update(data) {
    const h = this._normalizeHost(data.host);
    if (!h) return;

    const label = fmtTS(h.timestamp || data.timestamp);

    const push = (arr, val) => {
      arr.push(val);
      if (arr.length > this.MAX_POINTS) arr.shift();
    };

    push(this.buffers.labels,    label);
    push(this.buffers.cpu,       h.cpu || 0);
    push(this.buffers.ram,       h.mem_used || 0);
    push(this.buffers.diskRead,  h.disk_read_bytes_per_sec || 0);
    push(this.buffers.diskWrite, h.disk_write_bytes_per_sec || 0);
    push(this.buffers.netRx,     h.net_rx_bytes_per_sec || 0);
    push(this.buffers.netTx,     h.net_tx_bytes_per_sec || 0);

    // Update metric cards
    this._updateCards(h);

    // Update charts
    const setData = (chart, labels, ...dataSets) => {
      if (!chart) return;
      chart.data.labels = labels;
      dataSets.forEach((ds, i) => { chart.data.datasets[i].data = ds; });
      chart.update('none');
    };

    setData(this.charts.cpu,  this.buffers.labels, this.buffers.cpu);
    setData(this.charts.ram,  this.buffers.labels, this.buffers.ram);
    setData(this.charts.disk, this.buffers.labels, this.buffers.diskRead, this.buffers.diskWrite);
    setData(this.charts.net,  this.buffers.labels, this.buffers.netRx, this.buffers.netTx);

    // Update container live stats in the containers table
    // data.containers from SSE is a map: {container_id → metrics}, not an array
    if (data.containers && typeof data.containers === 'object' && !Array.isArray(data.containers)) {
      const containerArray = Object.entries(data.containers).map(([id, metrics]) => {
        const normalized = this._normalizeContainer(metrics);
        return { container_id: id, ...normalized };
      });
      ContainersView.updateLiveStats(containerArray);
    } else if (data.containers && Array.isArray(data.containers)) {
      ContainersView.updateLiveStats(data.containers);
    }
  },

  _updateCards(h) {
    const memTotal = this._serverInfo ? this._serverInfo.memory_total : 0;
    const memPct = memTotal ? ((h.mem_used / memTotal) * 100) : (h.mem_percent || 0);

    this._setCard('mc-cpu', fmtPercent(h.cpu), h.cpu || 0, 100, '');

    this._setCard('mc-ram',
      fmtPercent(memPct),
      memPct, 100,
      memTotal ? `${fmtBytes(h.mem_used)} / ${fmtBytes(memTotal)}` : fmtBytes(h.mem_used));

    el('mc-disk-read').textContent  = `↓ ${fmtBytes(h.disk_read_bytes_per_sec || 0)}/s`;
    el('mc-disk-write').textContent = `↑ ${fmtBytes(h.disk_write_bytes_per_sec || 0)}/s`;

    el('mc-net-rx').textContent  = `↓ ${fmtBytes(h.net_rx_bytes_per_sec || 0)}/s`;
    el('mc-net-tx').textContent  = `↑ ${fmtBytes(h.net_tx_bytes_per_sec || 0)}/s`;
  },

  _setCard(id, valueText, valuePct, max, sub) {
    el(`${id}-val`).textContent = valueText;
    const pct = Math.min(100, (valuePct / max) * 100);
    el(`${id}-bar`).style.width = `${pct}%`;
    const subEl = document.getElementById(`${id}-sub`);
    if (subEl) subEl.textContent = sub;
  },

  destroy() {
    SSE.disconnect();
    this._destroyCharts();
    this._initBuffers();
  },
};

// ═══════════════════════════════════════════════
// HISTORY METRICS module
// ═══════════════════════════════════════════════
const HistoryMetrics = {
  charts: {},
  _serverUUID: null,
  _range: '1h',

  _destroyCharts() {
    ['cpu', 'ram', 'disk', 'net'].forEach(k => {
      this.charts[k] = ChartFactory.destroy(this.charts[k]);
    });
  },

  _shouldShowMinMax() {
    const rangesWithMinMax = ['5m', '15m', '30m', '1h', '6h', '12h', '24h', '7d', '30d'];
    return rangesWithMinMax.includes(this._range);
  },

  _initCharts() {
    this._destroyCharts();
    const C = ChartFactory.COLORS;
    const showMinMax = this._shouldShowMinMax();

    if (showMinMax) {
      this.charts.cpu = ChartFactory.create('chart-hist-cpu',
        [
          { label: 'CPU max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'CPU avg', borderColor: C.blue, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'CPU min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2 }
        ],
        v => `${v.toFixed(2)}%`
      );
      this.charts.ram = ChartFactory.create('chart-hist-ram',
        [
          { label: 'RAM max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'RAM avg', borderColor: C.blue, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'RAM min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2 }
        ],
        v => fmtBytes(v)
      );
      this.charts.disk = ChartFactory.create('chart-hist-disk',
        [
          { label: 'Odczyt max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'Odczyt avg', borderColor: C.blue, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'Odczyt min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'Zapis max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2, borderDash: [5, 5] },
          { label: 'Zapis avg', borderColor: C.accent, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'Zapis min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2, borderDash: [5, 5] }
        ],
        v => `${fmtBytesShort(v)}/s`
      );
      this.charts.net = ChartFactory.create('chart-hist-net',
        [
          { label: 'RX max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'RX avg', borderColor: C.blue, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'RX min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2 },
          { label: 'TX max', borderColor: C.red + '80', borderWidth: 1, fill: false, tension: 0.2, borderDash: [5, 5] },
          { label: 'TX avg', borderColor: C.accent, borderWidth: 2, fill: true, tension: 0.3 },
          { label: 'TX min', borderColor: C.green + '80', borderWidth: 1, fill: false, tension: 0.2, borderDash: [5, 5] }
        ],
        v => `${fmtBytesShort(v)}/s`
      );
    } else {
      this.charts.cpu = ChartFactory.create('chart-hist-cpu',
        [{ label: 'CPU', borderColor: C.accent }],
        v => `${v.toFixed(2)}%`
      );
      this.charts.ram = ChartFactory.create('chart-hist-ram',
        [{ label: 'RAM', borderColor: C.blue }],
        v => fmtBytes(v)
      );
      this.charts.disk = ChartFactory.create('chart-hist-disk',
        [{ label: 'Odczyt', borderColor: C.green }, { label: 'Zapis', borderColor: C.yellow }],
        v => `${fmtBytesShort(v)}/s`
      );
      this.charts.net = ChartFactory.create('chart-hist-net',
        [{ label: 'RX', borderColor: C.purple }, { label: 'TX', borderColor: C.red }],
        v => `${fmtBytesShort(v)}/s`
      );
    }
  },

  async load(uuid, range) {
    this._serverUUID = uuid;
    this._range = range || '1h';

    showEl('history-loading');
    hideEl('history-error');

    try {
      const data = await API.get(`/api/metrics/history/servers/${uuid}?range=${this._range}`);
      this._initCharts();
      this._renderHost(data.host);
      this._renderContainers(data.containers || []);
    } catch (e) {
      const errEl = document.getElementById('history-error');
      errEl.textContent = `Błąd ładowania historii: ${e.message}`;
      showEl('history-error');
    } finally {
      hideEl('history-loading');
    }
  },

  _isRaw(point) {
    return 'cpu' in point; // RawHostMetricPoint has `cpu`, HistoricalMetricPoint has `cpu_avg`
  },

  _extractHost(points) {
    if (!points || points.length === 0) return { labels: [], cpu: [], cpuMin: [], cpuMax: [], ram: [], ramMin: [], ramMax: [], diskRead: [], diskWrite: [], diskWriteMin: [], diskWriteMax: [], netRx: [], netRxMin: [], netRxMax: [], netTx: [], netTxMin: [], netTxMax: [] };
    const raw = this._isRaw(points[0]);

    return {
      labels:        points.map(p => fmtTS(p.timestamp)),
      cpu:           points.map(p => raw ? (p.cpu || 0) : (p.cpu_avg || 0)),
      cpuMin:        points.map(p => raw ? (p.cpu || 0) : (p.cpu_min || 0)),
      cpuMax:        points.map(p => raw ? (p.cpu || 0) : (p.cpu_max || 0)),
      ram:           points.map(p => raw ? (p.mem_used || 0) : (p.mem_used_avg || 0)),
      ramMin:        points.map(p => raw ? (p.mem_used || 0) : (p.mem_used_min || 0)),
      ramMax:        points.map(p => raw ? (p.mem_used || 0) : (p.mem_used_max || 0)),
      diskRead:      points.map(p => raw ? (p.disk_read_bytes_per_sec || 0) : (p.disk_read_bytes_per_sec_avg || 0)),
      diskWrite:     points.map(p => raw ? (p.disk_write_bytes_per_sec || 0) : (p.disk_write_bytes_per_sec_avg || 0)),
      diskWriteMin:  points.map(p => raw ? (p.disk_write_bytes_per_sec || 0) : (p.disk_write_bytes_per_sec_min || p.disk_write_bytes_per_sec_avg || 0)),
      diskWriteMax:  points.map(p => raw ? (p.disk_write_bytes_per_sec || 0) : (p.disk_write_bytes_per_sec_max || p.disk_write_bytes_per_sec_avg || 0)),
      netRx:         points.map(p => raw ? (p.net_rx_bytes_per_sec || 0) : (p.net_rx_bytes_per_sec_avg || 0)),
      netRxMin:      points.map(p => raw ? (p.net_rx_bytes_per_sec || 0) : (p.net_rx_bytes_per_sec_min || p.net_rx_bytes_per_sec_avg || 0)),
      netRxMax:      points.map(p => raw ? (p.net_rx_bytes_per_sec || 0) : (p.net_rx_bytes_per_sec_max || p.net_rx_bytes_per_sec_avg || 0)),
      netTx:         points.map(p => raw ? (p.net_tx_bytes_per_sec || 0) : (p.net_tx_bytes_per_sec_avg || 0)),
      netTxMin:      points.map(p => raw ? (p.net_tx_bytes_per_sec || 0) : (p.net_tx_bytes_per_sec_min || p.net_tx_bytes_per_sec_avg || 0)),
      netTxMax:      points.map(p => raw ? (p.net_tx_bytes_per_sec || 0) : (p.net_tx_bytes_per_sec_max || p.net_tx_bytes_per_sec_avg || 0)),
    };
  },

  _renderHost(host) {
    const pts = host ? (host.points || []) : [];
    const d = this._extractHost(pts);
    const showMinMax = this._shouldShowMinMax();

    const setChart = (chart, labels, ...sets) => {
      if (!chart) return;
      chart.data.labels = labels;
      sets.forEach((s, i) => { if (chart.data.datasets[i]) chart.data.datasets[i].data = s; });
      chart.update();
    };

    if (showMinMax) {
      setChart(this.charts.cpu,  d.labels, d.cpuMax, d.cpu, d.cpuMin);
      setChart(this.charts.ram,  d.labels, d.ramMax, d.ram, d.ramMin);
      // Disk: dla odczytu mamy tylko avg (min/max replikowane z avg w backendzie), dla zapisu mamy min/max
      setChart(this.charts.disk, d.labels, 
        d.diskRead, d.diskRead, d.diskRead,  // Odczyt: min=max=avg (brak min/max w backendzie)
        d.diskWriteMax, d.diskWrite, d.diskWriteMin  // Zapis: min/max z backendu
      );
      // Net: mamy min/max dla RX i TX
      setChart(this.charts.net, d.labels, 
        d.netRxMax, d.netRx, d.netRxMin,
        d.netTxMax, d.netTx, d.netTxMin
      );
    } else {
      setChart(this.charts.cpu,  d.labels, d.cpu);
      setChart(this.charts.ram,  d.labels, d.ram);
      setChart(this.charts.disk, d.labels, d.diskRead, d.diskWrite);
      setChart(this.charts.net,  d.labels, d.netRx, d.netTx);
    }
  },

  _renderContainers(containers) {
    const wrap = document.getElementById('history-containers');
    wrap.innerHTML = '';

    if (!containers || containers.length === 0) {
      wrap.innerHTML = '<div style="color:var(--text3);font-size:13px;padding:12px 0">Brak danych kontenerów dla tego zakresu</div>';
      return;
    }

    const C = ChartFactory.COLORS;

    containers.forEach((c, idx) => {
      const card = document.createElement('div');
      card.className = 'history-container-card';

      const cpuId    = `hcc-cpu-${idx}`;
      const ramId    = `hcc-ram-${idx}`;

      card.innerHTML = `
        <div class="history-container-header">
          <div class="status-dot dot-green"></div>
          <div>
            <div class="history-container-name">${escHtml(c.name || c.container_id)}</div>
            <div class="history-container-image">${escHtml(c.image || '')}${c.project ? ' · ' + escHtml(c.project) : ''}</div>
          </div>
        </div>
        <div class="charts-grid">
          <div class="chart-card">
            <div class="chart-title">CPU %</div>
            <div class="chart-wrap"><canvas id="${cpuId}"></canvas></div>
          </div>
          <div class="chart-card">
            <div class="chart-title">RAM</div>
            <div class="chart-wrap"><canvas id="${ramId}"></canvas></div>
          </div>
        </div>
      `;
      wrap.appendChild(card);

      const pts = c.points || [];
      if (pts.length === 0) return;

      const raw = this._isRaw(pts[0]);
      const labels    = pts.map(p => fmtTS(p.timestamp));
      const cpuData   = pts.map(p => raw ? (p.cpu || 0) : (p.cpu_avg || 0));
      const ramData   = pts.map(p => raw ? (p.mem_used || 0) : (p.mem_used_avg || 0));

      const mkChart = (id, ds, fmt) => ChartFactory.create(id, ds, fmt);
      mkChart(cpuId, [{ label: 'CPU %', borderColor: C.accent, data: cpuData }], v => `${v.toFixed(2)}%`);
      const ramChart = mkChart(ramId, [{ label: 'RAM', borderColor: C.blue }], v => fmtBytes(v));
      if (ramChart) {
        ramChart.data.labels = labels;
        ramChart.data.datasets[0].data = ramData;
        ramChart.update();
      }

      // Set CPU chart data after creation
      const cpuChart = Chart.getChart(cpuId);
      if (cpuChart) {
        cpuChart.data.labels = labels;
        cpuChart.data.datasets[0].data = cpuData;
        cpuChart.update();
      }
    });
  },

  destroy() {
    this._destroyCharts();
  },
};

// ═══════════════════════════════════════════════
// CONTAINERS VIEW
// ═══════════════════════════════════════════════
const ContainersView = {
  _containers: [],
  _serverUUID: null,
  _liveStats: {}, // containerID → {cpu, mem_used}

  load(uuid, containers) {
    this._serverUUID = uuid;
    this._containers = containers || [];
    this._liveStats = {};

    const badge = document.getElementById('containers-count-badge');
    badge.textContent = this._containers.length;
    badge.classList.toggle('hidden', this._containers.length === 0);

    this.render(document.querySelector('.filter-btn.active')?.dataset.filter || 'all');
  },

  updateLiveStats(containerPoints) {
    // containerPoints from SSE: [{container_id, cpu, mem_used, ...}]
    (containerPoints || []).forEach(cp => {
      this._liveStats[cp.container_id] = cp;
    });

    // Update visible rows
    this._containers.forEach(c => {
      const stats = this._liveStats[c.container_id];
      if (!stats) return;

      const cpuCell = document.getElementById(`cpu-${c.container_id}`);
      const ramCell = document.getElementById(`ram-${c.container_id}`);
      if (cpuCell) cpuCell.textContent = fmtPercent(stats.cpu || 0);
      if (ramCell) ramCell.textContent = fmtBytes(stats.mem_used || 0);
    });
  },

  render(filter) {
    const tbody = document.getElementById('containers-tbody');
    const emptyEl = document.getElementById('containers-empty');
    tbody.innerHTML = '';

    let visible = this._containers;
    if (filter === 'running') {
      visible = this._containers.filter(c => this._isRunning(c));
    } else if (filter === 'stopped') {
      visible = this._containers.filter(c => !this._isRunning(c));
    }

    if (visible.length === 0) {
      showEl('containers-empty');
      return;
    }
    hideEl('containers-empty');

    visible.forEach(c => {
      const running = this._isRunning(c);
      const paused  = this._isPaused(c);
      const stats   = this._liveStats[c.container_id] || {};

      const status = running ? 'running' : (paused ? 'paused' : 'stopped');
      const statusLabel = { running: 'running', paused: 'paused', stopped: 'stopped' }[status];
      const dotClass  = { running: 'dot-green', paused: 'dot-yellow', stopped: 'dot-red' }[status];

      const tr = document.createElement('tr');
      tr.dataset.containerStatus = status;

      tr.innerHTML = `
        <td>
          <div class="status-cell">
            <span class="status-dot ${dotClass}"></span>
            <span class="status-label ${status}">${statusLabel}</span>
          </div>
        </td>
        <td>
          <div class="container-name">${escHtml(c.name)}</div>
          <div class="container-id">${escHtml(c.container_id.slice(0, 12))}</div>
        </td>
        <td><div class="container-image">${escHtml(c.image || '—')}</div></td>
        <td><div class="container-project">${escHtml(c.project || '—')}${c.service ? ' / ' + escHtml(c.service) : ''}</div></td>
        <td><span class="container-metric" id="cpu-${escAttr(c.container_id)}">${stats.cpu !== undefined ? fmtPercent(stats.cpu) : '—'}</span></td>
        <td><span class="container-metric" id="ram-${escAttr(c.container_id)}">${stats.mem_used !== undefined ? fmtBytes(stats.mem_used) : '—'}</span></td>
        <td><span class="container-lastseen">${relTime(c.last_seen)}</span></td>
        <td>${this._actionBtns(c, status)}</td>
      `;

      // Bind action buttons
      tr.querySelectorAll('[data-action]').forEach(btn => {
        btn.addEventListener('click', () => this._doAction(c, btn.dataset.action, btn));
      });

      tbody.appendChild(tr);
    });
  },

  _isRunning(c) {
    const stats = this._liveStats[c.container_id];
    if (stats) return true; // has live data → running
    // Fallback: recently seen (within last 10s of last_seen)
    const diff = (Date.now() - new Date(c.last_seen).getTime()) / 1000;
    return diff < 15;
  },

  _isPaused(c) {
    // We can't reliably detect pause from current data; would need explicit status
    return false;
  },

  _actionBtns(c, status) {
    if (status === 'running') {
      return `
        <div class="action-btns">
          <button class="act-btn act-update"  data-action="update"  title="Aktualizuj">⬆ Update</button>
          <button class="act-btn act-restart" data-action="restart" title="Restart">↺ Restart</button>
          <button class="act-btn act-stop"    data-action="stop"    title="Zatrzymaj">■ Stop</button>
          <button class="act-btn act-pause"   data-action="pause"   title="Wstrzymaj">⏸ Pause</button>
          <button class="act-btn act-remove"  data-action="remove"  title="Usuń">✕ Remove</button>
        </div>`;
    }
    if (status === 'paused') {
      return `
        <div class="action-btns">
          <button class="act-btn act-update"  data-action="update"  title="Aktualizuj">⬆ Update</button>
          <button class="act-btn act-unpause" data-action="unpause" title="Wznów">▶ Unpause</button>
          <button class="act-btn act-stop"    data-action="stop"    title="Zatrzymaj">■ Stop</button>
          <button class="act-btn act-remove"  data-action="remove"  title="Usuń">✕ Remove</button>
        </div>`;
    }
    // stopped
    return `
      <div class="action-btns">
        <button class="act-btn act-start"  data-action="start"  title="Uruchom">▶ Start</button>
        <button class="act-btn act-remove" data-action="remove" title="Usuń (force)">✕ Remove</button>
      </div>`;
  },

  async _doAction(c, action, btn) {
    const uuid = this._serverUUID;
    if (!uuid) return;

    const confirmActions = ['remove', 'update'];
    if (confirmActions.includes(action)) {
      const messages = {
        remove: `Czy na pewno chcesz usunąć kontener "${c.name}"?\n\nOperacja jest nieodwracalna.`,
        update: `Czy na pewno chcesz zaktualizować kontener "${c.name}"?\n\nAkcja pobierze najnowszą wersję obrazu Docker i zrestartuje kontener.`
      };
      if (!confirm(messages[action])) return;
    }

    // Disable all action buttons in the row
    const row = btn.closest('tr');
    row && row.querySelectorAll('[data-action]').forEach(b => { b.disabled = true; });

    try {
      await API.post(`/api/servers/${uuid}/containers/${c.container_id}/command`, { action });
      Toast.success(`Akcja "${action}" dla ${c.name} wykonana pomyślnie`);

      // Refresh container list after a short delay
      setTimeout(async () => {
        try {
          const data = await API.get(`/api/servers/${uuid}`);
          this.load(uuid, data.containers || []);
        } catch { /* ignore */ }
      }, 1500);
    } catch (e) {
      Toast.error(`Błąd akcji "${action}": ${e.message}`);
      row && row.querySelectorAll('[data-action]').forEach(b => { b.disabled = false; });
    }
  },
};

// ═══════════════════════════════════════════════
// SIDEBAR — server list + live/all SSE
// ═══════════════════════════════════════════════
const Sidebar = {
  _servers: [],
  _liveAll: {},  // uuid → {cpu, memory}
  _refreshTimer: null,
  _sseAllCtrl: null,

  async init() {
    await this.refresh();
    this._startPolling();
    this._startLiveAll();
  },

  _startPolling() {
    clearInterval(this._refreshTimer);
    this._refreshTimer = setInterval(() => this.refresh(), 10000);
  },

  async refresh() {
    try {
      const servers = await API.get('/api/servers');
      this._servers = Array.isArray(servers) ? servers : [];
      this.render();
      this._updateStatus(true);
    } catch (e) {
      this._updateStatus(false);
    }
  },

  _startLiveAll() {
    if (this._sseAllCtrl) { this._sseAllCtrl.abort(); }
    const ctrl = new AbortController();
    this._sseAllCtrl = ctrl;
    const url = `${Config.BASE}/api/metrics/live/all`;

    (async () => {
      try {
        const res = await fetch(url, {
          signal: ctrl.signal,
          headers: {
            'Authorization': `Bearer ${Auth.getToken()}`,
            'Accept': 'text/event-stream',
          },
        });
        if (!res.ok) {
          // Schedule reconnect on HTTP error (not on abort)
          setTimeout(() => { if (!ctrl.signal.aborted) this._startLiveAll(); }, 3000);
          return;
        }

        const reader = res.body.getReader();
        const dec = new TextDecoder();
        let buf = '';

        while (true) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += dec.decode(value, { stream: true });
          const lines = buf.split('\n');
          buf = lines.pop();

          for (const line of lines) {
            if (line.startsWith('data:')) {
              try {
                const json = JSON.parse(line.slice(5).trim());
                this._handleLiveAll(json);
              } catch { /* skip */ }
            }
          }
        }

        // Stream ended cleanly — reconnect
        if (!ctrl.signal.aborted) {
          setTimeout(() => this._startLiveAll(), 1000);
        }
      } catch (e) {
        if (e.name !== 'AbortError') {
          // Network error — reconnect after delay
          setTimeout(() => { if (!ctrl.signal.aborted) this._startLiveAll(); }, 3000);
        }
      }
    })();
  },

  _handleLiveAll(data) {
    if (!data.servers) return;
    data.servers.forEach(s => {
      // Normalize: memory field is mem_used in bytes, compute percentage against server total
      this._liveAll[s.uuid] = s;
    });
    this._updateSidebarStats();
  },

  _updateSidebarStats() {
    this._servers.forEach(s => {
      const live = this._liveAll[s.uuid];
      const cpuEl  = document.getElementById(`sb-cpu-${s.uuid}`);
      const memEl  = document.getElementById(`sb-mem-${s.uuid}`);
      if (live && cpuEl) cpuEl.textContent = `CPU ${fmtPercent(live.cpu)}`;
      if (live && memEl) {
        // memory field is mem_used (bytes) — format as bytes shorthand
        memEl.textContent = `MEM ${fmtBytes(live.memory)}`;
      }
    });
  },

  render() {
    const list = document.getElementById('servers-list');
    list.innerHTML = '';

    if (this._servers.length === 0) {
      list.innerHTML = `<div style="padding:16px;color:var(--text3);font-size:13px">Brak serwerów</div>`;
      return;
    }

    const activeUUID = ServerView.currentUUID;

    this._servers.forEach(s => {
      const live = this._liveAll[s.uuid];
      const online = (Date.now() - new Date(s.last_seen).getTime()) < 10000;
      const dotCls = online ? 'dot-green' : 'dot-gray';

      const item = document.createElement('div');
      item.className = `server-item${s.uuid === activeUUID ? ' active' : ''}`;
      item.dataset.uuid = s.uuid;
      item.innerHTML = `
        <span class="status-dot ${dotCls}"></span>
        <div class="server-item-info">
          <div class="server-item-name">${escHtml(s.hostname)}</div>
          <div class="server-item-meta">${escHtml(s.platform || '')} ${s.approved ? '' : '· <span style="color:var(--yellow)">nieaktywny</span>'}</div>
        </div>
        <div class="server-item-stats">
          <span class="mini-stat${live ? ' live' : ''}" id="sb-cpu-${s.uuid}">${live ? `CPU ${fmtPercent(live.cpu)}` : '—'}</span>
          <span class="mini-stat${live ? ' live' : ''}" id="sb-mem-${s.uuid}">${live ? `MEM ${fmtBytes(live.memory)}` : '—'}</span>
        </div>
      `;
      item.addEventListener('click', () => ServerView.load(s.uuid));
      list.appendChild(item);
    });
  },

  markActive(uuid) {
    document.querySelectorAll('.server-item').forEach(el => {
      el.classList.toggle('active', el.dataset.uuid === uuid);
    });
  },

  _updateStatus(ok) {
    const dot  = document.querySelector('#sidebar-status .conn-dot');
    const text = document.getElementById('sidebar-status-text');
    if (dot)  dot.classList.toggle('connected', ok);
    if (text) text.textContent = ok ? `${this._servers.length} serwerów` : 'Błąd połączenia';
  },

  destroy() {
    clearInterval(this._refreshTimer);
    if (this._sseAllCtrl) { this._sseAllCtrl.abort(); this._sseAllCtrl = null; }
  },
};

// ═══════════════════════════════════════════════
// SERVER VIEW — loads and renders server detail
// ═══════════════════════════════════════════════
const ServerView = {
  currentUUID: null,
  _activeTab: 'live',
  _serverData: null,

  async load(uuid) {
    this.currentUUID = uuid;
    Sidebar.markActive(uuid);

    hideEl('view-welcome');
    showEl('view-server');

    // Reset tabs to Live
    this._switchTab('live');

    // Show loading state
    el('sv-hostname').textContent = 'Ładowanie...';
    el('sv-meta').innerHTML = '';

    // Disconnect previous SSE
    LiveMetrics.destroy();

    try {
      const data = await API.get(`/api/servers/${uuid}`);
      this._serverData = data;
      this._renderHeader(data.server);
      await this._loadLive(data.server);
      ContainersView.load(uuid, data.containers || []);
    } catch (e) {
      el('sv-hostname').textContent = 'Błąd ładowania';
      Toast.error(`Nie udało się wczytać serwera: ${e.message}`);
    }
  },

  _renderHeader(server) {
    el('sv-hostname').textContent = server.hostname || server.uuid;

    const online = (Date.now() - new Date(server.last_seen).getTime()) < 15000;
    const dot = el('sv-online-dot');
    dot.className = `status-dot ${online ? 'dot-green' : 'dot-gray'}`;

    // Approve badge / button
    if (!server.approved) {
      showEl('sv-approve-badge');
      showEl('sv-btn-approve');
    } else {
      hideEl('sv-approve-badge');
      hideEl('sv-btn-approve');
    }

    // Meta row
    const meta = [
      { label: 'CPU',      val: `${server.cpu_model || '?'} (${server.cpu_cores || '?'} rdzenie)` },
      { label: 'RAM',      val: fmtBytes(server.memory_total || 0) },
      { label: 'Platforma',val: server.platform || '?' },
      { label: 'Kernel',   val: server.kernel || '?' },
      { label: 'Arch',     val: server.architecture || '?' },
      { label: 'UUID',     val: `<span style="font-family:monospace;font-size:11px">${server.uuid}</span>` },
      { label: 'Ostatnio', val: relTime(server.last_seen) },
    ];
    el('sv-meta').innerHTML = meta.map(m =>
      `<span class="meta-item"><strong>${m.label}:</strong> ${m.val}</span>`
    ).join('');
  },

  async _loadLive(server) {
    await LiveMetrics.init(server, server.uuid);

    const sseStatus = document.getElementById('live-sse-status');

    SSE.connect(
      server.uuid,
      (data) => {
        LiveMetrics.update(data);
      },
      (status) => {
        if (!sseStatus) return;
        const statusMap = {
          connecting:   { cls: '',          txt: '<span class="spinner-ring spinner-sm"></span> Łączenie ze streamem...' },
          connected:    { cls: 'connected', txt: '● Live — strumień aktywny' },
          disconnected: { cls: '',          txt: '○ Stream rozłączony' },
          error:        { cls: 'error',     txt: '✕ Błąd streamu — spróbuj odświeżyć' },
        };
        const s = statusMap[status] || statusMap.error;
        sseStatus.className = `sse-status ${s.cls}`;
        sseStatus.innerHTML = s.txt;
      }
    );
  },

  _switchTab(tab) {
    this._activeTab = tab;

    // Update tab buttons
    document.querySelectorAll('.tab-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.tab === tab);
    });

    // Show/hide tab content
    ['live', 'history', 'containers'].forEach(t => {
      const el2 = document.getElementById(`tab-${t}`);
      if (el2) el2.classList.toggle('hidden', t !== tab);
    });

    // On tab switch to history — load data automatically
    if (tab === 'history' && this.currentUUID) {
      const activeRange = document.querySelector('.range-btn.active')?.dataset.range || '1h';
      HistoryMetrics.load(this.currentUUID, activeRange);
    }
  },

  async approve() {
    if (!this.currentUUID) return;
    try {
      await API.put(`/api/servers/${this.currentUUID}/approve`);
      Toast.success('Agent aktywowany pomyślnie');
      await this.load(this.currentUUID);
    } catch (e) {
      Toast.error(`Błąd aktywacji: ${e.message}`);
    }
  },

  async deleteServer() {
    if (!this.currentUUID) return;
    const uuid = this.currentUUID;

    const modalText = document.getElementById('modal-delete-text');
    if (modalText && this._serverData?.server) {
      modalText.textContent = `Czy na pewno chcesz usunąć serwer "${this._serverData.server.hostname}"? Operacja jest nieodwracalna i usunie wszystkie dane metryk.`;
    }
    showEl('modal-delete');

    return new Promise(resolve => {
      const onConfirm = async () => {
        cleanup();
        hideEl('modal-delete');
        try {
          await API.del(`/api/servers/${uuid}`);
          Toast.success('Serwer usunięty');

          // Clear server view
          this.currentUUID = null;
          LiveMetrics.destroy();
          hideEl('view-server');
          showEl('view-welcome');

          // Refresh sidebar
          await Sidebar.refresh();
        } catch (e) {
          Toast.error(`Błąd usuwania: ${e.message}`);
        }
        resolve();
      };

      const onCancel = () => {
        cleanup();
        hideEl('modal-delete');
        resolve();
      };

      const cleanup = () => {
        document.getElementById('modal-confirm').removeEventListener('click', onConfirm);
        document.getElementById('modal-cancel').removeEventListener('click', onCancel);
      };

      document.getElementById('modal-confirm').addEventListener('click', onConfirm);
      document.getElementById('modal-cancel').addEventListener('click', onCancel);
    });
  },
};

// ═══════════════════════════════════════════════
// LOGIN PAGE
// ═══════════════════════════════════════════════
const LoginPage = {
  _setupMode: false,

  async init() {
    // Check backend status
    try {
      const status = await API.get('/api/auth/status');
      if (status.authenticated && Auth.isLoggedIn()) {
        App.goToApp();
        return;
      }
      if (status.setup_required) {
        this._setupMode = true;
        showEl('setup-banner');
        el('login-btn-label').textContent = 'Utwórz konto';
      }
    } catch (e) {
      // Backend not available — show login anyway
      console.warn('Auth status check failed:', e.message);
    }
  },

  async submit(username, password) {
    const btnLabel  = el('login-btn-label');
    const btnSpin   = el('login-btn-spinner');
    const errBanner = el('login-error');

    hideEl('login-error');
    btnLabel.classList.add('hidden');
    showEl('login-btn-spinner');
    el('login-btn').disabled = true;

    try {
      const endpoint = this._setupMode ? '/api/setup' : '/api/login';
      const data = await API.post(endpoint, { username, password });

      if (data.token) {
        Auth.setToken(data.token);
        App.goToApp();
      } else {
        throw new Error('Serwer nie zwrócił tokenu');
      }
    } catch (e) {
      errBanner.textContent = e.message;
      showEl('login-error');
    } finally {
      showEl('login-btn-label');
      hideEl('login-btn-spinner');
      el('login-btn').disabled = false;
    }
  },
};

// ═══════════════════════════════════════════════
// APP — main controller
// ═══════════════════════════════════════════════
const App = {
  goToLogin() {
    Auth.clearToken();
    showPage('page-login');
  },

  async goToApp() {
    showPage('page-app');
    await Sidebar.init();
  },
};

// ═══════════════════════════════════════════════
// DOM HELPERS
// ═══════════════════════════════════════════════
function el(id)         { return document.getElementById(id); }
function showEl(id)     { const e = el(id); if (e) e.classList.remove('hidden'); }
function hideEl(id)     { const e = el(id); if (e) e.classList.add('hidden'); }
function showPage(id)   { document.querySelectorAll('.page').forEach(p => p.classList.add('hidden')); showEl(id); }

function escHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}
function escAttr(str) {
  if (!str) return '';
  return String(str).replace(/[^a-zA-Z0-9_-]/g, '_');
}

// ═══════════════════════════════════════════════
// EVENT BINDINGS
// ═══════════════════════════════════════════════
document.addEventListener('DOMContentLoaded', async () => {

  // ── Login form ──────────────────────────────
  el('login-form')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const username = el('inp-username').value.trim();
    const password = el('inp-password').value;
    await LoginPage.submit(username, password);
  });

  // ── Logout ──────────────────────────────────
  el('btn-logout')?.addEventListener('click', () => {
    SSE.disconnect();
    Sidebar.destroy();
    LiveMetrics.destroy();
    App.goToLogin();
  });

  // ── Server: approve button ───────────────────
  el('sv-btn-approve')?.addEventListener('click', () => ServerView.approve());

  // ── Server: delete button ───────────────────
  el('sv-btn-delete')?.addEventListener('click', () => ServerView.deleteServer());

  // ── Tabs ─────────────────────────────────────
  document.querySelectorAll('.tab-btn').forEach(btn => {
    btn.addEventListener('click', () => ServerView._switchTab(btn.dataset.tab));
  });

  // ── History range buttons ────────────────────
  document.querySelectorAll('.range-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.range-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      if (ServerView.currentUUID) {
        HistoryMetrics.load(ServerView.currentUUID, btn.dataset.range);
      }
    });
  });

  // ── History refresh ──────────────────────────
  el('btn-refresh-history')?.addEventListener('click', () => {
    const range = document.querySelector('.range-btn.active')?.dataset.range || '1h';
    if (ServerView.currentUUID) HistoryMetrics.load(ServerView.currentUUID, range);
  });

  // ── Container filters ────────────────────────
  document.querySelectorAll('.filter-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      ContainersView.render(btn.dataset.filter);
    });
  });

  // ── Containers refresh ───────────────────────
  el('btn-refresh-containers')?.addEventListener('click', async () => {
    if (!ServerView.currentUUID) return;
    showEl('containers-loading');
    try {
      const data = await API.get(`/api/servers/${ServerView.currentUUID}`);
      ContainersView.load(ServerView.currentUUID, data.containers || []);
    } catch (e) {
      Toast.error(`Błąd odświeżania: ${e.message}`);
    } finally {
      hideEl('containers-loading');
    }
  });

  // ── Modal cancel (fallback) ──────────────────
  el('modal-delete')?.addEventListener('click', (e) => {
    if (e.target === el('modal-delete')) hideEl('modal-delete');
  });

  // ── INIT ─────────────────────────────────────
  if (Auth.isLoggedIn()) {
    await App.goToApp();
  } else {
    await LoginPage.init();
    showPage('page-login');
  }
});
