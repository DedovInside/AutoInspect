# AutoInspect

![logo-core](img/logo-core.png)

**AutoInspect** - веб-сервис для автоматизированного анализа повреждений автомобилей по изображениям.

Пользователь загружает фотографии автомобиля, система определяет видимые детали и повреждения, сопоставляет повреждения с конкретными элементами кузова и формирует структурированный результат анализа. Пользователь может просмотреть историю осмотров, подобрать подходящий автосервис и создать заявку на ремонт.

**Демо:** https://auto-inspect-bay.vercel.app/

## Возможности

- загрузка изображений автомобиля и запуск анализа;
- сегментация автомобильных деталей и повреждений;
- сопоставление повреждений с конкретными деталями;
- история анализов и просмотр результатов;
- WebSocket-уведомления о завершении обработки;
- OAuth-аутентификация через Яндекс ID;
- роли пользователей: `USER`, `SERVICE`, `ADMIN`;
- профиль автосервиса с изображениями и специализациями;
- подбор автосервисов по результатам анализа;
- создание и обработка заявок на ремонт;
- административное управление ML-моделями и заявками на обучение.

## Архитектура

AutoInspect построен как модульная система из frontend-приложения, backend API, ML-микросервиса и инфраструктурных сервисов.

```text
Frontend
  |
  | REST API / WebSocket
  v
Backend API ---- PostgreSQL
  |   \
  |    \---- Redis / PubSub
  |     \
  |      \--- MinIO / S3
  |
  | AnalysisRequest
  v
Kafka request topic
  |
  v
ML-service
  |
  | AnalysisResult
  v
Kafka result topic
  |
  v
Backend worker ---- PostgreSQL
  |
  v
Redis Pub/Sub -> WebSocket -> Frontend
```

### Компоненты

| Компонент | Назначение                                                                     |
|---|--------------------------------------------------------------------------------|
| `frontend/` | Веб-интерфейс для пользователей, автосервисов и администраторов.               |
| `backend/` | REST API, WebSocket, бизнес-логика, хранение данных и интеграция с ML-service. |
| `ml-service/` | Асинхронный ML-worker для обработки задач анализа через Kafka и MinIO.         |
| `ml/` | Исследовательская и обучающая часть ML-системы.                                |
| `proto/` | Protobuf-контракты обмена между backend и ML-service.                          |
| `deployments/` | Docker Compose окружение для локальной инфраструктуры.                         |

## Как работает анализ

1. Пользователь загружает изображения автомобиля.
2. Backend сохраняет изображения в MinIO и создаёт задачу анализа.
3. Backend выбирает подходящую модель сегментации деталей: универсальную или специализированную.
4. Задача отправляется в Kafka.
5. ML-service скачивает изображения и ML-артефакты, запускает inference и публикует результат.
6. Backend worker сохраняет результат в PostgreSQL.
7. Frontend получает WebSocket-уведомление и отображает результат пользователю.

## ML-система

ML-часть AutoInspect построена как каскад моделей компьютерного зрения:

```text
Image
  |
  | parts segmentation
  v
Part masks
  |
  | damage segmentation
  v
Damage masks
  |
  | matching: damage -> part
  v
Structured analysis result
```

### Основные ML-модули

| Модуль | Назначение |
|---|---|
| `View Model` | Классификация ракурса автомобиля. Используется как вспомогательный инструмент при подготовке датасетов. |
| `Part Segmentation` | Сегментация видимых деталей автомобиля: двери, бамперы, крылья, фары, колёса и другие элементы. |
| `Damage Segmentation` | Сегментация повреждений автомобиля и выделение зон дефектов. |
| `Matching Pipeline` | Сопоставление масок повреждений с масками деталей и формирование итогового результата. |

В production-сценарии ML-service использует связку `parts + damage`: отдельно определяет детали и повреждения, затем связывает найденные повреждения с соответствующими деталями автомобиля.

## ML-артефакты

ML-артефакты вынесены из исходного кода и хранятся отдельно:

- датасеты и модели публикуются на Hugging Face;
- обучение и эксперименты воспроизводятся через Kaggle notebooks;
- метрики и сравнение экспериментов ведутся в Comet;
- runtime-модели и конфиги для сервиса загружаются в MinIO/S3.

Полезные ссылки:

- [ML README](./ml/README.md)
- [View model](https://huggingface.co/mitbersh/car-view-classification)
- [View dataset](https://huggingface.co/datasets/mitbersh/car-view-classification)
- [Car parts segmentation dataset](https://huggingface.co/datasets/mitbersh/car-parts-segmentation-yolo)
- [Car damage segmentation dataset](https://huggingface.co/datasets/mitbersh/car-damage-segmentation-yolo)

## Технологический стек

### Frontend

- React
- React Router
- Vite

### Backend

- Go
- Gin
- PostgreSQL
- Redis
- Kafka
- MinIO / S3
- Protocol Buffers
- JWT
- Yandex ID OAuth
- WebSocket

### ML-service и ML

- Python
- PyTorch
- YOLOv26
- OpenCV / NumPy
- Kafka
- MinIO / S3
- Protobuf
- Hugging Face
- Kaggle
- Comet

### Infrastructure

- Docker
- Docker Compose
- PostgreSQL
- Redis
- Kafka
- MinIO

## Структура репозитория

```text
AutoInspect/
├── backend/        # Go backend: API, worker, migrator, domain logic
├── frontend/       # React/Vite frontend
├── ml-service/     # Python ML worker for Kafka/S3 inference
├── ml/             # ML research, training, datasets and notebooks
├── deployments/    # Docker Compose configuration
├── proto/          # Protobuf contracts
└── img/            # Project images and logos
```

## Роли пользователей

| Роль | Возможности |
|---|---|
| `USER` | Создание анализов, просмотр истории, подбор автосервисов, создание заявок на ремонт. |
| `SERVICE` | Управление профилем автосервиса, специализациями и заявками на ремонт. |
| `ADMIN` | Управление заявками, ML-моделями и административными сценариями. |

## Особенности реализации

### Асинхронный inference

ML-инференс не выполняется внутри HTTP-запроса. Backend создаёт задачу, публикует сообщение в Kafka и возвращает пользователю статус обработки. Результат сохраняется позднее backend worker-ом.

### Protobuf-контракты

Backend и ML-service обмениваются сообщениями `AnalysisRequest` и `AnalysisResult` через Kafka. Общие структуры сообщений описаны в `proto/`.

### Presigned URLs

Backend не отдаёт изображения напрямую. Для доступа к изображениям анализа и изображениям автосервиса формируются временные presigned URL.

### Специализированные модели

AutoInspect поддерживает универсальную модель сегментации деталей и специализированные модели, адаптированные под конкретные автомобили или домены изображений.

## Документация компонентов

- [Frontend README](./frontend/README.md)
- [Backend README](./backend/README.md)
- [ML-service README](./ml-service/README.md)
- [ML README](./ml/README.md)

## Команда

- **Дедов Иван Андреевич** - backend, инфраструктура, интеграция сервисов;
- **Гусева Полина Дмитриевна** - frontend;
- **Бершицкий Дмитрий Александрович** - ML-подсистема.


