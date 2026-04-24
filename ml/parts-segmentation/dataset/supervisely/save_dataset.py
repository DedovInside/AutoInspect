from __future__ import annotations

import argparse
import csv
import json
from collections import Counter, defaultdict
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Set

REQUIRED_COLUMNS = {"image", "annotation", "view", "split"}
ALLOWED_SPLITS = {"train", "val", "test"}
IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}


@dataclass
class DatasetStats:
    split_counter: Counter
    split_view_counter: Dict[str, Counter]
    sample_count: int
    dataset_format: str


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    default_images_dir = script_dir.parent / "images" / "out"

    parser = argparse.ArgumentParser(
        description="Validate and upload parts-segmentation dataset folder to Hugging Face Hub."
    )
    parser.add_argument(
        "--repo-id",
        required=True,
        help='Hugging Face dataset repo id, for example "username/repo-name".',
    )
    parser.add_argument(
        "--images-dir",
        type=Path,
        default=default_images_dir,
        help="Path to local dataset folder (default: ../images/out).",
    )
    parser.add_argument(
        "--split-csv",
        type=Path,
        default=None,
        help="Optional path to split.csv for legacy format (img/, ann/, split.csv).",
    )
    parser.add_argument(
        "--token",
        default=None,
        help="HF token. If omitted, huggingface_hub will use local auth cache/env vars.",
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
        "--num-workers",
        type=int,
        default=None,
        help="Number of parallel upload workers for upload_large_folder().",
    )
    parser.add_argument(
        "--upload-mode",
        choices=("datasetdict", "raw-folder"),
        default="datasetdict",
        help="Upload mode. datasetdict gives reliable image previews in Data Studio.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Only validate and print plan without uploading.",
    )
    parser.add_argument(
        "--print-format",
        action="store_true",
        help="Print expected dataset format and exit.",
    )
    return parser.parse_args()


def print_format_guide() -> None:
    print("Expected folder format (new):")
    print("  images/out/")
    for split in sorted(ALLOWED_SPLITS):
        print(f"    {split}/img/<image files>")
        print(f"    {split}/ann/<matching annotation json files>")
        print(f"    {split}/img_info/<matching img_info json files>")
    print("    meta.json")
    print()
    print("Legacy format is also supported:")
    print("  images/")
    print("    img/<image files>")
    print("    ann/<matching annotation json files>")
    print("    split.csv")
    print()
    print("Legacy split.csv columns:")
    print("  image,annotation,view,split")
    print("Example row:")
    print("  img/sample.png,ann/sample.png.json,front-left,train")
    print()
    print("Upload modes:")
    print("  datasetdict (default): pushes typed image dataset for better Data Studio preview")
    print("  raw-folder: uploads folder as-is via upload_large_folder")


def _assert(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def _validate_repo_id(repo_id: str) -> None:
    _assert("/" in repo_id and len(repo_id.split("/", 1)[0]) > 0 and len(repo_id.split("/", 1)[1]) > 0,
            f'Invalid --repo-id "{repo_id}". Expected format: "username/dataset-name".')


def _read_rows(split_csv: Path) -> Iterable[dict]:
    with split_csv.open("r", encoding="utf-8", newline="") as file:
        reader = csv.DictReader(file)
        header = set(reader.fieldnames or [])
        missing = REQUIRED_COLUMNS - header
        _assert(not missing, f"split.csv is missing columns: {sorted(missing)}")
        for row in reader:
            yield row


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


def find_annotation_for_image(image_path: Path, img_dir: Path, ann_dir: Path) -> Path | None:
    rel_from_img = image_path.relative_to(img_dir)
    candidates = [
        ann_dir / f"{rel_from_img.as_posix()}.json",
        ann_dir / f"{image_path.name}.json",
    ]
    for candidate in candidates:
        if candidate.exists() and candidate.is_file():
            return candidate
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


def validate_legacy_dataset(images_dir: Path, split_csv: Path) -> DatasetStats:
    img_dir = images_dir / "img"
    ann_dir = images_dir / "ann"

    _assert(images_dir.exists() and images_dir.is_dir(), f"images dir not found: {images_dir}")
    _assert(img_dir.exists() and img_dir.is_dir(), f"img dir not found: {img_dir}")
    _assert(ann_dir.exists() and ann_dir.is_dir(), f"ann dir not found: {ann_dir}")
    _assert(split_csv.exists() and split_csv.is_file(), f"split.csv not found: {split_csv}")

    split_counter: Counter = Counter()
    split_view_counter: Dict[str, Counter] = defaultdict(Counter)

    seen_images = set()
    row_count = 0

    for idx, row in enumerate(_read_rows(split_csv), start=2):
        row_count += 1

        image_rel = (row.get("image") or "").strip()
        ann_rel = (row.get("annotation") or "").strip()
        view = (row.get("view") or "").strip() or "unknown"
        split = (row.get("split") or "").strip()

        _assert(split in ALLOWED_SPLITS, f"line {idx}: invalid split '{split}', expected one of {sorted(ALLOWED_SPLITS)}")
        _assert(image_rel.startswith("img/"), f"line {idx}: image path must start with 'img/': {image_rel}")
        _assert(ann_rel.startswith("ann/"), f"line {idx}: annotation path must start with 'ann/': {ann_rel}")

        image_abs = images_dir / Path(image_rel)
        ann_abs = images_dir / Path(ann_rel)

        _assert(image_abs.exists() and image_abs.is_file(), f"line {idx}: image file not found: {image_rel}")
        _assert(ann_abs.exists() and ann_abs.is_file(), f"line {idx}: annotation file not found: {ann_rel}")

        _assert(image_rel not in seen_images, f"line {idx}: duplicate image path in split.csv: {image_rel}")
        seen_images.add(image_rel)

        split_counter[split] += 1
        split_view_counter[split][view] += 1

    _assert(row_count > 0, "split.csv has no rows")

    return DatasetStats(
        split_counter=split_counter,
        split_view_counter=dict(split_view_counter),
        sample_count=row_count,
        dataset_format="legacy_csv",
    )


def validate_split_dirs_dataset(images_dir: Path) -> DatasetStats:
    _assert(images_dir.exists() and images_dir.is_dir(), f"dataset dir not found: {images_dir}")

    for split in ALLOWED_SPLITS:
        split_dir = images_dir / split
        _assert(split_dir.exists() and split_dir.is_dir(), f"split dir not found: {split_dir}")
        for subset in ("img", "ann", "img_info"):
            subset_dir = split_dir / subset
            _assert(
                subset_dir.exists() and subset_dir.is_dir(),
                f"{split}/{subset} dir not found: {subset_dir}",
            )

    meta_path = images_dir / "meta.json"
    _assert(meta_path.exists() and meta_path.is_file(), f"meta.json not found: {meta_path}")
    car_view_tag_ids = load_car_view_tag_ids(meta_path)

    split_counter: Counter = Counter()
    split_view_counter: Dict[str, Counter] = defaultdict(Counter)
    seen_images = set()
    sample_count = 0

    for split in sorted(ALLOWED_SPLITS):
        split_dir = images_dir / split
        img_dir = split_dir / "img"
        ann_dir = split_dir / "ann"
        img_info_dir = split_dir / "img_info"

        image_files = sorted(
            p for p in img_dir.rglob("*") if p.is_file() and p.suffix.lower() in IMAGE_EXTENSIONS
        )

        for image_abs in image_files:
            sample_count += 1

            image_rel = image_abs.relative_to(img_dir).as_posix()
            _assert(image_rel not in seen_images, f"duplicate image in different splits: {image_rel}")
            seen_images.add(image_rel)

            ann_abs = find_annotation_for_image(image_abs, img_dir, ann_dir)
            if ann_abs is None:
                raise ValueError(f"missing annotation for {split}/img/{image_rel}")

            img_info_abs = find_img_info_for_image(image_abs, img_dir, img_info_dir)
            if img_info_abs is None:
                raise ValueError(f"missing img_info for {split}/img/{image_rel}")

            view = extract_car_view_from_img_info(img_info_abs, car_view_tag_ids)
            split_counter[split] += 1
            split_view_counter[split][view] += 1

    _assert(sample_count > 0, "dataset has no images in split folders")

    return DatasetStats(
        split_counter=split_counter,
        split_view_counter=dict(split_view_counter),
        sample_count=sample_count,
        dataset_format="split_dirs",
    )


def validate_dataset(images_dir: Path, split_csv: Path) -> DatasetStats:
    has_split_dirs = all((images_dir / split).exists() for split in ALLOWED_SPLITS)
    if has_split_dirs:
        return validate_split_dirs_dataset(images_dir)

    if split_csv.exists() and split_csv.is_file():
        return validate_legacy_dataset(images_dir, split_csv)

    raise ValueError(
        "Could not detect dataset format. Expected either '\n"
        "1) split dirs: <images-dir>/{train,val,test}/{img,ann,img_info} + meta.json\n"
        "or\n"
        "2) legacy: <images-dir>/img, <images-dir>/ann, and split.csv"
    )


def print_distribution(stats: DatasetStats) -> None:
    split_counter = stats.split_counter
    split_view_counter = stats.split_view_counter

    print("Split sizes:")
    for split in sorted(ALLOWED_SPLITS):
        print(f"  {split}: {split_counter.get(split, 0)}")

    print("\nView distribution by split:")
    for split in sorted(ALLOWED_SPLITS):
        total = split_counter.get(split, 0)
        print(f"  {split}:")
        if total == 0:
            print("    (empty)")
            continue
        for view, count in split_view_counter.get(split, {}).most_common():
            share = 100.0 * count / total
            print(f"    {view}: {count} ({share:.2f}%)")


def _read_text_file(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def _collect_records_split_dirs(images_dir: Path) -> Dict[str, List[dict]]:
    records_by_split: Dict[str, List[dict]] = {split: [] for split in sorted(ALLOWED_SPLITS)}
    car_view_tag_ids = load_car_view_tag_ids(images_dir / "meta.json")

    for split in sorted(ALLOWED_SPLITS):
        split_dir = images_dir / split
        img_dir = split_dir / "img"
        ann_dir = split_dir / "ann"
        img_info_dir = split_dir / "img_info"

        image_files = sorted(
            p for p in img_dir.rglob("*") if p.is_file() and p.suffix.lower() in IMAGE_EXTENSIONS
        )
        for image_abs in image_files:
            image_rel = image_abs.relative_to(img_dir).as_posix()

            ann_abs = find_annotation_for_image(image_abs, img_dir, ann_dir)
            if ann_abs is None:
                raise ValueError(f"missing annotation for {split}/img/{image_rel}")

            img_info_abs = find_img_info_for_image(image_abs, img_dir, img_info_dir)
            if img_info_abs is None:
                raise ValueError(f"missing img_info for {split}/img/{image_rel}")

            records_by_split[split].append(
                {
                    "image": str(image_abs),
                    "split": split,
                    "view": extract_car_view_from_img_info(img_info_abs, car_view_tag_ids),
                    "image_rel": image_rel,
                    "annotation_rel": ann_abs.relative_to(split_dir).as_posix(),
                    "img_info_rel": img_info_abs.relative_to(split_dir).as_posix(),
                    "annotation_json": _read_text_file(ann_abs),
                    "img_info_json": _read_text_file(img_info_abs),
                }
            )

    return records_by_split


def _collect_records_legacy(images_dir: Path, split_csv: Path) -> Dict[str, List[dict]]:
    records_by_split: Dict[str, List[dict]] = {split: [] for split in sorted(ALLOWED_SPLITS)}

    for row in _read_rows(split_csv):
        image_rel = (row.get("image") or "").strip()
        ann_rel = (row.get("annotation") or "").strip()
        split = (row.get("split") or "").strip()
        view = (row.get("view") or "").strip() or "unknown"

        if split not in ALLOWED_SPLITS:
            continue

        image_abs = images_dir / Path(image_rel)
        ann_abs = images_dir / Path(ann_rel)
        records_by_split[split].append(
            {
                "image": str(image_abs),
                "split": split,
                "view": view,
                "image_rel": image_rel,
                "annotation_rel": ann_rel,
                "annotation_json": _read_text_file(ann_abs),
            }
        )

    return records_by_split


def upload_datasetdict_to_hf(
    repo_id: str,
    images_dir: Path,
    split_csv: Path,
    dataset_format: str,
    token: str | None,
    revision: str,
    private: bool,
) -> None:
    from huggingface_hub import HfApi

    try:
        datasets_module = __import__("datasets", fromlist=["Dataset", "DatasetDict", "Image"])
    except ImportError as error:
        raise RuntimeError(
            "datasetdict upload mode requires the 'datasets' package. "
            "Install dependencies from dataset/requirements.txt"
        ) from error

    Dataset = datasets_module.Dataset
    DatasetDict = datasets_module.DatasetDict
    Image = datasets_module.Image

    if dataset_format == "split_dirs":
        records_by_split = _collect_records_split_dirs(images_dir)
    else:
        records_by_split = _collect_records_legacy(images_dir, split_csv)

    dataset_splits = {}
    for split in sorted(ALLOWED_SPLITS):
        records = records_by_split.get(split, [])
        if not records:
            continue
        ds = Dataset.from_list(records)
        dataset_splits[split] = ds.cast_column("image", Image())

    _assert(bool(dataset_splits), "No samples to upload after conversion to DatasetDict")
    dataset_dict = DatasetDict(dataset_splits)

    api = HfApi(token=token)
    api.create_repo(repo_id=repo_id, repo_type="dataset", private=private, exist_ok=True)
    dataset_dict.push_to_hub(repo_id=repo_id, token=token, private=private, revision=revision)


def upload_raw_folder_to_hf(
    repo_id: str,
    images_dir: Path,
    token: str | None,
    revision: str,
    private: bool,
    num_workers: int | None,
) -> None:
    from huggingface_hub import HfApi

    api = HfApi(token=token)
    api.create_repo(repo_id=repo_id, repo_type="dataset", private=private, exist_ok=True)

    # upload_large_folder is recommended for large dataset folders and resumable uploads.
    api.upload_large_folder(
        repo_id=repo_id,
        repo_type="dataset",
        folder_path=str(images_dir),
        revision=revision,
        num_workers=num_workers,
    )


def main() -> None:
    # Run: python save_dataset.py --repo-id mitbersh/car-parts-segmentation-raw --images-dir ../../images/out --upload-mode datasetdict
    args = parse_args()

    if args.print_format:
        print_format_guide()
        return

    _validate_repo_id(args.repo_id)

    images_dir = args.images_dir.expanduser().resolve()
    split_csv = (args.split_csv or (images_dir / "split.csv")).expanduser().resolve()

    stats = validate_dataset(images_dir=images_dir, split_csv=split_csv)

    print(f"Validation passed for: {images_dir}")
    print(f"Detected format: {stats.dataset_format}")
    print(f"Total samples: {stats.sample_count}")
    print_distribution(stats)

    if args.dry_run:
        print("\n[dry-run] Upload skipped.")
        print(f"[dry-run] Target repo: {args.repo_id} (revision: {args.revision})")
        print(f"[dry-run] Upload mode: {args.upload_mode}")
        return

    if args.upload_mode == "datasetdict":
        upload_datasetdict_to_hf(
            repo_id=args.repo_id,
            images_dir=images_dir,
            split_csv=split_csv,
            dataset_format=stats.dataset_format,
            token=args.token,
            revision=args.revision,
            private=args.private,
        )
    else:
        upload_raw_folder_to_hf(
            repo_id=args.repo_id,
            images_dir=images_dir,
            token=args.token,
            revision=args.revision,
            private=args.private,
            num_workers=args.num_workers,
        )
    print("\nUpload completed.")


if __name__ == "__main__":
    main()

