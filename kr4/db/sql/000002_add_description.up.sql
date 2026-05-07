-- Миграция 2 (UP): добавление поля description (NOT NULL DEFAULT '')
ALTER TABLE products ADD COLUMN description TEXT NOT NULL DEFAULT '';
