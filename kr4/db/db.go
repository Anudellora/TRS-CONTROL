// Package db: подключение к SQLite и запуск версионных SQL-миграций (Task 9.1).
// Аналог связки SQLAlchemy + Alembic, но на Go:
//   - gorm.io/gorm + github.com/glebarez/sqlite — ORM-драйвер (pure-Go, без CGO)
//   - собственный runner миграций с таблицей schema_migrations
package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed sql/*.sql
var sqlFS embed.FS

// InitDB открывает базу, прогоняет все pending-миграции и возвращает *gorm.DB.
func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	if err := runMigrations(sqlDB); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return db, nil
}

// Seed добавляет две начальных записи если таблица пуста (Task 9.1, шаг 5).
func Seed(db *gorm.DB) {
	var count int64
	db.Table("products").Count(&count)
	if count > 0 {
		return
	}
	db.Exec(`INSERT INTO products (title, price, count, description) VALUES
		('Ноутбук',  89999.00, 10, 'Мощный игровой ноутбук'),
		('Наушники', 4999.99,  50, 'Беспроводные наушники с шумоподавлением')`)
	log.Println("[seed] добавлены 2 начальные записи в products")
}

// ── Runner миграций ───────────────────────────────────────────────────────────

type migration struct {
	version int
	name    string
	upSQL   string
}

func runMigrations(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER  PRIMARY KEY,
		name       TEXT     NOT NULL,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	migs, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migs {
		var cnt int
		if err := db.QueryRow(
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?", m.version,
		).Scan(&cnt); err != nil {
			return err
		}
		if cnt > 0 {
			log.Printf("[migrate] v%d (%s): уже применена, пропускаем", m.version, m.name)
			continue
		}

		log.Printf("[migrate] применяем v%d (%s)…", m.version, m.name)
		if _, err := db.Exec(m.upSQL); err != nil {
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		if _, err := db.Exec(
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)", m.version, m.name,
		); err != nil {
			return err
		}
		log.Printf("[migrate] v%d (%s): готово", m.version, m.name)
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	upFiles := make(map[string]string)

	entries, err := fs.ReadDir(sqlFS, "sql")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		data, err := sqlFS.ReadFile("sql/" + e.Name())
		if err != nil {
			return nil, err
		}
		key := strings.TrimSuffix(e.Name(), ".up.sql")
		upFiles[key] = string(data)
	}

	keys := make([]string, 0, len(upFiles))
	for k := range upFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]migration, 0, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, "_", 2)
		if len(parts) < 2 {
			continue
		}
		var ver int
		fmt.Sscanf(parts[0], "%d", &ver)
		result = append(result, migration{version: ver, name: parts[1], upSQL: upFiles[key]})
	}
	return result, nil
}
