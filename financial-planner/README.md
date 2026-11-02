# Сервисы Gateway и Ledger

Проект содержит два Go-модуля:

- `financial-planner/gateway` — HTTP-сервис Gateway;
- `financial-planner/ledger` — сервис Ledger с бюджетами и транзакциями в памяти.

## Требования

- установлен Go версии, указанной в файлах `go.mod`;
- команды выполняются из корня репозитория.

Проверить установленную версию Go:

```bash
go version
```

## Запуск Gateway

Перейдите в каталог сервиса и запустите его:

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
| PUT | `/api/budgets` | Заглушка обновления бюджета (`501`) |

`POST /api/transactions` принимает JSON с полями `amount`, `category`, `description`, `date` и возвращает `201 Created` с `TransactionResponse`. Некорректный JSON или ошибка валидации возвращают `400 Bad Request`, превышение бюджета — `409 Conflict` с `{"error":"budget exceeded"}`, неожиданные ошибки — `500 Internal Server Error`. Ошибки возвращаются в JSON с полем `error`.

`GET /api/transactions` возвращает `200 OK` и JSON-массив `TransactionResponse`. При отсутствии транзакций возвращается `[]`.

`POST /api/budgets` принимает JSON с полями `category`, `limit`, проверяет и сохраняет бюджет. При успехе возвращает `201 Created` с `BudgetResponse`, при некорректном JSON или ошибке валидации — `400 Bad Request` с JSON `{"error":"описание ошибки"}`. Неожиданные ошибки возвращаются как `500 Internal Server Error`.

`GET /api/budgets` возвращает `200 OK` и JSON-массив `BudgetResponse`. При отсутствии бюджетов возвращается `[]`. Ledger отдаёт копию значений внутреннего map под блокировкой чтения; порядок элементов не гарантируется.

`PUT /api/budgets` пока возвращает `501 Not Implemented`. Неподдерживаемые методы возвращают `405 Method Not Allowed`, неизвестные пути — `404 Not Found`.

### Примеры cURL

Запустите Gateway и выполните команды в другом терминале. Флаг `-i` выводит статус и заголовки ответа. До добавления данных `GET /api/transactions` и `GET /api/budgets` возвращают `[]`.

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

Повторный `POST /api/budgets` с той же категорией заменяет её бюджет. Gateway стартует с пустым хранилищем; файл `budgets.json` загружает только команда Ledger. Данные двух процессов не разделяются и теряются при перезапуске.

## Запуск Ledger

Из корня репозитория перейдите в каталог Ledger и запустите команду:

```bash
cd financial-planner/ledger
go run ./cmd/ledger
```

Доменный пакет `ledger` импортируется из Gateway. Точка входа Ledger находится в `ledger/cmd/ledger`.

При запуске Ledger:

1. открывает файл `budgets.json`;
2. загружает из него бюджеты в память;
3. добавляет тестовые транзакции;
4. отклоняет транзакции, превышающие бюджет категории;
5. выводит сообщение о загрузке бюджетов, результаты валидации, ошибки и сохранённые транзакции в консоль.

Ledger нужно запускать именно из каталога `financial-planner/ledger`, поскольку относительный путь `budgets.json` вычисляется от текущего рабочего каталога.

Все бюджеты и транзакции хранятся в оперативной памяти и удаляются после завершения программы.

## Файл бюджетов

Начальные бюджеты задаются в `financial-planner/ledger/budgets.json` в виде массива JSON-объектов:

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
- `Period` — период действия бюджета.

При отсутствии файла или некорректном JSON Ledger выводит сообщение об ошибке и завершает работу. Если сумма сохранённых транзакций категории вместе с новой транзакцией превышает `Limit`, новая транзакция не сохраняется.

## HTTP DTO

Пакет `gateway/internal/api` отделяет JSON-формат HTTP от доменных моделей:

- `CreateTransactionRequest`: `amount`, `category`, `description`, `date`;
- `TransactionResponse`: `id`, `amount`, `category`, `description`, `date`;
- `CreateBudgetRequest`: `category`, `limit`;
- `BudgetResponse`: `category`, `limit`, `period`.

Методы запросов `ToDomain()` создают доменные объекты. Функции `NewTransactionResponse()` и `NewBudgetResponse()` создают DTO ответов. Проверка данных остаётся в Ledger. Идентификатор транзакции и период бюджета не задаются этими запросами и при преобразовании остаются нулевыми.

В `gateway/go.mod` настроены зависимость `ledger` и локальная замена `replace ledger => ../ledger`.

## Тесты и покрытие

Каждый сервис является отдельным Go-модулем, поэтому проверять их нужно отдельно:

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

Достигнутое покрытие операторов:

| Область | Покрытие |
| --- | --- |
| Ledger, все пакеты модуля | 47,7% |
| Gateway, все пакеты модуля | 75,8% |
| Доменный пакет `ledger` | 82,6% |
| HTTP-пакет `gateway/internal/api` | 90,2% |

Итоги по модулям взяты из `go tool cover -func=cover.out` и учитывают точки входа `main`, покрытые на 0%. Значения по отдельным пакетам выводит `go test`.

Для подробного дерева сценариев запустите `go test -v ./...`, для проверки гонок и изоляции — `go test -race -shuffle=on -count=3 ./...` в каждом модуле. Независимые unit-тесты выполняются параллельно; сценарии с общим Ledger выполняются последовательно и очищают хранилище через `Reset()` и `t.Cleanup()`.
