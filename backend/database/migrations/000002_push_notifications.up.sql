-- Add push notification support

-- Push subscriptions table
CREATE TABLE IF NOT EXISTS push_subscriptions (
    id INTEGER PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT UNIQUE NOT NULL,
    p256dh TEXT NOT NULL,
    auth TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Add notification preferences to users
ALTER TABLE users ADD COLUMN notify_new_requests INTEGER DEFAULT 1;
ALTER TABLE users ADD COLUMN notify_device_status INTEGER DEFAULT 1;

-- VAPID keys storage (single row table for app config)
CREATE TABLE IF NOT EXISTS app_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

