# Ledger

Ledger хранит бюджеты и расходы в PostgreSQL. `SetBudget` выполняет UPSERT по категории; добавление расхода проверяет бюджет в SQL-транзакции. Отчёты по категориям кешируются в Redis на **30 секунд**. Gateway использует этот пакет, сохраняя прежние HTTP-эндпойнты бюджетов и транзакций.

Нужны Go 1.27.0 и Docker. Команды подготовки ниже выполняются из каталога `financial-planner`.

## PostgreSQL, Redis и создание БД

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

Продолжайте после ответа PostgreSQL `accepting connections` и Redis `PONG`; во время начальной инициализации повторите проверки готовности.

`POSTGRES_DB=cashapp` создаёт БД при первом запуске нового контейнера. Если контейнеры уже созданы, запустите их командой `docker start cashcraft-postgres cashcraft-redis` вместо повторного `docker run`.

## Окружение и миграции Goose

Задайте эти переменные в каждом терминале, где запускаете Ledger или Gateway:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
export REDIS_ADDR='localhost:6379'
export REDIS_DB='0'
export REDIS_PASSWORD=''
```

`DATABASE_URL` имеет приоритет над `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_NAME`. Значения по умолчанию: `localhost:5432`, `postgres/postgres`, БД `cashapp`, `sslmode=disable`. Redis по умолчанию использует `localhost:6379`, БД `0`, без пароля. `.env` автоматически не читается.

Установите Goose и примените миграции **до** первого запуска сервиса и после изменения схемы:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
export PATH="$(go env GOPATH)/bin:$PATH"
goose -dir ledger/migrations validate
goose -dir ledger/migrations postgres "$DATABASE_URL" up
goose -dir ledger/migrations postgres "$DATABASE_URL" status
```

Откат последней миграции:

```bash
goose -dir ledger/migrations postgres "$DATABASE_URL" down
```

Первый `down` удаляет таблицу `expenses`, второй — `budgets`. Откат удаляет данные соответствующей таблицы.

## Запуск Ledger

После подготовки из `financial-planner`:

```bash
cd ledger
go run ./cmd/ledger
```

При старте выполняются `sql.Open("pgx", dsn)`, настройка пула (10 открытых, 5 простаивающих соединений, срок жизни 30 минут), `PingContext` и Redis `PING`. Успех виден по сообщениям `Redis connected: ...` и `Ledger connected to PostgreSQL`. CLI выводит сохранённые бюджеты и транзакции и завершает работу. HTTP-сервер запускается отдельно в Gateway.

Данные сохраняются между запусками. Период бюджета не хранится в текущей SQL-схеме, поэтому поле `period` в HTTP-ответе остаётся пустой строкой.

## Проверка через Gateway и cURL

В другом терминале повторите экспорт переменных выше и запустите из `financial-planner`:

```bash
cd gateway
go run .
```

Выполняйте примеры на новой мигрированной БД:

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

Дважды запросите отчёт подряд:

```bash
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
```

Оба запроса возвращают `200` и `[{"category":"еда","total":450}]`. Если готового ключа ещё нет, в терминале Gateway первый запрос логируется как `report summary cache miss`, второй — как `report summary cache hit`. Ключ `report:summary:2025-09-01:2025-09-30` содержит JSON с TTL 30 секунд. Срок жизни можно посмотреть отдельно:

```bash
docker exec cashcraft-redis redis-cli TTL report:summary:2025-09-01:2025-09-30
```

После истечения TTL следующий запрос пересчитывает отчёт через SQL. Обе границы диапазона включены. После добавления расходов старый отчёт может сохраняться до истечения TTL. При недоступном Redis используется PostgreSQL.

## Тесты

Из каталога `financial-planner/ledger`:

```bash
export TEST_DATABASE_URL="$DATABASE_URL"
go test -race -shuffle=on ./...
```

Интеграционные тесты создают и удаляют отдельные схемы; роли нужно право `CREATE SCHEMA`. Без `TEST_DATABASE_URL` SQL-сценарии пропускаются. Кеш проверяется через miniredis, включая TTL без ожидания таймера.
