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

## Тесты и покрытие

Из корня репозитория проверьте каждый модуль отдельно:

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

Покрытие тестами (операторов):

| Область | Покрытие |
| --- | --- |
| Ledger, все пакеты модуля | 47,7% |
| Gateway, все пакеты модуля | 75,8% |
| Доменный пакет `ledger` | 82,6% |
| HTTP-пакет `gateway/internal/api` | 90,2% |

Итоги модулей соответствуют строке `total` из `go tool cover -func=cover.out` и включают точки входа `main` с покрытием 0%. Значения отдельных пакетов выводит `go test`.

`go test -v ./...` показывает сгруппированные подслучаи. `go test -race -shuffle=on -count=3 ./...` в каждом модуле проверяет гонки и повторяемость сценариев. Независимые unit-подтесты используют `t.Parallel()`, тесты с общим хранилищем выполняются последовательно с `ledger.Reset()` перед сценарием и в `t.Cleanup()`.

Запуск демонстрационной команды Ledger из каталога `financial-planner/ledger`:

```bash
go run ./cmd/ledger
```
