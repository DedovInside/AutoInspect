from __future__ import annotations

import argparse
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}


@dataclass(frozen=True)
class ValidationReport:
    split_image_counts: Dict[str, int]
    split_object_counts: Dict[str, int]
    total_images: int
    total_objects: int


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    default_source_dir = script_dir.parent / "images" / "CarDD_YOLO"

    parser = argparse.ArgumentParser(
        description="Validate and upload CarDD YOLO dataset to Hugging Face Hub as parquet-backed DatasetDict."
    )
    parser.add_argument(
        "--repo-id",
        required=True,
        help='Hugging Face dataset repo id, for example "username/repo-name".',
    )
    parser.add_argument(
        "--source-dir",
        type=Path,
        default=default_source_dir,
        help="Path to YOLO dataset root (default: ../images/CarDD_YOLO).",
    )
    parser.add_argument(
        "--token",
        default=None,
        help="HF token. If omitted, uses HF_TOKEN/HUGGINGFACE_HUB_TOKEN or local auth cache.",
    )
    parser.add_argument(
        "--revision",
        default="main",
        help="Target branch/tag/commit. Default: main.",
    )
    parser.add_argument(
        "--private",
        action="store_true",
        help="Create repo as private if it does not exist.",
    )
    parser.add_argument(
        "--allow-missing-labels",
        action="store_true",
        help="Allow images without label txt files. They will be exported with empty objects.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Only validate and print stats without uploading.",
    )
    return parser.parse_args()


def _assert(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def resolve_token(cli_token: str | None) -> str | None:
    if cli_token:
        return cli_token

    env_token = os.getenv("HF_TOKEN") or os.getenv("HUGGINGFACE_HUB_TOKEN")
    if env_token:
        return env_token

    try:
        from huggingface_hub import get_token
    except ImportError:
        return None

    return get_token()


def iter_files(root: Path) -> Iterable[Path]:
    for path in root.rglob("*"):
        if path.is_file():
            yield path


def discover_splits(source_dir: Path) -> List[str]:
    images_root = source_dir / "images"
    labels_root = source_dir / "labels"

    _assert(images_root.exists() and images_root.is_dir(), f"Missing directory: {images_root}")
    _assert(labels_root.exists() and labels_root.is_dir(), f"Missing directory: {labels_root}")

    image_splits = {p.name for p in images_root.iterdir() if p.is_dir()}
    label_splits = {p.name for p in labels_root.iterdir() if p.is_dir()}
    common = sorted(image_splits & label_splits)

    _assert(bool(common), "No common splits found under images/ and labels/.")
    return common


def parse_yolo_segment_line(line: str, label_path: Path, line_no: int) -> dict:
    tokens = line.strip().split()
    _assert(len(tokens) >= 7, f"{label_path}:{line_no}: expected at least 7 tokens (class + 3 points), got {len(tokens)}")

    try:
        class_id = int(float(tokens[0]))
    except ValueError as error:
        raise ValueError(f"{label_path}:{line_no}: invalid class id '{tokens[0]}'") from error

    try:
        coords = [float(value) for value in tokens[1:]]
    except ValueError as error:
        raise ValueError(f"{label_path}:{line_no}: polygon coordinates must be numeric") from error

    _assert(len(coords) % 2 == 0, f"{label_path}:{line_no}: expected even number of coordinates, got {len(coords)}")

    polygon = [[coords[i], coords[i + 1]] for i in range(0, len(coords), 2)]
    _assert(len(polygon) >= 3, f"{label_path}:{line_no}: polygon must have at least 3 points")

    return {
        "class_id": class_id,
        "polygon": polygon,
        "raw_line": line.strip(),
    }


def parse_label_file(label_path: Path) -> List[dict]:
    if not label_path.exists():
        return []

    objects: List[dict] = []
    with label_path.open("r", encoding="utf-8") as file:
        for line_no, line in enumerate(file, start=1):
            stripped = line.strip()
            if not stripped:
                continue
            objects.append(parse_yolo_segment_line(stripped, label_path, line_no))

    return objects


def build_records(source_dir: Path, splits: List[str], allow_missing_labels: bool) -> tuple[Dict[str, List[dict]], ValidationReport]:
    images_root = source_dir / "images"
    labels_root = source_dir / "labels"

    records_by_split: Dict[str, List[dict]] = {split: [] for split in splits}
    split_image_counts: Dict[str, int] = {split: 0 for split in splits}
    split_object_counts: Dict[str, int] = {split: 0 for split in splits}

    for split in splits:
        images_dir = images_root / split
        labels_dir = labels_root / split

        image_files = sorted(
            path for path in iter_files(images_dir) if path.suffix.lower() in IMAGE_EXTENSIONS
        )
        _assert(bool(image_files), f"No images found in split '{split}': {images_dir}")

        for image_path in image_files:
            image_rel = image_path.relative_to(images_dir).as_posix()
            label_rel = Path(image_rel).with_suffix(".txt")
            label_path = labels_dir / label_rel

            if not label_path.exists() and not allow_missing_labels:
                raise ValueError(
                    f"Missing label for image '{split}/{image_rel}'. Expected: {label_path}"
                )

            objects = parse_label_file(label_path)
            label_text = "\n".join(obj["raw_line"] for obj in objects)

            records_by_split[split].append(
                {
                    "image": str(image_path),
                    "split": split,
                    "image_rel": image_rel,
                    "label_rel": label_rel.as_posix(),
                    "label_exists": label_path.exists(),
                    "label_text": label_text,
                    "object_count": len(objects),
                    "objects": objects,
                }
            )

            split_image_counts[split] += 1
            split_object_counts[split] += len(objects)

    total_images = sum(split_image_counts.values())
    total_objects = sum(split_object_counts.values())
    _assert(total_images > 0, "Dataset is empty: no images found.")

    report = ValidationReport(
        split_image_counts=split_image_counts,
        split_object_counts=split_object_counts,
        total_images=total_images,
        total_objects=total_objects,
    )
    return records_by_split, report


def print_report(source_dir: Path, report: ValidationReport) -> None:
    print(f"Validated YOLO dataset: {source_dir}")
    print("Split statistics:")
    for split in sorted(report.split_image_counts):
        images = report.split_image_counts[split]
        objects = report.split_object_counts[split]
        print(f"  {split}: images={images}, objects={objects}")
    print(f"Total images: {report.total_images}")
    print(f"Total objects: {report.total_objects}")


def upload_datasetdict(
    repo_id: str,
    records_by_split: Dict[str, List[dict]],
    token: str,
    revision: str,
    private: bool,
) -> None:
    from huggingface_hub import HfApi

    try:
        datasets_module = __import__("datasets", fromlist=["Dataset", "DatasetDict", "Image"])
    except ImportError as error:
        raise RuntimeError(
            "This script requires the 'datasets' package. Install dependencies from dataset/requirements.txt"
        ) from error

    Dataset = datasets_module.Dataset
    DatasetDict = datasets_module.DatasetDict
    Image = datasets_module.Image

    dataset_splits = {}
    for split, records in sorted(records_by_split.items()):
        if not records:
            continue
        split_dataset = Dataset.from_list(records).cast_column("image", Image())
        dataset_splits[split] = split_dataset

    _assert(bool(dataset_splits), "No records to upload after conversion.")

    api = HfApi(token=token)
    api.create_repo(repo_id=repo_id, repo_type="dataset", private=private, exist_ok=True)
    DatasetDict(dataset_splits).push_to_hub(
        repo_id=repo_id,
        token=token,
        private=private,
        revision=revision,
    )


def main() -> None:
    # Run: python upload_dataset.py --repo-id mitbersh/car-damage-segmentation --source-dir "..\images\CarDD_YOLO"
    args = parse_args()
    source_dir = args.source_dir.expanduser().resolve()

    splits = discover_splits(source_dir)
    records_by_split, report = build_records(
        source_dir=source_dir,
        splits=splits,
        allow_missing_labels=args.allow_missing_labels,
    )
    print_report(source_dir, report)

    if args.dry_run:
        print("Dry-run enabled: upload skipped.")
        print(f"Target repo: {args.repo_id} (revision: {args.revision})")
        return

    token = resolve_token(args.token)
    if not token:
        raise RuntimeError(
            "HF token not found. Pass --token or set HF_TOKEN/HUGGINGFACE_HUB_TOKEN."
        )

    upload_datasetdict(
        repo_id=args.repo_id,
        records_by_split=records_by_split,
        token=token,
        revision=args.revision,
        private=args.private,
    )
    print("Upload completed.")


if __name__ == "__main__":
    main()

