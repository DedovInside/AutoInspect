from __future__ import annotations

from pathlib import Path
import logging
import time

import boto3
from botocore.exceptions import BotoCoreError, ClientError

from app.settings import Settings
from app.storage.paths import job_file_path, model_cache_path

logger = logging.getLogger(__name__)


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
        if not key:
            raise ValueError("S3 key is required")
        if key.startswith("models/"):
            target_path = model_cache_path(self._settings.cache_dir, key)
        else:
            target_path = job_file_path(self._settings.cache_dir, self._correlation_id, key)

        if target_path.exists():
            return target_path

        target_path.parent.mkdir(parents=True, exist_ok=True)
        retries = max(1, int(self._settings.s3_download_retries))
        backoff = float(self._settings.s3_download_backoff_sec)
        for attempt in range(1, retries + 1):
            try:
                self._client.download_file(self._settings.s3_bucket, key, str(target_path))
                return target_path
            except (BotoCoreError, ClientError) as exc:
                logger.warning("S3 download failed (attempt %s/%s) for key=%s: %s", attempt, retries, key, exc)
                if attempt >= retries:
                    raise
                time.sleep(backoff * attempt)
        return target_path

    def ensure_exists(self, key: str) -> None:
        if not key:
            raise ValueError("S3 key is required")
        try:
            self._client.head_object(Bucket=self._settings.s3_bucket, Key=key)
        except ClientError as exc:
            raise FileNotFoundError(f"S3 key not found: {key}") from exc
