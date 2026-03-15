-- Один барбер может иметь несколько Telegram-аккаунтов: все привязываются к одному barber_id.
-- Расписание и записи общие для всех аккаунтов.

CREATE TABLE IF NOT EXISTS barber_telegram_ids (
  barber_id INTEGER NOT NULL REFERENCES barbers(id),
  telegram_id INTEGER NOT NULL UNIQUE,
  PRIMARY KEY (barber_id, telegram_id)
);

CREATE INDEX IF NOT EXISTS idx_barber_telegram_ids_telegram ON barber_telegram_ids(telegram_id);

-- Привязываем все текущие barbers к одному каноническому (минимальный id)
INSERT OR IGNORE INTO barber_telegram_ids (barber_id, telegram_id)
SELECT (SELECT id FROM barbers ORDER BY id LIMIT 1), telegram_id FROM barbers;

-- Переносим расписание и визиты к каноническому барберу
UPDATE barber_working_days
SET barber_id = (SELECT id FROM barbers ORDER BY id LIMIT 1)
WHERE barber_id != (SELECT id FROM barbers ORDER BY id LIMIT 1);

UPDATE visits
SET barber_id = (SELECT id FROM barbers ORDER BY id LIMIT 1)
WHERE barber_id != (SELECT id FROM barbers ORDER BY id LIMIT 1);

-- Удаляем лишние строки барберов (оставляем канонического — с минимальным id)
DELETE FROM barbers
WHERE id > (SELECT MIN(id) FROM barbers);
