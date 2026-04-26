# AutoInspect Car View Classifier

[![Project](https://img.shields.io/badge/Project-AutoInspect-black?logo=github)](https://github.com/DedovInside/AutoInspect/tree/ml/ml)
[![Hugging Face](https://img.shields.io/badge/Hugging%20Face-car--view-yellow?logo=huggingface)](https://huggingface.co/mitbersh/car-view-classification)
[![PyTorch](https://img.shields.io/badge/PyTorch-ResNet18%20Two--Head-ee4c2c?logo=pytorch&logoColor=white)](https://pytorch.org/)
[![Task](https://img.shields.io/badge/Task-Car%20View%20Classification-blue)]()
[![Classes](https://img.shields.io/badge/Classes-9-success)]()

Модель классификации ракурса автомобиля из проекта **AutoInspect**.

## О модели

- **Backbone:** `ResNet18`, предобученный на ImageNet
- **Архитектура:** two-head classifier
  - horizontal head: `left`, `center`, `right`
  - vertical head: `front`, `center`, `back`
- **Итоговый ракурс** собирается из предсказаний по двум осям: `front-left`, `back-right`, `left`, `front` и т.д.

Ранее `View Model` рассматривалась как часть каскада:

```text
View Model -> Part Segmentation -> Damage Segmentation
```

Сейчас она не является обязательной частью runtime-пайплайна, потому что текущая `YOLOv26-s` модель для `Part Segmentation General` достаточно хорошо различает left/right детали.

При этом `View Model` остается важной частью проекта, потому что использовалась для подготовки и обогащения датасета. На ее основе были сделаны инструменты:

- https://github.com/brshtsk/SuperviselyPerspective
- https://github.com/brshtsk/SuperviselyPartsTags

## Быстрый старт

#### 1. Установка зависимостей для инференса

```bash
pip install -r requirements.infer.txt
```

#### 2. Скачать модель

```bash
python -c "from huggingface_hub import hf_hub_download; hf_hub_download(repo_id='mitbersh/car-view-classification', filename='car_view_model.pth', local_dir='.')"
```

#### 3. Запустить инференс

```bash
python infer_perspective.py --image "path/to/car.jpg" --model "car_view_model.pth" --top-k 3 --device auto
```

## Классы

```python
CLASS_NAMES = [
    "back",
    "back-left",
    "back-right",
    "front",
    "front-left",
    "front-right",
    "left",
    "other",
    "right",
]
```

## Датасет

Публичный датасет: [mitbersh/car-view-classification dataset](https://huggingface.co/datasets/mitbersh/car-view-classification)

Датасет состоит из **synthetic** и **real** частей:

- **Synthetic:** изображения автомобилей из Carvana, наложенные на разные фоны
- **Real:** реальные фотографии повреждённых автомобилей, релевантные домену AutoInspect

Источники:
- Carvana: https://www.kaggle.com/competitions/carvana-image-masking-challenge
- Real damaged cars: https://humansintheloop.org/resources/datasets/car-parts-and-car-damages-dataset/
- Скрипт сборки датасета: [prepare_dataset.py](./dataset/prepare_dataset.py)

Реальная часть была изначально размечена baseline-моделью, обученной на Carvana, а затем вручную скорректирована.

## Обучение

Обучение проводилось в [Kaggle](https://www.kaggle.com/code/brshtskmit/train-car-view-classifier-resnet18) с логированием в [Comet](https://www.comet.com/brshtsk/car-perspective/view/new/panels).

- end-to-end fine-tuning
- оптимизатор: `AdamW`
- валидация: **real-only split**
- основная метрика отбора: `val_axes_macro_f1`

Лучшая конфигурация:

```text
lr=1.17e-04 | bs=16 | rsw=2.00 | wd=3.0e-06
```

Лучший результат:

```text
val_axes_macro_f1 = 0.993908
```

## Файлы

- `train_perspective.py` - обучение
- `infer_perspective.py` - инференс
- `requirements.txt` - зависимости для обучения
- `requirements.infer.txt` - зависимости для инференса
