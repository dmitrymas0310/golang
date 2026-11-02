# Gateway

Gateway обслуживает прежние HTTP-эндпойнты транзакций и бюджетов. Он вызывает доменный пакет Ledger: данные хранятся в общей PostgreSQL, а результаты отчётов — в Redis с TTL **30 секунд**. CLI Ledger проверяет подключение и показывает данные; отдельный HTTP-процесс Ledger не требуется.

Нужны Go 1.27.0 и Docker. Команды подготовки выполняются из каталога `financial-planner`.

## Создание БД и запуск Redis

```bash
docker run --name cashcraft-postgres \
  -p 127.0.0.1:5432:5432 \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=cashapp \
  -d postgres:18

docker run --name cashcraft-redis \
  -p 127.0.0.1:6379:6379 \
  -d redis:7-alpine

docker exec cashcraft-postgres pg_isready -U postgres -d cashapp
docker exec cashcraft-redis redis-cli PING
```

Продолжайте после ответа PostgreSQL `accepting connections` и Redis `PONG`; при начальной инициализации повторите проверки готовности.

БД `cashapp` создаётся из `POSTGRES_DB` при первом запуске нового контейнера. Для уже существующих контейнеров используйте `docker start cashcraft-postgres cashcraft-redis` вместо повторного создания.

## Окружение, миграции и старт

В каждом терминале сервисов задайте одинаковые параметры:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
export REDIS_ADDR='localhost:6379'
export REDIS_DB='0'
export REDIS_PASSWORD=''
```

Пустой `REDIS_PASSWORD` означает подключение без пароля; для Redis с авторизацией укажите его значение. `DATABASE_URL` имеет приоритет над отдельными `DB_*`. `.env` автоматически не загружается.

Из `financial-planner` установите Goose и примените схему:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
export PATH="$(go env GOPATH)/bin:$PATH"
goose -dir ledger/migrations postgres "$DATABASE_URL" up
goose -dir ledger/migrations postgres "$DATABASE_URL" status
```

Откат последней миграции (удаляет соответствующую таблицу вместе с данными):

```bash
goose -dir ledger/migrations postgres "$DATABASE_URL" down
```

Сначала проверьте Ledger из `financial-planner`:

```bash
cd ledger
go run ./cmd/ledger
```

Успешный старт логирует подключения к PostgreSQL и Redis, выводит данные и завершает CLI. Затем в новом терминале с теми же переменными из `financial-planner` запустите Gateway:

```bash
cd gateway
go run .
```

Сервер слушает `:8080`. Ответы имеют `Content-Type: application/json; charset=utf-8`, ошибки — формат `{"error":"…"}`. При недоступном Redis отчёты читаются из PostgreSQL.

## Эндпойнты

| Метод | Путь | Результат |
| --- | --- | --- |
| GET | `/ping` | `200`, проверка сервера |
| POST | `/api/budgets` | `201`, создание или обновление бюджета категории |
| GET | `/api/budgets` | `200`, бюджеты по категории |
| POST | `/api/transactions` | `201`, расход со сгенерированным ID |
| GET | `/api/transactions` | `200`, расходы по дате и ID по убыванию |
| GET | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | `200`, суммы по категориям, JSON-кеш 30 секунд |
| PUT | `/api/budgets` | `501`, заглушка |

Ошибки валидации и диапазона дат возвращают `400`, превышение бюджета — `409` с `{"error":"budget exceeded"}`, внутренние ошибки — `500`.

## cURL и эффект кеша

Выполняйте по порядку на новой мигрированной БД:

```bash
curl -i http://localhost:8080/ping

curl -i -X POST http://localhost:8080/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"еда","limit":1000}'

curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'

curl -i http://localhost:8080/api/budgets
curl -i http://localhost:8080/api/transactions
```

Отчёт запросите дважды подряд:

```bash
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
```

Ожидаемый ответ обоих запросов — `200` и `[{"category":"еда","total":450}]`. В консоли Gateway видны `report summary cache miss` при первом запросе без готового ключа и `report summary cache hit` при повторном. HTTP-тела совпадают, вычисление второго отчёта выполняется через Redis. TTL ключа можно проверить командой:

```bash
docker exec cashcraft-redis redis-cli TTL report:summary:2025-09-01:2025-09-30
```

Суммы включают обе даты. Ключ живёт 30 секунд; после его истечения выполняется новый SQL-запрос. После записи новых расходов отчёт может оставаться прежним до истечения TTL.

Негативные проверки:

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":600,"category":"еда","date":"2025-09-10"}'

curl -i 'http://localhost:8080/api/reports/summary?from=bad&to=2025-09-30'
```

После расхода `450` при бюджете `1000` первый запрос возвращает `409`; второй — `400`.

## Тесты

Из `financial-planner/gateway`:

```bash
export TEST_DATABASE_URL="$DATABASE_URL"
go test -race -shuffle=on ./...
```

SQL/HTTP-сценарии используют отдельные временные схемы PostgreSQL и miniredis. Без `TEST_DATABASE_URL` интеграционные тесты пропускаются, unit-тесты выполняются.
