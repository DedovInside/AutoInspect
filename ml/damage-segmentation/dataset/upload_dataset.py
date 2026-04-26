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
        description="Validate and upload CarDD YOLO dataset to Hugging Face Hub."
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
        help="Path to YOLO dataset root. Default: ../images/CarDD_YOLO",
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
        help="Allow images without label txt files.",
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

    _assert(images_root.is_dir(), f"Missing directory: {images_root}")
    _assert(labels_root.is_dir(), f"Missing directory: {labels_root}")

    image_splits = {path.name for path in images_root.iterdir() if path.is_dir()}
    label_splits = {path.name for path in labels_root.iterdir() if path.is_dir()}

    splits = sorted(image_splits & label_splits)
    _assert(bool(splits), "No common splits found under images/ and labels/.")

    return splits


def count_label_objects(label_path: Path) -> int:
    if not label_path.exists():
        return 0

    count = 0
    with label_path.open("r", encoding="utf-8") as file:
        for line in file:
            if line.strip():
                count += 1

    return count


def validate_yolo_dataset(
    source_dir: Path,
    splits: List[str],
    allow_missing_labels: bool,
) -> ValidationReport:
    images_root = source_dir / "images"
    labels_root = source_dir / "labels"

    split_image_counts: Dict[str, int] = {}
    split_object_counts: Dict[str, int] = {}

    for split in splits:
        images_dir = images_root / split
        labels_dir = labels_root / split

        image_files = sorted(
            path
            for path in iter_files(images_dir)
            if path.suffix.lower() in IMAGE_EXTENSIONS
        )

        _assert(bool(image_files), f"No images found in split '{split}': {images_dir}")

        image_count = 0
        object_count = 0

        for image_path in image_files:
            image_rel = image_path.relative_to(images_dir)
            label_path = labels_dir / image_rel.with_suffix(".txt")

            if not label_path.exists() and not allow_missing_labels:
                raise ValueError(
                    f"Missing label for image '{split}/{image_rel.as_posix()}'. "
                    f"Expected: {label_path}"
                )

            image_count += 1
            object_count += count_label_objects(label_path)

        split_image_counts[split] = image_count
        split_object_counts[split] = object_count

    total_images = sum(split_image_counts.values())
    total_objects = sum(split_object_counts.values())

    _assert(total_images > 0, "Dataset is empty: no images found.")

    return ValidationReport(
        split_image_counts=split_image_counts,
        split_object_counts=split_object_counts,
        total_images=total_images,
        total_objects=total_objects,
    )


def print_report(source_dir: Path, report: ValidationReport) -> None:
    print(f"Validated YOLO dataset: {source_dir}")
    print("Split statistics:")

    for split in sorted(report.split_image_counts):
        images = report.split_image_counts[split]
        objects = report.split_object_counts[split]
        print(f"  {split}: images={images}, objects={objects}")

    print(f"Total images: {report.total_images}")
    print(f"Total objects: {report.total_objects}")


def upload_yolo_folder(
    repo_id: str,
    source_dir: Path,
    token: str | None,
    revision: str,
    private: bool,
) -> None:
    from huggingface_hub import HfApi

    api = HfApi(token=token)

    api.create_repo(
        repo_id=repo_id,
        repo_type="dataset",
        private=private,
        exist_ok=True,
    )

    api.upload_folder(
        repo_id=repo_id,
        repo_type="dataset",
        folder_path=str(source_dir),
        path_in_repo=".",
        revision=revision,
        commit_message="Upload YOLO dataset",
    )


def main() -> None:
    # Run: python upload_dataset.py --repo-id mitbersh/car-damage-segmentation --source-dir "../images/CarDD_YOLO"

    args = parse_args()
    source_dir = args.source_dir.expanduser().resolve()

    splits = discover_splits(source_dir)
    report = validate_yolo_dataset(
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

    upload_yolo_folder(
        repo_id=args.repo_id,
        source_dir=source_dir,
        token=token,
        revision=args.revision,
        private=args.private,
    )

    print("Upload completed.")


if __name__ == "__main__":
    main()