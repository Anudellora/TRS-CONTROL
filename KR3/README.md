# Контрольная работа №3

## Установка и запуск

1. Создайте базу данных PostgreSQL:
```bash
createdb kr3
```

2. Запустите сервер:
```bash
go mod tidy
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=kr3 \
MODE=DEV DOCS_USER=admin DOCS_PASSWORD=secret go run .
```

Сервер запустится на `http://localhost:8080`.

## Эндпоинты

### Задание 6.1 — Базовая аутентификация
```bash
curl -u admin:secret http://localhost:8080/basic-login
```

### Задание 6.2 + 6.5 — Регистрация с хешированием и rate limiter
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"qwerty123"}' \
  http://localhost:8080/register
```

### Задание 6.3 — Документация (DEV/PROD)
```bash
# DEV — требует Basic Auth
curl -u admin:secret http://localhost:8080/docs

# PROD — вернёт 404
MODE=PROD go run .
curl http://localhost:8080/docs
```

### Задание 6.4 + 6.5 — JWT логин
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"qwerty123"}' \
  http://localhost:8080/login
```

### Задание 6.4 — Защищённый ресурс
```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/protected_resource
```

### Задание 7.1 — RBAC
```bash
# Регистрация с ролью admin
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"bob","password":"pass123","role":"admin"}' \
  http://localhost:8080/register

# Логин и получение токена
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"bob","password":"pass123"}' \
  http://localhost:8080/login

# Доступ к ресурсам по ролям
curl -H "Authorization: Bearer <token>" http://localhost:8080/admin/resource
curl -H "Authorization: Bearer <token>" http://localhost:8080/user/resource
curl -H "Authorization: Bearer <token>" http://localhost:8080/guest/resource
```

### Задание 8.1 — SQLite регистрация
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"username":"test_user","password":"12345"}' \
  http://localhost:8080/db/register
```

### Задание 8.2 — CRUD Todo
```bash
# Создать
curl -X POST -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","description":"Milk, eggs, bread"}' \
  http://localhost:8080/todos

# Получить
curl http://localhost:8080/todos/1

# Обновить
curl -X PUT -H "Content-Type: application/json" \
  -d '{"title":"Buy groceries","description":"Milk, eggs","completed":true}' \
  http://localhost:8080/todos/1

# Удалить
curl -X DELETE http://localhost:8080/todos/1
```
