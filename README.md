# Go: Financial Planner

Проект [financial-planner](financial-planner/README.md) содержит HTTP Gateway и доменный пакет Ledger. Модуль Gateway импортирует Ledger через локальный `replace` в `go.mod`.

## Запуск

Нужен Go 1.27.0. Выполните из корня репозитория:

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
| PUT | `/api/budgets` | Заглушка, `501 Not Implemented` |

Транзакция принимается с полями `amount`, `category`, `description`, `date`. Сумма должна быть положительной, категория и дата — непустыми. Ответ содержит `id` и те же поля; генерация ID пока не реализована, поэтому у созданной через API транзакции `id` равен `0`.

Бюджет принимается с полями `category`, `limit`. Категория должна быть непустой, лимит — положительным. Ответ содержит `category`, `limit`, `period`; период не задаётся этим запросом и остаётся пустой строкой.

Статусы ошибок: `400` — некорректный JSON или валидация, `409` — превышение бюджета, `500` — внутренняя ошибка. Неизвестные пути возвращают `404`, неподдерживаемые методы — `405`.

## Примеры cURL

Выполняйте команды по порядку после запуска Gateway. Флаг `-i` показывает статус и заголовки.

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

Данные хранятся в памяти и теряются после остановки Gateway. Gateway стартует с пустым хранилищем. Файл бюджетов загружает только отдельная команда Ledger, её память не разделяется с Gateway.

## Проверка

Из корня репозитория проверьте каждый модуль отдельно:

```bash
cd financial-planner/gateway
go test -race ./...
```

```bash
cd financial-planner/ledger
go test ./...
```

Запуск демонстрационной команды Ledger из каталога `financial-planner/ledger`:

```bash
go run ./cmd/ledger
```
