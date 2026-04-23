# AutoInspect - ML

![logo-ml](../img/logo-ml.png)

ML-часть AutoInspect - это каскад моделей компьютерного зрения:
- классификация ракурса автомобиля (`View Model`),
- сегментация деталей (`Part Segmentation`),
- сегментация повреждений (`Damage Segmentation`).

Проект сделан с прицелом на production: общая модель + дообучение под конкретные марки и типы кузова.

**Автор**: Бершицкий Дмитрий Александрович

## Что уже готово

| Модуль | Назначение | Статус | Артефакты |
|---|---|---|---|
| View Model | Классификация ракурса фото авто | `READY` | Hugging Face, Kaggle, Comet |
| Part Segmentation | Сегментация деталей авто (coarse/tuned/detailed) | `IN PROGRESS` | Датасет и пайплайн в работе |
| Damage Segmentation | Сегментация зон повреждений | `IN PROGRESS` | Базовый датасет подготовлен |

> Сейчас полностью доступна модель 1 (View Model). Модули 2-3 находятся в активной разработке.

## Архитектура ML-каскада

1. `View Model` определяет ракурс авто (`front-left`, `back-right` и т.д.).
2. `Part Segmentation` сегментирует релевантные детали с учетом ракурса.
3. `Damage Segmentation` выделяет повреждения и связывает их с деталями.

Итог: модельный стек определяет **что повреждено**, **где расположено** и **к какому элементу относится**.

## 1) View Model (готово)

### Fine-tuned ResNet18: классификация ракурса

- Модель (Hugging Face): https://huggingface.co/mitbersh/car-view-classification
- Инференс-ноутбук (Kaggle): https://www.kaggle.com/code/brshtskmit/infer-car-view
- Обучение (Kaggle): https://www.kaggle.com/code/brshtskmit/train-view-model
- Метрики и эксперименты (Comet): https://www.comet.com/brshtsk/car-perspective
- Датасет (Hugging Face): https://huggingface.co/datasets/mitbersh/car-view-classification

На базе `View Model` сделаны инструменты для подготовки датасета сегментации:
- [SuperviselyPerspective](https://github.com/brshtsk/SuperviselyPerspective)
- [SuperviselyPartsTags](https://github.com/brshtsk/SuperviselyPartsTags)

## 2) Part Segmentation (в разработке)

### 2a. Coarse (General)

Базовая сегментация крупных частей авто (дверь, бампер, крыло и т.д.).
Используется как универсальный сегментатор и как база для specialized-моделей.

- Адаптированный датасет (Supervisely): https://app.supervisely.com/projects/373665/datasets/1128482
- Исходный набор: [Humans In The Loop](https://humansintheloop.org/resources/datasets/car-parts-and-car-damages-dataset/)
- Для обогащения разметки использовались:
  - [SuperviselyPerspective](https://github.com/brshtsk/SuperviselyPerspective)
  - [SuperviselyPartsTags](https://github.com/brshtsk/SuperviselyPartsTags)

### 2b. Coarse (Tuned)

Дообучаемая версия `Coarse (General)` под конкретные авто/домены.

- Статус: `IN PROGRESS`
- План: шаблон датасета + notebook дообучения + базовые метрики

### 2c. Detailed

Детальная сегментация мелких элементов (ниже уровня крупных панелей).

- Статус: `IN PROGRESS`
- План: расширенная схема классов + пайплайн дообучения + сравнение с coarse-моделью

## 3) Damage Segmentation (в разработке)

Отдельный контур сегментации повреждений с последующей привязкой к деталям авто.

- Базовый датасет: https://app.supervisely.com/projects/373665/datasets/1128482
- Статус: `IN PROGRESS`
- План: baseline-модель + валидация на реальных кейсах

## Дорожная карта

- [x] Релиз `View Model` и публичных артефактов
- [ ] Релиз `Part Segmentation Coarse (General)`
- [ ] Релиз `Part Segmentation Coarse (Tuned)`
- [ ] Релиз `Part Segmentation Detailed`
- [ ] Релиз `Damage Segmentation`
- [ ] Единый end-to-end пайплайн оценки повреждений
