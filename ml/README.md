# AutoInspect: ML-часть

**Исполнитель**: Бершицкий Дмитрий Александрович

В проекте используется композиция из 3 моделей:

## 1. Классификация ракурса фотографии

### Датасет

Собран синтетический датасет: https://huggingface.co/datasets/mitbersh/car-position

### Модель

Обученные модели и их метрики: https://www.comet.com/brshtsk/car-perspective

Лучший результат у модели с гиперпараметрами lr = 3.66e-05, batch_size = 32: 

### Быстрый инференс на пользовательском фото

Для классификации одного изображения используйте скрипт `perspective-classification/infer_perspective.py`.

Пример запуска:

```bash
python perspective-classification/infer_perspective.py --image ./example.jpg --model ./best_car_view_model.pth --data-dir ./car_position_dataset --top-k 3
```

Скрипт:
- приводит изображение к квадрату (padding цветом среднего RGB изображения),
- ресайзит до нужного размера модели,
- выводит top-1 класс и уверенность,
- опционально печатает top-k вероятностей.

## 2. Сегментация деталей

С fine-tuning под конкретные модели авто.

## 3. Сегментация повреждений

С классификацией степени повреждения.