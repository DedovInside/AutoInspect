from __future__ import annotations

from functools import lru_cache
from pathlib import Path

from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    kafka_bootstrap_servers: str = "localhost:9092"
    kafka_request_topic: str = "analysis.requested.v1"
    kafka_result_topic: str = "analysis.completed.v1"
    kafka_consumer_group: str = "autoinspect-ml-service"

    s3_endpoint_url: str = "http://localhost:9000"
    s3_access_key: str = "minioadmin"
    s3_secret_key: str = "minioadmin"
    s3_bucket: str = "autoinspect"

    local_cache_dir: str = "/tmp/autoinspect-cache"

    damage_model_s3_key: str = "models/general/damage_segmentation.pt"
    damage_config_s3_key: str = "models/general/damage_inference_config.json"

    model_config = SettingsConfigDict(env_file=".env", env_prefix="", case_sensitive=False)

    @property
    def cache_dir(self) -> Path:
        return Path(self.local_cache_dir)


@lru_cache
def get_settings() -> Settings:
    return Settings()

