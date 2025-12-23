// ========================================
// Watchtower Extension - Background Service Worker
// ========================================

// Configuration
const CONFIG_KEY = 'watchtower_config';
const PATTERNS_KEY = 'watchtower_patterns';
const SYNC_INTERVAL = 2 * 60 * 1000; // 2 minutes (fallback)
const HEARTBEAT_INTERVAL = 1; // 1 minute
const WS_RECONNECT_INTERVAL = 1; // 1 minute - check/reconnect WebSocket

// Default configuration
const DEFAULT_CONFIG = {
    apiUrl: 'http://localhost:8080',
    token: '',
    lastSync: null,
    lastHeartbeat: null
};

// WebSocket connection state
let ws = null;
let wsConnected = false;

// ========================================
// Storage Helpers
// ========================================

async function getConfig() {
    const result = await chrome.storage.local.get(CONFIG_KEY);
    return { ...DEFAULT_CONFIG, ...result[CONFIG_KEY] };
}

async function setConfig(config) {
    await chrome.storage.local.set({ [CONFIG_KEY]: config });
}

async function getPatterns() {
    const result = await chrome.storage.local.get(PATTERNS_KEY);
    return result[PATTERNS_KEY] || { allow: [], deny: [] };
}

async function setPatterns(patterns) {
    await chrome.storage.local.set({ [PATTERNS_KEY]: patterns });
}

// ========================================
// API Communication
// ========================================

async function fetchPatterns() {
    const config = await getConfig();
    
    if (!config.token || !config.apiUrl) {
        console.log('Watchtower: Not configured');
        return null;
    }
    
    try {
        const response = await fetch(`${config.apiUrl}/api/patterns`, {
            headers: {
                'Authorization': `Bearer ${config.token}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const data = await response.json();
        
        // Organize patterns by type
        // Note: Backend already filters out expired patterns, so we just categorize here
        const patterns = {
            allow: [],
            deny: []
        };
        
        for (const pattern of (data.patterns || [])) {
            if (pattern.type === 'allow') {
                patterns.allow.push(pattern.pattern);
            } else if (pattern.type === 'deny') {
                patterns.deny.push(pattern.pattern);
            }
        }
        
        await setPatterns(patterns);
        
        // Update last sync time
        config.lastSync = new Date().toISOString();
        await setConfig(config);
        
        console.log('Watchtower: Patterns synced', patterns);
        return patterns;
    } catch (error) {
        console.error('Watchtower: Failed to fetch patterns', error);
        return null;
    }
}

async function submitRequest(url) {
    const config = await getConfig();
    
    if (!config.token || !config.apiUrl) {
        console.log('Watchtower: Not configured');
        return false;
    }
    
    try {
        // Extract suggested pattern (domain + path prefix)
        const parsed = new URL(url);
        const suggestedPattern = parsed.hostname + '/*';
        
        const response = await fetch(`${config.apiUrl}/api/requests`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${config.token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                url: url,
                suggested_pattern: suggestedPattern
            })
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        console.log('Watchtower: Request submitted for', url);
        return true;
    } catch (error) {
        console.error('Watchtower: Failed to submit request', error);
        return false;
    }
}

// ========================================
// Heartbeat (Canary Check)
// ========================================

async function sendHeartbeat() {
    const config = await getConfig();
    
    if (!config.token || !config.apiUrl) {
        return false;
    }
    
    try {
        const response = await fetch(`${config.apiUrl}/api/heartbeat`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${config.token}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        // Update last heartbeat time
        config.lastHeartbeat = new Date().toISOString();
        await setConfig(config);
        
        console.log('Watchtower: Heartbeat sent');
        return true;
    } catch (error) {
        console.error('Watchtower: Failed to send heartbeat', error);
        return false;
    }
}

// Set up the uninstall URL to notify the backend when extension is removed
async function setupUninstallUrl() {
    const config = await getConfig();
    
    if (config.token && config.apiUrl) {
        const uninstallUrl = `${config.apiUrl}/api/uninstall?token=${encodeURIComponent(config.token)}`;
        chrome.runtime.setUninstallURL(uninstallUrl);
        console.log('Watchtower: Uninstall URL configured');
    }
}

// ========================================
// WebSocket Connection
// ========================================

async function connectWebSocket() {
    const config = await getConfig();
    
    if (!config.token || !config.apiUrl) {
        console.log('Watchtower: WebSocket not connecting - not configured');
        return;
    }
    
    // Close existing connection if any
    if (ws) {
        ws.close();
        ws = null;
        wsConnected = false;
    }
    
    // Convert HTTP URL to WebSocket URL
    const wsUrl = config.apiUrl.replace(/^http/, 'ws') + '/api/ws?token=' + encodeURIComponent(config.token);
    
    try {
        console.log('Watchtower: Connecting WebSocket...');
        ws = new WebSocket(wsUrl);
        
        ws.onopen = () => {
            wsConnected = true;
            console.log('Watchtower: WebSocket connected');
        };
        
        ws.onmessage = async (event) => {
            try {
                const message = JSON.parse(event.data);
                console.log('Watchtower: WebSocket message received', message.type);
                
                if (message.type === 'patterns_updated') {
                    // Update patterns from WebSocket push
                    const data = message.data;
                    const patterns = {
                        allow: [],
                        deny: []
                    };
                    
                    for (const pattern of (data.patterns || [])) {
                        if (pattern.type === 'allow') {
                            patterns.allow.push(pattern.pattern);
                        } else if (pattern.type === 'deny') {
                            patterns.deny.push(pattern.pattern);
                        }
                    }
                    
                    await setPatterns(patterns);
                    
                    // Update last sync time
                    const cfg = await getConfig();
                    cfg.lastSync = new Date().toISOString();
                    await setConfig(cfg);
                    
                    console.log('Watchtower: Patterns updated via WebSocket', patterns);
                }
            } catch (error) {
                console.error('Watchtower: Failed to parse WebSocket message', error);
            }
        };
        
        ws.onclose = (event) => {
            wsConnected = false;
            ws = null;
            console.log('Watchtower: WebSocket closed', event.code, event.reason);
        };
        
        ws.onerror = (error) => {
            console.error('Watchtower: WebSocket error', error);
            wsConnected = false;
        };
    } catch (error) {
        console.error('Watchtower: Failed to connect WebSocket', error);
        wsConnected = false;
        ws = null;
    }
}

function disconnectWebSocket() {
    if (ws) {
        ws.close();
        ws = null;
        wsConnected = false;
        console.log('Watchtower: WebSocket disconnected');
    }
}

async function ensureWebSocketConnected() {
    const config = await getConfig();
    if (config.token && !wsConnected) {
        await connectWebSocket();
    }
}

// ========================================
// Pattern Matching
// ========================================

function patternToRegex(pattern) {
    // Convert glob-style pattern to regex
    // Trailing /* or * at end matches everything (common user expectation)
    // ** matches any characters including /
    // * in the middle matches any characters except /
    
    let regex = pattern
        .replace(/[.+?^${}()|[\]\\]/g, '\\$&') // Escape special regex chars except *
        .replace(/\*\*/g, '{{DOUBLE_STAR}}');  // Temporarily replace **
    
    // Handle trailing /* or trailing * - these should match everything including subpaths
    // e.g., "example.com/*" should match "example.com/page/subpage"
    if (regex.endsWith('/*')) {
        // Remove the trailing /* and add optional path matching
        regex = regex.slice(0, -2) + '(?:/.*)?';
    } else if (regex.endsWith('*') && !regex.endsWith('{{DOUBLE_STAR}}')) {
        // Trailing * without / - match everything from this point
        regex = regex.slice(0, -1) + '.*';
    }
    
    // Replace remaining * with [^/]* (matches anything except /)
    regex = regex.replace(/\*/g, '[^/]*');
    
    // Replace ** placeholder with .* (matches anything including /)
    regex = regex.replace(/{{DOUBLE_STAR}}/g, '.*');
    
    return new RegExp(`^${regex}$`, 'i');
}

function matchesPattern(url, patterns) {
    try {
        const parsed = new URL(url);
        const urlToMatch = parsed.hostname + parsed.pathname + parsed.search;
        
        for (const pattern of patterns) {
            const regex = patternToRegex(pattern);
            if (regex.test(urlToMatch)) {
                return true;
            }
            // Also try matching just the hostname
            if (regex.test(parsed.hostname)) {
                return true;
            }
        }
        return false;
    } catch {
        return false;
    }
}

async function shouldBlockUrl(url) {
    // Skip extension pages, chrome pages, etc.
    if (!url.startsWith('http://') && !url.startsWith('https://')) {
        return false;
    }
    
    const config = await getConfig();
    if (!config.token) {
        return false;
    }
    
    const patterns = await getPatterns();
    
    // Check deny list first
    if (patterns.deny.length > 0 && matchesPattern(url, patterns.deny)) {
        return true;
    }
    
    // If there are allow patterns, URL must match one of them
    if (patterns.allow.length > 0) {
        if (!matchesPattern(url, patterns.allow)) {
            return true; // Not in allow list = blocked
        }
    }
    
    return false;
}

// ========================================
// Navigation Handling
// ========================================

chrome.webNavigation.onBeforeNavigate.addListener(async (details) => {
    // Only handle main frame navigations
    if (details.frameId !== 0) {
        return;
    }
    
    const url = details.url;
    
    // Skip our own blocked page
    if (url.includes(chrome.runtime.id)) {
        return;
    }
    
    const blocked = await shouldBlockUrl(url);
    
    if (blocked) {
        console.log('Watchtower: Blocking', url);
        
        // Redirect to blocked page
        const blockedPageUrl = chrome.runtime.getURL('blocked.html') + 
            '?url=' + encodeURIComponent(url);
        
        chrome.tabs.update(details.tabId, { url: blockedPageUrl });
    }
});

// ========================================
// Message Handling
// ========================================

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === 'getConfig') {
        getConfig().then(sendResponse);
        return true;
    }
    
    if (message.action === 'setConfig') {
        setConfig(message.config).then(() => {
            setupUninstallUrl(); // Update uninstall URL with new config
            connectWebSocket(); // Connect/reconnect WebSocket (handles heartbeat)
            fetchPatterns().then(() => sendResponse({ success: true }));
        });
        return true;
    }
    
    if (message.action === 'getPatterns') {
        getPatterns().then(sendResponse);
        return true;
    }
    
    if (message.action === 'syncPatterns') {
        fetchPatterns().then(patterns => {
            sendResponse({ success: !!patterns, patterns });
        });
        return true;
    }
    
    if (message.action === 'submitRequest') {
        submitRequest(message.url).then(success => {
            sendResponse({ success });
        });
        return true;
    }
    
    if (message.action === 'checkUrl') {
        shouldBlockUrl(message.url).then(blocked => {
            sendResponse({ blocked });
        });
        return true;
    }
    
    if (message.action === 'getStatus') {
        sendResponse({ 
            wsConnected,
            configured: true
        });
        return true;
    }
    
    if (message.action === 'reconnectWebSocket') {
        connectWebSocket().then(() => {
            sendResponse({ success: true, wsConnected });
        });
        return true;
    }
});

// ========================================
// Periodic Sync & Heartbeat
// ========================================

async function periodicSync() {
    const config = await getConfig();
    if (config.token) {
        await fetchPatterns();
    }
}

async function periodicHeartbeat() {
    const config = await getConfig();
    // Only send HTTP heartbeat if WebSocket is not connected
    // WebSocket handles heartbeat via ping/pong when connected
    if (config.token && !wsConnected) {
        console.log('Watchtower: Sending HTTP heartbeat (WebSocket not connected)');
        await sendHeartbeat();
    }
}

// Set up periodic alarms
chrome.alarms.create('syncPatterns', { periodInMinutes: 2 }); // Fallback sync every 2 min
chrome.alarms.create('heartbeat', { periodInMinutes: HEARTBEAT_INTERVAL });
chrome.alarms.create('wsReconnect', { periodInMinutes: WS_RECONNECT_INTERVAL });

chrome.alarms.onAlarm.addListener((alarm) => {
    if (alarm.name === 'syncPatterns') {
        periodicSync();
    }
    if (alarm.name === 'heartbeat') {
        periodicHeartbeat();
    }
    if (alarm.name === 'wsReconnect') {
        ensureWebSocketConnected();
    }
});

// Initial sync and WebSocket on startup
chrome.runtime.onStartup.addListener(() => {
    console.log('Watchtower: Extension startup');
    setupUninstallUrl();
    connectWebSocket(); // WebSocket handles patterns sync and heartbeat
    periodicSync(); // Fallback sync
});

// Sync when extension is installed or updated
chrome.runtime.onInstalled.addListener(() => {
    console.log('Watchtower: Extension installed/updated');
    setupUninstallUrl();
    connectWebSocket(); // WebSocket handles patterns sync and heartbeat
    periodicSync(); // Fallback sync
});

