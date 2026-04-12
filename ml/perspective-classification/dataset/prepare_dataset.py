import os
import random
import csv

import cv2
import numpy as np
from PIL import Image
from tqdm import tqdm

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_DIR = os.path.dirname(SCRIPT_DIR)

CAR_DIR = os.path.join(PROJECT_DIR, 'images', 'carvana', 'train')
MASK_DIR = os.path.join(PROJECT_DIR, 'images', 'carvana', 'train_masks')
BG_DIR = os.path.join(PROJECT_DIR, 'images', 'backgrounds')
HITL_DIR = os.path.join(PROJECT_DIR, 'images', 'hitl-views')
OUTPUT_DIR = os.path.join(SCRIPT_DIR, 'blended')
META_FILENAME = 'meta.csv'

AUGMENTATION_FACTOR = 1
SAVE_SIZE = 224
MAX_BG_ROTATION_DEG = 8
MAX_CAR_ROTATION_DEG = 7
MIN_MASK_RETENTION = 0.85
MIN_RANDOM_CROP_SCALE = 0.9

CATEGORIES = [
    'front',
    'front-left',
    'left',
    'back-left',
    'back',
    'back-right',
    'right',
    'front-right',
]


def get_view_from_filename(filename):
    """Определяем ракурс из имени файла Carvana и возвращаем название папки."""
    try:
        view_code = int(filename.split('_')[1].split('.')[0])
    except (IndexError, ValueError):
        return 'unknown'

    if view_code == 1:
        return 'front'
    if view_code in [2, 3, 4]:
        return 'front-left'
    if view_code == 5:
        return 'left'
    if view_code in [6, 7, 8]:
        return 'back-left'
    if view_code == 9:
        return 'back'
    if view_code in [10, 11, 12]:
        return 'back-right'
    if view_code == 13:
        return 'right'
    if view_code in [14, 15, 16]:
        return 'front-right'
    return 'unknown'


def normalize_view_name(raw_view_name):
    normalized = raw_view_name.strip().lower().replace('_', ' ').replace('-', ' ')
    normalized = ' '.join(normalized.split())
    mapping = {
        'front': 'front',
        'front left': 'front-left',
        'left': 'left',
        'back left': 'back-left',
        'back': 'back',
        'back right': 'back-right',
        'right': 'right',
        'front right': 'front-right',
    }
    return mapping.get(normalized, 'unknown')


def crop_car_content(img_car, img_mask, padding=20):
    """Обрезает изображение и маску по границам маски."""
    points = cv2.findNonZero(img_mask)
    if points is None:
        return img_car, img_mask

    x, y, w, h = cv2.boundingRect(points)
    h_img, w_img = img_car.shape[:2]

    x_new = max(0, x - padding)
    y_new = max(0, y - padding)
    w_new = min(w_img - x_new, w + 2 * padding)
    h_new = min(h_img - y_new, h + 2 * padding)

    cropped_car = img_car[y_new:y_new + h_new, x_new:x_new + w_new]
    cropped_mask = img_mask[y_new:y_new + h_new, x_new:x_new + w_new]
    return cropped_car, cropped_mask


def rotate_bound(img, angle, interpolation, border_mode, border_value=0):
    h, w = img.shape[:2]
    center = (w / 2.0, h / 2.0)
    matrix = cv2.getRotationMatrix2D(center, angle, 1.0)

    cos = abs(matrix[0, 0])
    sin = abs(matrix[0, 1])
    new_w = int((h * sin) + (w * cos))
    new_h = int((h * cos) + (w * sin))

    matrix[0, 2] += (new_w / 2.0) - center[0]
    matrix[1, 2] += (new_h / 2.0) - center[1]

    return cv2.warpAffine(
        img,
        matrix,
        (new_w, new_h),
        flags=interpolation,
        borderMode=border_mode,
        borderValue=border_value,
    )


def augment_car_rotation(img_car, img_mask):
    if random.random() < 0.5:
        return img_car, img_mask

    angle = random.uniform(-MAX_CAR_ROTATION_DEG, MAX_CAR_ROTATION_DEG)
    rotated_car = rotate_bound(
        img_car,
        angle,
        interpolation=cv2.INTER_LINEAR,
        border_mode=cv2.BORDER_CONSTANT,
        border_value=(0, 0, 0),
    )
    rotated_mask = rotate_bound(
        img_mask,
        angle,
        interpolation=cv2.INTER_NEAREST,
        border_mode=cv2.BORDER_CONSTANT,
        border_value=0,
    )
    return rotated_car, rotated_mask


def augment_background_rotation(img_bg):
    if random.random() < 0.5:
        return img_bg

    angle = random.uniform(-MAX_BG_ROTATION_DEG, MAX_BG_ROTATION_DEG)
    h_bg, w_bg = img_bg.shape[:2]
    center = (w_bg // 2, h_bg // 2)
    matrix = cv2.getRotationMatrix2D(center, angle, 1.0)
    return cv2.warpAffine(img_bg, matrix, (w_bg, h_bg), borderMode=cv2.BORDER_REFLECT)


def crop_with_mask_constraint(img, mask, min_retention=0.85, min_scale=0.9, attempts=40):
    total_car_pixels = np.count_nonzero(mask > 127)
    if total_car_pixels == 0:
        return img, mask

    h, w = img.shape[:2]
    for _ in range(attempts):
        crop_scale = random.uniform(min_scale, 1.0)
        crop_w = max(1, int(w * crop_scale))
        crop_h = max(1, int(h * crop_scale))

        if crop_w >= w or crop_h >= h:
            continue

        x = random.randint(0, w - crop_w)
        y = random.randint(0, h - crop_h)

        cropped_mask = mask[y:y + crop_h, x:x + crop_w]
        kept_ratio = np.count_nonzero(cropped_mask > 127) / total_car_pixels
        if kept_ratio >= min_retention:
            return img[y:y + crop_h, x:x + crop_w], cropped_mask

    return img, mask


def blend_images(img_car, img_mask, bg_path, target_size=224, scale=1.0):
    img_bg = cv2.imread(bg_path)
    if img_bg is None:
        return None

    if random.random() > 0.5:
        img_bg = cv2.flip(img_bg, 1)

    img_bg = augment_background_rotation(img_bg)

    h_car, w_car = img_car.shape[:2]
    square_side = int(max(h_car, w_car) * scale)
    square_side = max(square_side, target_size)

    h_bg_orig, w_bg_orig = img_bg.shape[:2]
    if h_bg_orig < square_side or w_bg_orig < square_side:
        resize_scale = max(square_side / h_bg_orig, square_side / w_bg_orig)
        new_w_bg = int(w_bg_orig * resize_scale * 1.1)
        new_h_bg = int(h_bg_orig * resize_scale * 1.1)
        img_bg = cv2.resize(img_bg, (new_w_bg, new_h_bg))

    if random.random() < 0.5:
        bg_zoom = random.uniform(1.0, 1.35)
        h_bg, w_bg = img_bg.shape[:2]
        img_bg = cv2.resize(img_bg, (int(w_bg * bg_zoom), int(h_bg * bg_zoom)))

    h_bg, w_bg = img_bg.shape[:2]
    min_x = int(w_bg * 0.2)
    max_x = int(w_bg * 0.8) - square_side
    max_y = int(min(h_bg - square_side, h_bg * 0.2))

    bg_x = random.randint(min_x, max_x) if max_x > min_x else 0
    bg_y = random.randint(0, max_y) if max_y > 0 else 0
    img_bg_cropped = img_bg[bg_y:bg_y + square_side, bg_x:bg_x + square_side]

    max_car_x = max(0, square_side - w_car)
    max_car_y = max(0, square_side - h_car)

    car_start_x = random.randint(0, max_car_x) if max_car_x > 0 else 0
    low_bound = int(max_car_y * 0.3)
    if low_bound < max_car_y:
        car_start_y = random.randint(low_bound, max_car_y)
    else:
        car_start_y = max_car_y // 2

    mask_binary = (img_mask > 127).astype(np.uint8) * 255
    mask_float = mask_binary.astype(float) / 255.0
    k = 5 if min(h_car, w_car) > 50 else 1
    mask_blurred = cv2.GaussianBlur(mask_float, (k, k), 0)
    mask_3ch = np.dstack([mask_blurred] * 3)

    roi = img_bg_cropped[car_start_y:car_start_y + h_car, car_start_x:car_start_x + w_car]
    foreground = img_car.astype(float) * mask_3ch
    background = roi.astype(float) * (1.0 - mask_3ch)
    composed_roi = (foreground + background).astype(np.uint8)
    img_bg_cropped[car_start_y:car_start_y + h_car, car_start_x:car_start_x + w_car] = composed_roi

    placed_mask = np.zeros((square_side, square_side), dtype=np.uint8)
    placed_mask[car_start_y:car_start_y + h_car, car_start_x:car_start_x + w_car] = mask_binary

    cropped_img, _ = crop_with_mask_constraint(
        img_bg_cropped,
        placed_mask,
        min_retention=MIN_MASK_RETENTION,
        min_scale=MIN_RANDOM_CROP_SCALE,
    )
    return cv2.resize(cropped_img, (target_size, target_size))


def resize_long_side(image, target_long_side=224):
    h, w = image.shape[:2]
    if h == 0 or w == 0:
        return image

    scale = target_long_side / max(h, w)
    new_w = max(1, int(round(w * scale)))
    new_h = max(1, int(round(h * scale)))
    return cv2.resize(image, (new_w, new_h), interpolation=cv2.INTER_AREA)


def get_unique_save_path(directory, filename):
    base, ext = os.path.splitext(filename)
    candidate = os.path.join(directory, filename)
    index = 1
    while os.path.exists(candidate):
        candidate = os.path.join(directory, f'{base}_{index}{ext}')
        index += 1
    return candidate


if AUGMENTATION_FACTOR != 1:
    raise ValueError('AUGMENTATION_FACTOR поддерживается только со значением 1')

os.makedirs(OUTPUT_DIR, exist_ok=True)
for category in CATEGORIES:
    os.makedirs(os.path.join(OUTPUT_DIR, category), exist_ok=True)

print(f'Папки созданы в: {OUTPUT_DIR}')

car_files = [f for f in os.listdir(CAR_DIR) if f.lower().endswith('.jpg')]
bg_files = [f for f in os.listdir(BG_DIR) if f.lower().endswith(('.jpg', '.png', '.jpeg'))]

print(f'Найдено машин Carvana: {len(car_files)}')
print(f'Найдено фонов: {len(bg_files)}')

meta_rows = []

synthetic_counter = 0
for car_file in tqdm(car_files, desc='Synthetic from Carvana'):
    mask_file = car_file.replace('.jpg', '_mask.gif')
    mask_path = os.path.join(MASK_DIR, mask_file)
    car_path = os.path.join(CAR_DIR, car_file)

    if not os.path.exists(mask_path):
        continue

    img_car = cv2.imread(car_path)
    if img_car is None:
        continue

    pil_mask = Image.open(mask_path).convert('L')
    img_mask = np.array(pil_mask)

    img_car, img_mask = crop_car_content(img_car, img_mask, padding=20)
    img_car, img_mask = augment_car_rotation(img_car, img_mask)

    view_name = get_view_from_filename(car_file)
    if view_name == 'unknown':
        continue

    random_bg = random.choice(bg_files)
    bg_path = os.path.join(BG_DIR, random_bg)

    if view_name in ['left', 'right']:
        scale = random.uniform(1.0, 1.05)
    elif view_name in ['front', 'back']:
        scale = random.uniform(1.15, 1.25)
    else:
        scale = random.uniform(1.05, 1.15)

    result_img = blend_images(img_car, img_mask, bg_path, target_size=SAVE_SIZE, scale=scale)
    if result_img is None:
        continue

    car_base_name = os.path.splitext(car_file)[0]
    save_name = f'synthetic_{car_base_name}.jpg'
    save_dir = os.path.join(OUTPUT_DIR, view_name)
    save_path = get_unique_save_path(save_dir, save_name)
    cv2.imwrite(save_path, result_img)

    meta_rows.append(
        {
            'filename': os.path.basename(save_path),
            'relative_path': os.path.relpath(save_path, OUTPUT_DIR),
            'view': view_name,
            'source': 'synthetic',
            'original_name': car_file,
        }
    )
    synthetic_counter += 1


def iter_hitl_files(hitl_root):
    for root, _, files in os.walk(hitl_root):
        view_name = normalize_view_name(os.path.basename(root))
        if view_name == 'unknown':
            continue

        for file_name in files:
            if file_name.lower().endswith(('.jpg', '.jpeg', '.png')):
                yield view_name, os.path.join(root, file_name)


real_counter = 0
for view_name, source_path in tqdm(list(iter_hitl_files(HITL_DIR)), desc='Real from HITL'):
    img_real = cv2.imread(source_path)
    if img_real is None:
        continue

    img_real = resize_long_side(img_real, target_long_side=SAVE_SIZE)

    source_name = os.path.splitext(os.path.basename(source_path))[0]
    save_name = f'real_{source_name}.jpg'
    save_dir = os.path.join(OUTPUT_DIR, view_name)
    save_path = get_unique_save_path(save_dir, save_name)
    cv2.imwrite(save_path, img_real)

    meta_rows.append(
        {
            'filename': os.path.basename(save_path),
            'relative_path': os.path.relpath(save_path, OUTPUT_DIR),
            'view': view_name,
            'source': 'real',
            'original_name': os.path.basename(source_path),
        }
    )
    real_counter += 1

meta_path = os.path.join(OUTPUT_DIR, META_FILENAME)
with open(meta_path, 'w', newline='', encoding='utf-8') as meta_file:
    writer = csv.DictWriter(
        meta_file,
        fieldnames=['filename', 'relative_path', 'view', 'source', 'original_name'],
    )
    writer.writeheader()
    writer.writerows(meta_rows)

print(f'Сохранено synthetic: {synthetic_counter}')
print(f'Сохранено real: {real_counter}')
print(f'Сохранено meta: {meta_path} (строк: {len(meta_rows)})')
