from __future__ import annotations

import argparse
import json
import random
import shutil
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, List, Set, Tuple

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
SEED = 84


@dataclass(frozen=True)
class Sample:
    image_rel: str
    annotation_rel: str
    img_info_rel: str
    view: str


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    project_parts_dir = script_dir.parent
    default_images_dir = project_parts_dir / "images/source"

    parser = argparse.ArgumentParser(
        description=(
            "Random and reproducible train/val/test split for Supervisely parts-segmentation dataset."
        )
    )
    parser.add_argument(
        "--images-dir",
        type=Path,
        default=default_images_dir,
        help="Path to directory that contains img/ and ann/ folders (default: ../images).",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=None,
        help="Where to save split folders (default: <images-dir>/out).",
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


def find_annotation_for_image(image_path: Path, img_dir: Path, ann_dir: Path) -> Path | None:
    # Prefer Supervisely-like mirrored layout, fallback to flat ann/ directory.
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
            return value.strip().lower()

        return "unknown"

    return "unknown"


def collect_samples(images_dir: Path) -> List[Sample]:
    img_dir = images_dir / "img"
    ann_dir = images_dir / "ann"
    img_info_dir = images_dir / "img_info"

    if not img_dir.exists() or not img_dir.is_dir():
        raise FileNotFoundError(f"Missing images directory: {img_dir}")
    if not ann_dir.exists() or not ann_dir.is_dir():
        raise FileNotFoundError(f"Missing annotations directory: {ann_dir}")
    if not img_info_dir.exists() or not img_info_dir.is_dir():
        raise FileNotFoundError(f"Missing img_info directory: {img_info_dir}")

    dataset_root = images_dir
    car_view_tag_ids = load_car_view_tag_ids(images_dir / "meta.json")

    image_files = sorted(
        p for p in img_dir.rglob("*") if p.is_file() and p.suffix.lower() in IMAGE_EXTENSIONS
    )
    if not image_files:
        raise RuntimeError(f"No images found in {img_dir}")

    missing_annotations: List[Path] = []
    missing_img_info: List[Path] = []
    samples: List[Sample] = []

    for image_path in image_files:
        annotation_path = find_annotation_for_image(image_path, img_dir, ann_dir)
        if annotation_path is None:
            missing_annotations.append(image_path)
            continue

        img_info_path = find_img_info_for_image(image_path, img_dir, img_info_dir)
        if img_info_path is None:
            missing_img_info.append(image_path)
            continue

        image_rel = image_path.relative_to(dataset_root).as_posix()
        annotation_rel = annotation_path.relative_to(dataset_root).as_posix()
        img_info_rel = img_info_path.relative_to(dataset_root).as_posix()
        view = extract_car_view_from_img_info(img_info_path, car_view_tag_ids)
        samples.append(
            Sample(
                image_rel=image_rel,
                annotation_rel=annotation_rel,
                img_info_rel=img_info_rel,
                view=view,
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

    if missing_img_info:
        preview = "\n".join(f"- {p.as_posix()}" for p in missing_img_info[:10])
        extra = "" if len(missing_img_info) <= 10 else f"\n... and {len(missing_img_info) - 10} more"
        raise RuntimeError(
            "Found images without matching img_info JSON files:\n"
            f"{preview}{extra}\n"
            "Expected img_info path patterns: img_info/<relative_image_path>.json or img_info/<image_name>.json"
        )

    return samples


def compute_split_counts(total: int, ratios: Dict[str, float]) -> Dict[str, int]:
    raw = {name: total * ratio for name, ratio in ratios.items()}
    counts = {name: int(value) for name, value in raw.items()}
    assigned = sum(counts.values())

    # Distribute rounding remainder by largest fractional parts.
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
        for subset in ("img", "ann", "img_info"):
            (output_dir / split / subset).mkdir(parents=True, exist_ok=True)


def copy_if_exists(src: Path, dst: Path) -> bool:
    if not src.exists() or not src.is_file():
        return False

    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dst)
    return True


def materialize_split(assignments: List[Tuple[Sample, str]], images_dir: Path, output_dir: Path) -> Dict[str, int]:
    prepare_output_dirs(output_dir)

    split_stats = {"train": 0, "val": 0, "test": 0}
    for sample, split in assignments:
        image_src = images_dir / sample.image_rel
        ann_src = images_dir / sample.annotation_rel
        img_info_src = images_dir / sample.img_info_rel

        image_dst = output_dir / split / sample.image_rel
        ann_dst = output_dir / split / sample.annotation_rel
        img_info_dst = output_dir / split / sample.img_info_rel

        copy_if_exists(image_src, image_dst)
        copy_if_exists(ann_src, ann_dst)
        copy_if_exists(img_info_src, img_info_dst)
        split_stats[split] += 1

    meta_src = images_dir / "meta.json"
    meta_dst = output_dir / "meta.json"
    copy_if_exists(meta_src, meta_dst)

    return split_stats


def compute_view_distribution(assignments: List[Tuple[Sample, str]]) -> Dict[str, Dict[str, float]]:
    split_counts: Dict[str, int] = {"train": 0, "val": 0, "test": 0}
    view_counts: Dict[str, Dict[str, int]] = {"train": {}, "val": {}, "test": {}}

    for sample, split in assignments:
        split_counts[split] += 1
        view_counts[split][sample.view] = view_counts[split].get(sample.view, 0) + 1

    distribution: Dict[str, Dict[str, float]] = {}
    for split in ("train", "val", "test"):
        total = split_counts[split]
        if total == 0:
            distribution[split] = {}
            continue

        distribution[split] = {
            view: (count / total) * 100.0
            for view, count in sorted(
                view_counts[split].items(), key=lambda item: (-item[1], item[0])
            )
        }

    return distribution


def format_view_distribution(distribution: Dict[str, Dict[str, float]]) -> str:
    lines: List[str] = []
    for split in ("train", "val", "test"):
        split_distribution = distribution[split]
        if not split_distribution:
            lines.append(f"{split}: no samples")
            continue

        parts = [f"{view}={share:.2f}%" for view, share in split_distribution.items()]
        lines.append(f"{split}: " + ", ".join(parts))

    return "\n".join(lines)


def main() -> None:
    # Run: python split.py --images-dir "..\images\source"
    args = parse_args()
    ratios = validate_ratios(args.train_ratio, args.val_ratio, args.test_ratio)
    output_dir = args.output_dir if args.output_dir is not None else args.images_dir / "out"

    samples = collect_samples(args.images_dir)
    assignments = assign_splits(samples, ratios, seed=SEED)
    split_stats = materialize_split(assignments, args.images_dir, output_dir)
    view_distribution = compute_view_distribution(assignments)

    print(f"Created split folders in: {output_dir}")
    print(f"Total samples: {len(assignments)}")
    print(
        "Counts: "
        f"train={split_stats['train']}, "
        f"val={split_stats['val']}, "
        f"test={split_stats['test']}"
    )
    print("View distribution by split (%):")
    print(format_view_distribution(view_distribution))


if __name__ == "__main__":
    main()
