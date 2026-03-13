-- Рабочие дни барбера: только перечисленные даты являются рабочими, у каждой своё окно времени. Шаг слотов 1 час (в коде).

CREATE TABLE IF NOT EXISTS barber_working_days (
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  work_date TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  PRIMARY KEY (barber_id, work_date)
);

CREATE INDEX IF NOT EXISTS idx_barber_working_days_barber ON barber_working_days(barber_id);
