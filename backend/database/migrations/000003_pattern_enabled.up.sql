-- Add enabled field to patterns
ALTER TABLE patterns ADD COLUMN enabled INTEGER DEFAULT 1;

