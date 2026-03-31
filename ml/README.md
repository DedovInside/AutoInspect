# AutoInspect: ML-часть

**Исполнитель**: Бершицкий Дмитрий Александрович

В проекте используется композиция из 3 моделей:

## 1. Классификация ракурса фотографии

### Датасет

Собран синтетический датасет: https://huggingface.co/datasets/mitbersh/car-view

### Модель

Лучший результат показала [модель](https://huggingface.co/mitbersh/car-view) с гиперпараметрами lr = 3.66e-05, batch_size = 32.
В [ноутбуке Infer Car View](https://www.kaggle.com/code/brshtskmit/infer-car-view) можно опробовать модель со своими фото.

Обучение модели происходило в [ноутбуке Train View Model](https://www.kaggle.com/code/brshtskmit/train-view-model).
Метрики можно посмотреть на [Comet-дешборде](https://www.comet.com/brshtsk/car-perspective).

## 2. Сегментация деталей

С fine-tuning под конкретные модели авто.

## 3. Сегментация повреждений

С классификацией степени повреждения.