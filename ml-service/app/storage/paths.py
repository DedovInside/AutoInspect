from __future__ import annotations

from pathlib import Path


def sanitize_key(key: str) -> str:
    return key.strip().replace("\\", "_").replace("/", "_")


def cache_models_dir(cache_dir: Path) -> Path:
    return cache_dir / "models"


def cache_jobs_dir(cache_dir: Path) -> Path:
    return cache_dir / "jobs"


def model_cache_path(cache_dir: Path, key: str) -> Path:
    return cache_models_dir(cache_dir) / sanitize_key(key)


def job_dir(cache_dir: Path, correlation_id: str) -> Path:
    safe_id = correlation_id.strip() or "job-unknown"
    return cache_jobs_dir(cache_dir) / safe_id


def job_file_path(cache_dir: Path, correlation_id: str, key: str) -> Path:
    return job_dir(cache_dir, correlation_id) / Path(key).name
