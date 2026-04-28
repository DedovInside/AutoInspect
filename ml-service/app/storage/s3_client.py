from __future__ import annotations

from pathlib import Path

import boto3

from app.settings import Settings
from app.storage.paths import job_file_path, model_cache_path


class S3Storage:
    def __init__(self, settings: Settings, correlation_id: str | None = None):
        self._settings = settings
        self._correlation_id = correlation_id or "job-unknown"
        self._client = boto3.client(
            "s3",
            endpoint_url=settings.s3_endpoint_url,
            aws_access_key_id=settings.s3_access_key,
            aws_secret_access_key=settings.s3_secret_key,
        )

    def download_to_cache(self, key: str) -> Path:
        if key.startswith("models/"):
            target_path = model_cache_path(self._settings.cache_dir, key)
        else:
            target_path = job_file_path(self._settings.cache_dir, self._correlation_id, key)

        if target_path.exists():
            return target_path

        target_path.parent.mkdir(parents=True, exist_ok=True)
        self._client.download_file(self._settings.s3_bucket, key, str(target_path))
        return target_path

