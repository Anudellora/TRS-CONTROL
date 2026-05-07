-- Миграция 2 (DOWN): откат — пересоздаём таблицу без столбца description
-- (SQLite не поддерживает DROP COLUMN в старых версиях, используем временную таблицу)
CREATE TABLE products_backup (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT    NOT NULL,
    price REAL    NOT NULL,
    count INTEGER NOT NULL
);
INSERT INTO products_backup (id, title, price, count)
    SELECT id, title, price, count FROM products;
DROP TABLE products;
ALTER TABLE products_backup RENAME TO products;
