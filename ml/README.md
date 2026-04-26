# AutoInspect - ML

[![ML](https://img.shields.io/badge/ML-Computer%20Vision-F99117)]()
[![Hugging Face](https://img.shields.io/badge/Hugging%20Face-Models%20%26%20Datasets-yellow?logo=huggingface)](https://huggingface.co/mitbersh)
[![Kaggle](https://img.shields.io/badge/Kaggle-Training%20Notebooks-20beff?logo=kaggle&logoColor=white)](https://www.kaggle.com/brshtskmit)
[![Comet](https://img.shields.io/badge/Comet-Experiment%20Tracking-000000)](https://www.comet.com/brshtsk)

![logo-ml](../img/logo-ml.png)

ML-часть **AutoInspect** - это набор моделей компьютерного зрения для автоматической оценки повреждений автомобиля по фотографии.

Проект построен с прицелом на production: есть general-модели, возможность дообучения под конкретные автомобили/домены и отдельный inference-сервис для интеграции с backend.

Автор: Бершицкий Дмитрий Александрович

---

## Статус модулей

| Модуль | Назначение | Статус |
|---|---|---|
| Part Segmentation General | Сегментация крупных деталей автомобиля | `READY` |
| Damage Segmentation | Сегментация повреждений | `READY` |
| View Model | Вспомогательная классификация ракурса / tooling для датасета | `READY / AUXILIARY` |
| ML Inference Service | Объединение Parts + Damages в JSON для backend | `IN PROGRESS` |
| Part Segmentation Tuned | Дообучение под конкретные авто/домены | `IN PROGRESS` |
| Detailed Segmentation | Детальная сегментация мелких элементов | `OPTIONAL / FUTURE` |

---

## Архитектура

Текущий ML-pipeline состоит из двух основных моделей и inference-сервиса:

```text
Input image
    ↓
Part Segmentation General
    ↓
Damage Segmentation
    ↓
Mask matching: damages -> parts
    ↓
Structured JSON response for backend
```

Главная задача пайплайна - определить:

- какие повреждения есть на автомобиле;
- где они расположены;
- к каким деталям относятся;
- сколько повреждений найдено на каждой детали.

---

## 1. Part Segmentation General

`Part Segmentation General` - основная модель сегментации крупных деталей автомобиля.

Модель определяет внешние элементы автомобиля:

- двери;
- бамперы;
- капот;
- багажник;
- крылья;
- фары;
- окна;
- колеса;
- зеркала;
- другие крупные кузовные элементы.

Для парных деталей учитывается сторона: `left` / `right`. Это важно, потому что AutoInspect должен не просто найти повреждение, а понять, какая именно деталь повреждена: например, `left front door`, `right headlight`, `left fender`.

### Артефакты

- Модель: https://huggingface.co/mitbersh/car-parts-segmentation
- Обучение: https://www.kaggle.com/code/brshtskmit/train-car-parts-segmentation-yolov26-s
- Датасет RAW/Supervisely: https://huggingface.co/datasets/mitbersh/car-parts-segmentation-raw
- Датасет YOLO: https://huggingface.co/datasets/mitbersh/car-parts-segmentation-yolo
- Исходный датасет HITL: https://humansintheloop.org/resources/datasets/car-parts-and-car-damages-dataset/

### Особенности

При обучении особое внимание уделялось корректному определению стороны деталей. Поэтому качество модели оценивалось не только через стандартные segmentation-метрики, но и через корректность распознавания `side` для парных элементов. Датасет HITL был адаптирован с помощью Supervisely Apps для добавления тегов `side` и подготовки масок для обучения.

---

## 2. Damage Segmentation

`Damage Segmentation` - модель сегментации повреждений автомобиля.

Она выделяет зоны повреждений и используется вместе с `Part Segmentation General`.

### Артефакты

- Модель: https://huggingface.co/mitbersh/car-damage-segmentation
- Датасет YOLO: https://huggingface.co/datasets/mitbersh/car-damage-segmentation-yolo
- Исходный датасет CArDD: https://cardd-ustc.github.io/

### Назначение

Модель используется для:

- поиска повреждений;
- получения масок повреждений;
- классификации типов повреждений;
- последующего сопоставления повреждений с деталями автомобиля.

---

## 3. ML Inference Service

`ML Inference Service` - это слой между ML-моделями и backend-частью приложения.

Сервис выполняет:

1. запуск `Part Segmentation General`;
2. запуск `Damage Segmentation`;
3. сопоставление масок повреждений с масками деталей;
4. агрегацию повреждений по деталям;
5. формирование итогового JSON-ответа для backend.

Главная логика:

```text
damage instance -> affected car parts
```

Например, если маска повреждения пересекается с масками `Hood` и `Front bumper`, сервис должен связать это повреждение с соответствующими деталями и confidence-значениями.

### Статус

Сервис находится в разработке.

Планируется:

- API для inference;
- batch inference;
- JSON-контракт;
- Docker-образ;
- интеграция с backend;
- поддержка `model_id` и `model_version`.

---

## 4. View Model

`View Model` - вспомогательная модель классификации ракурса автомобиля.

Она определяет ракурс фото, например:

- `front`;
- `rear`;
- `side`;
- `front-left`;
- `front-right`;
- `back-left`;
- `back-right`.

### Артефакты

- Модель: https://huggingface.co/mitbersh/car-view-classification
- Датасет: https://huggingface.co/datasets/mitbersh/car-view-classification
- Inference notebook: https://www.kaggle.com/code/brshtskmit/infer-car-view
- Training notebook: https://www.kaggle.com/code/brshtskmit/train-view-model
- Comet experiments: https://www.comet.com/brshtsk/car-perspective

### Роль в проекте

Ранее `View Model` рассматривалась как часть каскада:

```text
View Model -> Part Segmentation -> Damage Segmentation
```

Сейчас она не является обязательной частью runtime-пайплайна, потому что текущая `YOLOv26-s` модель для `Part Segmentation General` достаточно хорошо различает left/right детали.

При этом `View Model` остается важной частью проекта, потому что использовалась для подготовки и обогащения датасета. На ее основе были сделаны инструменты:

- https://github.com/brshtsk/SuperviselyPerspective
- https://github.com/brshtsk/SuperviselyPartsTags

---

## 5. Part Segmentation Tuned

`Part Segmentation Tuned` - направление для дообучения general-модели под конкретные автомобили, марки, типы кузова или домены.

Идея:

> General-модель работает как универсальный MVP-сегментатор, а tuned-модель может давать дополнительную точность в конкретном сервисном сценарии.

Примеры сценариев:

- популярная модель автомобиля;
- корпоративный клиент;
- страховой партнер;
- специфичные условия съемки.

### План

- подготовить датасет под конкретный автомобиль/домен;
- привести классы к общей coarse-схеме;
- запустить fine-tuning;
- сравнить `general` и `tuned`;
- проверить качество side-aware сегментации;
- сохранить отдельную версию модели;
- подключить через `model_id`.

---

## 6. Detailed Segmentation

`Detailed Segmentation` - опциональное направление для более точной сегментации мелких элементов автомобиля.

Этот модуль не входит в MVP.

Для MVP важнее стабильно решить основную задачу:

```text
damage -> affected coarse part
```

Detailed-модель может быть полезна позже для advanced/enterprise-сценариев или для автосервисов, которым нужна более глубокая детализация.

---

## Хранение артефактов

В проекте используются разные платформы для разных типов ML-артефактов.

### GitHub

GitHub используется как основная точка входа в проект.

В репозитории хранятся:

- README-документация;
- описание архитектуры;
- ссылки на модели;
- ссылки на датасеты;
- ссылки на notebooks;
- схемы классов;
- inference-контракты;
- production-код сервисов.

### Hugging Face

Hugging Face используется для хранения:

- моделей;
- датасетов;
- model cards;
- quick start guide;
- версий моделей.

### Kaggle

Kaggle используется для training notebooks:

- обучение;
- подготовка данных;
- метрики;
- визуализации;
- train/validation predictions.

### Comet

Comet используется для хранения экспериментов и метрик, где это применимо.

---

## Roadmap

### Done

- [x] View Model
- [x] View Model dataset
- [x] SuperviselyPerspective
- [x] SuperviselyPartsTags
- [x] Part Segmentation General
- [x] Damage Segmentation

### In Progress

- [ ] ML Inference Service
- [ ] Damage-to-part matching
- [ ] JSON-контракт для backend
- [ ] Batch inference
- [ ] Docker-образ inference-сервиса
- [ ] Интеграция с backend
- [ ] Pipeline для Part Segmentation Tuned

### Future / Optional

- [ ] Detailed Segmentation
- [ ] Fine-tuning под конкретные автомобили
- [ ] Поддержка нескольких `model_id`
- [ ] Поддержка нескольких версий моделей
- [ ] End-to-end evaluation pipeline

---
