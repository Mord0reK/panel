-- Add display_name, icon, and status columns to servers table
ALTER TABLE servers ADD COLUMN display_name TEXT;
ALTER TABLE servers ADD COLUMN icon TEXT;
ALTER TABLE servers ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

-- Ensure any existing rows have status set
UPDATE servers SET status = 'active' WHERE status IS NULL OR status = '';
