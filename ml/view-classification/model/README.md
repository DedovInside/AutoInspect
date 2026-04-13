# AutoInspect Car View Classifier

[![Project](https://img.shields.io/badge/Project-AutoInspect-black?logo=github)](https://github.com/DedovInside/AutoInspect/tree/ml/ml)
[![Hugging Face](https://img.shields.io/badge/Hugging%20Face-car--view-yellow?logo=huggingface)](https://huggingface.co/mitbersh/car-view)
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

Модель является частью пайплайна **AutoInspect**:

`view classification → car parts segmentation → damage segmentation`

## Быстрый старт

#### 1. Установка зависимостей для инференса

```bash
pip install -r requirements.infer.txt
```

#### 2. Скачать модель

```bash
python -c "from huggingface_hub import hf_hub_download; hf_hub_download(repo_id='mitbersh/car-view', filename='car_view_model.pth', local_dir='.')"
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

Публичный датасет: [mitbersh/car-view dataset](https://huggingface.co/datasets/mitbersh/car-view)

Датасет состоит из **synthetic** и **real** частей:

- **Synthetic:** изображения автомобилей из Carvana, наложенные на разные фоны
- **Real:** реальные фотографии повреждённых автомобилей, релевантные домену AutoInspect

Источники:
- Carvana: https://www.kaggle.com/competitions/carvana-image-masking-challenge
- Real damaged cars: https://humansintheloop.org/resources/datasets/car-parts-and-car-damages-dataset/

Реальная часть была изначально размечена baseline-моделью, обученной на Carvana, а затем вручную скорректирована.

## Обучение

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
