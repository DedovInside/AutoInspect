from __future__ import annotations

import argparse
from collections import Counter, defaultdict
import json
import random
import re
import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, List, Sequence, Set, Tuple

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
SEED = 84
MISSING_VIEW = "__missing_view__"
SPLITS = ("train", "val", "test")
VALID_SIDES = {"left", "right"}
SIDE_REQUIRED_CLASS_TITLES = {
    "Headlight",
    "Tail-light",
    "Mirror",
    "Front-window",
    "Back-window",
    "Front-door",
    "Back-door",
    "Front-wheel",
    "Back-wheel",
    "Fender",
    "Quarter-panel",
    "Rocker-panel",
}
SIDE_REQUIRED_CLASSES = {re.sub(r"\s+", "-", title.strip().lower()) for title in SIDE_REQUIRED_CLASS_TITLES}
FEATURE_WEIGHTS = {
    "class": 4.0,
    "base_class": 3.0,
    "side_class": 2.0,
    "side": 1.0,
}


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
    view: str
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
    parser.add_argument("--seed", type=int, default=SEED, help=f"Random seed (default: {SEED}).")
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


def normalize_view_name(view: str) -> str:
    cleaned = view.strip().lower()
    cleaned = re.sub(r"\s+", "-", cleaned)
    return cleaned if cleaned else MISSING_VIEW


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

    if side is None and base in SIDE_REQUIRED_CLASSES:
        raise Exception(
            f"Class '{class_title}' requires side tag with left/right, but tag is missing."
        )

    if side is not None and base not in SIDE_REQUIRED_CLASSES:
        raise Exception(
            f"Class '{class_title}' has side='{side}', but side is allowed only for: {sorted(SIDE_REQUIRED_CLASSES)}"
        )

    if side is None:
        return base
    if base.startswith("left_") or base.startswith("right_"):
        return base
    return f"{side}_{base}"


def find_annotation_for_image(image_path: Path, img_dir: Path, ann_dir: Path) -> Path | None:
    rel_from_img = image_path.relative_to(img_dir)
    mirrored = ann_dir / f"{rel_from_img.as_posix()}.json"
    if mirrored.exists():
        return mirrored

    flat = ann_dir / f"{image_path.name}.json"
    if flat.exists():
        return flat

    return None


def find_img_info_for_image(image_path: Path, img_dir: Path, img_info_dir: Path) -> Path | None:
    rel_from_img = image_path.relative_to(img_dir)
    candidates = [
        img_info_dir / f"{rel_from_img.as_posix()}.json",
        img_info_dir / rel_from_img,
        img_info_dir / f"{image_path.name}.json",
        img_info_dir / f"{image_path.stem}.json",
    ]

    for candidate in candidates:
        if candidate.exists() and candidate.is_file():
            return candidate

    return None


def load_car_view_tag_ids(meta_path: Path) -> Set[str]:
    if not meta_path.exists() or not meta_path.is_file():
        return set()

    with meta_path.open("r", encoding="utf-8") as file:
        data = json.load(file)

    tag_ids: Set[str] = set()
    for tag in data.get("tags", []):
        if not isinstance(tag, dict):
            continue
        if tag.get("name") != "car_view":
            continue
        if tag.get("id") is not None:
            tag_ids.add(str(tag["id"]))

    return tag_ids


def extract_car_view_from_img_info(img_info_path: Path, car_view_tag_ids: Set[str]) -> str:
    with img_info_path.open("r", encoding="utf-8") as file:
        data = json.load(file)

    tags = data.get("tags", [])
    for tag in tags:
        if not isinstance(tag, dict):
            continue

        tag_name = tag.get("name")
        tag_id = tag.get("tagId") if tag.get("tagId") is not None else tag.get("id")
        is_car_view = tag_name == "car_view"
        if not is_car_view and tag_id is not None and str(tag_id) in car_view_tag_ids:
            is_car_view = True
        if not is_car_view:
            continue

        value = parse_tag_value(tag)
        if value:
            return normalize_view_name(value)

        return MISSING_VIEW

    return MISSING_VIEW


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


def extract_view_value(tags: Sequence[object]) -> str | None:
    for tag in tags:
        if not isinstance(tag, dict):
            continue

        tag_name = str(tag.get("name", "")).strip().lower()
        if tag_name not in {"view", "car_view"}:
            continue

        value = parse_tag_value(tag)
        if value:
            return normalize_view_name(value)

    return None


def parse_annotation(annotation_path: Path) -> Tuple[int, int, str, Tuple[YoloObject, ...], int]:
    with annotation_path.open("r", encoding="utf-8") as file:
        data = json.load(file)

    size = data.get("size", {})
    width = int(size.get("width", 0))
    height = int(size.get("height", 0))
    if width <= 0 or height <= 0:
        raise RuntimeError(f"Invalid image size in annotation: {annotation_path}")

    objects: List[YoloObject] = []
    skipped = 0
    view = extract_view_value(data.get("tags", []))

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

    return width, height, view if view else MISSING_VIEW, tuple(objects), skipped


def collect_samples(images_dir: Path) -> Tuple[List[Sample], int]:
    img_dir = images_dir / "img"
    ann_dir = images_dir / "ann"
    img_info_dir = images_dir / "img_info"

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
    car_view_tag_ids = load_car_view_tag_ids(images_dir / "meta.json")
    has_img_info_dir = img_info_dir.exists() and img_info_dir.is_dir()

    for image_path in image_files:
        annotation_path = find_annotation_for_image(image_path, img_dir, ann_dir)
        if annotation_path is None:
            missing_annotations.append(image_path)
            continue

        width, height, ann_view, objects, skipped = parse_annotation(annotation_path)
        view = ann_view
        if view == MISSING_VIEW and has_img_info_dir:
            img_info_path = find_img_info_for_image(image_path, img_dir, img_info_dir)
            if img_info_path is not None:
                view = extract_car_view_from_img_info(img_info_path, car_view_tag_ids)

        skipped_objects_total += skipped
        samples.append(
            Sample(
                image_path=image_path,
                image_rel=image_path.relative_to(img_dir).as_posix(),
                width=width,
                height=height,
                view=view if view else MISSING_VIEW,
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


def split_class_name(class_name: str) -> Tuple[str, str]:
    if class_name.startswith("left_"):
        return class_name[5:], "left"
    if class_name.startswith("right_"):
        return class_name[6:], "right"
    return class_name, "neutral"


def make_feature_key(family: str, value: str) -> str:
    return f"{family}|{value}"


def split_feature_key(feature_key: str) -> Tuple[str, str]:
    family, _, value = feature_key.partition("|")
    return family, value


def sample_feature_counts(sample: Sample) -> Counter[str]:
    counts: Counter[str] = Counter()
    for obj in sample.objects:
        base_class, side = split_class_name(obj.class_name)
        counts[make_feature_key("class", obj.class_name)] += 1
        counts[make_feature_key("base_class", base_class)] += 1
        counts[make_feature_key("side", side)] += 1
        counts[make_feature_key("side_class", f"{side}:{base_class}")] += 1
    return counts


def assign_view_bucket(
    bucket_samples: Sequence[Sample], ratios: Dict[str, float], rng: random.Random
) -> List[Tuple[Sample, str]]:
    if not bucket_samples:
        return []

    target_images = compute_split_counts(len(bucket_samples), ratios)
    split_share = {
        split: (target_images[split] / len(bucket_samples)) if bucket_samples else 0.0 for split in SPLITS
    }

    per_sample_features = {sample: sample_feature_counts(sample) for sample in bucket_samples}
    total_features: Counter[str] = Counter()
    for counts in per_sample_features.values():
        total_features.update(counts)

    target_features: Dict[str, Dict[str, float]] = {split: {} for split in SPLITS}
    for split in SPLITS:
        for feature_key, total in total_features.items():
            target_features[split][feature_key] = total * split_share[split]

    shuffled = list(bucket_samples)
    rng.shuffle(shuffled)

    def rarity_score(sample: Sample) -> Tuple[float, int]:
        score = 0.0
        for feature_key, value in per_sample_features[sample].items():
            family, _ = split_feature_key(feature_key)
            total = total_features[feature_key]
            if total <= 0:
                continue
            score += FEATURE_WEIGHTS.get(family, 1.0) * (value / total)
        return score, len(sample.objects)

    ordered = sorted(shuffled, key=rarity_score, reverse=True)

    assigned_images = {split: 0 for split in SPLITS}
    assigned_features: Dict[str, Counter[str]] = {split: Counter() for split in SPLITS}
    assignments: List[Tuple[Sample, str]] = []

    for sample in ordered:
        candidates = [split for split in SPLITS if assigned_images[split] < target_images[split]]
        if not candidates:
            raise RuntimeError("No split has capacity while assigning view bucket.")

        if len(candidates) == 1:
            chosen_split = candidates[0]
        else:
            best_score: Tuple[float, float, float, float, float] | None = None
            chosen_split = candidates[0]
            sample_counts = per_sample_features[sample]

            for split in candidates:
                overflow_penalty = 0.0
                deficit_gain = 0.0
                abs_gap = 0.0

                for feature_key, count in sample_counts.items():
                    family, _ = split_feature_key(feature_key)
                    weight = FEATURE_WEIGHTS.get(family, 1.0)
                    target = target_features[split].get(feature_key, 0.0)
                    current = assigned_features[split][feature_key]
                    after = current + count

                    overflow_penalty += max(0.0, after - target) * weight
                    deficit_before = max(0.0, target - current)
                    deficit_after = max(0.0, target - after)
                    deficit_gain += (deficit_before - deficit_after) * weight
                    abs_gap += abs(target - after) * weight

                target_img = max(1, target_images[split])
                fill_ratio = (assigned_images[split] + 1) / target_img
                score = (overflow_penalty, -deficit_gain, fill_ratio, abs_gap, rng.random())

                if best_score is None or score < best_score:
                    best_score = score
                    chosen_split = split

        assignments.append((sample, chosen_split))
        assigned_images[chosen_split] += 1
        assigned_features[chosen_split].update(per_sample_features[sample])

    return assignments


def assign_splits(samples: List[Sample], ratios: Dict[str, float], seed: int) -> List[Tuple[Sample, str]]:
    by_view: Dict[str, List[Sample]] = defaultdict(list)
    for sample in samples:
        by_view[sample.view].append(sample)

    rng = random.Random(seed)
    assignments: List[Tuple[Sample, str]] = []
    for view in sorted(by_view.keys()):
        view_rng = random.Random(rng.randint(0, 2**31 - 1))
        assignments.extend(assign_view_bucket(by_view[view], ratios, view_rng))

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


def empty_counter_block() -> Dict[str, Any]:
    return {
        "image_count": 0,
        "object_count": 0,
        "class_counts": Counter(),
        "base_class_counts": Counter(),
        "side_counts": Counter(),
        "side_class_counts": Counter(),
        "view_counts": Counter(),
    }


def update_counter_block(block: Dict[str, Any], sample: Sample) -> None:
    block["image_count"] = int(block["image_count"]) + 1
    block["object_count"] = int(block["object_count"]) + len(sample.objects)
    block["view_counts"][sample.view] += 1

    for obj in sample.objects:
        base_class, side = split_class_name(obj.class_name)
        block["class_counts"][obj.class_name] += 1
        block["base_class_counts"][base_class] += 1
        block["side_counts"][side] += 1
        block["side_class_counts"][f"{side}:{base_class}"] += 1


def serialize_counter_block(block: Dict[str, Any]) -> Dict[str, Any]:
    return {
        "image_count": int(block["image_count"]),
        "object_count": int(block["object_count"]),
        "class_counts": dict(sorted(block["class_counts"].items())),
        "base_class_counts": dict(sorted(block["base_class_counts"].items())),
        "side_counts": dict(sorted(block["side_counts"].items())),
        "side_class_counts": dict(sorted(block["side_class_counts"].items())),
        "view_counts": dict(sorted(block["view_counts"].items())),
    }


def collect_rare_warnings(total_block: Dict[str, Any], active_split_count: int) -> List[str]:
    warnings: List[str] = []

    categories = {
        "view": total_block["view_counts"],
        "class": total_block["class_counts"],
        "base_class": total_block["base_class_counts"],
        "side": total_block["side_counts"],
        "side_class": total_block["side_class_counts"],
    }

    for category, counts in categories.items():
        for label, count in sorted(counts.items()):
            if count < active_split_count:
                warnings.append(
                    f"{category} '{label}' has count={count}, which is too rare to cover all {active_split_count} active splits"
                )

    return warnings


def build_split_report(
    assignments: Sequence[Tuple[Sample, str]], ratios: Dict[str, float], seed: int
) -> Dict[str, Any]:
    per_split_raw: Dict[str, Dict[str, Any]] = {split: empty_counter_block() for split in SPLITS}
    total_raw = empty_counter_block()

    for sample, split in assignments:
        update_counter_block(per_split_raw[split], sample)
        update_counter_block(total_raw, sample)

    active_split_count = sum(1 for value in ratios.values() if value > 0)

    return {
        "seed": seed,
        "ratios": ratios,
        "splits": {split: serialize_counter_block(per_split_raw[split]) for split in SPLITS},
        "totals": serialize_counter_block(total_raw),
        "warnings": collect_rare_warnings(total_raw, active_split_count),
    }


def write_split_report(output_dir: Path, report: Dict[str, Any]) -> Path:
    report_path = output_dir / "split_report.json"
    report_path.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    return report_path


def main() -> None:
    # Run: python export_and_split.py --images-dir "..\..\images\source" --output-dir "..\..\images\out"
    args = parse_args()
    ratios = validate_ratios(args.train_ratio, args.val_ratio, args.test_ratio)
    output_dir = args.output_dir if args.output_dir is not None else args.images_dir / "yolo"

    samples, skipped_objects = collect_samples(args.images_dir)
    assignments = assign_splits(samples, ratios, args.seed)

    class_names = collect_class_names(samples)
    class_to_id = {name: idx for idx, name in enumerate(class_names)}

    split_stats = materialize_yolo_dataset(assignments, output_dir, class_to_id)
    write_dataset_yaml(output_dir, class_names)
    split_report_path = write_split_report(output_dir, build_split_report(assignments, ratios, args.seed))

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
    print(f"split_report.json: {split_report_path}")


if __name__ == "__main__":
    main()

