from huggingface_hub import HfApi
import os
from pathlib import Path

api = HfApi()

folders = ["back", "back-left", "back-right", "front", "front-left",
           "front-right", "left", "other", "right"]

chunks_by_folder = {
    "back": 2,
    "back-left": 4,
    "back-right":   4,
    "front": 2,
    "front-left": 4,
    "front-right": 4,
    "left": 2,
    "other": 8,
    "right": 2,
}

base_path = "/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/blended"

# api.delete_folder(
#     path_in_repo="back",
#     repo_id="mitbersh/car-position",
#     repo_type="dataset",
#     commit_message="Delete back folder"
# )


def split_files_into_chunks(folder_path):
    """Разбивает файлы в папке на N частей"""
    num_chunks = chunks_by_folder[folder_path.split("/")[-1]]
    all_files = []
    for root, dirs, files in os.walk(folder_path):
        for file in files:
            if not file.startswith('.'):  # Игнорируем скрытые файлы
                full_path = os.path.join(root, file)
                rel_path = os.path.relpath(full_path, folder_path)
                all_files.append((full_path, rel_path))

    # Разбиваем на равные части
    chunk_size = len(all_files) // num_chunks + (1 if len(all_files) % num_chunks else 0)
    chunks = [all_files[i:i + chunk_size] for i in range(0, len(all_files), chunk_size)]

    return chunks


def upload_folder_in_chunks(folder_name):
    """Загружает папку по частям"""
    folder_path = f"{base_path}/{folder_name}"

    print(f"\n📁 Обработка папки: {folder_name}")

    chunks = split_files_into_chunks(folder_path)

    print(f"   Всего файлов: {sum(len(chunk) for chunk in chunks)}")
    print(f"   Разбито на {len(chunks)} коммитов")

    for i, chunk in enumerate(chunks, 1):
        try:
            print(f"\n   Коммит {i}/{len(chunks)} ({len(chunk)} файлов)...")

            operations = []
            for full_path, rel_path in chunk:
                # Формируем путь в репозитории
                path_in_repo = f"{folder_name}/{rel_path}"

                operations.append(
                    api.upload_file(
                        path_or_fileobj=full_path,
                        path_in_repo=path_in_repo,
                        repo_id="mitbersh/car-position",
                        repo_type="dataset",
                        commit_message=f"Upload {folder_name} (part {i}/{len(chunks)})",
                        create_pr=False,
                    )
                )

            print(f"   ✓ Коммит {i}/{len(chunks)} завершён")

        except Exception as e:
            print(f"   ✗ Ошибка в коммите {i}: {e}")
            return False

    print(f"✓ Папка {folder_name} полностью загружена!\n")
    return True


# Загружаем все папки
for folder in folders:
    upload_folder_in_chunks(folder)
