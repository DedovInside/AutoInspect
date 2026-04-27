"""End-to-end инференс AutoInspect."""

from __future__ import annotations

import argparse
import json
from datetime import datetime
from pathlib import Path
from typing import Any

from autoinspect_infer.matcher import build_batch_summary, build_image_result
from autoinspect_infer.yolo_utils import (
    DAMAGE_FILENAME,
    DAMAGE_REPO_ID,
    PARTS_FILENAME,
    PARTS_REPO_ID,
    collect_images,
    load_yolo,
    predict_one_image,
)


# Backend parameters
DEFAULT_MODEL_ID = None
DEFAULT_MODEL_VERSION = None


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run AutoInspect end-to-end inference and export backend JSON."
    )
    parser.add_argument("--source", required=True, help="Path to an image or directory with images.")
    parser.add_argument("--output", default="outputs/autoinspect_predictions.json", help="Path to output JSON.")

    parser.add_argument("--parts-model", default=None, help="Local parts YOLO .pt weights. If omitted, downloads from HF.")
    parser.add_argument("--damage-model", default=None, help="Local damage YOLO .pt weights. If omitted, downloads from HF.")
    parser.add_argument("--parts-repo-id", default=PARTS_REPO_ID)
    parser.add_argument("--damage-repo-id", default=DAMAGE_REPO_ID)
    parser.add_argument("--parts-filename", default=PARTS_FILENAME)
    parser.add_argument("--damage-filename", default=DAMAGE_FILENAME)

    parser.add_argument("--model-id", default=DEFAULT_MODEL_ID)
    parser.add_argument("--model-version", default=DEFAULT_MODEL_VERSION)
    parser.add_argument("--batch-id", default=None, help="External batch id. Auto-generated if omitted.")
    parser.add_argument("--image-uri-prefix", default=None, help="Optional prefix for image_uri, e.g. s3://bucket/path.")

    parser.add_argument("--parts-imgsz", type=int, default=768)
    parser.add_argument("--damage-imgsz", type=int, default=896)
    parser.add_argument("--parts-conf", type=float, default=0.25)
    parser.add_argument("--damage-conf", type=float, default=0.25)
    parser.add_argument("--iou", type=float, default=0.70)
    parser.add_argument("--device", default="auto", help="auto, cpu, 0, 1, 0,1, etc.")
    parser.add_argument("--max-det", type=int, default=300)
    parser.add_argument("--retina-masks", action="store_true", help="Use high-resolution masks from Ultralytics.")

    parser.add_argument("--min-overlap", type=float, default=0.05, help="Minimum share of damage mask inside a part mask.")
    parser.add_argument("--min-assignment-score", type=float, default=0.05, help="Minimum final part assignment confidence.")
    parser.add_argument("--max-parts-per-damage", type=int, default=3)

    return parser.parse_args()


def make_batch_id() -> str:
    return "batch_" + datetime.now().strftime("%Y_%m_%d_%H%M%S")


def make_image_id(index: int) -> str:
    return f"image_{index}"


def make_image_uri(image_path: Path, image_uri_prefix: str | None) -> str:
    if not image_uri_prefix:
        return str(image_path)
    return f"{image_uri_prefix.rstrip('/')}/{image_path.name}"


def main() -> None:
    args = parse_args()
    device = None if args.device == "auto" else args.device

    image_paths = collect_images(args.source)
    if not image_paths:
        raise ValueError(f"No images found in source: {args.source}")

    parts_model = load_yolo(args.parts_model, repo_id=args.parts_repo_id, filename=args.parts_filename)
    damage_model = load_yolo(args.damage_model, repo_id=args.damage_repo_id, filename=args.damage_filename)

    image_results: list[dict[str, Any]] = []
    for index, image_path in enumerate(image_paths, start=1):
        image_id = make_image_id(index)
        image_uri = make_image_uri(image_path, args.image_uri_prefix)

        parts_predictions = predict_one_image(
            model=parts_model,
            image_path=image_path,
            imgsz=args.parts_imgsz,
            conf=args.parts_conf,
            iou=args.iou,
            device=device,
            retina_masks=args.retina_masks,
            max_det=args.max_det,
        )
        damage_predictions = predict_one_image(
            model=damage_model,
            image_path=image_path,
            imgsz=args.damage_imgsz,
            conf=args.damage_conf,
            iou=args.iou,
            device=device,
            retina_masks=args.retina_masks,
            max_det=args.max_det,
        )

        image_results.append(
            build_image_result(
                image_id=image_id,
                image_uri=image_uri,
                parts_predictions=parts_predictions,
                damage_predictions=damage_predictions,
                min_overlap=args.min_overlap,
                min_assignment_score=args.min_assignment_score,
                max_parts_per_damage=args.max_parts_per_damage,
            )
        )

    response = {
        "model_id": args.model_id,
        "model_version": args.model_version,
        "batch_id": args.batch_id or make_batch_id(),
        "results": image_results,
        "batch_summary": build_batch_summary(image_results),
    }

    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    with output_path.open("w", encoding="utf-8") as f:
        json.dump(response, f, ensure_ascii=False, indent=2)

    print(f"Saved AutoInspect JSON to: {output_path}")


if __name__ == "__main__":
    main()
