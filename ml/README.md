# AutoInspect: ML-часть

**Исполнитель**: Бершицкий Дмитрий Александрович

В проекте используется композиция из 3 моделей:

## 1. Классификация ракурса фотографии

### Датасет

Собран [синтетический датасет](https://huggingface.co/datasets/mitbersh/car-view).

### Модель

Лучший результат показала [модель](https://huggingface.co/mitbersh/car-view) с гиперпараметрами lr = 3.66e-05, batch_size = 32.
В [ноутбуке Infer Car View](https://www.kaggle.com/code/brshtskmit/infer-car-view) можно опробовать модель.

Обучение модели происходило в [ноутбуке Train View Model](https://www.kaggle.com/code/brshtskmit/train-view-model).
Метрики можно посмотреть на [Comet-дешборде](https://www.comet.com/brshtsk/car-perspective).

## 2. Сегментация деталей

С fine-tuning под конкретные модели авто.

Для модели 2a выбран датасет _ с классами:

Уникальные детали:
* Back-bumper
* Back-door
* Back-wheel
* Back-window
* Back-windshield
* Fender
* Front-bumper
* Front-door
* Front-wheel
* Front-window
* Grille
* Headlight
* Hood
* License-plate
* Mirror
* Quarter-panel
* Rocker-panel
* Roof
* Tail-light
* Trunk
* Windshield

Уникальные повреждения:
* Broken part
* Corrosion
* Cracked
* Dent
* Flaking
* Missing part
* Paint chip
* Scratch

## 3. Сегментация повреждений

С классификацией степени повреждения.