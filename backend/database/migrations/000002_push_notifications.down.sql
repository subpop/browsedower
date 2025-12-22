-- Rollback push notification support

DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS app_config;

-- Note: SQLite doesn't support DROP COLUMN easily
-- These columns will remain but be unused if rolled back

