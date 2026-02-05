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
BG_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/backgrounds'  # Фоны для авто
OUTPUT_DIR = '/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/blended'  # Вывод

# Если 1 - то просто 1 к 1 замена. Если 3 - то каждая машина будет на 3 разных фонах.
AUGMENTATION_FACTOR = 1

os.makedirs(OUTPUT_DIR, exist_ok=True)

# Получаем списки файлов
# Carvana файлы имеют вид: 'id_01.jpg'. Маски: 'id_01_mask.gif'
car_files = [f for f in os.listdir(CAR_DIR) if f.endswith('.jpg')]
car_files = random.sample(car_files, 20)

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

    h, w, _ = img_car.shape

    # Случайный кроп из фона
    h_bg, w_bg, _ = img_bg.shape

    # Если фон меньше машины
    if h_bg < h or w_bg < w:
        scale = max(h / h_bg, w / w_bg) * 1.05
        img_bg = cv2.resize(img_bg, (0, 0), fx=scale, fy=scale)
        h_bg, w_bg, _ = img_bg.shape

    # Случайная точка вырезки
    start_x = random.randint(0, w_bg - w)
    start_y = random.randint(0, h_bg - h)
    img_bg_cropped = img_bg[start_y:start_y + h, start_x:start_x + w]

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
