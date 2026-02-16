// Configuration
const API_BASE = 'http://localhost:8080';
const TOKEN_KEY = 'api_token';

// SSE connections
let liveAllAbortController = null;
let liveServerAbortController = null;

// Get token from localStorage
function getToken() {
    return localStorage.getItem(TOKEN_KEY);
}

// Save token to localStorage
function saveToken(token) {
    localStorage.setItem(TOKEN_KEY, token);
    updateAuthStatus();
}

// Clear token
function clearToken() {
    localStorage.removeItem(TOKEN_KEY);
    updateAuthStatus();
    showResponse({ message: 'Token wyczyszczony' }, 200);
}

// Update auth status display
function updateAuthStatus() {
    const token = getToken();
    const statusEl = document.getElementById('authStatus');
    if (token) {
        statusEl.textContent = 'Zalogowany ✓';
        statusEl.style.background = 'rgba(40, 167, 69, 0.8)';
    } else {
        statusEl.textContent = 'Brak autoryzacji';
        statusEl.style.background = 'rgba(255, 255, 255, 0.2)';
    }
}

// Show response in the panel
function showResponse(data, status) {
    const statusEl = document.getElementById('responseStatus');
    const bodyEl = document.getElementById('responseBody');

    // Update status
    statusEl.textContent = `Status: ${status}`;
    statusEl.className = status >= 200 && status < 300 ? 'success' : 'error';

    // Update body
    if (typeof data === 'object') {
        bodyEl.textContent = JSON.stringify(data, null, 2);
    } else {
        bodyEl.textContent = data;
    }
}

// Clear response panel
function clearResponse() {
    document.getElementById('responseStatus').textContent = '';
    document.getElementById('responseBody').textContent = '';
}

// Generic API request
async function apiRequest(method, endpoint, body = null, requiresAuth = true) {
    const headers = {
        'Content-Type': 'application/json',
    };

    if (requiresAuth) {
        const token = getToken();
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }
    }

    const options = {
        method,
        headers,
    };

    if (body) {
        options.body = JSON.stringify(body);
    }

    try {
        const response = await fetch(API_BASE + endpoint, options);
        const data = await response.json().catch(() => ({}));

        showResponse(data, response.status);
        return { data, status: response.status, ok: response.ok };
    } catch (error) {
        showResponse({ error: error.message }, 0);
        return { data: { error: error.message }, status: 0, ok: false };
    }
}

// ===== AUTH ENDPOINTS =====

async function checkAuthStatus() {
    await apiRequest('GET', '/api/auth/status', null, false);
}

async function setupUser() {
    const username = document.getElementById('setupUsername').value;
    const password = document.getElementById('setupPassword').value;

    if (!username || !password) {
        showResponse({ error: 'Wypełnij oba pola' }, 400);
        return;
    }

    const result = await apiRequest('POST', '/api/setup', { username, password }, false);

    if (result.ok && result.data.token) {
        saveToken(result.data.token);
    }
}

async function loginUser() {
    const username = document.getElementById('loginUsername').value;
    const password = document.getElementById('loginPassword').value;

    if (!username || !password) {
        showResponse({ error: 'Wypełnij oba pola' }, 400);
        return;
    }

    const result = await apiRequest('POST', '/api/login', { username, password }, false);

    if (result.ok && result.data.token) {
        saveToken(result.data.token);
    }
}

// ===== SERVERS ENDPOINTS =====

async function getServers() {
    await apiRequest('GET', '/api/servers');
}

async function getServer() {
    const uuid = document.getElementById('serverUuid').value;
    if (!uuid) {
        showResponse({ error: 'Podaj UUID serwera' }, 400);
        return;
    }
    await apiRequest('GET', `/api/servers/${uuid}`);
}

async function approveServer() {
    const uuid = document.getElementById('approveUuid').value;
    if (!uuid) {
        showResponse({ error: 'Podaj UUID serwera' }, 400);
        return;
    }
    await apiRequest('PUT', `/api/servers/${uuid}/approve`);
}

async function deleteServer() {
    const uuid = document.getElementById('deleteUuid').value;
    if (!uuid) {
        showResponse({ error: 'Podaj UUID serwera' }, 400);
        return;
    }

    if (!confirm('Czy na pewno chcesz usunąć ten serwer?')) {
        return;
    }

    await apiRequest('DELETE', `/api/servers/${uuid}`);
}

// ===== COMMANDS ENDPOINTS =====

async function sendServerCommand() {
    const uuid = document.getElementById('cmdServerUuid').value;
    const action = document.getElementById('cmdAction').value;
    const target = document.getElementById('cmdTarget').value;

    if (!uuid || !action) {
        showResponse({ error: 'Podaj UUID serwera i akcję' }, 400);
        return;
    }

    await apiRequest('POST', `/api/servers/${uuid}/command`, {
        action,
        target: target || ''
    });
}

async function sendContainerCommand() {
    const uuid = document.getElementById('containerServerUuid').value;
    const containerId = document.getElementById('containerId').value;
    const action = document.getElementById('containerAction').value;

    if (!uuid || !containerId || !action) {
        showResponse({ error: 'Wypełnij wszystkie pola' }, 400);
        return;
    }

    await apiRequest('POST', `/api/servers/${uuid}/containers/${containerId}/command`, {
        action,
        target: ''
    });
}

// ===== METRICS ENDPOINTS =====

async function getServerMetrics() {
    const uuid = document.getElementById('metricsServerUuid').value;
    const range = document.getElementById('metricsServerRange').value;

    if (!uuid) {
        showResponse({ error: 'Podaj UUID serwera' }, 400);
        return;
    }

    await apiRequest('GET', `/api/metrics/history/servers/${uuid}?range=${range}`);
}

async function getContainerMetrics() {
    const uuid = document.getElementById('metricsContainerServerUuid').value;
    const containerId = document.getElementById('metricsContainerId').value;
    const range = document.getElementById('metricsContainerRange').value;

    if (!uuid || !containerId) {
        showResponse({ error: 'Podaj UUID serwera i ID kontenera' }, 400);
        return;
    }

    await apiRequest('GET', `/api/metrics/history/servers/${uuid}/containers/${containerId}?range=${range}`);
}

// ===== SSE ENDPOINTS =====

async function startSSEStream(url, statusElementId, abortControllerRef) {
    const token = getToken();
    if (!token) {
        showResponse({ error: 'Brak tokenu autoryzacji' }, 401);
        return null;
    }

    const controller = new AbortController();
    const statusEl = document.getElementById(statusElementId);

    try {
        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${token}`
            },
            signal: controller.signal
        });

        if (!response.ok) {
            statusEl.innerHTML = `<div style="color: #dc3545;">Błąd: ${response.status}</div>`;
            return null;
        }

        statusEl.innerHTML = '<div style="color: #28a745;">Połączono - czekam na dane...</div>';

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        const readStream = async () => {
            while (true) {
                try {
                    const { done, value } = await reader.read();
                    if (done) break;

                    buffer += decoder.decode(value, { stream: true });
                    const lines = buffer.split('\n');
                    buffer = lines.pop();

                    for (const line of lines) {
                        if (line.startsWith('data: ')) {
                            try {
                                const data = JSON.parse(line.slice(6));
                                const timestamp = new Date().toLocaleTimeString();
                                statusEl.innerHTML = `<div style="color: #007bff;">[${timestamp}]</div>` +
                                                     `<pre>${JSON.stringify(data, null, 2)}</pre>`;
                                statusEl.scrollTop = statusEl.scrollHeight;
                            } catch (e) {
                                statusEl.innerHTML += `<div style="color: #dc3545;">Błąd parsowania: ${e.message}</div>`;
                            }
                        }
                    }
                } catch (error) {
                    if (error.name === 'AbortError') {
                        statusEl.innerHTML = '<div style="color: #6c757d;">Rozłączono</div>';
                    } else {
                        statusEl.innerHTML += `<div style="color: #dc3545;">Błąd: ${error.message}</div>`;
                    }
                    break;
                }
            }
        };

        readStream();
        return controller;
    } catch (error) {
        statusEl.innerHTML = `<div style="color: #dc3545;">Błąd połączenia: ${error.message}</div>`;
        return null;
    }
}

async function startLiveAll() {
    if (liveAllAbortController) {
        showResponse({ warning: 'Połączenie już aktywne' }, 200);
        return;
    }

    const url = `${API_BASE}/api/metrics/live/all`;
    liveAllAbortController = await startSSEStream(url, 'liveAllStatus', liveAllAbortController);

    if (liveAllAbortController) {
        showResponse({ message: 'Rozpoczęto nasłuchiwanie SSE /api/metrics/live/all' }, 200);
    }
}

function stopLiveAll() {
    if (liveAllAbortController) {
        liveAllAbortController.abort();
        liveAllAbortController = null;
        showResponse({ message: 'Zatrzymano nasłuchiwanie SSE' }, 200);
    }
}

async function startLiveServer() {
    if (liveServerAbortController) {
        showResponse({ warning: 'Połączenie już aktywne' }, 200);
        return;
    }

    const uuid = document.getElementById('liveServerUuid').value;
    if (!uuid) {
        showResponse({ error: 'Podaj UUID serwera' }, 400);
        return;
    }

    const url = `${API_BASE}/api/metrics/live/servers/${uuid}`;
    liveServerAbortController = await startSSEStream(url, 'liveServerStatus', liveServerAbortController);

    if (liveServerAbortController) {
        showResponse({ message: `Rozpoczęto nasłuchiwanie SSE /api/metrics/live/servers/${uuid}` }, 200);
    }
}

function stopLiveServer() {
    if (liveServerAbortController) {
        liveServerAbortController.abort();
        liveServerAbortController = null;
        showResponse({ message: 'Zatrzymano nasłuchiwanie SSE' }, 200);
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    updateAuthStatus();

    // Clear token button
    document.getElementById('clearToken').addEventListener('click', clearToken);

    // Clean up SSE connections on page unload
    window.addEventListener('beforeunload', () => {
        stopLiveAll();
        stopLiveServer();
    });
});
