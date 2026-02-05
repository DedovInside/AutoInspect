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
OTHER_CLASS_FACTOR = 1   # Сколько "кропов" (деталей) делать с одного фото для класса 'other'


def get_view_from_filename(filename):
    """Определяем ракурс из имени файла Carvana и возвращаем название папки"""
    try:
        view_code = int(filename.split('_')[1].split('.')[0])
    except:
        return 'unknown'

    # Исправленная логика мэппинга
    if view_code == 1: return 'front'
    if view_code in [2, 3, 4]: return 'front-left'
    if view_code == 5: return 'left'
    if view_code in [6, 7, 8]: return 'back-left'
    if view_code == 9: return 'back'
    if view_code in [10, 11, 12]: return 'back-right'
    if view_code == 13: return 'right'
    if view_code in [14, 15, 16]: return 'front-right'

    return 'unknown'

def blend_images(img_car, img_mask, bg_path):
    """
    Смешивает картинку машины (или её кусок) с фоном.
    Принимает numpy array машины и маски, но путь к фону.
    """

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


def generate_other_crop(img_car, img_mask):
    """
    Пытается найти случайный кусок авто (20-30% размера),
    где маска занимает более 50% площади кропа.
    """
    h, w = img_car.shape[:2]

    for _ in range(50):
        # Случайный размер от 10% до 20%
        crop_ratio_h = random.uniform(0.1, 0.2)
        crop_ratio_w = random.uniform(0.1, 0.2)

        crop_h = int(h * crop_ratio_h)
        crop_w = int(w * crop_ratio_w)

        # Случайная координата
        if h - crop_h <= 0 or w - crop_w <= 0: continue
        y = random.randint(0, h - crop_h)
        x = random.randint(0, w - crop_w)

        # Вырезаем маску для проверки
        mask_crop = img_mask[y:y + crop_h, x:x + crop_w]

        # Считаем процент белых пикселей (автомобиля) в кропе
        # Маска у нас 0..255, считаем > 127 как "есть машина"
        pixels_count = crop_h * crop_w
        car_pixels = np.count_nonzero(mask_crop > 127)

        if (car_pixels / pixels_count) > 0.5:
            # Условие выполнено (>50% машины)
            car_crop = img_car[y:y + crop_h, x:x + crop_w]
            return car_crop, mask_crop

    return None, None


os.makedirs(OUTPUT_DIR, exist_ok=True)

categories = ['front', 'front-left', 'left', 'back-left', 'back',
              'back-right', 'right', 'front-right', 'other']

for cat in categories:
    os.makedirs(os.path.join(OUTPUT_DIR, cat), exist_ok=True)

print(f"Папки созданы в: {OUTPUT_DIR}")

# Получаем списки файлов
# Carvana файлы имеют вид: 'id_01.jpg'. Маски: 'id_01_mask.gif'
car_files = [f for f in os.listdir(CAR_DIR) if f.endswith('.jpg')]
# car_files = random.sample(car_files, 100)

bg_files = [f for f in os.listdir(BG_DIR) if f.endswith(('.jpg', '.png', '.jpeg'))]

print(f"Найдено машин: {len(car_files)}")
print(f"Найдено фонов: {len(bg_files)}")

counter = 0

for car_file in tqdm(car_files):
    mask_file = car_file.replace('.jpg', '_mask.gif')
    mask_path = os.path.join(MASK_DIR, mask_file)
    car_path = os.path.join(CAR_DIR, car_file)

    if not os.path.exists(mask_path):
        continue

    img_car = cv2.imread(car_path)
    pil_mask = Image.open(mask_path).convert('L')
    img_mask = np.array(pil_mask)

    if img_car is None: continue

    view_name = get_view_from_filename(car_file)
    if view_name == 'unknown':
        continue  # Пропускаем файлы с непонятным именем

    # Генерация основных ракурсов
    for i in range(AUGMENTATION_FACTOR):
        random_bg = random.choice(bg_files)
        bg_path = os.path.join(BG_DIR, random_bg)

        # Вызываем функцию смешивания, передавая массивы
        result_img = blend_images(img_car, img_mask, bg_path)

        if result_img is not None:
            save_name = f"{car_file.replace('.jpg', '')}_bg{i}.jpg"
            # Сохраняем в подпапку ракурса
            save_path = os.path.join(OUTPUT_DIR, view_name, save_name)
            cv2.imwrite(save_path, result_img)
            counter += 1

    # Генерация деталей
    for i in range(OTHER_CLASS_FACTOR):
        # Генерируем кроп
        crop_car, crop_mask = generate_other_crop(img_car, img_mask)

        if crop_car is not None:
            random_bg = random.choice(bg_files)
            bg_path = os.path.join(BG_DIR, random_bg)

            # Накладываем кроп на фон
            # Функция blend_images универсальна, ей всё равно, целая машина или кусок
            result_other = blend_images(crop_car, crop_mask, bg_path)

            if result_other is not None:
                save_name = f"{car_file.replace('.jpg', '')}_other_{i}.jpg"
                # Сохраняем в папку other
                save_path = os.path.join(OUTPUT_DIR, "other", save_name)
                cv2.imwrite(save_path, result_other)
                counter += 1