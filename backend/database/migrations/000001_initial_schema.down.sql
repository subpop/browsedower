-- Rollback initial schema
-- Drop tables in reverse dependency order

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS requests;
DROP TABLE IF EXISTS patterns;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS users;

