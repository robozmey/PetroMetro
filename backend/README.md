### Формат JSON

Схема метро содержит два массива: `stations` и `lines`.

- `stations`: `id` (уникальный положительный ID), `name` (название), `x`, `y` (координаты), `year` (необязательный год открытия).
- `lines`: `id` (уникальный положительный ID), `name` (название), `color` (цвет `#RRGGBB`), `stations` (ID станций в порядке следования).

Все ID станций в линиях должны существовать в `stations`. Название и slug города хранятся отдельно в таблице `cities`.

### Запуск

При запущенном Docker:

```powershell
docker compose up -d db
docker compose run --rm migrate
cd backend
$env:DATABASE_URL = 'postgres://podzemka:podzemka@localhost:5433/podzemka?sslmode=disable'
go run ./cmd/seedschemas
go run ./cmd/api
```

### Добавление города

Из папки `backend` при настроенном `DATABASE_URL`:

```powershell
go run ./cmd/importcity -file kazan.json -name "Казань" -slug kazan
```

Команда проверяет схему и сохраняет город в PostgreSQL. При повторном запуске с тем же slug существующая схема обновляется.