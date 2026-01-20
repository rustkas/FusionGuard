CREATE TABLE IF NOT EXISTS shots (
  shot_id TEXT PRIMARY KEY,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS telemetry_points (
  id SERIAL PRIMARY KEY,
  shot_id TEXT REFERENCES shots(shot_id),
  ts_unix_ns BIGINT NOT NULL,
  channel_name TEXT NOT NULL,
  value DOUBLE PRECISION NOT NULL,
  quality TEXT NOT NULL,
  UNIQUE(shot_id, ts_unix_ns, channel_name)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_shot_time ON telemetry_points(shot_id, ts_unix_ns);

CREATE TABLE IF NOT EXISTS risks (
  id SERIAL PRIMARY KEY,
  shot_id TEXT REFERENCES shots(shot_id),
  ts_unix_ns BIGINT NOT NULL,
  risk_h50 DOUBLE PRECISION NOT NULL,
  risk_h200 DOUBLE PRECISION NOT NULL,
  model_version TEXT,
  calibration_version TEXT,
  UNIQUE(shot_id, ts_unix_ns)
);

CREATE INDEX IF NOT EXISTS idx_risks_shot_time ON risks(shot_id, ts_unix_ns);

CREATE TABLE IF NOT EXISTS events (
  id SERIAL PRIMARY KEY,
  shot_id TEXT REFERENCES shots(shot_id),
  ts_unix_ns BIGINT NOT NULL,
  kind TEXT NOT NULL,
  message TEXT NOT NULL,
  severity TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_events_shot_time ON events(shot_id, ts_unix_ns);
