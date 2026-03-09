-- Особые дни: для конкретной даты своё рабочее время (вместо расписания по умолчанию).

CREATE TABLE IF NOT EXISTS barber_schedule_override (
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  work_date TEXT NOT NULL,
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  slot_step_min INTEGER NOT NULL,
  PRIMARY KEY (barber_id, work_date)
);

CREATE INDEX IF NOT EXISTS idx_barber_schedule_override_barber ON barber_schedule_override(barber_id);
