-- Миграция 1 (UP): создание таблицы products
CREATE TABLE IF NOT EXISTS products (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT    NOT NULL,
    price REAL    NOT NULL,
    count INTEGER NOT NULL
);
