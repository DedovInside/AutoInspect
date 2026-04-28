#!/usr/bin/env python3
"""Inference script for AutoInspect Car Damage Segmentation.

Example:
    python infer_damage.py \
        --source "path/to/car.jpg" \
        --model "car_damage_model.pt" \
        --imgsz 896 \
        --conf 0.25 \
        --iou 0.70 \
        --device auto \
        --save \
        --json "outputs/damage_predictions.json"
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any
from sympy.printing.pretty.pretty_symbology import line_width

from huggingface_hub import hf_hub_download
from ultralytics import YOLO


DEFAULT_REPO_ID = "mitbersh/car-damage-segmentation"
DEFAULT_MODEL_FILENAME = "car_damage_model.pt"
DEFAULT_PROJECT = "runs/damage_infer"
DEFAULT_NAME = "predict"


# Fallback only. In normal inference, class names are read from the YOLO weights.
CLASS_NAMES = [
    "crack",
    "dent",
    "glass_shatter",
    "lamp_broken",
    "scratch",
    "tire_flat",
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run AutoInspect car damage segmentation inference."
    )
    parser.add_argument(
        "--source",
        required=True,
        help="Path to an image, directory, video, or any Ultralytics-supported source.",
    )
    parser.add_argument(
        "--model",
        default=None,
        help=(
            "Path to local YOLO weights. If omitted, the script downloads "
            "car_damage_model.pt from Hugging Face."
        ),
    )
    parser.add_argument(
        "--repo-id",
        default=DEFAULT_REPO_ID,
        help="Hugging Face repo id used when --model is not provided.",
    )
    parser.add_argument(
        "--filename",
        default=DEFAULT_MODEL_FILENAME,
        help="Model filename in the Hugging Face repo used when --model is not provided.",
    )
    parser.add_argument("--imgsz", type=int, default=896, help="Inference image size.")
    parser.add_argument("--conf", type=float, default=0.25, help="Confidence threshold.")
    parser.add_argument("--iou", type=float, default=0.70, help="NMS IoU threshold.")
    parser.add_argument(
        "--device",
        default="auto",
        help="Device for inference: auto, cpu, 0, 1, 0,1, etc.",
    )
    parser.add_argument(
        "--project",
        default=DEFAULT_PROJECT,
        help="Directory for Ultralytics visual outputs.",
    )
    parser.add_argument(
        "--name",
        default=DEFAULT_NAME,
        help="Run name inside --project.",
    )
    parser.add_argument(
        "--json",
        default=None,
        help=(
            "Path to JSON output. If omitted, predictions are saved to "
            "<project>/<name>/predictions.json."
        ),
    )
    parser.add_argument(
        "--save",
        action="store_true",
        help="Save visualization images with predicted masks.",
    )
    parser.add_argument(
        "--save-txt",
        action="store_true",
        help="Also save Ultralytics txt labels.",
    )
    parser.add_argument(
        "--save-conf",
        action="store_true",
        help="Include confidences in Ultralytics txt labels when --save-txt is used.",
    )
    parser.add_argument(
        "--retina-masks",
        action="store_true",
        help="Use high-resolution segmentation masks in Ultralytics prediction.",
    )
    parser.add_argument(
        "--max-det",
        type=int,
        default=300,
        help="Maximum detections per image.",
    )
    parser.add_argument(
        "--exist-ok",
        action="store_true",
        help="Allow writing into an existing Ultralytics output directory.",
    )
    return parser.parse_args()


def resolve_weights(args: argparse.Namespace) -> str:
    if args.model:
        model_path = Path(args.model)
        if not model_path.exists():
            raise FileNotFoundError(f"Model weights not found: {model_path}")
        return str(model_path)

    return hf_hub_download(
        repo_id=args.repo_id,
        filename=args.filename,
        local_dir=".",
    )


def to_float_list(values: Any, ndigits: int = 6) -> list[float]:
    return [round(float(v), ndigits) for v in values]


def polygon_to_list(poly: Any, ndigits: int = 6) -> list[list[float]]:
    if poly is None:
        return []
    return [[round(float(x), ndigits), round(float(y), ndigits)] for x, y in poly]


def get_mask_area_px(masks: Any, idx: int) -> int | None:
    """Return mask area in pixels when raster masks are available."""
    if masks is None or getattr(masks, "data", None) is None:
        return None
    mask_tensor = masks.data[idx]
    return int(mask_tensor.detach().cpu().sum().item())


def result_to_record(result: Any) -> dict[str, Any]:
    height, width = result.orig_shape
    names = getattr(result, "names", None) or {i: name for i, name in enumerate(CLASS_NAMES)}

    record: dict[str, Any] = {
        "image": Path(result.path).name if result.path else None,
        "path": str(result.path) if result.path else None,
        "width": int(width),
        "height": int(height),
        "detections": [],
    }

    boxes = result.boxes
    masks = result.masks

    if boxes is None or len(boxes) == 0:
        return record

    class_ids = boxes.cls.detach().cpu().numpy().astype(int).tolist()
    confidences = boxes.conf.detach().cpu().numpy().tolist()
    xyxy_boxes = boxes.xyxy.detach().cpu().numpy().tolist()
    xywh_boxes = boxes.xywh.detach().cpu().numpy().tolist()

    mask_xy = masks.xy if masks is not None else [None] * len(class_ids)
    mask_xyn = masks.xyn if masks is not None else [None] * len(class_ids)

    for idx, class_id in enumerate(class_ids):
        fallback_name = CLASS_NAMES[class_id] if class_id < len(CLASS_NAMES) else str(class_id)
        class_name = names.get(class_id, fallback_name)

        detection = {
            "damage_id": idx,
            "class_id": int(class_id),
            "class_name": class_name,
            "confidence": round(float(confidences[idx]), 6),
            "bbox_xyxy": to_float_list(xyxy_boxes[idx]),
            "bbox_xywh": to_float_list(xywh_boxes[idx]),
            "mask_area_px": get_mask_area_px(masks, idx),
            "mask_polygon_xy": polygon_to_list(mask_xy[idx]),
            "mask_polygon_xyn": polygon_to_list(mask_xyn[idx]),
        }
        record["detections"].append(detection)

    return record


def main() -> None:
    args = parse_args()
    weights_path = resolve_weights(args)
    device = None if args.device == "auto" else args.device

    project_dir = Path(args.project)
    output_dir = project_dir / args.name
    output_dir.mkdir(parents=True, exist_ok=True)

    json_path = Path(args.json) if args.json else output_dir / "predictions.json"
    json_path.parent.mkdir(parents=True, exist_ok=True)

    model = YOLO(weights_path)

    results = model.predict(
        source=args.source,
        imgsz=args.imgsz,
        conf=args.conf,
        iou=args.iou,
        device=device,
        save=args.save,
        save_txt=args.save_txt,
        save_conf=args.save_conf,
        retina_masks=args.retina_masks,
        max_det=args.max_det,
        project=str(project_dir),
        name=args.name,
        exist_ok=args.exist_ok,
        verbose=True,
        line_width=1
    )

    records = [result_to_record(result) for result in results]

    with json_path.open("w", encoding="utf-8") as f:
        json.dump(records, f, ensure_ascii=False, indent=2)

    print(f"Saved JSON predictions to: {json_path}")
    if args.save:
        print(f"Saved visual predictions to: {output_dir}")


if __name__ == "__main__":
    main()
