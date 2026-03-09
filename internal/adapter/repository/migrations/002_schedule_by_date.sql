-- Расписание по умолчанию (одно на барбера) и выходные по датам.

CREATE TABLE IF NOT EXISTS barber_default_schedule (
  barber_id INTEGER PRIMARY KEY REFERENCES barbers(id),
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  slot_step_min INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS barber_days_off (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  off_date TEXT NOT NULL,
  UNIQUE(barber_id, off_date)
);

CREATE INDEX IF NOT EXISTS idx_barber_days_off_barber_date ON barber_days_off(barber_id, off_date);
