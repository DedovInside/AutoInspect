# ml-service

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

## parts_inference_config.json (этап 2)

ML-pipeline ожидает путь к `parts_inference_config.json` и использует секцию `inference` для настройки части модели (например `imgsz`, `conf`, `iou`, `max_det`, `retina_masks`, `device`).
