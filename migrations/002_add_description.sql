-- migrations/002_add_description.sql
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS description TEXT DEFAULT '';

