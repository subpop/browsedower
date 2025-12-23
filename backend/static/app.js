// ========================================
// Watchtower Admin - Application
// ========================================

const API_BASE = '/api';

// ========================================
// State
// ========================================

let currentPage = 'requests';
let devices = [];
let patterns = [];
let requests = [];
let users = [];
let pushSupported = false;
let pushSubscription = null;
let notificationPrefs = { notify_new_requests: true, notify_device_status: true };

// ========================================
// DOM Elements
// ========================================

const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => document.querySelectorAll(selector);

// ========================================
// API Helpers
// ========================================

async function api(endpoint, options = {}) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
        credentials: 'include',
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        ...options
    });
    
    if (response.status === 401) {
        showLoginScreen();
        throw new Error('Unauthorized');
    }
    
    const data = await response.json();
    
    if (!response.ok) {
        throw new Error(data.message || 'Request failed');
    }
    
    return data;
}

// ========================================
// Toast Notifications
// ========================================

function showToast(message, type = 'success') {
    const container = $('#toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.innerHTML = `
        <svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            ${type === 'success' 
                ? '<path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/>'
                : '<circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/>'
            }
        </svg>
        <span class="toast-message">${message}</span>
    `;
    container.appendChild(toast);
    
    setTimeout(() => {
        toast.style.animation = 'toastIn 0.3s ease reverse';
        setTimeout(() => toast.remove(), 300);
    }, 3000);
}

// ========================================
// Modal
// ========================================

function showModal(title, content) {
    $('#modal-title').textContent = title;
    $('#modal-body').innerHTML = content;
    $('#modal').classList.remove('hidden');
}

function hideModal() {
    $('#modal').classList.add('hidden');
}

// ========================================
// Authentication
// ========================================

function showSetupScreen() {
    $('#setup-screen').classList.remove('hidden');
    $('#login-screen').classList.add('hidden');
    $('#dashboard').classList.add('hidden');
}

function showLoginScreen() {
    $('#setup-screen').classList.add('hidden');
    $('#login-screen').classList.remove('hidden');
    $('#dashboard').classList.add('hidden');
}

function showDashboard() {
    $('#setup-screen').classList.add('hidden');
    $('#login-screen').classList.add('hidden');
    $('#dashboard').classList.remove('hidden');
    loadAllData();
    initPushNotifications();
}

async function login(username, password) {
    try {
        await api('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password })
        });
        showDashboard();
    } catch (error) {
        throw error;
    }
}

async function logout() {
    try {
        await api('/auth/logout', { method: 'POST' });
    } catch (error) {
        // Ignore errors
    }
    showLoginScreen();
}

async function checkSetupNeeded() {
    try {
        const response = await fetch(`${API_BASE}/setup/status`);
        const data = await response.json();
        return data.setup_needed;
    } catch (error) {
        console.error('Failed to check setup status:', error);
        return false;
    }
}

async function checkAuth() {
    // First check if initial setup is needed
    const needsSetup = await checkSetupNeeded();
    if (needsSetup) {
        showSetupScreen();
        return;
    }
    
    try {
        await api('/admin/requests?status=pending');
        showDashboard();
    } catch (error) {
        showLoginScreen();
    }
}

// ========================================
// Data Loading
// ========================================

async function loadAllData() {
    await Promise.all([
        loadRequests(),
        loadPatterns(),
        loadDevices(),
        loadUsers()
    ]);
}

async function loadRequests() {
    const filter = $('#request-filter').value;
    try {
        const data = await api(`/admin/requests?status=${filter}`);
        requests = data.requests || [];
        renderRequests();
        updatePendingBadge();
    } catch (error) {
        showToast('Failed to load requests', 'error');
    }
}

async function loadPatterns() {
    try {
        const data = await api('/admin/patterns');
        patterns = data.patterns || [];
        renderPatterns();
    } catch (error) {
        showToast('Failed to load patterns', 'error');
    }
}

async function loadDevices() {
    try {
        const data = await api('/admin/devices');
        devices = data.devices || [];
        renderDevices();
    } catch (error) {
        showToast('Failed to load devices', 'error');
    }
}

async function loadUsers() {
    try {
        const data = await api('/admin/users');
        users = data.users || [];
        renderUsers();
    } catch (error) {
        showToast('Failed to load users', 'error');
    }
}

function updatePendingBadge() {
    const pendingCount = requests.filter(r => r.status === 'pending').length;
    const badge = $('#pending-badge');
    if (pendingCount > 0) {
        badge.textContent = pendingCount;
        badge.classList.remove('hidden');
    } else {
        badge.classList.add('hidden');
    }
}

// ========================================
// Rendering
// ========================================

function renderRequests() {
    const container = $('#requests-list');
    
    if (requests.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M9 11l3 3L22 4"/>
                    <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/>
                </svg>
                <p>No requests found</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = requests.map(req => `
        <div class="card request-card" data-id="${req.id}">
            <div class="request-info">
                <h4>Access Request</h4>
                <div class="request-url">${escapeHtml(req.url)}</div>
                <div class="request-meta">
                    <span>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                            <line x1="8" y1="21" x2="16" y2="21"/>
                            <line x1="12" y1="17" x2="12" y2="21"/>
                        </svg>
                        ${escapeHtml(req.device_name)}
                    </span>
                    <span>
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <circle cx="12" cy="12" r="10"/>
                            <polyline points="12 6 12 12 16 14"/>
                        </svg>
                        ${formatDate(req.created_at)}
                    </span>
                    <span class="status-badge ${req.status}">${req.status}</span>
                </div>
            </div>
            ${req.status === 'pending' ? `
                <div class="request-actions">
                    <button class="btn btn-success btn-small" onclick="showApproveModal(${req.id}, '${escapeHtml(req.url)}', ${req.device_id})">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="20 6 9 17 4 12"/>
                        </svg>
                        Approve
                    </button>
                    <button class="btn btn-danger btn-small" onclick="denyRequest(${req.id})">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="18" y1="6" x2="6" y2="18"/>
                            <line x1="6" y1="6" x2="18" y2="18"/>
                        </svg>
                        Deny
                    </button>
                </div>
            ` : ''}
        </div>
    `).join('');
}

function renderPatterns() {
    const tbody = $('#patterns-tbody');
    
    if (patterns.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="6" style="text-align: center; padding: 3rem;">
                    <div class="empty-state">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                            <path d="M4 6h16M4 12h16M4 18h16"/>
                        </svg>
                        <p>No patterns defined</p>
                    </div>
                </td>
            </tr>
        `;
        return;
    }
    
    tbody.innerHTML = patterns.map(p => {
        const device = devices.find(d => d.id === p.device_id);
        const expiresText = p.expires_at 
            ? formatDate(p.expires_at)
            : 'Never';
        const isExpired = p.expires_at && new Date(p.expires_at) < new Date();
        const isEnabled = p.enabled !== false; // Default to true if undefined
        
        return `
            <tr class="pattern-row ${!isEnabled ? 'pattern-disabled' : ''}">
                <td data-label="Enabled">
                    <label class="toggle-switch" title="${isEnabled ? 'Disable pattern' : 'Enable pattern'}">
                        <input type="checkbox" ${isEnabled ? 'checked' : ''} onchange="togglePattern(${p.id}, this.checked)">
                        <span class="toggle-slider"></span>
                    </label>
                </td>
                <td data-label="Pattern"><span class="pattern-text">${escapeHtml(p.pattern)}</span></td>
                <td data-label="Type"><span class="type-badge ${p.type}">${p.type}</span></td>
                <td data-label="Device">${device ? escapeHtml(device.name) : 'Unknown'}</td>
                <td data-label="Expires"><span class="expires-text ${isExpired ? 'expired' : ''}">${expiresText}</span></td>
                <td data-label="Actions" class="pattern-actions">
                    <div class="action-buttons">
                        <button class="btn btn-secondary btn-small" onclick="showEditPatternModal(${p.id})">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/>
                                <path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/>
                            </svg>
                            Edit
                        </button>
                        <button class="btn btn-danger btn-small" onclick="deletePattern(${p.id})">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="3 6 5 6 21 6"/>
                                <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                            </svg>
                            Delete
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

function renderDevices() {
    const container = $('#devices-list');
    
    if (devices.length === 0) {
        container.innerHTML = `
            <div class="empty-state" style="grid-column: 1/-1;">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                    <line x1="8" y1="21" x2="16" y2="21"/>
                    <line x1="12" y1="17" x2="12" y2="21"/>
                </svg>
                <p>No devices registered</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = devices.map(d => {
        const statusClass = d.status || 'active';
        const statusIcon = getDeviceStatusIcon(statusClass);
        const lastSeenText = d.last_seen ? `Last seen ${formatDate(d.last_seen)}` : 'Never connected';
        
        return `
            <div class="card device-card" data-id="${d.id}">
                <div style="display: flex; gap: 1rem; align-items: flex-start;">
                    <div class="device-icon status-${statusClass}">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
                            <line x1="8" y1="21" x2="16" y2="21"/>
                            <line x1="12" y1="17" x2="12" y2="21"/>
                        </svg>
                    </div>
                    <div class="device-info">
                        <h4>
                            ${escapeHtml(d.name)}
                            <span class="device-status-badge ${statusClass}" title="${getStatusTitle(statusClass)}">
                                ${statusIcon}
                            </span>
                        </h4>
                        <p>${lastSeenText}</p>
                        <p style="font-size: 0.75rem; color: var(--text-muted);">Created ${formatDate(d.created_at)}</p>
                    </div>
                </div>
                <div class="card-actions">
                    <button class="btn btn-secondary btn-small" onclick="regenerateDeviceToken(${d.id}, '${escapeHtml(d.name)}')">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M23 4v6h-6M1 20v-6h6"/>
                            <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/>
                        </svg>
                        Regenerate Token
                    </button>
                    <button class="btn btn-danger btn-small" onclick="deleteDevice(${d.id})">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"/>
                            <path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/>
                        </svg>
                        Delete
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

function getDeviceStatusIcon(status) {
    switch (status) {
        case 'active':
            return '<span style="color: var(--accent-success);">●</span>';
        case 'inactive':
            return '<span style="color: var(--accent-warning);">●</span>';
        case 'uninstalled':
            return '<span style="color: var(--accent-danger);">●</span>';
        default:
            return '<span style="color: var(--text-muted);">●</span>';
    }
}

function getStatusTitle(status) {
    switch (status) {
        case 'active':
            return 'Active - Extension is running';
        case 'inactive':
            return 'Inactive - No recent heartbeat';
        case 'uninstalled':
            return 'Uninstalled - Extension was removed';
        default:
            return 'Unknown status';
    }
}

function renderUsers() {
    const container = $('#users-list');
    
    if (users.length === 0) {
        container.innerHTML = `
            <div class="empty-state" style="grid-column: 1/-1;">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2"/>
                    <circle cx="9" cy="7" r="4"/>
                </svg>
                <p>No users found</p>
            </div>
        `;
        return;
    }
    
    container.innerHTML = users.map(u => `
        <div class="card user-card" data-id="${u.id}">
            <div style="display: flex; gap: 1rem; align-items: flex-start;">
                <div class="user-icon">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                        <circle cx="12" cy="7" r="4"/>
                    </svg>
                </div>
                <div class="user-info">
                    <h4>${escapeHtml(u.username)}</h4>
                    <p>Created ${formatDate(u.created_at)}</p>
                </div>
            </div>
        </div>
    `).join('');
}

// ========================================
// Actions
// ========================================

function showApproveModal(requestId, url, deviceId) {
    const suggestedPattern = extractDomain(url);
    
    showModal('Approve Request', `
        <form id="approve-form">
            <div class="form-group">
                <label>URL</label>
                <input type="text" value="${escapeHtml(url)}" disabled style="opacity: 0.7;">
            </div>
            <div class="form-group">
                <label for="approve-pattern">Pattern</label>
                <input type="text" id="approve-pattern" value="${escapeHtml(suggestedPattern)}" required>
                <small style="color: var(--text-muted); font-size: 0.8rem;">
                    Use * as wildcard (e.g., *.example.com/*)
                </small>
            </div>
            <div class="form-group">
                <label for="approve-type">Type</label>
                <select id="approve-type" class="select" style="width: 100%;">
                    <option value="allow">Allow</option>
                    <option value="deny">Deny</option>
                </select>
            </div>
            <div class="form-group">
                <label for="approve-duration">Duration</label>
                <select id="approve-duration" class="select" style="width: 100%;">
                    <option value="15m">15 Minutes</option>
                    <option value="30m">30 Minutes</option>
                    <option value="1h" selected>1 Hour</option>
                    <option value="8h">8 Hours</option>
                    <option value="24h">24 Hours</option>
                    <option value="1w">1 Week</option>
                    <option value="custom">Custom...</option>
                    <option value="permanent">Permanent</option>
                </select>
            </div>
            <div id="approve-custom-duration" class="form-group" style="display: none;">
                <label for="approve-custom-minutes">Custom Duration (minutes)</label>
                <input type="number" id="approve-custom-minutes" min="1" max="525600" placeholder="Enter minutes">
            </div>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-success">Approve</button>
            </div>
        </form>
    `);
    
    // Toggle custom duration field
    $('#approve-duration').onchange = (e) => {
        $('#approve-custom-duration').style.display = e.target.value === 'custom' ? 'block' : 'none';
        if (e.target.value === 'custom') {
            $('#approve-custom-minutes').focus();
        }
    };
    
    $('#approve-form').onsubmit = async (e) => {
        e.preventDefault();
        const pattern = $('#approve-pattern').value;
        const type = $('#approve-type').value;
        const duration = $('#approve-duration').value;
        const customMinutes = duration === 'custom' ? parseInt($('#approve-custom-minutes').value) || 0 : 0;
        
        if (duration === 'custom' && customMinutes <= 0) {
            showToast('Please enter a valid number of minutes', 'error');
            return;
        }
        
        try {
            await api(`/admin/requests/${requestId}/approve`, {
                method: 'POST',
                body: JSON.stringify({ pattern, type, duration, custom_minutes: customMinutes })
            });
            hideModal();
            showToast('Request approved');
            loadRequests();
            loadPatterns();
        } catch (error) {
            showToast('Failed to approve request', 'error');
        }
    };
}

async function denyRequest(requestId) {
    if (!confirm('Are you sure you want to deny this request?')) return;
    
    try {
        await api(`/admin/requests/${requestId}/deny`, { method: 'POST' });
        showToast('Request denied');
        loadRequests();
    } catch (error) {
        showToast('Failed to deny request', 'error');
    }
}

async function deletePattern(patternId) {
    if (!confirm('Are you sure you want to delete this pattern?')) return;
    
    try {
        await api(`/admin/patterns/${patternId}`, { method: 'DELETE' });
        showToast('Pattern deleted');
        loadPatterns();
    } catch (error) {
        showToast('Failed to delete pattern', 'error');
    }
}

async function togglePattern(patternId, enabled) {
    try {
        await api(`/admin/patterns/${patternId}/toggle`, {
            method: 'POST',
            body: JSON.stringify({ enabled })
        });
        showToast(enabled ? 'Pattern enabled' : 'Pattern disabled');
        loadPatterns();
    } catch (error) {
        showToast('Failed to toggle pattern', 'error');
        loadPatterns(); // Reload to reset the toggle state
    }
}

function showEditPatternModal(patternId) {
    const pattern = patterns.find(p => p.id === patternId);
    if (!pattern) {
        showToast('Pattern not found', 'error');
        return;
    }
    
    const device = devices.find(d => d.id === pattern.device_id);
    
    showModal('Edit Pattern', `
        <form id="edit-pattern-form">
            <div class="form-group">
                <label>Device</label>
                <input type="text" value="${device ? escapeHtml(device.name) : 'Unknown'}" disabled style="opacity: 0.7;">
            </div>
            <div class="form-group">
                <label for="edit-pattern-text">Pattern</label>
                <input type="text" id="edit-pattern-text" value="${escapeHtml(pattern.pattern)}" required>
                <small style="color: var(--text-muted); font-size: 0.8rem;">
                    Use * as wildcard
                </small>
            </div>
            <div class="form-group">
                <label for="edit-pattern-type">Type</label>
                <select id="edit-pattern-type" class="select" style="width: 100%;">
                    <option value="allow" ${pattern.type === 'allow' ? 'selected' : ''}>Allow</option>
                    <option value="deny" ${pattern.type === 'deny' ? 'selected' : ''}>Deny</option>
                </select>
            </div>
            <div class="form-group">
                <label for="edit-pattern-duration">Duration</label>
                <select id="edit-pattern-duration" class="select" style="width: 100%;">
                    <option value="15m">15 Minutes (from now)</option>
                    <option value="30m">30 Minutes (from now)</option>
                    <option value="1h">1 Hour (from now)</option>
                    <option value="8h">8 Hours (from now)</option>
                    <option value="24h">24 Hours (from now)</option>
                    <option value="1w">1 Week (from now)</option>
                    <option value="custom">Custom...</option>
                    <option value="permanent" ${!pattern.expires_at ? 'selected' : ''}>Permanent</option>
                </select>
                ${pattern.expires_at ? `
                    <small style="color: var(--text-muted); font-size: 0.8rem;">
                        Current expiration: ${formatDate(pattern.expires_at)}
                    </small>
                ` : ''}
            </div>
            <div id="edit-custom-duration" class="form-group" style="display: none;">
                <label for="edit-custom-minutes">Custom Duration (minutes)</label>
                <input type="number" id="edit-custom-minutes" min="1" max="525600" placeholder="Enter minutes">
            </div>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Save Changes</button>
            </div>
        </form>
    `);
    
    // Toggle custom duration field
    $('#edit-pattern-duration').onchange = (e) => {
        $('#edit-custom-duration').style.display = e.target.value === 'custom' ? 'block' : 'none';
        if (e.target.value === 'custom') {
            $('#edit-custom-minutes').focus();
        }
    };
    
    $('#edit-pattern-form').onsubmit = async (e) => {
        e.preventDefault();
        const patternText = $('#edit-pattern-text').value;
        const type = $('#edit-pattern-type').value;
        const duration = $('#edit-pattern-duration').value;
        const customMinutes = duration === 'custom' ? parseInt($('#edit-custom-minutes').value) || 0 : 0;
        
        if (duration === 'custom' && customMinutes <= 0) {
            showToast('Please enter a valid number of minutes', 'error');
            return;
        }
        
        try {
            await api(`/admin/patterns/${patternId}`, {
                method: 'PUT',
                body: JSON.stringify({ pattern: patternText, type, duration, custom_minutes: customMinutes })
            });
            hideModal();
            showToast('Pattern updated');
            loadPatterns();
        } catch (error) {
            showToast('Failed to update pattern', 'error');
        }
    };
}

async function regenerateDeviceToken(deviceId, deviceName) {
    if (!confirm(`Are you sure you want to regenerate the token for "${deviceName}"?\n\nThe old token will stop working immediately. You will need to update the extension configuration with the new token.`)) {
        return;
    }
    
    try {
        const device = await api(`/admin/devices/${deviceId}/regenerate-token`, {
            method: 'POST'
        });
        
        showModal('New Token Generated', `
            <p style="margin-bottom: 1rem; color: var(--text-secondary);">
                A new token has been generated for "${escapeHtml(deviceName)}". Copy it below and update the browser extension configuration.
            </p>
            <div style="background: var(--bg-tertiary); padding: 1rem; border-radius: var(--radius-sm); margin-bottom: 1rem;">
                <code id="new-device-token" style="font-family: var(--font-mono); word-break: break-all; font-size: 0.85rem; color: var(--accent-primary);">
                    ${escapeHtml(device.token)}
                </code>
            </div>
            <p style="color: var(--accent-warning); font-size: 0.85rem;">
                <strong>Important:</strong> Save this token now. It won't be shown again.
            </p>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="copyToken('${device.token}')">Copy Token</button>
                <button type="button" class="btn btn-primary" onclick="hideModal()">Done</button>
            </div>
        `);
        
        showToast('Token regenerated successfully');
        loadDevices();
    } catch (error) {
        showToast('Failed to regenerate token', 'error');
    }
}

async function deleteDevice(deviceId) {
    if (!confirm('Are you sure you want to delete this device? All associated patterns will also be deleted.')) return;
    
    try {
        await api(`/admin/devices/${deviceId}`, { method: 'DELETE' });
        showToast('Device deleted');
        loadDevices();
        loadPatterns();
    } catch (error) {
        showToast('Failed to delete device', 'error');
    }
}

function showAddDeviceModal() {
    showModal('Add Device', `
        <form id="add-device-form">
            <div class="form-group">
                <label for="device-name">Device Name</label>
                <input type="text" id="device-name" placeholder="e.g., Work Laptop, Home PC" required>
            </div>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Create Device</button>
            </div>
        </form>
    `);
    
    $('#add-device-form').onsubmit = async (e) => {
        e.preventDefault();
        const name = $('#device-name').value;
        
        try {
            const device = await api('/admin/devices', {
                method: 'POST',
                body: JSON.stringify({ name })
            });
            
            // Show the token to the user
            showModal('Device Created', `
                <p style="margin-bottom: 1rem; color: var(--text-secondary);">
                    Device "${escapeHtml(name)}" has been created. Copy the token below and use it to configure the browser extension.
                </p>
                <div style="background: var(--bg-tertiary); padding: 1rem; border-radius: var(--radius-sm); margin-bottom: 1rem;">
                    <code id="new-device-token" style="font-family: var(--font-mono); word-break: break-all; font-size: 0.85rem; color: var(--accent-primary);">
                        ${escapeHtml(device.token)}
                    </code>
                </div>
                <p style="color: var(--accent-warning); font-size: 0.85rem;">
                    <strong>Important:</strong> Save this token now. It won't be shown again.
                </p>
                <div class="modal-actions">
                    <button type="button" class="btn btn-secondary" onclick="copyToken('${device.token}')">Copy Token</button>
                    <button type="button" class="btn btn-primary" onclick="hideModal(); loadDevices();">Done</button>
                </div>
            `);
        } catch (error) {
            showToast('Failed to create device', 'error');
        }
    };
}

function copyToken(token) {
    navigator.clipboard.writeText(token).then(() => {
        showToast('Token copied to clipboard');
    }).catch(() => {
        showToast('Failed to copy token', 'error');
    });
}

function showAddPatternModal() {
    if (devices.length === 0) {
        showToast('Please add a device first', 'error');
        return;
    }
    
    const deviceOptions = devices.map(d => 
        `<option value="${d.id}">${escapeHtml(d.name)}</option>`
    ).join('');
    
    showModal('Add Pattern', `
        <form id="add-pattern-form">
            <div class="form-group">
                <label for="pattern-device">Device</label>
                <select id="pattern-device" class="select" style="width: 100%;" required>
                    ${deviceOptions}
                </select>
            </div>
            <div class="form-group">
                <label for="pattern-text">Pattern</label>
                <input type="text" id="pattern-text" placeholder="e.g., *.example.com/*" required>
                <small style="color: var(--text-muted); font-size: 0.8rem;">
                    Use * as wildcard
                </small>
            </div>
            <div class="form-group">
                <label for="pattern-type">Type</label>
                <select id="pattern-type" class="select" style="width: 100%;">
                    <option value="allow">Allow</option>
                    <option value="deny">Deny</option>
                </select>
            </div>
            <div class="form-group">
                <label for="pattern-duration">Duration</label>
                <select id="pattern-duration" class="select" style="width: 100%;">
                    <option value="15m">15 Minutes</option>
                    <option value="30m">30 Minutes</option>
                    <option value="1h" selected>1 Hour</option>
                    <option value="8h">8 Hours</option>
                    <option value="24h">24 Hours</option>
                    <option value="1w">1 Week</option>
                    <option value="custom">Custom...</option>
                    <option value="permanent">Permanent</option>
                </select>
            </div>
            <div id="add-custom-duration" class="form-group" style="display: none;">
                <label for="add-custom-minutes">Custom Duration (minutes)</label>
                <input type="number" id="add-custom-minutes" min="1" max="525600" placeholder="Enter minutes">
            </div>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Add Pattern</button>
            </div>
        </form>
    `);
    
    // Toggle custom duration field
    $('#pattern-duration').onchange = (e) => {
        $('#add-custom-duration').style.display = e.target.value === 'custom' ? 'block' : 'none';
        if (e.target.value === 'custom') {
            $('#add-custom-minutes').focus();
        }
    };
    
    $('#add-pattern-form').onsubmit = async (e) => {
        e.preventDefault();
        const device_id = parseInt($('#pattern-device').value);
        const pattern = $('#pattern-text').value;
        const type = $('#pattern-type').value;
        const duration = $('#pattern-duration').value;
        const customMinutes = duration === 'custom' ? parseInt($('#add-custom-minutes').value) || 0 : 0;
        
        if (duration === 'custom' && customMinutes <= 0) {
            showToast('Please enter a valid number of minutes', 'error');
            return;
        }
        
        try {
            await api('/admin/patterns', {
                method: 'POST',
                body: JSON.stringify({ device_id, pattern, type, duration, custom_minutes: customMinutes })
            });
            hideModal();
            showToast('Pattern added');
            loadPatterns();
        } catch (error) {
            showToast('Failed to add pattern', 'error');
        }
    };
}

function showAddUserModal() {
    showModal('Add User', `
        <form id="add-user-form">
            <div class="form-group">
                <label for="new-username">Username</label>
                <input type="text" id="new-username" required autocomplete="off">
            </div>
            <div class="form-group">
                <label for="new-password">Password</label>
                <input type="password" id="new-password" required autocomplete="new-password">
            </div>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Create User</button>
            </div>
        </form>
    `);
    
    $('#add-user-form').onsubmit = async (e) => {
        e.preventDefault();
        const username = $('#new-username').value;
        const password = $('#new-password').value;
        
        try {
            await api('/admin/users', {
                method: 'POST',
                body: JSON.stringify({ username, password })
            });
            hideModal();
            showToast('User created');
            loadUsers();
        } catch (error) {
            showToast('Failed to create user', 'error');
        }
    };
}

// ========================================
// Navigation
// ========================================

function navigateTo(page) {
    currentPage = page;
    
    // Update nav items
    $$('.nav-item').forEach(item => {
        item.classList.toggle('active', item.dataset.page === page);
    });
    
    // Update pages
    $$('.page').forEach(p => {
        p.classList.toggle('hidden', p.id !== `page-${page}`);
    });
    
    // Reload data for the page
    switch (page) {
        case 'requests':
            loadRequests();
            break;
        case 'patterns':
            loadPatterns();
            break;
        case 'devices':
            loadDevices();
            break;
        case 'users':
            loadUsers();
            break;
    }
}

// ========================================
// Utilities
// ========================================

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateStr) {
    if (!dateStr) return 'Unknown';
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    const absDiff = Math.abs(diff);
    const isFuture = diff < 0;
    
    // Less than a minute
    if (absDiff < 60000) {
        return isFuture ? 'in less than a minute' : 'Just now';
    }
    
    // Less than an hour
    if (absDiff < 3600000) {
        const mins = Math.floor(absDiff / 60000);
        return isFuture 
            ? `in ${mins} min${mins > 1 ? 's' : ''}`
            : `${mins} min${mins > 1 ? 's' : ''} ago`;
    }
    
    // Less than a day
    if (absDiff < 86400000) {
        const hours = Math.floor(absDiff / 3600000);
        return isFuture
            ? `in ${hours} hour${hours > 1 ? 's' : ''}`
            : `${hours} hour${hours > 1 ? 's' : ''} ago`;
    }
    
    // Less than a week
    if (absDiff < 604800000) {
        const days = Math.floor(absDiff / 86400000);
        return isFuture
            ? `in ${days} day${days > 1 ? 's' : ''}`
            : `${days} day${days > 1 ? 's' : ''} ago`;
    }
    
    // Format as date
    return date.toLocaleDateString(undefined, {
        month: 'short',
        day: 'numeric',
        year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
    });
}

function extractDomain(url) {
    try {
        const parsed = new URL(url);
        return parsed.hostname + '/*';
    } catch {
        return url;
    }
}

// ========================================
// Push Notifications
// ========================================

async function initPushNotifications() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        console.log('Push notifications not supported');
        return;
    }
    
    pushSupported = true;
    
    try {
        // Register service worker
        const registration = await navigator.serviceWorker.register('/admin/sw.js');
        console.log('Service worker registered');
        
        // Check existing subscription
        pushSubscription = await registration.pushManager.getSubscription();
        
        // Load notification preferences
        await loadNotificationPrefs();
        
        updatePushUI();
    } catch (error) {
        console.error('Push initialization failed:', error);
    }
}

async function loadNotificationPrefs() {
    try {
        notificationPrefs = await api('/admin/notifications/prefs');
    } catch (error) {
        console.error('Failed to load notification prefs:', error);
    }
}

function updatePushUI() {
    const indicator = $('#notification-indicator');
    if (indicator) {
        if (pushSubscription) {
            indicator.classList.add('active');
            indicator.title = 'Notifications enabled';
        } else {
            indicator.classList.remove('active');
            indicator.title = 'Notifications disabled';
        }
    }
}

async function subscribePush() {
    if (!pushSupported) {
        showToast('Push notifications not supported in this browser', 'error');
        return;
    }
    
    try {
        // Request notification permission
        const permission = await Notification.requestPermission();
        if (permission !== 'granted') {
            showToast('Notification permission denied', 'error');
            return;
        }
        
        // Get VAPID public key
        const vapidResponse = await api('/admin/push/vapid-key');
        const vapidPublicKey = vapidResponse.publicKey;
        
        // Convert VAPID key to Uint8Array
        const applicationServerKey = urlBase64ToUint8Array(vapidPublicKey);
        
        // Subscribe
        const registration = await navigator.serviceWorker.ready;
        pushSubscription = await registration.pushManager.subscribe({
            userVisibleOnly: true,
            applicationServerKey
        });
        
        // Send subscription to server
        await api('/admin/push/subscribe', {
            method: 'POST',
            body: JSON.stringify(pushSubscription.toJSON())
        });
        
        updatePushUI();
        showToast('Notifications enabled');
    } catch (error) {
        console.error('Push subscription failed:', error);
        showToast('Failed to enable notifications', 'error');
    }
}

async function unsubscribePush() {
    if (!pushSubscription) return;
    
    try {
        const endpoint = pushSubscription.endpoint;
        await pushSubscription.unsubscribe();
        pushSubscription = null;
        
        // Remove from server
        await api('/admin/push/unsubscribe', {
            method: 'POST',
            body: JSON.stringify({ endpoint })
        });
        
        updatePushUI();
        showToast('Notifications disabled');
    } catch (error) {
        console.error('Push unsubscription failed:', error);
        showToast('Failed to disable notifications', 'error');
    }
}

async function updateNotificationPrefs(notifyNewRequests, notifyDeviceStatus) {
    try {
        await api('/admin/notifications/prefs', {
            method: 'PUT',
            body: JSON.stringify({
                notify_new_requests: notifyNewRequests,
                notify_device_status: notifyDeviceStatus
            })
        });
        
        notificationPrefs = { notify_new_requests: notifyNewRequests, notify_device_status: notifyDeviceStatus };
        showToast('Notification preferences updated');
    } catch (error) {
        showToast('Failed to update preferences', 'error');
    }
}

function showChangePasswordModal() {
    showModal('Change Password', `
        <form id="change-password-form">
            <div class="form-group">
                <label for="current-password">Current Password</label>
                <input type="password" id="current-password" required autocomplete="current-password">
            </div>
            <div class="form-group">
                <label for="new-password">New Password</label>
                <input type="password" id="new-password" required autocomplete="new-password" minlength="8">
                <span class="input-hint">Minimum 8 characters</span>
            </div>
            <div class="form-group">
                <label for="confirm-new-password">Confirm New Password</label>
                <input type="password" id="confirm-new-password" required autocomplete="new-password">
            </div>
            <p id="change-password-error" class="error-message"></p>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" onclick="hideModal()">Cancel</button>
                <button type="submit" class="btn btn-primary">Change Password</button>
            </div>
        </form>
    `);
    
    $('#change-password-form').onsubmit = async (e) => {
        e.preventDefault();
        const currentPassword = $('#current-password').value;
        const newPassword = $('#new-password').value;
        const confirmPassword = $('#confirm-new-password').value;
        
        // Client-side validation
        if (newPassword.length < 8) {
            $('#change-password-error').textContent = 'New password must be at least 8 characters';
            return;
        }
        
        if (newPassword !== confirmPassword) {
            $('#change-password-error').textContent = 'Passwords do not match';
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE}/auth/change-password`, {
                method: 'POST',
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    current_password: currentPassword,
                    new_password: newPassword,
                    confirm_password: confirmPassword
                })
            });
            
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || 'Failed to change password');
            }
            
            hideModal();
            showToast('Password changed successfully', 'success');
        } catch (error) {
            $('#change-password-error').textContent = error.message;
        }
    };
}

function showNotificationSettingsModal() {
    const subscribed = !!pushSubscription;
    
    showModal('Notification Settings', `
        <div style="margin-bottom: 1.5rem;">
            <h4 style="margin-bottom: 0.75rem; font-size: 0.9rem; color: var(--text-secondary);">Browser Notifications</h4>
            <p style="margin-bottom: 1rem; color: var(--text-muted); font-size: 0.85rem;">
                ${!pushSupported 
                    ? 'Push notifications are not supported in this browser.'
                    : subscribed 
                        ? 'Push notifications are enabled on this browser.'
                        : 'Enable push notifications to receive alerts about new access requests and device status changes.'
                }
            </p>
            ${pushSupported ? `
                <button type="button" class="btn ${subscribed ? 'btn-danger' : 'btn-primary'}" 
                        onclick="${subscribed ? 'unsubscribePush()' : 'subscribePush()'}; hideModal();">
                    ${subscribed ? 'Disable Notifications' : 'Enable Notifications'}
                </button>
            ` : ''}
        </div>
        
        <div style="border-top: 1px solid var(--border-color); padding-top: 1.5rem;">
            <h4 style="margin-bottom: 0.75rem; font-size: 0.9rem; color: var(--text-secondary);">Notification Types</h4>
            <p style="margin-bottom: 1rem; color: var(--text-muted); font-size: 0.85rem;">
                Choose which notifications you want to receive:
            </p>
            <form id="notification-prefs-form">
                <label style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 0.75rem; cursor: pointer;">
                    <input type="checkbox" id="pref-new-requests" ${notificationPrefs.notify_new_requests ? 'checked' : ''} 
                           style="width: 18px; height: 18px;">
                    <span>New access requests</span>
                </label>
                <label style="display: flex; align-items: center; gap: 0.75rem; margin-bottom: 1rem; cursor: pointer;">
                    <input type="checkbox" id="pref-device-status" ${notificationPrefs.notify_device_status ? 'checked' : ''} 
                           style="width: 18px; height: 18px;">
                    <span>Device status changes (inactive, uninstalled)</span>
                </label>
                <div class="modal-actions" style="padding-top: 1rem;">
                    <button type="button" class="btn btn-secondary" onclick="hideModal()">Close</button>
                    <button type="submit" class="btn btn-primary">Save Preferences</button>
                </div>
            </form>
        </div>
    `);
    
    $('#notification-prefs-form').onsubmit = async (e) => {
        e.preventDefault();
        const notifyNewRequests = $('#pref-new-requests').checked;
        const notifyDeviceStatus = $('#pref-device-status').checked;
        await updateNotificationPrefs(notifyNewRequests, notifyDeviceStatus);
        hideModal();
    };
}

function urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
        .replace(/-/g, '+')
        .replace(/_/g, '/');
    
    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);
    
    for (let i = 0; i < rawData.length; ++i) {
        outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
}

// ========================================
// Event Listeners
// ========================================

document.addEventListener('DOMContentLoaded', () => {
    // Setup form
    $('#setup-form').onsubmit = async (e) => {
        e.preventDefault();
        const username = $('#setup-username').value;
        const password = $('#setup-password').value;
        const confirmPassword = $('#setup-confirm-password').value;
        
        // Client-side validation
        if (password.length < 8) {
            $('#setup-error').textContent = 'Password must be at least 8 characters';
            return;
        }
        
        if (password !== confirmPassword) {
            $('#setup-error').textContent = 'Passwords do not match';
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE}/setup/create-user`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    username,
                    password,
                    confirm_password: confirmPassword
                })
            });
            
            const data = await response.json();
            
            if (!response.ok) {
                throw new Error(data.error || 'Setup failed');
            }
            
            showToast('Account created successfully! Please sign in.', 'success');
            showLoginScreen();
            $('#setup-error').textContent = '';
        } catch (error) {
            $('#setup-error').textContent = error.message;
        }
    };
    
    // Login form
    $('#login-form').onsubmit = async (e) => {
        e.preventDefault();
        const username = $('#username').value;
        const password = $('#password').value;
        
        try {
            await login(username, password);
            $('#login-error').textContent = '';
        } catch (error) {
            $('#login-error').textContent = 'Invalid username or password';
        }
    };
    
    // Logout button
    $('#logout-btn').onclick = logout;
    
    // Notification settings
    const notifBtn = $('#notification-btn');
    if (notifBtn) notifBtn.onclick = showNotificationSettingsModal;
    
    // Change password
    const changePwBtn = $('#change-password-btn');
    if (changePwBtn) changePwBtn.onclick = showChangePasswordModal;
    
    // Mobile menu toggle
    const menuToggle = $('#menu-toggle');
    const sidebar = $('.sidebar');
    const overlay = $('#sidebar-overlay');
    
    if (menuToggle) {
        menuToggle.onclick = () => {
            sidebar.classList.toggle('open');
            overlay.classList.toggle('active');
        };
    }
    
    if (overlay) {
        overlay.onclick = () => {
            sidebar.classList.remove('open');
            overlay.classList.remove('active');
        };
    }
    
    // Navigation (close mobile menu after navigation)
    $$('.nav-item').forEach(item => {
        item.onclick = () => {
            navigateTo(item.dataset.page);
            // Close mobile menu
            if (window.innerWidth <= 768) {
                sidebar.classList.remove('open');
                overlay.classList.remove('active');
            }
        };
    });
    
    // Modal close
    $('.modal-backdrop').onclick = hideModal;
    $('.modal-close').onclick = hideModal;
    
    // Filter change
    $('#request-filter').onchange = loadRequests;
    
    // Refresh button
    $('#refresh-requests').onclick = loadRequests;
    
    // Add buttons
    $('#add-device-btn').onclick = showAddDeviceModal;
    $('#add-pattern-btn').onclick = showAddPatternModal;
    $('#add-user-btn').onclick = showAddUserModal;
    
    // Check auth on load
    checkAuth();
});

// Keyboard shortcuts
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        hideModal();
    }
});

