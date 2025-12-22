// ========================================
// Browsedower Extension - Popup
// ========================================

const $ = (selector) => document.querySelector(selector);

// ========================================
// Message Handling
// ========================================

function showMessage(text, type = 'success') {
    const msg = $('#message');
    msg.textContent = text;
    msg.className = `message ${type}`;
    msg.classList.remove('hidden');
    
    setTimeout(() => {
        msg.classList.add('hidden');
    }, 3000);
}

// ========================================
// UI Updates
// ========================================

async function updateUI() {
    // Get config
    const config = await sendMessage({ action: 'getConfig' });
    
    // Update form fields
    $('#api-url').value = config.apiUrl || '';
    $('#token').value = config.token || '';
    
    // Update connection status
    const connStatus = $('#connection-status');
    if (config.token) {
        connStatus.textContent = 'Connected';
        connStatus.className = 'status-value connected';
    } else {
        connStatus.textContent = 'Not configured';
        connStatus.className = 'status-value disconnected';
    }
    
    // Update last sync
    const lastSync = $('#last-sync');
    if (config.lastSync) {
        lastSync.textContent = formatDate(config.lastSync);
    } else {
        lastSync.textContent = 'Never';
    }
    
    // Update admin link
    $('#admin-link').href = (config.apiUrl || 'http://localhost:8080') + '/admin/';
    
    // Get patterns
    const patterns = await sendMessage({ action: 'getPatterns' });
    
    // Update stats
    $('#allow-count').textContent = patterns.allow?.length || 0;
    $('#deny-count').textContent = patterns.deny?.length || 0;
}

function formatDate(dateStr) {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now - date;
    
    if (diff < 60000) return 'Just now';
    if (diff < 3600000) {
        const mins = Math.floor(diff / 60000);
        return `${mins}m ago`;
    }
    if (diff < 86400000) {
        const hours = Math.floor(diff / 3600000);
        return `${hours}h ago`;
    }
    
    return date.toLocaleDateString();
}

// ========================================
// Message Communication
// ========================================

function sendMessage(message) {
    return new Promise((resolve) => {
        chrome.runtime.sendMessage(message, resolve);
    });
}

// ========================================
// Event Handlers
// ========================================

async function saveConfig() {
    const apiUrl = $('#api-url').value.trim();
    const token = $('#token').value.trim();
    
    if (!apiUrl) {
        showMessage('Please enter an API URL', 'error');
        return;
    }
    
    if (!token) {
        showMessage('Please enter a device token', 'error');
        return;
    }
    
    try {
        const result = await sendMessage({
            action: 'setConfig',
            config: { apiUrl, token }
        });
        
        if (result.success) {
            showMessage('Settings saved and synced!');
            updateUI();
        } else {
            showMessage('Failed to sync patterns', 'error');
        }
    } catch (error) {
        showMessage('Error saving settings', 'error');
    }
}

async function syncPatterns() {
    try {
        const result = await sendMessage({ action: 'syncPatterns' });
        
        if (result.success) {
            showMessage('Patterns synced!');
            updateUI();
        } else {
            showMessage('Failed to sync patterns', 'error');
        }
    } catch (error) {
        showMessage('Error syncing patterns', 'error');
    }
}

// ========================================
// Initialization
// ========================================

document.addEventListener('DOMContentLoaded', () => {
    // Initial UI update
    updateUI();
    
    // Event listeners
    $('#save-btn').addEventListener('click', saveConfig);
    $('#sync-btn').addEventListener('click', syncPatterns);
    
    // Admin link click
    $('#admin-link').addEventListener('click', async (e) => {
        e.preventDefault();
        const config = await sendMessage({ action: 'getConfig' });
        const url = (config.apiUrl || 'http://localhost:8080') + '/admin/';
        chrome.tabs.create({ url });
    });
});

