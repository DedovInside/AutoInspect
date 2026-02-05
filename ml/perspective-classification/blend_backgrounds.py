# Для тренировки модели по классификации ракурса фото автомобиля помимо основого датасета с авто используем датасет
# от Carvana и наклкдываем фоны с видами улиц (чтобы модель не обучалась на идеальных фото из салона с белым фоном)

import cv2
import numpy as np
import os
from PIL import Image
import random
from tqdm import tqdm

CAR_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/cars-showroom/train'  # Фото авто Carvana
MASK_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/cars-showroom/train_masks'  # Маски Carvana
BG_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/new-backgrounds-cropped'  # Фоны для авто
OUTPUT_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/blended'  # Вывод

# Если 1 - то просто 1 к 1 замена. Если 3 - то каждая машина будет на 3 разных фонах.
AUGMENTATION_FACTOR = 2

os.makedirs(OUTPUT_DIR, exist_ok=True)

# Получаем списки файлов
# Carvana файлы имеют вид: 'id_01.jpg'. Маски: 'id_01_mask.gif'
car_files = [f for f in os.listdir(CAR_DIR) if f.endswith('.jpg')]

bg_files = [f for f in os.listdir(BG_DIR) if f.endswith(('.jpg', '.png', '.jpeg'))]

print(f"Найдено машин: {len(car_files)}")
print(f"Найдено фонов: {len(bg_files)}")

if len(bg_files) == 0:
    print("Ошибка: Папка с фонами пуста!")
    exit()


def blend_images(car_path, mask_path, bg_path):
    img_car = cv2.imread(car_path)

    pil_mask = Image.open(mask_path).convert('L')  # L = черно-белый
    img_mask = np.array(pil_mask)

    img_bg = cv2.imread(bg_path)

    if img_car is None or img_bg is None:
        return None

    h_car, w_car, _ = img_car.shape
    h_bg, w_bg, _ = img_bg.shape

    # Делаем размер фона всегда в 1.5 раза больше, чем машина
    scale_bg = 1.5
    new_bg_h = int(h_car * scale_bg)
    new_bg_w = int(w_car * scale_bg)

    img_bg = cv2.resize(img_bg, (new_bg_w, new_bg_h))
    h_bg, w_bg, _ = img_bg.shape

    # Случайная позиция для размещения машины на фоне
    max_start_x = max(int(w_bg * 0.1), int(w_bg * 0.2))
    max_start_y = 0

    start_x = random.randint(0, max_start_x) if max_start_x > 0 else 0
    start_y = random.randint(0, max_start_y) if max_start_y > 0 else 0

    # Вырезаем часть фона под размер машины
    img_bg_cropped = img_bg[start_y:start_y + h_car, start_x:start_x + w_car]

    # Нормализуем маску 0..1
    mask_float = img_mask.astype(float) / 255.0

    # Размываем края маски (Blur) для мягкого смешивания
    mask_blurred = cv2.GaussianBlur(mask_float, (5, 5), 0)

    # Делаем маску 3-канальной
    alpha = np.dstack([mask_blurred] * 3)

    # Формула смешивания: Car * Alpha + Bg * (1 - Alpha)
    foreground = img_car.astype(float) * alpha
    background = img_bg_cropped.astype(float) * (1.0 - alpha)

    final_image = (foreground + background).astype(np.uint8)

    return final_image


counter = 0

for car_file in tqdm(car_files):
    mask_file = car_file.replace('.jpg', '_mask.gif')
    mask_path = os.path.join(MASK_DIR, mask_file)
    car_path = os.path.join(CAR_DIR, car_file)

    if not os.path.exists(mask_path):
        continue

    # Делаем N вариантов с разными фонами
    for i in range(AUGMENTATION_FACTOR):
        # Берем случайный фон
        random_bg = random.choice(bg_files)
        bg_path = os.path.join(BG_DIR, random_bg)

        result_img = blend_images(car_path, mask_path, bg_path)

        if result_img is not None:
            # Сохраняем с новым именем
            save_name = f"{car_file.replace('.jpg', '')}_blend_{i}.jpg"
            cv2.imwrite(os.path.join(OUTPUT_DIR, save_name), result_img)
            counter += 1
