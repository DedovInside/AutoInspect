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

AUGMENTATION_FACTOR = 2 # Если 1 - то просто 1 к 1 замена. Если 3 - то каждая машина будет на 3 разных фонах.
OTHER_CLASS_FACTOR = 1 # Сколько "кропов" (деталей) делать с одного фото для класса 'other'
SAVE_SIZE = 256


def get_view_from_filename(filename):
    """Определяем ракурс из имени файла Carvana и возвращаем название папки"""
    try:
        view_code = int(filename.split('_')[1].split('.')[0])
    except:
        return 'unknown'

    if view_code == 1: return 'front'
    if view_code in [2, 3, 4]: return 'front-left'
    if view_code == 5: return 'left'
    if view_code in [6, 7, 8]: return 'back-left'
    if view_code == 9: return 'back'
    if view_code in [10, 11, 12]: return 'back-right'
    if view_code == 13: return 'right'
    if view_code in [14, 15, 16]: return 'front-right'

    return 'unknown'


def crop_car_content(img_car, img_mask, padding=20):
    """
    Обрезает изображение и маску по границам маски (убирает лишнюю пустоту).
    padding: отступ в пикселях, чтобы не резать впритык к кузову.
    """
    points = cv2.findNonZero(img_mask)

    if points is None:
        return img_car, img_mask  # Если маска пустая, возвращаем как было

    x, y, w, h = cv2.boundingRect(points)

    h_img, w_img = img_car.shape[:2]

    x_new = max(0, x - padding)
    y_new = max(0, y - padding)
    w_new = min(w_img - x_new, w + 2 * padding)
    h_new = min(h_img - y_new, h + 2 * padding)

    # Обрезаем
    cropped_car = img_car[y_new: y_new + h_new, x_new: x_new + w_new]
    cropped_mask = img_mask[y_new: y_new + h_new, x_new: x_new + w_new]

    return cropped_car, cropped_mask


def blend_images(img_car, img_mask, bg_path, target_size=256, add_extra_background=True, scale=1.0):
    img_bg = cv2.imread(bg_path)

    if random.random() > 0.5:
        img_bg = cv2.flip(img_bg, 1)

    if img_bg is None:
        return None

    h_car, w_car = img_car.shape[:2]

    # Размер стороны квадрата
    square_side = int(max(h_car, w_car) * scale)
    square_side = max(square_side, target_size)  # Чтобы не было меньше целевого размера

    h_bg_orig, w_bg_orig = img_bg.shape[:2]
    if h_bg_orig < square_side or w_bg_orig < square_side:
        resize_scale = max(square_side / h_bg_orig, square_side / w_bg_orig)
        new_w_bg = int(w_bg_orig * resize_scale * 1.1)
        new_h_bg = int(h_bg_orig * resize_scale * 1.1)
        img_bg = cv2.resize(img_bg, (new_w_bg, new_h_bg))

    h_bg, w_bg = img_bg.shape[:2]

    # A. Случайный поворот фона (Tilt)
    if random.random() < 0.5:
        angle = random.uniform(-10, 10)  # Угол наклона +/- 10 градусов
        center = (w_bg // 2, h_bg // 2)
        M = cv2.getRotationMatrix2D(center, angle, 1.0)

        img_bg = cv2.warpAffine(img_bg, M, (w_bg, h_bg), borderMode=cv2.BORDER_REFLECT)

    # B. Случайный зум фона (Scale background)
    if random.random() < 0.5:
        bg_zoom = random.uniform(1.0, 1.5)
        new_w = int(w_bg * bg_zoom)
        new_h = int(h_bg * bg_zoom)
        img_bg = cv2.resize(img_bg, (new_w, new_h))
        # Размеры изменились, обновляем переменные
        h_bg, w_bg = img_bg.shape[:2]

    min_x = int(w_bg * 0.2)
    max_x = int(w_bg * 0.8) - square_side
    max_y = int(min(h_bg - square_side, h_bg * 0.2))

    bg_x = random.randint(min_x, max_x) if max_x > min_x else 0
    bg_y = random.randint(0, max_y) if max_y > 0 else 0
    img_bg_cropped = img_bg[bg_y:bg_y + square_side, bg_x:bg_x + square_side]

    max_car_x = square_side - w_car
    max_car_y = square_side - h_car

    if add_extra_background:
        car_start_x = random.randint(0, max_car_x)
        low_bound = int(max_car_y * 0.3)
        if low_bound < max_car_y:
            car_start_y = random.randint(low_bound, max_car_y)
        else:
            car_start_y = max_car_y // 2  # Если места совсем нет
    else:
        car_start_x = max_car_x // 2
        car_start_y = max_car_y // 2

    # Нормализуем маску машины
    mask_float = img_mask.astype(float) / 255.0
    k = 5 if min(h_car, w_car) > 50 else 1
    mask_blurred = cv2.GaussianBlur(mask_float, (k, k), 0)
    mask_3ch = np.dstack([mask_blurred] * 3)

    # Вырезаем область из фона (ROI), куда встанет машина
    roi = img_bg_cropped[car_start_y:car_start_y + h_car, car_start_x:car_start_x + w_car]

    # Смешиваем (Машина * mask + Фон * (1-mask))
    foreground = img_car.astype(float) * mask_3ch
    background = roi.astype(float) * (1.0 - mask_3ch)
    dst = (foreground + background).astype(np.uint8)

    # Вставляем смешанный кусок обратно в квадратный фон
    img_bg_cropped[car_start_y:car_start_y + h_car, car_start_x:car_start_x + w_car] = dst

    final_output = cv2.resize(img_bg_cropped, (target_size, target_size))

    return final_output


def generate_other_crop(img_car, img_mask):
    """
    Пытается найти случайный кусок авто (15-25% размера),
    где маска занимает более 50% площади кропа.
    """
    h, w = img_car.shape[:2]

    for _ in range(50):
        # Случайный размер от 15% до 25%
        crop_ratio = random.uniform(0.15, 0.25)

        crop_h = max(int(max(h, w) * crop_ratio), 256)
        crop_w = max(int(max(h, w) * crop_ratio), 256)

        # Случайная координата
        if h - crop_h <= 0 or w - crop_w <= 0: continue
        y = random.randint(0, h - crop_h)
        x = random.randint(0, w - crop_w)

        # Вырезаем маску для проверки
        mask_crop = img_mask[y:y + crop_h, x:x + crop_w]

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

    img_car, img_mask = crop_car_content(img_car, img_mask, padding=20)

    if img_car is None: continue

    view_name = get_view_from_filename(car_file)
    if view_name == 'unknown':
        print('unknown view!')
        continue  # Пропускаем файлы с непонятным именем

    # Генерация основных ракурсов
    for i in range(AUGMENTATION_FACTOR):
        random_bg = random.choice(bg_files)
        bg_path = os.path.join(BG_DIR, random_bg)

        # Вызываем функцию смешивания, передавая массивы
        if view_name in ['left', 'right']:
            scale = random.uniform(1.0, 1.05)
        elif view_name in ['front', 'back']:
            scale = random.uniform(1.2, 1.3)
        else:
            scale = random.uniform(1.05, 1.2)
        result_img = blend_images(img_car, img_mask, bg_path, scale=scale)

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

            result_other = blend_images(crop_car, crop_mask, bg_path, add_extra_background=False, scale=1.0)

            if result_other is not None:
                save_name = f"{car_file.replace('.jpg', '')}_other_{i}.jpg"
                # Сохраняем в папку other
                save_path = os.path.join(OUTPUT_DIR, "other", save_name)
                cv2.imwrite(save_path, result_other)
                counter += 1