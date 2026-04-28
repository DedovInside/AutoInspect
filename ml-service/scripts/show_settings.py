from __future__ import annotations

import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT_DIR))

from app.settings import get_settings


def main() -> None:
    settings = get_settings()
    print("kafka_bootstrap_servers=", settings.kafka_bootstrap_servers)
    print("kafka_request_topic=", settings.kafka_request_topic)
    print("kafka_result_topic=", settings.kafka_result_topic)
    print("s3_endpoint_url=", settings.s3_endpoint_url)
    print("s3_bucket=", settings.s3_bucket)
    print("local_cache_dir=", settings.local_cache_dir)
    print("damage_model_s3_key=", settings.damage_model_s3_key)
    print("damage_config_s3_key=", settings.damage_config_s3_key)


if __name__ == "__main__":
    main()
