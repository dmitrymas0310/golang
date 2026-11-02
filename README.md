# Go: Financial Planner

Проект [financial-planner](financial-planner/README.md) содержит HTTP Gateway и доменный пакет Ledger. Модуль Gateway импортирует Ledger через локальный `replace` в `go.mod`.

Отдельные инструкции: [Ledger](financial-planner/ledger/README.md), [Gateway](financial-planner/gateway/README.md). Они включают создание БД, миграции Goose и запуск с PostgreSQL/Redis.

## Запуск

Нужен Go 1.27.0. Выполните из корня репозитория:

Сначала подготовьте PostgreSQL и примените миграции из раздела ниже. В терминале Gateway задайте `DATABASE_URL` той же базы, что и у Ledger.

```bash
cd financial-planner/gateway
go run .
```

Сервер слушает `:8080`. Все ответы имеют `Content-Type: application/json; charset=utf-8`; ошибки возвращаются в формате `{"error":"описание ошибки"}`. Каждый запрос логируется с методом, путём и длительностью обработки.

## Эндпоинты

| Метод | Путь | Результат |
| --- | --- | --- |
| GET | `/ping` | `200`, `{"message":"pong"}` |
| GET | `/api/transactions` | `200`, массив `TransactionResponse`, без записей — `[]` |
| POST | `/api/transactions` | `201`, созданный `TransactionResponse` |
| GET | `/api/budgets` | `200`, массив `BudgetResponse`, без записей — `[]` |
| POST | `/api/budgets` | `201`, сохранённый `BudgetResponse`; бюджет той же категории заменяется |
| GET | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | `200`, суммы по категориям, Redis TTL 30 секунд |
| PUT | `/api/budgets` | Заглушка, `501 Not Implemented` |

Транзакция принимается с полями `amount`, `category`, `description`, `date`. Сумма должна быть положительной, категория — непустой, дата — строкой `YYYY-MM-DD`. Ответ содержит сгенерированный PostgreSQL `id` и остальные поля. Список транзакций сортируется по дате и ID по убыванию.

Бюджет принимается с полями `category`, `limit`. Категория должна быть непустой, лимит — положительным. Ответ содержит `category`, `limit`, `period`; период не задаётся этим запросом и остаётся пустой строкой.

Бюджет сохраняется через UPSERT по категории, список сортируется по категории. Для категории без бюджета расходы разрешены без лимита. Проверка лимита и вставка выполняются в одной SQL-транзакции с блокировкой строки бюджета, денежное сравнение — через `NUMERIC`.

Статусы ошибок: `400` — некорректный JSON или валидация, `409` — превышение бюджета, `500` — внутренняя ошибка. Неизвестные пути возвращают `404`, неподдерживаемые методы — `405`.

## Примеры cURL

Выполняйте команды по порядку после запуска Gateway с новой мигрированной БД. Флаг `-i` показывает статус и заголовки.

Проверка сервера (`200`):

```bash
curl -i http://localhost:8080/ping
```

Создание бюджета (`201`):

```bash
curl -i -X POST http://localhost:8080/api/budgets \
  -H 'Content-Type: application/json' \
  -d '{"category":"еда","limit":1000}'
```

Создание транзакции (`201`):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":450,"category":"еда","description":"ланч","date":"2025-09-10"}'
```

Получение списков (`200`):

```bash
curl -i http://localhost:8080/api/transactions
curl -i http://localhost:8080/api/budgets
```

Отчёт за сентябрь дважды подряд (`200`):

```bash
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
```

После приведённой транзакции оба ответа содержат `[{"category":"еда","total":450}]`. В консоли Gateway видны `report summary cache miss` для первого запроса без готового ключа и `report summary cache hit` для второго. Неверные или отсутствующие `from`/`to` возвращают `400`.

Ошибка валидации (`400`, положительная сумма обязательна):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":0,"category":"еда","description":"ланч","date":"2025-09-10"}'
```

Превышение бюджета после транзакции `450` при лимите `1000` (`409`, `{"error":"budget exceeded"}`):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":600,"category":"еда","description":"ужин","date":"2025-09-10"}'
```

Данные сохраняются в PostgreSQL между перезапусками сервисов. Ledger и Gateway используют общую БД при одинаковом DSN. CLI Ledger читает существующие бюджеты и расходы и выводит их в консоль; демонстрационные записи и `budgets.json` автоматически не загружаются.

## Миграции PostgreSQL

Каталог миграций: `financial-planner/ledger/migrations`. Каждая SQL-миграция [Goose](https://github.com/pressly/goose) содержит создание (`-- +goose Up`) и откат (`-- +goose Down`):

- `00001_create_budgets.sql` создаёт `budgets`: `id SERIAL PRIMARY KEY`, уникальная обязательная `category`, положительный `limit_amount NUMERIC(14,2)`.
- `00002_create_expenses.sql` создаёт `expenses`: `id SERIAL PRIMARY KEY`, ненулевой `amount NUMERIC(14,2)`, обязательные `category` и `date DATE`, необязательная `description`.

Установка утилиты:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
export PATH="$(go env GOPATH)/bin:$PATH"
goose -version
```

Запустите PostgreSQL и заранее создайте базу `cashapp`. Из корня репозитория укажите DSN своей локальной БД (ниже значения по умолчанию Ledger):

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
goose -dir financial-planner/ledger/migrations validate
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" up
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" status
```

Откат последней миграции:

```bash
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" down
```

Первый `down` удаляет `expenses`, повторный — `budgets`. Для отката обеих миграций сразу:

```bash
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" down-to 0
```

Откат удаляет соответствующие таблицы вместе с данными. Миграции применяются перед первым запуском Ledger и после изменения схемы; приложение не запускает их автоматически.

## Подключение Ledger к PostgreSQL

`ledger.InitDB` использует `ledger/internal/db`: вызывает `sql.Open("pgx", dsn)` и `PingContext` с таймаутом 5 секунд. Пул доступен как `ledger.DB` и закрывается при завершении сервиса. Успешное подключение логируется, при ошибке сервис завершает запуск с ненулевым кодом.

Приоритет имеет непустой `DATABASE_URL`. Если его нет, DSN собирается из переменных `DB_HOST` (`localhost`), `DB_PORT` (`5432`), `DB_USER` (`postgres`), `DB_PASS` (`postgres`, если переменная не задана), `DB_NAME` (`cashapp`) с `sslmode=disable`. Явно пустой `DB_PASS` сохраняется. Для других настроек SSL используйте полный `DATABASE_URL`.

Порядок запуска: PostgreSQL и Redis → `goose ... up` → Ledger → Gateway. После успешного `up` в том же терминале:

```bash
cd financial-planner/ledger
go run ./cmd/ledger
```

Затем в другом терминале из корня репозитория:

```bash
cd financial-planner/gateway
go run .
```

Для проверки API выполните cURL-примеры выше. Переменные окружения должны быть заданы в обоих терминалах сервисов; `.env` автоматически не читается. Для Goose передавайте DSN той же базы, даже если Ledger использует отдельные `DB_*`.

Ledger возвращает ошибки чтения и записи из БД; Gateway отображает внутренние ошибки в `500` с прежним JSON-форматом. `AddTransaction(Transaction) error` сохранена; `CreateTransaction(Transaction) (Transaction, error)` дополнительно возвращает сохранённую запись с ID для HTTP-ответа. `ListBudgets` и `ListTransactions` возвращают срез и ошибку.

## Redis-кеш отчётов

Запуск Redis 7:

```bash
docker run -p 127.0.0.1:6379:6379 --name cashcraft-redis -d redis:7-alpine
docker exec cashcraft-redis redis-cli PING
```

Пакет `ledger/internal/cache` использует официальный клиент `github.com/redis/go-redis/v9`. Конфигурация читается из `REDIS_ADDR` (по умолчанию `localhost:6379`), `REDIS_DB` (`0`, неотрицательное целое), `REDIS_PASSWORD` (пустая строка). `ledger.InitDB(ctx)` автоматически подключает кеш и логирует успешный `PING`, поэтому код Gateway менять не нужно. При недоступном Redis сервис продолжает использовать PostgreSQL.

`ledger.GetReportSummary(ctx, from, to) ([]ledger.ReportSummary, error)` принимает даты `YYYY-MM-DD`, включая обе границы. SQL-запрос суммирует расходы по категориям и сортирует результат по категории. Каждый элемент содержит `category` и `total`:

```go
summary, err := ledger.GetReportSummary(ctx, "2026-11-01", "2026-11-30")
```

```json
[{"category":"Food","total":450}]
```

Ключ `report:summary:<from>:<to>` хранит JSON с TTL **30 секунд**. Попадание возвращает результат из Redis; промах запускает SQL-запрос и сохраняет результат. Пустой отчёт хранится как `[]`. Повреждённый кеш или ошибка Redis приводят к чтению PostgreSQL. После изменения расходов уже закешированный отчёт обновляется по истечении TTL.

## Тесты и покрытие

Из корня репозитория проверьте каждый модуль отдельно:

Для SQL- и HTTP-интеграционных тестов в каждом тестовом терминале задайте:

```bash
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
```

Роли нужно право `CREATE SCHEMA`: тесты создают отдельную схему для каждого сценария, применяют те же SQL-миграции и удаляют эту схему в `t.Cleanup()`. Общие таблицы БД не очищаются. Если `TEST_DATABASE_URL` не задана, интеграционные тесты пропускаются, а unit-тесты выполняются.

Кеш проверяется через `miniredis`, включая hit/miss, авторизацию и TTL без ожидания таймера. Проверки SQL-агрегации отчётов используют `TEST_DATABASE_URL`; внешний Redis для тестов не требуется.

```bash
cd financial-planner/gateway
go test ./... -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
go tool cover -func=cover.out
```

```bash
cd financial-planner/ledger
go test ./... -cover -coverprofile=cover.out
go tool cover -html=cover.out -o cover.html
go tool cover -func=cover.out
```

Каждый блок запускайте из корня репозитория в отдельном терминале. В каталоге модуля создаются `cover.out` и `cover.html`; HTML-отчёт можно открыть браузером. Эти файлы исключены из Git.

Покрытие тестами (операторов), зафиксированное до добавления подключения к БД:

| Область | Покрытие |
| --- | --- |
| Ledger, все пакеты модуля | 47,7% |
| Gateway, все пакеты модуля | 75,8% |
| Доменный пакет `ledger` | 82,6% |
| HTTP-пакет `gateway/internal/api` | 90,2% |

Итоги модулей соответствуют строке `total` из `go tool cover -func=cover.out` и включают точки входа `main` с покрытием 0%. Значения отдельных пакетов выводит `go test`.

`go test -v ./...` показывает сгруппированные подслучаи. `go test -race -shuffle=on -count=3 ./...` в каждом модуле проверяет гонки и повторяемость сценариев. Независимые unit-подтесты используют `t.Parallel()`, интеграционные сценарии остаются последовательными из-за общего указателя `ledger.DB`. `Reset() error` очищает таблицы текущей схемы через `TRUNCATE ... RESTART IDENTITY` и предназначен только для тестов.

Запуск демонстрационной команды Ledger из каталога `financial-planner/ledger`:

```bash
go run ./cmd/ledger
```
