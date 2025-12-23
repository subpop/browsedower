// ========================================
// Watchtower Extension - Blocked Page
// ========================================

const $ = (selector) => document.querySelector(selector);

// ========================================
// URL Parsing
// ========================================

function getBlockedUrl() {
    const params = new URLSearchParams(window.location.search);
    return params.get('url') || '';
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
// UI Updates
// ========================================

function showSuccess() {
    $('#request-form').classList.add('hidden');
    $('#success-message').classList.remove('hidden');
    $('#error-message').classList.add('hidden');
}

function showError() {
    $('#error-message').classList.remove('hidden');
    $('#request-btn').disabled = false;
    $('#request-btn').innerHTML = `
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 2L11 13"/>
            <path d="M22 2L15 22L11 13L2 9L22 2Z"/>
        </svg>
        Request Access
    `;
}

function showLoading() {
    $('#request-btn').disabled = true;
    $('#request-btn').innerHTML = `
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation: spin 1s linear infinite;">
            <path d="M21 12a9 9 0 11-6.219-8.56"/>
        </svg>
        Submitting...
    `;
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

async function submitRequest(e) {
    e.preventDefault();

    const url = getBlockedUrl();
    if (!url) {
        showError();
        return;
    }

    showLoading();

    try {
        const result = await sendMessage({
            action: 'submitRequest',
            url: url
        });

        if (result.success) {
            showSuccess();
        } else {
            showError();
        }
    } catch (error) {
        console.error('Failed to submit request:', error);
        showError();
    }
}

function goBack() {
    if (window.history.length > 1) {
        window.history.back();
    } else {
        window.close();
    }
}

// ========================================
// Initialization
// ========================================

document.addEventListener('DOMContentLoaded', () => {
    const blockedUrl = getBlockedUrl();

    // Display blocked URL as clickable link
    const urlElement = $('#blocked-url');
    urlElement.textContent = blockedUrl || 'Unknown URL';
    if (blockedUrl) {
        urlElement.href = blockedUrl;
    }

    // Suggest pattern
    $('#pattern').value = extractDomain(blockedUrl);

    // Event listeners
    $('#request-form').addEventListener('submit', submitRequest);
    $('#go-back-btn').addEventListener('click', goBack);
});

// Add spin animation for loading
const style = document.createElement('style');
style.textContent = `
    @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
    }
`;
document.head.appendChild(style);

