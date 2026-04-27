# ml-service

## Генерация Protobuf (этап 1)

`ml-service` использует protobuf-контракты из `proto/analysis/v1/` и генерирует Python-классы в `app/generated/analysis/v1/`.

### Установка минимальных зависимостей

```bash
pip install -r requirements.infer.txt
```

### Генерация Python protobuf-классов

```bash
./scripts/generate_proto.sh
```

Скрипт генерирует:
- `app/generated/analysis/v1/request_pb2.py`
- `app/generated/analysis/v1/result_pb2.py`

Примечания:
- Не редактируйте сгенерированные файлы вручную.
- Перезапускайте генерацию после любых изменений в `request.proto` или `result.proto`.
