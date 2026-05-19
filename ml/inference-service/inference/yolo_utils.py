from __future__ import annotations

import re
from pathlib import Path
from typing import Any

import numpy as np
from huggingface_hub import hf_hub_download
from ultralytics import YOLO

from .schema import ImagePredictions, InstancePrediction

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}

SIDE_PREFIX_RE = re.compile(r"^(left|right)_(.+)$")


PARTS_REPO_ID = "mitbersh/car-parts-segmentation"
PARTS_FILENAME = "parts_segmentation.pt"
DAMAGE_REPO_ID = "mitbersh/car-damage-segmentation"
DAMAGE_FILENAME = "damage_segmentation.pt"


def resolve_weights(model_path: str | None, repo_id: str, filename: str) -> str:
    """Берет веса из локального пути или скачивает из Hugging Face Hub"""
    if model_path:
        path = Path(model_path)
        if not path.exists():
            raise FileNotFoundError(f"Model weights not found: {path}")
        return str(path)

    return hf_hub_download(repo_id=repo_id, filename=filename, local_dir=".")


def load_yolo(model_path: str | None, repo_id: str, filename: str) -> YOLO:
    weights_path = resolve_weights(model_path, repo_id=repo_id, filename=filename)
    return YOLO(weights_path)


def collect_images(source: str) -> list[Path]:
    path = Path(source)
    if path.is_file():
        return [path]
    if path.is_dir():
        return sorted(p for p in path.rglob("*") if p.suffix.lower() in IMAGE_EXTENSIONS)
    raise FileNotFoundError(f"Source not found or unsupported: {source}")


def normalize_name(value: str) -> str:
    return value.strip().lower().replace(" ", "-")


def split_part_name(raw_name: str) -> tuple[str, str | None]:
    """left_front-door > (front-door, left)"""
    name = normalize_name(raw_name)
    match = SIDE_PREFIX_RE.match(name)
    if not match:
        return name, None
    side, base_name = match.groups()
    return base_name, side


def to_int_bbox(xyxy: list[float]) -> list[int]:
    x1, y1, x2, y2 = xyxy
    return [int(round(x1)), int(round(y1)), int(round(x2)), int(round(y2))]


def polygon_to_int_list(poly: Any) -> list[list[int]]:
    if poly is None:
        return []
    return [[int(round(float(x))), int(round(float(y)))] for x, y in poly]


def resize_mask_to_image(mask: np.ndarray, height: int, width: int) -> np.ndarray:
    import cv2

    mask = np.asarray(mask)
    if mask.shape[:2] != (height, width):
        mask = cv2.resize(mask.astype("uint8"), (width, height), interpolation=cv2.INTER_NEAREST)
    return mask.astype(bool)


def result_to_predictions(result: Any) -> ImagePredictions:
    height, width = result.orig_shape
    names = getattr(result, "names", {}) or {}
    boxes = result.boxes
    masks = result.masks

    image_path = Path(result.path) if getattr(result, "path", None) else None
    predictions: list[InstancePrediction] = []

    if boxes is None or len(boxes) == 0:
        return ImagePredictions(image_path=image_path, width=int(width), height=int(height), instances=[])

    class_ids = boxes.cls.detach().cpu().numpy().astype(int).tolist()
    confidences = boxes.conf.detach().cpu().numpy().tolist()
    xyxy_boxes = boxes.xyxy.detach().cpu().numpy().tolist()

    mask_polygons = masks.xy if masks is not None else [None] * len(class_ids)
    mask_data = masks.data.detach().cpu().numpy() if masks is not None and getattr(masks, "data", None) is not None else None

    for idx, class_id in enumerate(class_ids):
        raw_name = names.get(class_id, str(class_id))
        mask = None
        if mask_data is not None:
            mask = resize_mask_to_image(mask_data[idx], int(height), int(width))

        predictions.append(
            InstancePrediction(
                class_id=int(class_id),
                class_name=normalize_name(str(raw_name)),
                confidence=round(float(confidences[idx]), 6),
                bbox_xyxy=to_int_bbox(xyxy_boxes[idx]),
                polygon_xy=polygon_to_int_list(mask_polygons[idx]),
                mask=mask,
            )
        )

    return ImagePredictions(image_path=image_path, width=int(width), height=int(height), instances=predictions)


def predict_one_image(model: YOLO, image_path: Path, imgsz: int, conf: float, iou: float, device: str | None, retina_masks: bool, max_det: int) -> ImagePredictions:
    results = model.predict(
        source=str(image_path),
        imgsz=imgsz,
        conf=conf,
        iou=iou,
        device=device,
        retina_masks=retina_masks,
        max_det=max_det,
        save=False,
        verbose=False,
    )
    return result_to_predictions(results[0])
