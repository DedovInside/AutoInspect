from __future__ import annotations

import argparse
import os
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, Sequence

from huggingface_hub import HfApi, get_token

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
SPLITS = ("train", "val", "test")


@dataclass(frozen=True)
class ValidationReport:
    image_counts: Dict[str, int]
    total_images: int


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    parts_root = script_dir.parents[2]
    default_source_dir = parts_root / "images" / "yolo"

    parser = argparse.ArgumentParser(
        description="Validate and upload a YOLO dataset folder to Hugging Face Hub."
    )
    parser.add_argument("--repo-id", required=True, help="Hugging Face dataset repo id, e.g. username/my-yolo")
    parser.add_argument(
        "--source-dir",
        type=Path,
        default=default_source_dir,
        help="Path to YOLO dataset root (default: ../images/yolo relative to parts-segmentation).",
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
        default="Upload YOLO dataset",
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
        help="Optional glob pattern to include (repeatable). Example: --include 'images/**'",
    )
    parser.add_argument(
        "--exclude",
        action="append",
        default=None,
        help="Optional glob pattern to exclude (repeatable). Example: --exclude '**/*.cache'",
    )
    parser.add_argument("--dry-run", action="store_true", help="Validate only, do not upload anything.")
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
    if "/" in cleaned:
        return cleaned

    whoami_info = api.whoami()
    username = whoami_info.get("name")
    if not isinstance(username, str) or not username:
        raise Exception("Cannot resolve Hugging Face username from token. Use repo-id as username/repo.")

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


def validate_dataset_yaml(source_dir: Path) -> None:
    dataset_yaml = source_dir / "dataset.yaml"
    if not dataset_yaml.exists() or not dataset_yaml.is_file():
        raise Exception(f"Missing dataset.yaml: {dataset_yaml}")

    lines = [line.strip() for line in dataset_yaml.read_text(encoding="utf-8").splitlines() if line.strip()]
    required_prefixes = ("train:", "val:", "test:", "names:")
    missing = [prefix for prefix in required_prefixes if not any(line.startswith(prefix) for line in lines)]
    if missing:
        raise Exception(f"dataset.yaml is missing required keys: {missing}")


def iter_files(root: Path) -> Iterable[Path]:
    for path in root.rglob("*"):
        if path.is_file():
            yield path


def validate_split_pairs(source_dir: Path, split: str) -> int:
    images_root = source_dir / "images" / split
    labels_root = source_dir / "labels" / split

    if not images_root.exists() or not images_root.is_dir():
        raise Exception(f"Missing images split directory: {images_root}")
    if not labels_root.exists() or not labels_root.is_dir():
        raise Exception(f"Missing labels split directory: {labels_root}")

    images = [p for p in iter_files(images_root) if p.suffix.lower() in IMAGE_EXTENSIONS]
    labels = list(iter_files(labels_root))

    image_rel_stems = {p.relative_to(images_root).with_suffix("").as_posix() for p in images}
    label_rel_stems = {p.relative_to(labels_root).with_suffix("").as_posix() for p in labels if p.suffix.lower() == ".txt"}

    missing_labels = sorted(image_rel_stems - label_rel_stems)
    extra_labels = sorted(label_rel_stems - image_rel_stems)

    if missing_labels:
        preview = "\n".join(f"- {item}" for item in missing_labels[:20])
        extra = "" if len(missing_labels) <= 20 else f"\n... and {len(missing_labels) - 20} more"
        raise Exception(f"Split '{split}' has images without labels:\n{preview}{extra}")

    if extra_labels:
        preview = "\n".join(f"- {item}" for item in extra_labels[:20])
        extra = "" if len(extra_labels) <= 20 else f"\n... and {len(extra_labels) - 20} more"
        raise Exception(f"Split '{split}' has labels without images:\n{preview}{extra}")

    return len(images)


def validate_yolo_dataset(source_dir: Path) -> ValidationReport:
    if not source_dir.exists() or not source_dir.is_dir():
        raise Exception(f"Dataset source directory does not exist: {source_dir}")

    validate_dataset_yaml(source_dir)

    image_counts: Dict[str, int] = {}
    for split in SPLITS:
        image_counts[split] = validate_split_pairs(source_dir, split)

    total = sum(image_counts.values())
    if total == 0:
        raise Exception("YOLO dataset is empty: no images found in train/val/test.")

    return ValidationReport(image_counts=image_counts, total_images=total)


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


def main() -> None:
    # Run: python save_dataset.py --repo-id "car-parts-segmentation-yolo" --source-dir "..\..\images\out"
    args = parse_args()
    source_dir = args.source_dir.resolve()

    report = validate_yolo_dataset(source_dir)
    print(f"Validated YOLO dataset: {source_dir}")
    print(
        "Split image counts: "
        f"train={report.image_counts['train']}, "
        f"val={report.image_counts['val']}, "
        f"test={report.image_counts['test']}"
    )
    print(f"Total images: {report.total_images}")

    if args.dry_run:
        print("Dry-run enabled: upload skipped.")
        return

    token = resolve_token(args.token)
    if not token:
        raise Exception(
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

