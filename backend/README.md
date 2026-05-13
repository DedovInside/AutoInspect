# AutoInspect Backend

Backend-часть AutoInspect — серверное приложение на Go, которое связывает клиентское веб-приложение, ML-сервис и инфраструктурные компоненты системы. Компонент отвечает за аутентификацию пользователей, управление задачами анализа автомобиля, хранение результатов, работу с автосервисами, заявками на ремонт и административными сценариями.

AutoInspect предназначен для автоматизированного анализа изображений автомобилей: пользователь загружает фотографии и сведения об автомобиле, backend создаёт задачу анализа, передаёт её ML-сервису через Kafka, получает результат, сохраняет его и уведомляет пользователя.

## Основные возможности

- OAuth-аутентификация через Яндекс ID.
- Выпуск и обновление access/refresh JWT-токенов.
- Ролевая модель доступа: `user`, `car_service`, `admin`.
- Создание задач анализа автомобиля с загрузкой изображений в MinIO.
- Идемпотентное создание анализа через заголовок `Idempotency-Key`.
- Выбор специализированной или универсальной модели сегментации деталей.
- Проверка ML-артефактов в MinIO перед постановкой задачи.
- Асинхронная интеграция с ML-сервисом через Kafka и Protobuf.
- Обработка результатов анализа отдельным worker-процессом.
- Обогащение результата русскоязычными названиями деталей и повреждений.
- WebSocket-уведомления о завершении анализа.
- Управление заявками на роль автосервиса.
- Создание и редактирование профиля автосервиса.
- Управление изображениями и специализацией автосервиса.
- Подбор автосервисов по результату анализа.
- Создание, отмена, принятие и отклонение заявок на ремонт.
- Административное управление ML-артефактами.
- Заявки на добавление или обучение новых ML-моделей.
- Миграции PostgreSQL.

## Архитектура

Backend реализован как модульное приложение с разделением на транспортный, сервисный и инфраструктурный слои.

```text
cmd/
  api/       HTTP REST API и WebSocket
  worker/    обработка результатов анализа из Kafka
  migrator/  применение миграций PostgreSQL

internal/
  api/        маршруты, handlers, DTO, middleware
  service/    бизнес-логика
  domain/     доменные модели и ошибки
  repository/ работа с PostgreSQL и MinIO
  broker/     Kafka producer/consumer
  cache/      Redis
  notify/     Redis Pub/Sub уведомления
  config/     загрузка и валидация конфигурации
  proto/      Protobuf-контракты
```

Общая схема взаимодействия:

```text
Frontend
   |
   | REST API / WebSocket
   v
Backend API ---- PostgreSQL
   |  \          Redis
   |   \         MinIO
   |
   | protobuf task
   v
Kafka request topic
   |
   v
ML-service
   |
   | protobuf result
   v
Kafka result topic
   |
   v
Backend worker ---- PostgreSQL
       |
       v
Redis Pub/Sub -> Backend API -> WebSocket -> Frontend
```

## Процессы

### API-процесс

`cmd/api` запускает HTTP-сервер и обрабатывает:

- REST API-запросы клиентского приложения;
- WebSocket-соединения;
- OAuth-сценарии;
- загрузку изображений и ML-артефактов;
- создание задач анализа;
- публикацию задач в Kafka;
- подписку на Redis Pub/Sub для отправки WebSocket-уведомлений.

### Worker-процесс

`cmd/worker` читает сообщения из Kafka topic результатов анализа и:

- парсит protobuf-результат;
- находит задачу анализа по `correlation_id`;
- сохраняет результат в PostgreSQL;
- переводит задачу в статус `completed` или `failed`;
- публикует событие завершения через Redis Pub/Sub.

### Migrator-процесс

`cmd/migrator` применяет миграции PostgreSQL с помощью `golang-migrate`.

## Основной сценарий анализа

1. Пользователь проходит OAuth-аутентификацию через Яндекс ID.
2. Frontend отправляет запрос `POST /v1/analyses` с данными автомобиля и изображениями.
3. Backend проверяет JWT, входные данные и `Idempotency-Key`.
4. Изображения загружаются в MinIO.
5. Backend выбирает подходящую модель сегментации деталей:
   - специализированную модель, если она подходит по марке, модели, поколению и году;
   - универсальную модель, если специализированной нет.
6. Backend проверяет наличие `.pt`, `parts_inference_config.json` и `parts_catalog.json` в MinIO.
7. В PostgreSQL создаётся запись `analysis_jobs`.
8. Backend формирует protobuf-сообщение и публикует задачу в Kafka.
9. ML-сервис выполняет анализ и отправляет protobuf-результат в Kafka.
10. Worker получает результат, сохраняет его в PostgreSQL и публикует уведомление.
11. Frontend получает WebSocket-событие и запрашивает результат через REST API.

## Используемые технологии

- Go
- Gin
- PostgreSQL
- Redis
- Apache Kafka
- MinIO / S3-compatible storage
- Protocol Buffers
- Docker / Docker Compose
- sqlc
- golang-migrate
- Yandex ID OAuth
- JWT
- WebSocket

## Конфигурация

Конфигурация загружается из переменных окружения или из файла `.env`, если он найден рядом с запускаемым процессом.

Обязательные переменные:

```env
DATABASE_URL=postgres://postgres:postgres@localhost:5433/autoinspect?sslmode=disable
JWT_SECRET=change-me-at-least-16-chars
YANDEX_CLIENT_ID=...
YANDEX_CLIENT_SECRET=...
```

Часто используемые переменные:

```env
ENVIRONMENT=development
HTTP_HOST=0.0.0.0
HTTP_PORT=8080
WS_ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000,http://localhost:8080

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_NOTIFY_CHANNEL=notify:analysis:job

KAFKA_BROKERS=localhost:9092
KAFKA_TOPIC_ANALYSIS_REQUEST=autoinspect.analysis.request
KAFKA_TOPIC_ANALYSIS_RESULT=autoinspect.analysis.result
KAFKA_TOPIC_DLQ=autoinspect.analysis.dlq
KAFKA_CONSUMER_GROUP_ID=autoinspect-backend

S3_ENDPOINT=http://localhost:9000
S3_PUBLIC_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_REGION=us-east-1
S3_USE_PATH_STYLE=true
S3_BUCKET_UPLOADS=autoinspect-uploads
S3_BUCKET_MODELS=autoinspect-models
S3_BUCKET_RESULTS=autoinspect-results
S3_PRESIGNED_URL_TTL=15m
```

Для Docker Compose часть значений переопределяется в `deployments/docker-compose.yml`.

## Запуск через Docker Compose

Из корня проекта:

```powershell
docker compose -f deployments\docker-compose.yml up --build
```

В составе окружения поднимаются:

- PostgreSQL;
- Redis;
- Kafka;
- MinIO;
- backend migrator;
- backend API;
- backend worker;
- ML-service.

API будет доступен по адресу:

```text
http://localhost:8080
```

Проверка работоспособности:

```powershell
curl.exe http://localhost:8080/health
```

MinIO Console:

```text
http://localhost:9001
```

Логин и пароль по умолчанию:

```text
minioadmin / minioadmin
```

## Локальный запуск backend без Docker

Перед запуском должны быть доступны PostgreSQL, Redis, Kafka и MinIO.

Применить миграции:

```powershell
cd backend
go run ./cmd/migrator up
```

Запустить API:

```powershell
cd backend
go run ./cmd/api
```

Запустить worker:

```powershell
cd backend
go run ./cmd/worker
```

## Миграции базы данных

Миграции находятся в директории `migrations`.

Основные команды:

```powershell
go run ./cmd/migrator up
go run ./cmd/migrator down
go run ./cmd/migrator steps 1
go run ./cmd/migrator steps -1
go run ./cmd/migrator version
```

В миграциях создаются таблицы для:

- пользователей и OAuth-идентичностей;
- refresh-сессий;
- ML-моделей;
- задач анализа;
- заявок на обучение моделей;
- заявок на роль автосервиса;
- профилей автосервисов;
- изображений автосервисов;
- типов повреждений и категорий деталей;
- специализаций автосервисов;
- заявок на ремонт.

## Protobuf

Контракты обмена между backend и ML-сервисом описаны в `proto`.

Backend публикует задачу анализа в Kafka topic `autoinspect.analysis.request`, а результат получает из `autoinspect.analysis.result`.

Ключом Kafka-сообщения используется `correlation_id`, который связывает запись в `analysis_jobs` с сообщениями Kafka.

## Основные REST API группы

### Auth

- `GET /v1/auth/yandex/start`
- `GET /v1/auth/yandex/callback`
- `POST /v1/auth/oauth/yandex`
- `POST /v1/auth/refresh`
- `GET /v1/auth/me`
- `POST /v1/auth/logout`

### Analyses

- `POST /v1/analyses`
- `GET /v1/analyses`
- `GET /v1/analyses/:id`
- `GET /v1/analyses/:id/images/:idx`
- `GET /v1/analyses/:id/car-services`
- `GET /v1/analyses/ws`

### Model training requests

- `POST /v1/model-training-requests`
- `GET /v1/model-training-requests`
- `GET /v1/admin/model-training-requests`
- `PATCH /v1/admin/model-training-requests/:id/status`

### Car service applications

- `POST /v1/car-service-applications`
- `GET /v1/car-service-applications/current`
- `GET /v1/car-service-applications`
- `GET /v1/admin/car-service-applications`
- `PATCH /v1/admin/car-service-applications/:id/approve`
- `PATCH /v1/admin/car-service-applications/:id/reject`

### Car service profile

- `GET /v1/car-service/profile`
- `PATCH /v1/car-service/profile`
- `PATCH /v1/car-service/profile/active`
- `POST /v1/car-service/profile/images`
- `GET /v1/car-service/profile/images`
- `PATCH /v1/car-service/profile/images/:id/primary`
- `DELETE /v1/car-service/profile/images/:id`
- `GET /v1/car-service/profile/specialization-options`
- `GET /v1/car-service/profile/specializations`
- `PUT /v1/car-service/profile/specializations`

### Repair requests

- `POST /v1/repair-requests`
- `GET /v1/repair-requests`
- `GET /v1/repair-requests/:id`
- `PATCH /v1/repair-requests/:id/cancel`
- `GET /v1/car-service/repair-requests`
- `GET /v1/car-service/repair-requests/:id`
- `PATCH /v1/car-service/repair-requests/:id/accept`
- `PATCH /v1/car-service/repair-requests/:id/reject`

### Models

- `GET /v1/models/specialized`
- `POST /v1/admin/models`
- `GET /v1/admin/models`
- `PATCH /v1/admin/models/:id/deactivate`

## Тестирование

Запуск всех Go-тестов:

```powershell
cd backend
go test ./...
```

Если нужно явно задать локальный кеш сборки:

```powershell
$env:GOCACHE='C:\AutoInspect\.gocache'
go test ./...
```

## Особенности реализации

### Идемпотентность

Создание анализа, заявок и загрузка ML-артефактов используют `Idempotency-Key`.
Это защищает систему от повторной отправки одного и того же запроса при сетевых сбоях или повторном нажатии пользователем.

### Асинхронная обработка анализа

ML-инференс не выполняется внутри HTTP-запроса. Backend создаёт задачу, публикует сообщение в Kafka и сразу возвращает пользователю статус `pending`.
Результат сохраняется позднее worker-процессом.

### Компенсация при загрузке ML-артефактов

При административной загрузке модели backend сначала сохраняет файлы в MinIO, а затем создаёт запись в PostgreSQL.
Если запись в PostgreSQL создать не удалось, уже загруженные файлы удаляются из MinIO.

### Presigned URL

Backend не отдаёт изображения напрямую. Для доступа к изображениям анализа и изображениям автосервиса формируются временные presigned URL.

### Обогащение результата анализа

Результат ML-сервиса содержит технические коды деталей и повреждений.
Backend обогащает результат русскоязычными названиями, родительскими категориями деталей и сведениями о стороне детали.

## Статусы

Задача анализа:

- `pending`
- `completed`
- `failed`

Заявка на роль автосервиса:

- `pending`
- `approved`
- `rejected`

Заявка на обучение модели:

- `pending`
- `approved`
- `rejected`
- `in_progress`
- `completed`

Заявка на ремонт:

- `pending`
- `accepted`
- `rejected`
- `canceled`
