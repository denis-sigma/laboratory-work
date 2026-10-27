# TripGo — репозиторий для лабораторных работ

Заготовка курса «Разработка микросервисов на Go». Здесь вы делаете все пять
работ: каждая следующая продолжает предыдущую, переписывать сервис с нуля не
нужно.

## Что делать сразу

1. **Fork** этого репозитория к себе. Форк нужен, чтобы преподаватели видели
   список всех работ курса одной страницей.
2. Заведите модуль:

```bash
git clone git@github.com:<ваш-логин>/<ваш-репозиторий>.git
cd <ваш-репозиторий>
go mod init github.com/<ваш-логин>/<ваш-репозиторий>
```

Путь модуля потом не меняется — иначе придётся править все импорты. Проще всего
взять адрес своего репозитория, каким бы он ни был.

Дальше — [`homework/docs/getting-started.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/getting-started.md)
в репозитории курса: инструменты, окружение, миграции, вид сданной работы.

## Где что лежит

| Что | Где |
|---|---|
| Задания, документация, контракты | [`course-go-autumn-2026/course`](https://github.com/course-go-autumn-2026/course) |
| Слайды и записи лекций | [`lections/`](https://github.com/course-go-autumn-2026/course/tree/main/lections) |
| Как оценивают, дедлайны, порядок сдачи | [`homework/docs/grading.md`](https://github.com/course-go-autumn-2026/course/blob/main/homework/docs/grading.md) |
| Локальное окружение и утилита `tripgoctl` | [`course-go-autumn-2026/course-infra`](https://github.com/course-go-autumn-2026/course-infra) |

Задания появляются по мере курса, каждое — после своей пары лекций.

## Как сдавать

Ветка `homework/NN` от `main`, pull request в `main` своего форка, ссылка
ментору до дедлайна. Подробно — в `grading.md` репозитория курса.

## Чужие работы

Форки видны всем, включая ваши. Смотреть чужие решения, пока идёт курс, —
плохая идея: одинаковый код виден сразу, а разбираться на защите придётся
самому.

## Лабораторная работа 1

### Требования

- Go 1.27+
- Docker Desktop
- WSL2 Ubuntu
- tripgoctl

### Запуск окружения

```bash
tripgoctl cluster start
tripgoctl environment start
tripgoctl connect
```

### Миграции

```bash
make migrate
```

### Запуск сервиса

```bash
make run
```

Сервис запускается на адресе из `HTTP_ADDR`.

### Проверка

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

### Тесты

```bash
make test
```

### Генерация API

```bash
make generate
```

### Переменные окружения

Настройки описаны в `.env.example`.

Настоящий `.env` создаётся локальным окружением и в Git не добавляется.

### Решения

- Для транзакций выбран уровень изоляции `ReadCommitted`: для операций этой
  работы достаточно видеть только зафиксированные данные, а PostgreSQL
  блокирует конфликтующие изменения строк.
- `TxManager` открывает транзакцию и кладёт её в `context.Context`. Репозитории
  берут исполнителя запросов из контекста: при наличии транзакции используют её,
  иначе работают через пул соединений.
- Создание поездки и запись первого перехода статуса выполняются внутри одной
  транзакции. Ошибка любой вставки приводит к откату обеих операций.
- Запрет двух активных поездок одного водителя обеспечивается частичным
  уникальным индексом `trips_active_driver_uidx` по `driver_id` для строк со
  статусом `active`. Ошибка PostgreSQL `23505` преобразуется в `driver_busy`.

Для запуска в `.env` должны быть заданы `HTTP_ADDR`, `LOG_LEVEL`,
`SHUTDOWN_TIMEOUT`, `DATABASE_URL`, `DATABASE_MAX_CONNS`, `DATABASE_MIN_CONNS`,
`DATABASE_MAX_CONN_LIFETIME`, `DATABASE_CONNECT_TIMEOUT`,
`DATABASE_QUERY_TIMEOUT`, а также таймауты HTTP-сервера:
`HTTP_READ_TIMEOUT`, `HTTP_READ_HEADER_TIMEOUT`, `HTTP_WRITE_TIMEOUT` и
`HTTP_IDLE_TIMEOUT`.

## Лабораторная работа 2

### Архитектура

Сервис разделён на слои:

- `internal/domain` — доменные модели и ошибки;
- `internal/usecase` — бизнес-логика поездок и интерфейсы зависимостей;
- `internal/repository` — работа с данными поездок;
- `internal/infra` — PostgreSQL, транзакции, логирование, трассировка и метрики;
- `internal/transport/http` — HTTP-обработчики, middleware и преобразование ошибок;
- `internal/container` — сборка зависимостей приложения.

Зависимости создаются в `internal/container`. Бизнес-логика не зависит напрямую
от PostgreSQL, HTTP и OpenTelemetry.

### HTTP middleware

Middleware подключены в следующем порядке:

```text
Recover → Tracing → Logging → Metrics → HTTP router
```

- `Recover` перехватывает `panic` и возвращает `500`;
- `Tracing` создаёт OpenTelemetry trace для HTTP-запроса;
- `Logging` записывает метод, путь, статус, длительность, `trace_id` и `span_id`;
- `Metrics` записывает длительность, количество активных запросов и HTTP-статус.

### Observability

Сервис использует OpenTelemetry:

- трассировки отправляются в Jaeger через OTLP;
- HTTP-запросы и PostgreSQL-запросы представлены отдельными spans;
- метрики отправляются в Prometheus через OTel Collector;
- логи выводятся в JSON-формате через `log/slog`;
- бизнес-метрики показывают количество созданных и завершённых поездок.

Основные метрики:

```text
http_server_request_duration_seconds
http_server_active_requests
tripgo_trips_created_total
tripgo_trips_completed_total
```

Инструменты observability доступны по адресам:

```text
Jaeger:     http://localhost:22686
Prometheus: http://localhost:22909
Grafana:    http://localhost:22300
```

### Проверка лабораторной работы 2

```bash
make test
make test-integration
make lint
```

Проверка HTTP-метрик в Prometheus:

```promql
http_server_request_duration_seconds_count
```

Проверка бизнес-метрик:

```promql
tripgo_trips_created_total
tripgo_trips_completed_total
```
