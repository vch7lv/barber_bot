-- Схема БД для barber_bot. SQLite, время в UTC (unix timestamp где INTEGER).

CREATE TABLE IF NOT EXISTS clients (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  telegram_id INTEGER NOT NULL UNIQUE,
  name TEXT NOT NULL DEFAULT '',
  username TEXT NOT NULL DEFAULT '',
  contact TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS barbers (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  telegram_id INTEGER NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS services (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  price_cents INTEGER NOT NULL,
  duration_min INTEGER NOT NULL,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS visits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  client_id INTEGER NOT NULL REFERENCES clients(id),
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  starts_at INTEGER NOT NULL,
  duration_min INTEGER NOT NULL,
  created_at INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT 'scheduled'
);

CREATE TABLE IF NOT EXISTS visit_services (
  visit_id INTEGER NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
  service_id INTEGER NOT NULL REFERENCES services(id),
  PRIMARY KEY (visit_id, service_id)
);

CREATE TABLE IF NOT EXISTS schedule_settings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  day_of_week INTEGER NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  slot_step_min INTEGER NOT NULL,
  UNIQUE(barber_id, day_of_week)
);

CREATE TABLE IF NOT EXISTS bans (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  client_telegram_id INTEGER NOT NULL UNIQUE,
  banned_at INTEGER NOT NULL,
  reason TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  at INTEGER NOT NULL,
  event_type TEXT NOT NULL,
  payload TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_visits_client_id ON visits(client_id);
CREATE INDEX IF NOT EXISTS idx_visits_barber_starts ON visits(barber_id, starts_at);
CREATE INDEX IF NOT EXISTS idx_visits_status ON visits(status);
CREATE INDEX IF NOT EXISTS idx_audit_log_at ON audit_log(at);
