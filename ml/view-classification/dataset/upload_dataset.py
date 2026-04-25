from __future__ import annotations

import argparse
import csv
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, Sequence

from huggingface_hub import HfApi, get_token

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
VIEWS = (
    "back",
    "back-left",
    "back-right",
    "front",
    "front-left",
    "front-right",
    "left",
    "right",
)
SPLITS = ("train", "val", "test")
SOURCES = ("synthetic", "real")
REQUIRED_META_COLUMNS = ("filename", "relative_path", "view", "source", "original_name", "split")

SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_SOURCE_DIR = SCRIPT_DIR / "blended"


@dataclass(frozen=True)
class ValidationReport:
    rows_count: int
    image_files_count: int
    split_counts: Dict[str, int]
    view_counts: Dict[str, int]
    source_counts: Dict[str, int]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Validate and upload a car-view classification dataset folder to Hugging Face Hub."
    )
    parser.add_argument("--repo-id", required=True, help="Hugging Face dataset repo id, e.g. username/car-view")
    parser.add_argument(
        "--source-dir",
        "--base-path",
        dest="source_dir",
        type=Path,
        default=DEFAULT_SOURCE_DIR,
        help="Path to prepared dataset root with view folders and meta.csv (default: ./blended).",
    )
    parser.add_argument(
        "--token",
        default=None,
        help="HF token. If omitted, uses HF_TOKEN/HUGGINGFACE_HUB_TOKEN or token from `hf auth login`.",
    )

    visibility_group = parser.add_mutually_exclusive_group()
    visibility_group.add_argument("--private", action="store_true", help="Create repository as private.")
    visibility_group.add_argument("--public", action="store_true", help="Create repository as public.")

    parser.add_argument("--revision", default="main", help="Hub revision/branch (default: main).")
    parser.add_argument(
        "--path-in-repo",
        default="",
        help="Optional subfolder path inside repo where dataset will be uploaded.",
    )
    parser.add_argument(
        "--commit-message",
        default="Upload car-view classification dataset",
        help="Commit message for upload.",
    )
    parser.add_argument(
        "--create-only",
        action="store_true",
        help="Create dataset repository only and exit without uploading files.",
    )
    parser.add_argument(
        "--include",
        action="append",
        default=None,
        help="Optional glob pattern to include (repeatable). Example: --include 'front/**'",
    )
    parser.add_argument(
        "--exclude",
        action="append",
        default=None,
        help="Optional glob pattern to exclude (repeatable). Example: --exclude '**/*.cache'",
    )
    parser.add_argument("--dry-run", action="store_true", help="Validate only, do not create repo or upload anything.")
    parser.add_argument(
        "--allow-extra-files",
        action="store_true",
        help="Allow image files that are present in source-dir but missing from meta.csv.",
    )
    return parser.parse_args()


def resolve_token(cli_token: str | None) -> str | None:
    if cli_token:
        return cli_token
    return os.getenv("HF_TOKEN") or os.getenv("HUGGINGFACE_HUB_TOKEN") or get_token()


def resolve_private_flag(args: argparse.Namespace) -> bool | None:
    if args.private:
        return True
    if args.public:
        return False
    return None


def resolve_repo_id(api: HfApi, repo_id: str) -> str:
    cleaned = repo_id.strip()
    if not cleaned:
        raise ValueError("repo-id is empty.")
    if "/" in cleaned:
        return cleaned

    whoami_info = api.whoami()
    username = whoami_info.get("name")
    if not isinstance(username, str) or not username:
        raise RuntimeError("Cannot resolve Hugging Face username from token. Use repo-id as username/repo.")

    return f"{username}/{cleaned}"


def create_dataset_repo(api: HfApi, repo_id: str, private: bool | None, token: str | None) -> str:
    create_repo_kwargs = {
        "repo_id": repo_id,
        "repo_type": "dataset",
        "exist_ok": True,
        "token": token,
    }
    if private is not None:
        create_repo_kwargs["private"] = private

    created = api.create_repo(**create_repo_kwargs)
    return str(created)


def iter_files(root: Path) -> Iterable[Path]:
    for path in root.rglob("*"):
        if path.is_file() and not path.name.startswith("."):
            yield path


def iter_image_files(root: Path) -> Iterable[Path]:
    for path in iter_files(root):
        if path.suffix.lower() in IMAGE_EXTENSIONS:
            yield path


def read_meta_rows(meta_path: Path) -> list[dict[str, str]]:
    with meta_path.open("r", newline="", encoding="utf-8") as meta_file:
        reader = csv.DictReader(meta_file)
        fieldnames = reader.fieldnames or []
        missing_columns = [column for column in REQUIRED_META_COLUMNS if column not in fieldnames]
        if missing_columns:
            raise ValueError(f"meta.csv is missing required columns: {missing_columns}")

        return [dict(row) for row in reader]


def format_preview(items: Sequence[str], limit: int = 20) -> str:
    preview = "\n".join(f"- {item}" for item in items[:limit])
    extra = "" if len(items) <= limit else f"\n... and {len(items) - limit} more"
    return f"{preview}{extra}"


def validate_meta_row(source_dir: Path, row: dict[str, str], row_number: int) -> tuple[str, str, str, str]:
    filename = row.get("filename", "").strip()
    relative_path = row.get("relative_path", "").strip().replace("\\", "/")
    view = row.get("view", "").strip()
    source = row.get("source", "").strip()
    split = row.get("split", "").strip()

    if not filename:
        raise ValueError(f"meta.csv row {row_number}: empty filename")
    if not relative_path:
        raise ValueError(f"meta.csv row {row_number}: empty relative_path")
    if Path(relative_path).is_absolute() or ".." in Path(relative_path).parts:
        raise ValueError(f"meta.csv row {row_number}: unsafe relative_path: {relative_path}")
    if Path(relative_path).name != filename:
        raise ValueError(
            f"meta.csv row {row_number}: filename does not match relative_path basename: "
            f"filename={filename}, relative_path={relative_path}"
        )
    if view not in VIEWS:
        raise ValueError(f"meta.csv row {row_number}: unknown view '{view}'. Expected one of: {list(VIEWS)}")
    if source not in SOURCES:
        raise ValueError(f"meta.csv row {row_number}: unknown source '{source}'. Expected one of: {list(SOURCES)}")
    if split not in SPLITS:
        raise ValueError(f"meta.csv row {row_number}: unknown split '{split}'. Expected one of: {list(SPLITS)}")

    path_parts = Path(relative_path).parts
    if not path_parts or path_parts[0] != view:
        raise ValueError(
            f"meta.csv row {row_number}: relative_path must start with the view folder. "
            f"view={view}, relative_path={relative_path}"
        )

    image_path = source_dir / relative_path
    if not image_path.exists() or not image_path.is_file():
        raise FileNotFoundError(f"meta.csv row {row_number}: image file not found: {image_path}")
    if image_path.suffix.lower() not in IMAGE_EXTENSIONS:
        raise ValueError(f"meta.csv row {row_number}: unsupported image extension: {image_path}")

    return relative_path, view, source, split


def validate_classification_dataset(source_dir: Path, allow_extra_files: bool = False) -> ValidationReport:
    if not source_dir.exists() or not source_dir.is_dir():
        raise FileNotFoundError(f"Dataset source directory does not exist: {source_dir}")

    meta_path = source_dir / "meta.csv"
    if not meta_path.exists() or not meta_path.is_file():
        raise FileNotFoundError(f"Missing meta.csv: {meta_path}")

    for view in VIEWS:
        view_dir = source_dir / view
        if not view_dir.exists() or not view_dir.is_dir():
            raise FileNotFoundError(f"Missing view directory: {view_dir}")

    rows = read_meta_rows(meta_path)
    if not rows:
        raise ValueError("meta.csv is empty.")

    seen_relative_paths: set[str] = set()
    split_counts = {split: 0 for split in SPLITS}
    view_counts = {view: 0 for view in VIEWS}
    source_counts = {source: 0 for source in SOURCES}

    for index, row in enumerate(rows, start=2):
        relative_path, view, source, split = validate_meta_row(source_dir, row, row_number=index)
        if relative_path in seen_relative_paths:
            raise ValueError(f"meta.csv has duplicate relative_path: {relative_path}")

        seen_relative_paths.add(relative_path)
        split_counts[split] += 1
        view_counts[view] += 1
        source_counts[source] += 1

    image_relative_paths = {path.relative_to(source_dir).as_posix() for path in iter_image_files(source_dir)}
    missing_from_meta = sorted(image_relative_paths - seen_relative_paths)
    if missing_from_meta and not allow_extra_files:
        raise ValueError(
            "source-dir contains image files that are not listed in meta.csv:\n"
            f"{format_preview(missing_from_meta)}"
        )

    total_rows = len(rows)
    if total_rows == 0:
        raise ValueError("Dataset is empty: no rows found in meta.csv.")

    if split_counts["train"] == 0:
        raise ValueError("Dataset has no train images according to meta.csv.")
    if split_counts["val"] == 0:
        raise ValueError("Dataset has no val images according to meta.csv.")

    return ValidationReport(
        rows_count=total_rows,
        image_files_count=len(image_relative_paths),
        split_counts=split_counts,
        view_counts=view_counts,
        source_counts=source_counts,
    )


def upload_dataset(
    source_dir: Path,
    repo_id: str,
    token: str | None,
    private: bool | None,
    revision: str,
    path_in_repo: str,
    commit_message: str,
    allow_patterns: Sequence[str] | None,
    ignore_patterns: Sequence[str] | None,
) -> tuple[str, str]:
    api = HfApi(token=token)
    resolved_repo_id = resolve_repo_id(api, repo_id)

    create_dataset_repo(api=api, repo_id=resolved_repo_id, private=private, token=token)

    commit_info = api.upload_folder(
        repo_id=resolved_repo_id,
        repo_type="dataset",
        folder_path=str(source_dir),
        revision=revision,
        commit_message=commit_message,
        token=token,
        path_in_repo=path_in_repo or None,
        allow_patterns=list(allow_patterns) if allow_patterns else None,
        ignore_patterns=list(ignore_patterns) if ignore_patterns else None,
    )
    commit_ref = getattr(commit_info, "commit_url", "") or getattr(commit_info, "oid", "") or "uploaded"
    return resolved_repo_id, commit_ref


def print_validation_report(source_dir: Path, report: ValidationReport) -> None:
    print(f"Validated classification dataset: {source_dir}")
    print(f"Rows in meta.csv: {report.rows_count}")
    print(f"Image files on disk: {report.image_files_count}")
    print(
        "Split counts: "
        + ", ".join(f"{split}={report.split_counts[split]}" for split in SPLITS if report.split_counts[split] > 0)
    )
    print("Source counts: " + ", ".join(f"{source}={report.source_counts[source]}" for source in SOURCES))
    print("View counts:")
    for view in VIEWS:
        print(f"  {view}: {report.view_counts[view]}")


def main() -> None:
    # Run: python upload_dataset.py --repo-id "mitbersh/car-view-classification" --source-dir "./blended"
    args = parse_args()
    source_dir = args.source_dir.expanduser().resolve()

    report = validate_classification_dataset(
        source_dir=source_dir,
        allow_extra_files=args.allow_extra_files,
    )
    print_validation_report(source_dir, report)

    if args.dry_run:
        print("Dry-run enabled: upload skipped.")
        return

    token = resolve_token(args.token)
    if not token:
        raise RuntimeError(
            "HF token not found. Pass --token, set HF_TOKEN/HUGGINGFACE_HUB_TOKEN, or run `hf auth login`."
        )

    private = resolve_private_flag(args)
    api = HfApi(token=token)
    resolved_repo_id = resolve_repo_id(api, args.repo_id)
    create_ref = create_dataset_repo(api=api, repo_id=resolved_repo_id, private=private, token=token)
    print(f"Repository is ready: {resolved_repo_id}")
    print(f"Repository URL: {create_ref}")

    if args.create_only:
        print("Create-only enabled: upload skipped.")
        return

    uploaded_repo_id, result = upload_dataset(
        source_dir=source_dir,
        repo_id=resolved_repo_id,
        token=token,
        private=private,
        revision=args.revision,
        path_in_repo=args.path_in_repo.strip("/"),
        commit_message=args.commit_message,
        allow_patterns=args.include,
        ignore_patterns=args.exclude,
    )

    print(f"Uploaded to dataset repo: {uploaded_repo_id}")
    if result:
        print(f"Hub commit: {result}")


if __name__ == "__main__":
    main()
