# Сервисы Gateway и Ledger

Проект содержит два Go-модуля:

- `financial-planner/gateway` — HTTP-сервис Gateway;
- `financial-planner/ledger` — доменная логика и SQL-хранилище бюджетов и транзакций PostgreSQL.

Пошаговые инструкции находятся в [README Ledger](ledger/README.md) и [README Gateway](gateway/README.md): создание БД, Goose, переменные PostgreSQL/Redis и проверка API.

## Требования

- установлен Go версии, указанной в файлах `go.mod`;
- команды выполняются из корня репозитория.

Проверить установленную версию Go:

```bash
go version
```

## Запуск Gateway

Перейдите в каталог сервиса и запустите его:

Перед запуском примените миграции и задайте `DATABASE_URL` в этом терминале (см. раздел «Миграции PostgreSQL»). Gateway использует те же функции Ledger и ту же БД.

```bash
cd financial-planner/gateway
go run .
```

Gateway будет доступен по адресу `http://localhost:8080`.

Все запросы проходят через `LoggingMiddleware`: в консоль выводятся HTTP-метод, исходный путь и длительность обработки. Это включает `/ping`, маршруты `/api` и ответы на неизвестные пути или неподдерживаемые методы.

Ответы имеют заголовок `Content-Type: application/json; charset=utf-8`. `JSONMiddleware` приводит к JSON также стандартные ответы маршрутизатора и возвращает `500` при панике обработчика. Ошибки имеют единый формат `{"error":"описание ошибки"}`: `400` для неверного JSON и валидации, `409` для превышения бюджета, `500` для внутренних ошибок.

Для проверки в другом терминале выполните:

```bash
curl http://localhost:8080/ping
```

Ожидаемый ответ:

```json
{"message":"pong"}
```

Остановить сервис можно сочетанием клавиш `Ctrl+C`.

### Маршруты API

Маршруты регистрируются в `gateway/internal/api/router.go` с общим префиксом `/api`:

| Метод | Путь | Операция |
| --- | --- | --- |
| GET | `/api/transactions` | Список транзакций |
| POST | `/api/transactions` | Создание транзакции |
| GET | `/api/budgets` | Список бюджетов |
| POST | `/api/budgets` | Создание бюджета |
| GET | `/api/reports/summary?from=YYYY-MM-DD&to=YYYY-MM-DD` | Отчёт по категориям, кеш Redis 30 секунд |
| PUT | `/api/budgets` | Заглушка обновления бюджета (`501`) |

`POST /api/transactions` принимает JSON с полями `amount`, `category`, `description`, `date` и возвращает `201 Created` с `TransactionResponse`. Некорректный JSON или ошибка валидации возвращают `400 Bad Request`, превышение бюджета — `409 Conflict` с `{"error":"budget exceeded"}`, неожиданные ошибки — `500 Internal Server Error`. Ошибки возвращаются в JSON с полем `error`.

`GET /api/transactions` возвращает `200 OK` и JSON-массив `TransactionResponse`, отсортированный по дате и ID по убыванию. При отсутствии транзакций возвращается `[]`. `POST` возвращает ID, сгенерированный PostgreSQL; дата должна иметь формат `YYYY-MM-DD`.

`POST /api/budgets` принимает JSON с полями `category`, `limit`, проверяет и сохраняет бюджет. При успехе возвращает `201 Created` с `BudgetResponse`, при некорректном JSON или ошибке валидации — `400 Bad Request` с JSON `{"error":"описание ошибки"}`. Неожиданные ошибки возвращаются как `500 Internal Server Error`.

`GET /api/budgets` возвращает `200 OK` и JSON-массив `BudgetResponse`, отсортированный по категории. При отсутствии бюджетов возвращается `[]`.

`GET /api/reports/summary` возвращает `200 OK` и массив `{"category":"…","total":…}` за включённый интервал `from`–`to`. Параметры обязательны и имеют формат `YYYY-MM-DD`; неправильный диапазон возвращает `400`.

`PUT /api/budgets` пока возвращает `501 Not Implemented`. Неподдерживаемые методы возвращают `405 Method Not Allowed`, неизвестные пути — `404 Not Found`.

### Примеры cURL

Запустите Gateway с новой мигрированной БД и выполните команды в другом терминале. Флаг `-i` выводит статус и заголовки ответа. До добавления данных `GET /api/transactions` и `GET /api/budgets` возвращают `[]`.

Проверка сервера и создание бюджета (`200` и `201`):

```bash
curl -i http://localhost:8080/ping

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

Отчёт дважды подряд (`200`):

```bash
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
curl -i 'http://localhost:8080/api/reports/summary?from=2025-09-01&to=2025-09-30'
```

Для новой БД после расхода `450` оба ответа содержат `[{"category":"еда","total":450}]`. В консоли Gateway первый запрос без готового ключа отмечается как `report summary cache miss`, повторный — как `report summary cache hit`.

Проверка ошибки валидации (`400`):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":0,"category":"еда","description":"ланч","date":"2025-09-10"}'
```

После бюджета `1000` и транзакции `450` следующая транзакция превышает оставшийся лимит (`409`, `{"error":"budget exceeded"}`):

```bash
curl -i -X POST http://localhost:8080/api/transactions \
  -H 'Content-Type: application/json' \
  -d '{"amount":600,"category":"еда","description":"ужин","date":"2025-09-10"}'
```

Повторный `POST /api/budgets` с той же категорией обновляет её бюджет через UPSERT. Данные сохраняются между перезапусками. Ledger и Gateway используют общую PostgreSQL при одинаковом DSN.

## Запуск Ledger

Сначала настройте PostgreSQL и примените миграции, как описано в разделе «Миграции PostgreSQL». Затем из корня репозитория перейдите в каталог Ledger и запустите команду:

```bash
cd financial-planner/ledger
go run ./cmd/ledger
```

Доменный пакет `ledger` импортируется из Gateway. Точка входа Ledger находится в `ledger/cmd/ledger`.

При запуске Ledger:

1. открывает пул PostgreSQL через `sql.Open("pgx", dsn)` и проверяет подключение через `PingContext` с таймаутом 5 секунд;
2. при успехе пишет `Ledger connected to PostgreSQL`, при ошибке завершает процесс с ненулевым кодом;
3. читает существующие бюджеты и расходы из PostgreSQL;
4. выводит их в консоль и завершает работу, закрывая пул соединений.

CLI Ledger показывает содержимое БД. HTTP-сервер запускается отдельно в Gateway.

Все бюджеты и транзакции хранятся в PostgreSQL.

## Файл бюджетов

Файл `financial-planner/ledger/budgets.json` содержит пример для явного импорта через `ledger.LoadBudgets(io.Reader)` и автоматически при запуске не загружается:

```json
[
  {
    "Category": "Food",
    "Limit": 2000,
    "Period": "2026-10"
  },
  {
    "Category": "Transport",
    "Limit": 500,
    "Period": "2026-10"
  }
]
```

Поля бюджета:

- `Category` — категория расходов;
- `Limit` — максимальная сумма транзакций категории;
- `Period` — поле доменной модели; текущая схема БД его не хранит.

`LoadBudgets` возвращает ошибки чтения, валидации и записи в БД. Если сумма расходов категории вместе с новой транзакцией превышает `Limit`, новая запись не сохраняется. Категория без бюджета допускает расходы без лимита.

## HTTP DTO

Пакет `gateway/internal/api` отделяет JSON-формат HTTP от доменных моделей:

- `CreateTransactionRequest`: `amount`, `category`, `description`, `date`;
- `TransactionResponse`: `id`, `amount`, `category`, `description`, `date`;
- `CreateBudgetRequest`: `category`, `limit`;
- `BudgetResponse`: `category`, `limit`, `period`.

Методы запросов `ToDomain()` создают доменные объекты. Функции `NewTransactionResponse()` и `NewBudgetResponse()` создают DTO ответов. Проверка данных остаётся в Ledger. PostgreSQL генерирует ID транзакции; HTTP-обработчик получает сохранённую запись через `CreateTransaction`. Прежняя функция `AddTransaction(Transaction) error` также доступна. `period` не задаётся запросом и возвращается пустой строкой, поскольку в таблице `budgets` такого столбца нет.

В `gateway/go.mod` настроены зависимость `ledger` и локальная замена `replace ledger => ../ledger`.

## Миграции PostgreSQL

SQL-миграции находятся в `financial-planner/ledger/migrations`. Используется [Goose](https://github.com/pressly/goose): каждый файл содержит секции `-- +goose Up` и `-- +goose Down`.

- `00001_create_budgets.sql` — таблица `budgets` с уникальной категорией и положительным `limit_amount`.
- `00002_create_expenses.sql` — таблица `expenses` с ненулевой суммой и обязательной датой.

Установите Goose и добавьте каталог Go-утилит в `PATH`:

```bash
go install github.com/pressly/goose/v3/cmd/goose@v3.29.0
export PATH="$(go env GOPATH)/bin:$PATH"
goose -version
```

Нужна запущенная PostgreSQL с заранее созданной базой `cashapp`. Из корня репозитория задайте DSN своей локальной БД (ниже значения по умолчанию Ledger), затем примените миграции:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
goose -dir financial-planner/ledger/migrations validate
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" up
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" status
```

Откат последней миграции удаляет `expenses`; повторный `down` удаляет `budgets`:

```bash
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" down
```

Для отката всех миграций одной командой:

```bash
goose -dir financial-planner/ledger/migrations postgres "$DATABASE_URL" down-to 0
```

Откат удаляет таблицы вместе с данными. `ledger.InitDB` использует `ledger/internal/db`, проверенный `*sql.DB` доступен как `ledger.DB`. Автоматическое применение миграций при старте не выполняется.

### Настройка подключения и порядок запуска

Непустая переменная `DATABASE_URL` имеет приоритет. Если она не задана или пуста, Ledger собирает DSN из переменных:

| Переменная | По умолчанию |
| --- | --- |
| `DB_HOST` | `localhost` |
| `DB_PORT` | `5432` |
| `DB_USER` | `postgres` |
| `DB_PASS` | `postgres`, если переменная не задана; явно пустой пароль сохраняется |
| `DB_NAME` | `cashapp` |

В собранном DSN используется `sslmode=disable`. Для других настроек SSL передайте полный `DATABASE_URL`. Для Goose используйте DSN той же базы, даже если Ledger настроен через отдельные `DB_*`.

Порядок запуска: PostgreSQL и Redis → `goose ... up` → Ledger → Gateway. После успешной миграции в этом же терминале:

```bash
cd financial-planner/ledger
go run ./cmd/ledger
```

В другом терминале из корня репозитория запустите Gateway и выполните приведённые выше cURL-запросы:

```bash
cd financial-planner/gateway
go run .
```

Переменные окружения задавайте в обоих терминалах сервисов; `.env` автоматически не загружается. Контракты HTTP сохраняются. Ошибки чтения и записи БД возвращаются из Ledger, а Gateway отвечает `500` в прежнем JSON-формате.

`SetBudget` выполняет UPSERT по категории. `CreateTransaction` проверяет бюджет и вставляет расход в одной SQL-транзакции с `FOR UPDATE` на строке бюджета; одновременные расходы одной категории последовательно проверяют лимит. Сравнение денежных сумм выполняется в PostgreSQL как `NUMERIC`. Для категории без бюджета вставка разрешена.

## Redis-кеш отчётов

Запустите Redis локально через Docker:

```bash
docker run -p 127.0.0.1:6379:6379 --name cashcraft-redis -d redis:7-alpine
docker exec cashcraft-redis redis-cli PING
```

Параметры окружения:

| Переменная | По умолчанию |
| --- | --- |
| `REDIS_ADDR` | `localhost:6379` |
| `REDIS_DB` | `0`, целое неотрицательное число |
| `REDIS_PASSWORD` | пустая строка |

Настройка выполняется в `ledger/internal/cache` через официальный `github.com/redis/go-redis/v9`. `ledger.InitDB(ctx)` также подключает Redis, поэтому существующий Gateway получает кеш без изменений обработчиков. После успешного `PING` выводится `Redis connected: addr=... db=...`. CLI Ledger закрывает клиент через `CloseCache()` при завершении.

Функция `GetReportSummary(ctx, from, to)` принимает две строки `YYYY-MM-DD` и возвращает `([]ReportSummary, error)`. Она агрегирует `SUM(amount) GROUP BY category` по включённому интервалу дат, сортируя строки по категории:

```go
summary, err := ledger.GetReportSummary(ctx, "2026-11-01", "2026-11-30")
```

Пример результата в JSON:

```json
[{"category":"Food","total":450}]
```

Ключ кеша: `report:summary:2026-11-01:2026-11-30`, значение — JSON, TTL — **30 секунд**. При попадании результат возвращается из Redis, при промахе — вычисляется через PostgreSQL и сохраняется в Redis. Для пустого отчёта кешируется `[]`.

Ошибка Redis, невалидный JSON в кеше или ошибка записи кеша не ломают SQL-отчёт: используется PostgreSQL. Если Redis недоступен при старте, сервис пишет сообщение и продолжает работу с БД. Уже сохранённый отчёт после добавления расходов может оставаться прежним до истечения TTL; затем пересчитывается.

## Тесты и покрытие

Каждый сервис является отдельным Go-модулем, поэтому проверять их нужно отдельно:

Для SQL- и HTTP-интеграционных сценариев задайте `TEST_DATABASE_URL` в каждом тестовом терминале. Роль должна иметь право `CREATE SCHEMA`. Каждый сценарий создаёт отдельную схему, применяет встроенные SQL-миграции и удаляет только эту схему в `t.Cleanup()`:

```bash
export TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/cashapp?sslmode=disable'
```

Без `TEST_DATABASE_URL` интеграционные сценарии пропускаются; unit-тесты валидации, DSN, DTO и ошибок продолжают выполняться. `Reset() error` очищает таблицы текущей схемы через `TRUNCATE ... RESTART IDENTITY` и предназначен только для тестов.

Тесты кеша используют `miniredis`: проверяют настройки, авторизацию, hit/miss, TTL, повреждённый JSON и отказ Redis. Истечение TTL моделируется без ожидания 30 секунд. Тесты отчётов с агрегацией дополнительно используют PostgreSQL из `TEST_DATABASE_URL`.

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

Каждый блок запускайте из корня репозитория в отдельном терминале. HTML-отчёт `cover.html` создаётся в каталоге соответствующего модуля; его можно открыть браузером. `cover.out` и `cover.html` исключены из Git.

Покрытие операторов, зафиксированное на этапе тестирования HTTP API (до добавления подключения к БД):

| Область | Покрытие |
| --- | --- |
| Ledger, все пакеты модуля | 47,7% |
| Gateway, все пакеты модуля | 75,8% |
| Доменный пакет `ledger` | 82,6% |
| HTTP-пакет `gateway/internal/api` | 90,2% |

Итоги по модулям взяты из `go tool cover -func=cover.out` и учитывают точки входа `main`, покрытые на 0%. Значения по отдельным пакетам выводит `go test`.

Для подробного дерева сценариев запустите `go test -v ./...`, для проверки гонок и изоляции — `go test -race -shuffle=on -count=3 ./...` в каждом модуле. Независимые unit-тесты выполняются параллельно; сценарии с общим указателем `ledger.DB` остаются последовательными и используют отдельные тестовые схемы.
