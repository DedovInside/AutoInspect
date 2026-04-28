# ml-service

## Кратко о сервисе

ML‑микросервис принимает `AnalysisRequest` (protobuf) из Kafka, выполняет инференс (parts + damage) и публикует `AnalysisResult`. Транспорт (Kafka/S3) отделен от ML‑логики; ML‑pipeline - чистая функция по локальным путям.

## Интеграция для backend

### Контракты

- protobuf: `proto/analysis/v1/request.proto`, `proto/analysis/v1/result.proto`
- генерация: `./scripts/generate_proto.sh` → `app/generated/analysis/v1/*.py`

### Kafka

- request topic: `KAFKA_REQUEST_TOPIC`
- result topic: `KAFKA_RESULT_TOPIC`
- key для сообщений: `correlation_id`

### Request (AnalysisRequest)

Поля:
- `correlation_id`, `user_id`
- `car_info`
- `image_s3_keys[]`
- `parts_model_s3_key`
- `parts_config_s3_key`

### Result (AnalysisResult)

Поля:
- `correlation_id`, `status`, `error_message`
- `model_id`, `model_version`, `batch_id`
- `results[]` → `ImageAnalysisResult`

`parts_summary` **не отправляется** (считается на backend при необходимости).

### Конфиги

- `parts_config.json` - секция `inference`
- `damage_config.json` - секция `inference` + секция `matching`

Пример `damage_config.json` (matching внутри):

```json
{
  "inference": {
    "imgsz": 896,
    "conf": 0.25,
    "iou": 0.7,
    "max_det": 300,
    "retina_masks": true,
    "device": "auto"
  },
  "matching": {
    "min_overlap": 0.05,
    "min_assignment_score": 0.05,
    "max_parts_per_damage": 3
  }
}
```

### S3/MinIO и кэш

- скачивание в локальный кэш: `app/storage/s3_client.py`
- модели кэшируются по ключу
- изображения - в каталоге job по `correlation_id`

### Ошибки

- если обработка запроса падает, публикуется `AnalysisResult` со `status="failed"` и `error_message`

### Docker/Compose

Сервис собирается из `Dockerfile` и запускает `python -m app.kafka.worker`.

Пример подключения в `backend/deployments/docker-compose.yml`:

```yaml
  ml-service:
    build:
      context: ../ml-service
      dockerfile: Dockerfile
    container_name: autoinspect-ml-service
    environment:
      KAFKA_BOOTSTRAP_SERVERS: kafka:29092
      KAFKA_REQUEST_TOPIC: autoinspect.analysis.request
      KAFKA_RESULT_TOPIC: autoinspect.analysis.result
      KAFKA_CONSUMER_GROUP: autoinspect-ml-service
      S3_ENDPOINT_URL: http://minio:9000
      S3_ACCESS_KEY: minioadmin
      S3_SECRET_KEY: minioadmin
      S3_BUCKET: autoinspect
      LOCAL_CACHE_DIR: /cache
      DAMAGE_MODEL_S3_KEY: models/general/damage_segmentation.pt
      DAMAGE_CONFIG_S3_KEY: models/general/damage_inference_config.json
    volumes:
      - ml_cache:/cache
    depends_on:
      - kafka
      - minio
    restart: unless-stopped
```

В корневом разделе `volumes` добавьте:

```yaml
  ml_cache:
```

## Настройки окружения

Смотри `.env.example`. Ключевые переменные:
- Kafka: `KAFKA_BOOTSTRAP_SERVERS`, `KAFKA_REQUEST_TOPIC`, `KAFKA_RESULT_TOPIC`, `KAFKA_CONSUMER_GROUP`
- S3: `S3_ENDPOINT_URL`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- cache: `LOCAL_CACHE_DIR`
- damage defaults: `DAMAGE_MODEL_S3_KEY`, `DAMAGE_CONFIG_S3_KEY`

## Локальные команды (без Kafka/MinIO)

```bash
python scripts/build_request.py --out .\tmp\request.bin
python scripts/run_mock_flow.py --request .\tmp\request.bin --out .\tmp\result.bin
python scripts/run_real_inference.py --source "path\to\images" --parts-model "path\to\parts.pt" --damage-model "path\to\damage.pt" --parts-config "path\to\parts_config.json" --damage-config "path\to\damage_config.json" --out .\tmp\result.bin
```

ToDo: убрать PartsSummary отовсюду