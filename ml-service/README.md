# ml-service

## Что уже сделано

- protobuf-контракты `AnalysisRequest`/`AnalysisResult` и генерация Python-классов
- mapper: protobuf bytes ↔ dataclasses
- mock pipeline для проверки E2E без ML-моделей
- Kafka worker (mock) для будущего подключения транспорта
- настройки окружения через `.env`
- локальный кэш для S3-артефактов

## Генерация Protobuf (этап 1)

`ml-service` использует protobuf-контракты из `proto/analysis/v1/` и генерирует Python-классы в `app/generated/analysis/v1/`.

### Установка минимальных зависимостей

```bash
pip install -r requirements.txt
```

### Генерация Python protobuf-классов

```bash
./scripts/generate_proto.sh
```

Скрипт генерирует:
- `app/generated/analysis/v1/request_pb2.py`
- `app/generated/analysis/v1/result_pb2.py`

## parts_config.json (этап 2)

ML-pipeline ожидает путь к `parts_config.json` и использует секцию `inference` для настройки части модели (например `imgsz`, `conf`, `iou`, `max_det`, `retina_masks`, `device`).

## damage_config.json (этап 2.5)

Параметры композиции parts↔damage находятся внутри `damage_config.json` в секции `matching`:

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

## Переменные окружения (этап 3)

Пример находится в `.env.example`.

## Проверка настроек

```bash
python scripts/show_settings.py
```

## Проверка protobuf без Kafka (CLI)

### 1) Сформировать protobuf-запрос

```bash
python scripts/build_request.py --out .\tmp\request.bin
```

Можно переопределить поля через аргументы:

```bash
python scripts/build_request.py --out .\tmp\request.bin --correlation-id corr-123 --user-id user-42 --image-s3-keys uploads/a.jpg uploads/b.jpg --parts-model-s3-key models/v1/parts_segmentation.pt --parts-config-s3-key configs/parts_config.json --car-make Toyota --car-model Camry --car-generation XV70 --car-year 2022
```

### 2) Прогнать mock pipeline и вывести результат

```bash
python scripts/run_mock_flow.py --request .\tmp\request.bin --out .\tmp\result.bin
```

Скрипт выведет JSON-представление `AnalysisResult` и сохранит protobuf результат в `result.bin`.

## Локальный запуск реального инференса без Kafka/MinIO

Если у тебя есть локальные `.pt`, `parts_config.json` и `damage_config.json`, можно прогнать реальный адаптер так:

```bash
python scripts/run_real_inference.py --source "path\to\images" --parts-model "path\to\parts.pt" --damage-model "path\to\damage.pt" --parts-config "path\to\parts_config.json" --damage-config "path\to\damage_config.json" --out .\tmp\result.bin
```

Скрипт выведет JSON результата в stdout и при `--out` сохранит protobuf.

## Mock Kafka worker (этап 6)

`app/kafka/worker.py` слушает `KAFKA_REQUEST_TOPIC`, парсит protobuf-запрос и публикует результат в `KAFKA_RESULT_TOPIC`.

```bash
python -m app.kafka.worker
```
