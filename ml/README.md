# AutoInspect: ML-часть

**Исполнитель**: Бершицкий Дмитрий Александрович

![logo-ml.png](../img/logo-ml.png)

ML-часть AutoInspect - это каскад из нескольких моделей:
отдельная ResNet-классификация ракурса автомобиля, YoLo-сегментация деталей (coarse/tuned/detailed)
и отдельная YoLo-сегментация повреждений. Модели позволяют определять тип дефекта и затронутые детали авто.
Доступна общая модель, а также функционал для дообучения модели под конкретные авто, что позволяет получить максимальную точность.
Ниже - подробный разбор каждой модели в проекте:

## 1. View Model

### Fine-tuned ResNet 18: Классификация ракурса фотографии

* Модель Pythorch (Hugging face): https://huggingface.co/mitbersh/car-view
* Ноутбук, чтобы протестировать модель (Kaggle): https://www.kaggle.com/code/brshtskmit/infer-car-view
* Ноутбук с обучением (Kaggle): https://www.kaggle.com/code/brshtskmit/train-view-model
* Дешборд с метриками моделей с разными гиперпараметрами (Comet ML): https://www.comet.com/brshtsk/car-perspective
* Собственный Car View Dataset (Hugging face): https://huggingface.co/datasets/mitbersh/car-view
* На основе View Model реализованы Supervisely Apps для доразметки датасета под Part Segmentation Model:
[SuperviselyPerspective](https://github.com/brshtsk/SuperviselyPerspective)
и [SuperviselyPartsTags](https://github.com/brshtsk/SuperviselyPartsTags)

## 2. Part Segmentation Model

### Fine-tuned YoLo26-segm: Сегментация частей авто

Разбита на 3 части:

### 2a. Part Segmentation Coarse (General)

Общая модель сегментации крупных деталей автомобиля. Используется как база для Tuned моделей или как универсальный сегментатор

* Адаптированный Parts&Damages Dataset (Supervisely): https://app.supervisely.com/projects/373665/datasets/1128482.
Базируется на датасете от [Humans In The Loop](https://humansintheloop.org/resources/datasets/car-parts-and-car-damages-dataset/).
Были использованы Supervisely Apps [SuperviselyPerspective](https://github.com/brshtsk/SuperviselyPerspective)
и [SuperviselyPartsTags](https://github.com/brshtsk/SuperviselyPartsTags) для обогащения датасета и добавления меток
left/right на детали

### 2b. Part Segmentation Coarse (Tuned)

Специализированная модель сегментации крупных деталей автомобиля под конкретные авто. Базируется на Part Segmentation Coarse (General)

* Требует датасет формата X для дообучения
* Использует General модель как базу
* Ноутбук для дообучения: X

### 2c. Part Segmentation Detailed

Специализированная модель детальной сегментации под конкретные авто. Базируется на Part Segmentation Coarse (General)

* Работает не на грубом уровне (дверь/бампер/крыло), а находит мелкие элементы (в зависимости от детализированности датасета)
* Требует датасет формата X для дообучения
* Использует General модель как базу
* Ноутбук для дообучения: X

## 3. Damage Segmentation

Отдельная модель сегментации повреждений.

* Адаптированный Parts&Damages Dataset (Supervisely): https://app.supervisely.com/projects/373665/datasets/1128482