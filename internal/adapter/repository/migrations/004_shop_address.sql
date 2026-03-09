-- Адрес салона (текст + опционально фото). Одна запись на бота.

CREATE TABLE IF NOT EXISTS shop_address (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  address_text TEXT NOT NULL DEFAULT '',
  address_photo_file_id TEXT
);

INSERT OR IGNORE INTO shop_address (id, address_text) VALUES (1, '');
