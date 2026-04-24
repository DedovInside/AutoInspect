from __future__ import annotations

import argparse
import json
import random
import re
import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Sequence, Tuple

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
SEED = 84
VALID_SIDES = {"left", "right"}


@dataclass(frozen=True)
class YoloObject:
    class_name: str
    polygon: Tuple[Tuple[float, float], ...]


@dataclass(frozen=True)
class Sample:
    image_path: Path
    image_rel: str
    width: int
    height: int
    objects: Tuple[YoloObject, ...]


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    project_parts_dir = script_dir.parent
    default_images_dir = project_parts_dir / "images/source"

    parser = argparse.ArgumentParser(
        description="Convert Supervisely parts-segmentation dataset to YOLO format and split train/val/test."
    )
    parser.add_argument(
        "--images-dir",
        type=Path,
        default=default_images_dir,
        help="Path to Supervisely dataset with img/ and ann/ folders (default: ../images/source).",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=None,
        help="Where to save YOLO dataset (default: <images-dir>/yolo).",
    )
    parser.add_argument("--train-ratio", type=float, default=0.8, help="Train split ratio.")
    parser.add_argument("--val-ratio", type=float, default=0.1, help="Validation split ratio.")
    parser.add_argument("--test-ratio", type=float, default=0.1, help="Test split ratio.")
    return parser.parse_args()


def validate_ratios(train_ratio: float, val_ratio: float, test_ratio: float) -> Dict[str, float]:
    ratios = {"train": train_ratio, "val": val_ratio, "test": test_ratio}
    if any(v < 0 for v in ratios.values()):
        raise ValueError("All split ratios must be non-negative.")

    ratio_sum = sum(ratios.values())
    if abs(ratio_sum - 1.0) > 1e-9:
        raise ValueError(f"Split ratios must sum to 1.0, got {ratio_sum}.")

    if all(v == 0 for v in ratios.values()):
        raise ValueError("At least one split ratio must be greater than 0.")

    return ratios


def parse_tag_value(tag: dict) -> str | None:
    value = tag.get("value")
    if isinstance(value, str):
        return value
    if isinstance(value, dict):
        for key in ("value", "label", "classTitle", "title"):
            nested = value.get(key)
            if nested:
                return str(nested)
    if value is not None:
        return str(value)

    for key in ("label", "classTitle", "title"):
        if tag.get(key):
            return str(tag[key])

    return None


def normalize_class_name(class_title: str) -> str:
    cleaned = class_title.strip().lower()
    return re.sub(r"\s+", "-", cleaned)


def extract_side_value(tags: Sequence[object]) -> str | None:
    for tag in tags:
        if not isinstance(tag, dict):
            continue
        if tag.get("name") != "side":
            continue

        value = parse_tag_value(tag)
        if not value:
            continue

        side = value.strip().lower()
        if side in VALID_SIDES:
            return side

    return None


def compose_yolo_class_name(class_title: str, side: str | None) -> str:
    base = normalize_class_name(class_title)
    if side is None:
        return base
    if base.startswith("left-") or base.startswith("right-"):
        return base
    return f"{side}-{base}"


def find_annotation_for_image(image_path: Path, img_dir: Path, ann_dir: Path) -> Path | None:
    rel_from_img = image_path.relative_to(img_dir)
    mirrored = ann_dir / f"{rel_from_img.as_posix()}.json"
    if mirrored.exists():
        return mirrored

    flat = ann_dir / f"{image_path.name}.json"
    if flat.exists():
        return flat

    return None


def parse_polygon_points(raw_points: object) -> Tuple[Tuple[float, float], ...]:
    if not isinstance(raw_points, list):
        return ()

    parsed: List[Tuple[float, float]] = []
    for point in raw_points:
        if not isinstance(point, list) and not isinstance(point, tuple):
            continue
        if len(point) < 2:
            continue
        x, y = point[0], point[1]
        if not isinstance(x, (int, float)) or not isinstance(y, (int, float)):
            continue
        parsed.append((float(x), float(y)))

    return tuple(parsed)


def parse_annotation(annotation_path: Path) -> Tuple[int, int, Tuple[YoloObject, ...], int]:
    with annotation_path.open("r", encoding="utf-8") as file:
        data = json.load(file)

    size = data.get("size", {})
    width = int(size.get("width", 0))
    height = int(size.get("height", 0))
    if width <= 0 or height <= 0:
        raise RuntimeError(f"Invalid image size in annotation: {annotation_path}")

    objects: List[YoloObject] = []
    skipped = 0

    for obj in data.get("objects", []):
        if not isinstance(obj, dict):
            skipped += 1
            continue

        class_title = obj.get("classTitle")
        if not isinstance(class_title, str) or not class_title.strip():
            skipped += 1
            continue

        points = obj.get("points", {})
        exterior = points.get("exterior") if isinstance(points, dict) else None
        polygon = parse_polygon_points(exterior)
        if len(polygon) < 3:
            skipped += 1
            continue

        side = extract_side_value(obj.get("tags", []))
        class_name = compose_yolo_class_name(class_title, side)
        objects.append(YoloObject(class_name=class_name, polygon=polygon))

    return width, height, tuple(objects), skipped


def collect_samples(images_dir: Path) -> Tuple[List[Sample], int]:
    img_dir = images_dir / "img"
    ann_dir = images_dir / "ann"

    if not img_dir.exists() or not img_dir.is_dir():
        raise FileNotFoundError(f"Missing images directory: {img_dir}")
    if not ann_dir.exists() or not ann_dir.is_dir():
        raise FileNotFoundError(f"Missing annotations directory: {ann_dir}")

    image_files = sorted(
        p for p in img_dir.rglob("*") if p.is_file() and p.suffix.lower() in IMAGE_EXTENSIONS
    )
    if not image_files:
        raise RuntimeError(f"No images found in {img_dir}")

    missing_annotations: List[Path] = []
    samples: List[Sample] = []
    skipped_objects_total = 0

    for image_path in image_files:
        annotation_path = find_annotation_for_image(image_path, img_dir, ann_dir)
        if annotation_path is None:
            missing_annotations.append(image_path)
            continue

        width, height, objects, skipped = parse_annotation(annotation_path)
        skipped_objects_total += skipped
        samples.append(
            Sample(
                image_path=image_path,
                image_rel=image_path.relative_to(img_dir).as_posix(),
                width=width,
                height=height,
                objects=objects,
            )
        )

    if missing_annotations:
        preview = "\n".join(f"- {p.as_posix()}" for p in missing_annotations[:10])
        extra = "" if len(missing_annotations) <= 10 else f"\n... and {len(missing_annotations) - 10} more"
        raise RuntimeError(
            "Found images without matching annotation JSON files:\n"
            f"{preview}{extra}\n"
            "Expected ann path patterns: ann/<relative_image_path>.json or ann/<image_name>.json"
        )

    return samples, skipped_objects_total


def compute_split_counts(total: int, ratios: Dict[str, float]) -> Dict[str, int]:
    raw = {name: total * ratio for name, ratio in ratios.items()}
    counts = {name: int(value) for name, value in raw.items()}
    assigned = sum(counts.values())

    remainder = total - assigned
    if remainder > 0:
        order = sorted(raw.keys(), key=lambda k: (raw[k] - counts[k], ratios[k]), reverse=True)
        for i in range(remainder):
            counts[order[i % len(order)]] += 1

    return counts


def assign_splits(samples: List[Sample], ratios: Dict[str, float], seed: int) -> List[Tuple[Sample, str]]:
    rng = random.Random(seed)
    shuffled = list(samples)
    rng.shuffle(shuffled)

    counts = compute_split_counts(len(shuffled), ratios)
    boundaries = {
        "train": counts["train"],
        "val": counts["train"] + counts["val"],
    }

    assignments: List[Tuple[Sample, str]] = []
    for idx, sample in enumerate(shuffled):
        if idx < boundaries["train"]:
            split = "train"
        elif idx < boundaries["val"]:
            split = "val"
        else:
            split = "test"
        assignments.append((sample, split))

    return assignments


def prepare_output_dirs(output_dir: Path) -> None:
    if output_dir.exists():
        shutil.rmtree(output_dir)

    for split in ("train", "val", "test"):
        (output_dir / "images" / split).mkdir(parents=True, exist_ok=True)
        (output_dir / "labels" / split).mkdir(parents=True, exist_ok=True)


def collect_class_names(samples: Sequence[Sample]) -> List[str]:
    classes = {obj.class_name for sample in samples for obj in sample.objects}
    return sorted(classes)


def clamp(value: float, minimum: float, maximum: float) -> float:
    return max(minimum, min(value, maximum))


def polygon_to_yolo_line(class_id: int, polygon: Sequence[Tuple[float, float]], width: int, height: int) -> str:
    coords: List[str] = []
    for x, y in polygon:
        x_norm = clamp(x / width, 0.0, 1.0)
        y_norm = clamp(y / height, 0.0, 1.0)
        coords.append(f"{x_norm:.6f}")
        coords.append(f"{y_norm:.6f}")

    return f"{class_id} " + " ".join(coords)


def write_dataset_yaml(output_dir: Path, class_names: Sequence[str]) -> None:
    lines = [
        "path: .",
        "train: images/train",
        "val: images/val",
        "test: images/test",
        "names:",
    ]
    for idx, name in enumerate(class_names):
        lines.append(f"  {idx}: {json.dumps(name)}")

    (output_dir / "dataset.yaml").write_text("\n".join(lines) + "\n", encoding="utf-8")


def materialize_yolo_dataset(
    assignments: Sequence[Tuple[Sample, str]], output_dir: Path, class_to_id: Dict[str, int]
) -> Dict[str, int]:
    prepare_output_dirs(output_dir)

    split_stats = {"train": 0, "val": 0, "test": 0}
    for sample, split in assignments:
        image_dst = output_dir / "images" / split / sample.image_rel
        label_dst = output_dir / "labels" / split / Path(sample.image_rel).with_suffix(".txt")

        image_dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(sample.image_path, image_dst)

        label_lines = [
            polygon_to_yolo_line(
                class_to_id[obj.class_name],
                obj.polygon,
                sample.width,
                sample.height,
            )
            for obj in sample.objects
        ]

        label_dst.parent.mkdir(parents=True, exist_ok=True)
        label_dst.write_text("\n".join(label_lines) + ("\n" if label_lines else ""), encoding="utf-8")
        split_stats[split] += 1

    return split_stats


def main() -> None:
    # Run: python export_and_split.py --images-dir "..\images\source" --output-dir "..\images\out"
    args = parse_args()
    ratios = validate_ratios(args.train_ratio, args.val_ratio, args.test_ratio)
    output_dir = args.output_dir if args.output_dir is not None else args.images_dir / "yolo"

    samples, skipped_objects = collect_samples(args.images_dir)
    assignments = assign_splits(samples, ratios, SEED)

    class_names = collect_class_names(samples)
    class_to_id = {name: idx for idx, name in enumerate(class_names)}

    split_stats = materialize_yolo_dataset(assignments, output_dir, class_to_id)
    write_dataset_yaml(output_dir, class_names)

    print(f"Created YOLO dataset in: {output_dir}")
    print(f"Total images: {len(assignments)}")
    print(
        "Counts: "
        f"train={split_stats['train']}, "
        f"val={split_stats['val']}, "
        f"test={split_stats['test']}"
    )
    print(f"Classes: {len(class_names)}")
    print(f"Skipped invalid objects: {skipped_objects}")
    print(f"dataset.yaml: {output_dir / 'dataset.yaml'}")


if __name__ == "__main__":
    main()

