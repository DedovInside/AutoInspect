# AutoInspect - ML

[![ML](https://img.shields.io/badge/ML-Computer%20Vision-F99117)]()
[![Hugging Face](https://img.shields.io/badge/Hugging%20Face-Models%20%26%20Datasets-yellow?logo=huggingface)](https://huggingface.co/mitbersh)
[![Kaggle](https://img.shields.io/badge/Kaggle-Training%20Notebooks-20beff?logo=kaggle&logoColor=white)](https://www.kaggle.com/brshtskmit)
[![Comet](https://img.shields.io/badge/Comet-Experiment%20Tracking-000000)](https://www.comet.com/brshtsk)

![logo-ml](../img/logo-ml.png)

ML-часть **AutoInspect** - это набор моделей компьютерного зрения для автоматической оценки повреждений автомобиля по фотографии.

Проект построен с прицелом на production: есть general-модели, возможность адаптации под конкретные автомобили/домены и отдельный inference-сервис для интеграции с backend.

Автор: Бершицкий Дмитрий Александрович

---

## Модули

| Модуль | Назначение                                                  | Статус |
|---|-------------------------------------------------------------|---|
| Part Segmentation General | Сегментация крупных деталей автомобиля               | `READY` |
| Damage Segmentation | Сегментация повреждений автомобиля                          | `READY` |
| View Model | Вспомогательная классификация ракурса / tooling для датасета | `READY / AUXILIARY` |
| ML Inference Service | Объединение Parts + Damages в JSON для backend              | `READY` |
| Specialized Parts Segmentation | Сегментация деталей под конкретное авто / домен / автопарк  | `READY` |

---

## Архитектура

Текущий ML-pipeline состоит из двух основных моделей и inference-сервиса:

```text
Input image
    ↓
Parts Segmentation
    ├── General Parts model: coarse parts
    └── Specialized Parts model: detailed classes for a specific vehicle/domain
    ↓
Damage Segmentation
    ↓
Mask matching: damages -> parts
    ↓
Structured JSON response for backend
```

В базовом сценарии используется универсальная модель `Part Segmentation General`, которая выделяет основные детали автомобиля.  
Для конкретных автомобилей или доменов может использоваться `Specialized Parts Segmentation`, обученная на отдельном датасете. Такая модель работает не с coarse-классами General-модели, а со всеми классами specialized-датасета, то есть фактически дает более детальную сегментацию под конкретный тип автомобиля.

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
- Comet: https://www.comet.com/brshtsk/car-parts-test

### Особенности

При обучении особое внимание уделялось корректному определению стороны деталей. Поэтому качество модели оценивалось не только через стандартные segmentation-метрики, но и через корректность распознавания `side` для парных элементов. Датасет HITL был адаптирован с помощью Supervisely Apps для добавления тегов `side` и подготовки масок для обучения.

---

## 2. Damage Segmentation

`Damage Segmentation` - модель сегментации повреждений автомобиля.

Используется вместе с `Part Segmentation` (`General` или `Specialized`) и выделяет зоны повреждений:

- вмятины;
- царапины;
- трещины;
- разбитое стекло;
- поврежденные фары;
- спущенные колеса.

### Артефакты

- Модель: https://huggingface.co/mitbersh/car-damage-segmentation
- Обучение: https://www.kaggle.com/code/brshtskmit/train-car-damage-segmentation-yolov26-m-cardd
- Датасет YOLO: https://huggingface.co/datasets/mitbersh/car-damage-segmentation-yolo
- Исходный датасет CArDD: https://cardd-ustc.github.io/
- Comet: https://www.comet.com/brshtsk/car-damage-test

### Особенности

При обучении особое внимание уделялось корректному определению мелких повреждений. При оценке качества модели учитывались специализированные метрики по распознаванию `tiny` (до 0.05% площади изображения) и `small` (от 0.05% до 0.25%) повреждений.

---

## 3. ML Inference Service

`ML Inference Service` - это слой между ML-моделями и backend-частью приложения.

Сервис выполняет:

1. запуск `Part Segmentation` (`General` или `Specialized`);
2. запуск `Damage Segmentation`;
3. сопоставление масок повреждений с масками деталей;
4. агрегацию повреждений по деталям;
5. формирование итогового JSON-ответа для backend.

Главная логика:

```text
damage instance -> affected car parts
```

Например, если маска повреждения пересекается с масками `Hood` и `Front bumper`, сервис должен связать это повреждение с соответствующими деталями и confidence-значениями.

---

## 4. View Model

`View Model` - вспомогательная модель классификации ракурса автомобиля.

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

## 5. Specialized Parts Segmentation

`Specialized Parts Segmentation` - адаптация general-модели под конкретные автомобили, марки, типы кузова или домены.

Идея:

> General-модель работает как универсальный сегментатор, а specialized-модель может давать дополнительную точность в конкретном сервисном сценарии.

Примеры сценариев:

- популярная модель автомобиля;
- корпоративный клиент;
- страховой партнер;
- специфичные условия съемки.

### Обучение и артефакты

- Гайд по обучению specialized модели с использованием pretrained-весов general: https://www.kaggle.com/code/brshtskmit/guide-train-specialized-parts-segmentation
- Пример обучения specialized-s из general-s: https://www.kaggle.com/code/brshtskmit/train-specialized-s-from-general-s-896
- Пример обучения specialized-m из general-m: https://www.kaggle.com/code/brshtskmit/train-specialized-m-from-general-m-896

### Датасет и апробация

Пайплайн адаптации тестировался на VW Polo 5:

- Датасет: https://huggingface.co/datasets/mitbersh/specialized-parts-yolo

---

## Хранение артефактов

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
- [x] ML Inference Service
- [x] Damage-to-part matching
- [x] JSON-контракт для backend
- [x] Batch inference
- [x] Docker-образ inference-сервиса
- [x] Интеграция с backend
- [x] Specialized Segmentation

### In Progress

- [ ] —
