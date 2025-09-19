-- players: one row per device/account
CREATE TABLE IF NOT EXISTS players (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id TEXT UNIQUE NOT NULL,
  nickname TEXT,
  elo INTEGER NOT NULL DEFAULT 1000,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- matches: one row per completed or in-progress match
CREATE TABLE IF NOT EXISTS matches (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  mode TEXT NOT NULL CHECK (mode IN ("casual", "ranked")),
  state_snapshot JSONB,
  winner_player_id UUID REFERENCES players(id),
  ended_at TIMESTAMPTZ
);

-- match_players: relation and per-player rating delta
CREATE TABLE IF NOT EXISTS match_players (
  match_id UUID REFERENCES matches(id) ON DELETE CASCADE,
  player_id UUID REFERENCES players(id) ON DELETE CASCADE,
  symbol TEXT NOT NULL CHECK (symbol IN ("X", "O")),
  rating_before INTEGER,
  rating_after INTEGER,
  PRIMARY KEY (match_id, player_id)
);

-- leaderboards (materialized style for fast reads)
CREATE TABLE IF NOT EXISTS leaderboard_alltime (
  player_id UUID PRIMARY KEY REFERENCES players(id) ON DELETE CASCADE,
  wins INTEGER NOT NULL DEFAULT 0,
  losses INTEGER NOT NULL DEFAULT 0,
  elo INTEGER NOT NULL DEFAULT 1000
);

CREATE TABLE IF NOT EXISTS leaderboard_daily (
  player_id UUID REFERENCES players(id) ON DELETE CASCADE,
  period DATE NOT NULL,
  wins INTEGER NOT NULL DEFAULT 0,
  losses INTEGER NOT NULL DEFAULT 0,
  elo INTEGER NOT NULL DEFAULT 1000,
  PRIMARY KEY (player_id, period)
);
