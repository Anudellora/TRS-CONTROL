# Контрольная работа №4 — Go-реализация

Стек: **Go 1.22**, **Gin** (HTTP-фреймворк), **GORM + SQLite** (ORM), собственный runner миграций, **gofakeit** (генерация тестовых данных).

---

## Соответствие заданиям

| Задание | Что реализовано | Файлы |
|---------|-----------------|-------|
| **9.1** | Модель `Product` (id, title, price, count → +description), 2 SQL-миграции, seeder с 2 записями | `models/product.go`, `db/db.go`, `db/sql/` |
| **10.1** | Кастомные исключения `CustomErrorA`/`CustomErrorB`, middleware-обработчик, модели ответов, demo-эндпоинты | `apperrors/errors.go`, `handlers/router.go` |
| **10.2** | Модель `UserIn` с валидацией (username, age>18, email, password 8–16 симв., phone?), кастомный обработчик ошибок валидации | `models/user.go`, `apperrors/errors.go` |
| **11.1** | Unit-тесты трёх эндпоинтов (POST/GET/DELETE) через `httptest`, различные сценарии + граничные случаи | `handlers/user_test.go` |
| **11.2** | Параллельные тесты с `t.Parallel()`, `httptest.Server` + `http.Client` (ааналог httpx.AsyncClient+ASGITransport), данные от **gofakeit** (аналог Python Faker), изоляция состояния через `store.Reset()` | `handlers/user_faker_test.go` |

---

## Установка и запуск

```bash
# 1. Загрузить зависимости
go mod tidy

# 2. Запустить сервер (автоматически применяет обе миграции и добавляет 2 продукта)
go run .
```

Сервер поднимается на `http://localhost:8080`.

---

## Проверка функциональности

### Task 9.1 — Миграции и продукты

```bash
# Получить список продуктов (2 записи добавлены при старте)
curl http://localhost:8080/products

# Создать новый продукт (поле description появилось в миграции 2)
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"title":"Телефон","price":29999,"count":5,"description":"Флагманский смартфон"}'

# Получить продукт по ID
curl http://localhost:8080/products/1
```

Миграции выполняются автоматически при старте — в консоли видно:
```
[migrate] применяем v1 (create_products)…
[migrate] v1 (create_products): готово
[migrate] применяем v2 (add_description)…
[migrate] v2 (add_description): готово
```

### Task 10.1 — Кастомные исключения

```bash
# CustomErrorA (400) — условие не выполнено
curl "http://localhost:8080/demo/error-a"
# → {"code":400,"type":"CustomErrorA","message":"Обязательное условие не выполнено"}

# CustomErrorA — условие выполнено
curl "http://localhost:8080/demo/error-a?condition=ok"
# → {"message":"Условие выполнено"}

# CustomErrorB (404) — ресурс не найден
curl "http://localhost:8080/demo/error-b"
# → {"code":404,"type":"CustomErrorB","message":"Ресурс с указанным ID не найден"}
```

### Task 10.2 — Валидация пользователя

```bash
# Корректные данные → 201
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"ivan","age":25,"email":"ivan@mail.ru","password":"Secret123"}'

# Невалидный возраст → 422
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"ivan","age":16,"email":"ivan@mail.ru","password":"Secret123"}'

# Неверный e-mail → 422
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"ivan","age":25,"email":"not-email","password":"Secret123"}'

# Короткий пароль → 422
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"username":"ivan","age":25,"email":"ivan@mail.ru","password":"abc"}'
```

### Tasks 11.1 / 11.2 — Запуск тестов

```bash
# Все тесты
go test ./handlers/... -v

# Только быстрые unit-тесты (Task 11.1)
go test ./handlers/... -run TestCreate -v
go test ./handlers/... -run TestGet -v
go test ./handlers/... -run TestDelete -v

# Тесты с Faker (Task 11.2)
go test ./handlers/... -run TestFaker -v

# Параллельные тесты с флагом race-detector
go test ./handlers/... -race -v
```

---

## Структура проекта

```
kr4/
├── main.go                         — точка входа
├── models/
│   ├── product.go                  — GORM-модель Product
│   └── user.go                     — UserIn (валидация) / UserOut
├── apperrors/
│   └── errors.go                   — CustomErrorA/B, middleware, валидация
├── db/
│   ├── db.go                       — подключение к БД + runner миграций
│   └── sql/
│       ├── 000001_create_products.up.sql
│       ├── 000001_create_products.down.sql
│       ├── 000002_add_description.up.sql
│       └── 000002_add_description.down.sql
├── handlers/
│   ├── router.go                   — настройка маршрутов
│   ├── product.go                  — CRUD продуктов
│   ├── user.go                     — CRUD пользователей + UserStore
│   ├── user_test.go                — Task 11.1: базовые unit-тесты
│   └── user_faker_test.go          — Task 11.2: параллельные тесты + Faker
└── README.md
```

---

## Переменные окружения

Приложение не требует переменных окружения. БД создаётся автоматически в файле `products.db`.
