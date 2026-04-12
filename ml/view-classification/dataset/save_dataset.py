import argparse
import math
from pathlib import Path

from huggingface_hub import CommitOperationAdd, CommitOperationDelete, HfApi

api = HfApi()

FOLDERS = [
    "back",
    "back-left",
    "back-right",
    "front",
    "front-left",
    "front-right",
    "left",
    "right",
]

CHUNKS_BY_FOLDER = {
    "back": 2,
    "back-left": 4,
    "back-right": 4,
    "front": 2,
    "front-left": 4,
    "front-right": 4,
    "left": 2,
    "right": 2,
}

SCRIPT_DIR = Path(__file__).resolve().parent
DEFAULT_BASE_PATH = SCRIPT_DIR / "blended"


def list_folder_files(folder_path: Path):
    files = []
    for file_path in folder_path.rglob("*"):
        if file_path.is_file() and not file_path.name.startswith("."):
            rel_path = file_path.relative_to(folder_path).as_posix()
            files.append((file_path, rel_path))
    return files


def split_files_into_chunks(folder_name: str, folder_path: Path):
    """Разбивает файлы папки на фиксированное число частей."""
    all_files = list_folder_files(folder_path)
    if not all_files:
        return []

    num_chunks = CHUNKS_BY_FOLDER.get(folder_name, 1)
    chunk_size = int(math.ceil(len(all_files) / num_chunks))
    return [all_files[i:i + chunk_size] for i in range(0, len(all_files), chunk_size)]


def clear_repo(repo_id: str, dry_run: bool):
    repo_files = api.list_repo_files(repo_id=repo_id, repo_type="dataset")
    if not repo_files:
        print("Репозиторий уже пустой, очистка не требуется.")
        return

    print(f"В репозитории найдено {len(repo_files)} файлов для удаления.")
    if dry_run:
        print("[dry-run] Очистка пропущена. Примеры файлов:")
        for sample in repo_files[:10]:
            print(f"  - {sample}")
        return

    operations = [CommitOperationDelete(path_in_repo=path) for path in repo_files]
    api.create_commit(
        repo_id=repo_id,
        repo_type="dataset",
        operations=operations,
        commit_message="Clear dataset repository before full refresh",
    )
    print("Очистка репозитория завершена.")


def upload_folder_in_chunks(repo_id: str, folder_name: str, base_path: Path, dry_run: bool):
    """Загружает папку по частям."""
    folder_path = base_path / folder_name
    if not folder_path.exists():
        print(f"Папка не найдена, пропускаю: {folder_path}")
        return True

    chunks = split_files_into_chunks(folder_name, folder_path)
    total = sum(len(chunk) for chunk in chunks)

    print(f"\nОбработка папки: {folder_name}")
    print(f"  Всего файлов: {total}")
    print(f"  Коммитов: {len(chunks)}")

    for i, chunk in enumerate(chunks, 1):
        operations = []
        for full_path, rel_path in chunk:
            operations.append(
                CommitOperationAdd(
                    path_in_repo=f"{folder_name}/{rel_path}",
                    path_or_fileobj=str(full_path),
                )
            )

        if dry_run:
            print(f"  [dry-run] Коммит {i}/{len(chunks)}: {len(chunk)} файлов")
            continue

        try:
            api.create_commit(
                repo_id=repo_id,
                repo_type="dataset",
                operations=operations,
                commit_message=f"Upload {folder_name} (part {i}/{len(chunks)})",
            )
            print(f"  Коммит {i}/{len(chunks)} завершен")
        except Exception as error:
            print(f"  Ошибка в коммите {i}: {error}")
            return False

    return True


def upload_meta(repo_id: str, base_path: Path, dry_run: bool):
    meta_path = base_path / "meta.csv"
    if not meta_path.exists():
        print(f"meta.csv не найден: {meta_path}")
        return False

    if dry_run:
        print(f"[dry-run] Загрузка meta.csv: {meta_path}")
        return True

    api.create_commit(
        repo_id=repo_id,
        repo_type="dataset",
        operations=[
            CommitOperationAdd(path_in_repo="meta.csv", path_or_fileobj=str(meta_path))
        ],
        commit_message="Upload meta.csv",
    )
    print("meta.csv загружен в корень репозитория")
    return True


def parse_args():
    parser = argparse.ArgumentParser(description="Upload local dataset to Hugging Face.")
    parser.add_argument("--repo-id", default="mitbersh/car-view", help="HF dataset repo id")
    parser.add_argument(
        "--base-path",
        default=str(DEFAULT_BASE_PATH),
        help="Путь к локальной папке blended",
    )
    parser.add_argument(
        "--clear-repo",
        action="store_true",
        help="Удалить все файлы в HF dataset перед загрузкой",
    )
    parser.add_argument(
        "--confirm-clear",
        default="",
        help="Для очистки укажи точный repo_id еще раз",
    )
    parser.add_argument("--dry-run", action="store_true", help="Только показать действия")
    return parser.parse_args()


def main():
    r'''Запуск: python save_dataset.py --clear-repo --confirm-clear "mitbersh/car-view"'''
    args = parse_args()
    base_path = Path(args.base_path).expanduser().resolve()

    if not base_path.exists():
        raise FileNotFoundError(f"Локальная папка не найдена: {base_path}")

    if args.clear_repo:
        if args.confirm_clear != args.repo_id:
            raise ValueError("Для очистки укажи --confirm-clear с точным значением --repo-id")
        clear_repo(repo_id=args.repo_id, dry_run=args.dry_run)

    for folder in FOLDERS:
        success = upload_folder_in_chunks(
            repo_id=args.repo_id,
            folder_name=folder,
            base_path=base_path,
            dry_run=args.dry_run,
        )
        if not success:
            raise RuntimeError(f"Загрузка остановлена на папке: {folder}")

    if not upload_meta(repo_id=args.repo_id, base_path=base_path, dry_run=args.dry_run):
        raise RuntimeError("Не удалось загрузить meta.csv")

    print("Готово")


if __name__ == "__main__":
    main()
