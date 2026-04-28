from __future__ import annotations

import json
from pathlib import Path

from confluent_kafka import KafkaException

from app.contracts.mapper import build_analysis_result_message, build_failed_result, parse_analysis_request
from app.inference.adapter import AutoInspectPipeline
from app.kafka.consumer import create_consumer
from app.kafka.producer import create_producer
from app.settings import get_settings
from app.storage.s3_client import S3Storage


class KafkaWorker:
    def __init__(self) -> None:
        self._settings = get_settings()
        self._consumer = create_consumer(self._settings)
        self._producer = create_producer(self._settings)
        self._pipeline = AutoInspectPipeline()

    def _read_model_meta(self, config_path: Path) -> tuple[str, str]:
        try:
            payload = json.loads(config_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return "unknown", "unknown"
        if not isinstance(payload, dict):
            return "unknown", "unknown"
        model_id = str(payload.get("model_id") or "unknown")
        model_version = str(payload.get("model_version") or "unknown")
        return model_id, model_version

    def _prepare_artifacts(self, task) -> tuple[list[Path], list[str], Path, Path, Path, Path, str, str]:
        storage = S3Storage(self._settings, correlation_id=task.correlation_id)

        image_paths = [storage.download_to_cache(key) for key in task.image_s3_keys]
        image_uris = [f"s3://{self._settings.s3_bucket}/{key}" for key in task.image_s3_keys]

        parts_model_path = storage.download_to_cache(task.parts_model_s3_key)
        parts_config_path = storage.download_to_cache(task.parts_config_s3_key)

        damage_model_path = storage.download_to_cache(self._settings.damage_model_s3_key)
        damage_config_path = storage.download_to_cache(self._settings.damage_config_s3_key)

        model_id, model_version = self._read_model_meta(parts_config_path)
        return (
            image_paths,
            image_uris,
            parts_model_path,
            damage_model_path,
            parts_config_path,
            damage_config_path,
            model_id,
            model_version,
        )

    def run_forever(self) -> None:
        self._consumer.subscribe([self._settings.kafka_request_topic])

        try:
            while True:
                msg = self._consumer.poll(1.0)
                if msg is None:
                    continue
                if msg.error():
                    raise KafkaException(msg.error())

                try:
                    task = parse_analysis_request(msg.value())
                    (
                        image_paths,
                        image_uris,
                        parts_model_path,
                        damage_model_path,
                        parts_config_path,
                        damage_config_path,
                        model_id,
                        model_version,
                    ) = self._prepare_artifacts(task)

                    result = self._pipeline.analyze_batch(
                        image_paths=image_paths,
                        image_uris=image_uris,
                        parts_model_path=parts_model_path,
                        damage_model_path=damage_model_path,
                        parts_inference_config_path=parts_config_path,
                        damage_inference_config_path=damage_config_path,
                        model_id=model_id,
                        model_version=model_version,
                        batch_id=task.correlation_id,
                        correlation_id=task.correlation_id,
                    )
                    payload = build_analysis_result_message(result).SerializeToString()
                    self._producer.produce(
                        self._settings.kafka_result_topic,
                        key=task.correlation_id,
                        value=payload,
                    )
                    self._producer.poll(0)
                    self._consumer.commit(message=msg, asynchronous=False)
                except Exception as exc:  # noqa: BLE001
                    correlation_id = ""
                    try:
                        correlation_id = task.correlation_id
                    except Exception:  # noqa: BLE001
                        pass
                    failed = build_failed_result(correlation_id, str(exc))
                    payload = build_analysis_result_message(failed).SerializeToString()
                    self._producer.produce(
                        self._settings.kafka_result_topic,
                        key=correlation_id,
                        value=payload,
                    )
                    self._producer.poll(0)
                    self._consumer.commit(message=msg, asynchronous=False)
        finally:
            self._producer.flush(5)
            self._consumer.close()


def main() -> None:
    worker = KafkaWorker()
    worker.run_forever()


if __name__ == "__main__":
    main()
