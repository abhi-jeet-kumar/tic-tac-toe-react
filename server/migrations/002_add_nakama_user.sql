ALTER TABLE players ADD COLUMN IF NOT EXISTS nakama_user_id TEXT UNIQUE;

-- backfill could be done via RPC on login; keep nullable for now

