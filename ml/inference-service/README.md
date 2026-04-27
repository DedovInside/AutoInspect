# AutoInspect ML Inference Service

Отдельный inference-слой для AutoInspect: запускает две runtime-модели и возвращает единый JSON для backend.

```text
Input image(s)
    ↓
Part Segmentation General
    ↓
Damage Segmentation
    ↓
Mask matching: damage mask -> part mask
    ↓
Structured JSON response
```

## Принцип работы

1. Загружает `Part Segmentation General`.
2. Загружает `Damage Segmentation`.
3. Запускает обе модели на одном изображении или папке изображений.
4. Для каждого damage instance считает пересечение damage mask с part masks.
5. Формирует JSON-контракт для backend:
   - `damage_instances`
   - `parts_summary`
   - `batch_summary`

Side-aware part class вида `left_front-door` превращается в:

```json
{
  "name": "front-door",
  "side": "left"
}
```

А нейтральный класс вида `hood` остается:

```json
{
  "name": "hood"
}
```

## Быстрый старт

#### 1. Установка зависимостей для инференса

```bash
pip install -r requirements.infer.txt
```

#### 2. Скачать модели

```bash
python -c "from huggingface_hub import hf_hub_download; hf_hub_download(repo_id='mitbersh/car-parts-segmentation', filename='car_parts_model.pt', local_dir='.')"
python -c "from huggingface_hub import hf_hub_download; hf_hub_download(repo_id='mitbersh/car-damage-segmentation', filename='car_damage_model.pt', local_dir='.')"
```

#### 3. Запустить инференс

```bash
python infer_autoinspect.py --source "path/to/car.jpg" --parts-model "car_parts_model.pt" --damage-model "car_damage_model.pt" --output "outputs/autoinspect_predictions.json" --retina-masks
```

#### 4. Визуализировать

```bash
python visualize_predictions.py --predictions "outputs/autoinspect_predictions.json" --output-dir "outputs/visualized"
```

## Основные параметры

```bash
--parts-imgsz 768
--damage-imgsz 896
--parts-conf 0.25
--damage-conf 0.25
--iou 0.70
--device auto
--retina-masks
--min-overlap 0.05
--min-assignment-score 0.05
--max-parts-per-damage 3
```

## Как считается связь damage -> part

Для каждой пары `damage mask` и `part mask` считается:

```text
overlap_ratio = intersection(damage_mask, part_mask) / area(damage_mask)
assignment_confidence = overlap_ratio * part_model_confidence
```

`confidence` внутри `damage.parts[]` - confidence связи “это повреждение относится к этой детали”.

## Выходной формат

```json
{
  "model_id": "general",
  "model_version": "v1.3.0",
  "batch_id": "batch_2026_04_25_001",
  "results": [
    {
      "image_id": "image_1",
      "image_uri": "s3://bucket/uploads/car_front.jpg",
      "image": {
        "width": 642,
        "height": 370
      },
      "damage_instances": [
        {
          "id": "image_1_damage_1",
          "damage_type": "dent",
          "polygon": [[303, 206], [304, 193], [309, 181]],
          "bbox": [303, 177, 359, 219],
          "confidence": 0.94,
          "parts": [
            {
              "name": "hood",
              "confidence": 0.92,
              "overlap_ratio": 0.96
            }
          ]
        }
      ],
      "parts_summary": [
        {
          "name": "hood",
          "damage_count": 1,
          "damage_types": {
            "dent": 1
          }
        }
      ]
    }
  ],
  "batch_summary": {
    "image_count": 1,
    "damage_count": 1,
    "damage_types": {
      "dent": 1
    },
    "parts": [
      {
        "name": "hood"
      }
    ]
  }
}
```
