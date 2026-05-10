from __future__ import annotations

import json
import logging
import time
from pathlib import Path

from confluent_kafka import KafkaException

import app.logging_config  # noqa: F401
from app.contracts.mapper import build_analysis_result_message, build_failed_result, parse_analysis_request
from app.health import HealthServer, HealthState
from app.inference.adapter import AutoInspectPipeline
from app.kafka.consumer import create_consumer
from app.kafka.producer import create_producer
from app.settings import get_settings
from app.storage.s3_client import S3Storage

logger = logging.getLogger(__name__)


class KafkaWorker:
    def __init__(self) -> None:
        self._settings = get_settings()
        self._consumer = create_consumer(self._settings)
        self._producer = create_producer(self._settings)
        self._pipeline = AutoInspectPipeline()
        self._health = HealthState()
        self._health_server: HealthServer | None = None

        if self._settings.health_enabled:
            self._health_server = HealthServer(
                self._settings.health_host,
                self._settings.health_port,
                self._health.snapshot,
            )
            self._health_server.start()
            logger.info("Health endpoint started on %s:%s", self._settings.health_host, self._settings.health_port)

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
        image_uris = [f"s3://{storage.bucket_for_key(key)}/{key}" for key in task.image_s3_keys]

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

    def _produce_with_retry(self, key: str, payload: bytes) -> None:
        retries = max(1, int(self._settings.kafka_produce_retries))
        backoff = float(self._settings.kafka_retry_backoff_ms) / 1000.0
        for attempt in range(1, retries + 1):
            try:
                logger.info(
                    "Publishing analysis result to Kafka topic=%s key=%s payload_bytes=%s attempt=%s/%s",
                    self._settings.kafka_result_topic,
                    key,
                    len(payload),
                    attempt,
                    retries,
                )
                self._producer.produce(
                    self._settings.kafka_result_topic,
                    key=key,
                    value=payload,
                )
                remaining = self._producer.flush(10)
                if remaining > 0:
                    raise RuntimeError(f"{remaining} Kafka message(s) were not delivered before timeout")
                return
            except Exception as exc:  # noqa: BLE001
                logger.warning("Kafka produce failed (attempt %s/%s): %s", attempt, retries, exc)
                if attempt >= retries:
                    raise
                time.sleep(backoff * attempt)

    def run_forever(self) -> None:
        self._consumer.subscribe([self._settings.kafka_request_topic])
        logger.info("Kafka worker started on topic=%s", self._settings.kafka_request_topic)

        try:
            while True:
                msg = self._consumer.poll(1.0)
                if msg is None:
                    continue
                if msg.error():
                    err = msg.error()
                    if getattr(err, "fatal", lambda: False)():
                        raise KafkaException(err)
                    logger.warning("Kafka consumer error: %s", err)
                    continue

                try:
                    logger.info(
                        "Received Kafka message topic=%s partition=%s offset=%s key=%s payload_bytes=%s",
                        msg.topic(),
                        msg.partition(),
                        msg.offset(),
                        msg.key().decode("utf-8", errors="replace") if msg.key() else "",
                        len(msg.value() or b""),
                    )
                    task = parse_analysis_request(msg.value())
                    logger.info(
                        "Parsed analysis task correlation_id=%s user_id=%s images=%s parts_model_s3_key=%s parts_config_s3_key=%s car=%s/%s/%s/%s",
                        task.correlation_id,
                        task.user_id,
                        len(task.image_s3_keys),
                        task.parts_model_s3_key,
                        task.parts_config_s3_key,
                        task.car_info.make,
                        task.car_info.model,
                        task.car_info.generation,
                        task.car_info.year,
                    )
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
                    logger.info(
                        "Prepared analysis artifacts correlation_id=%s images=%s parts_model=%s damage_model=%s parts_config=%s damage_config=%s model_id=%s model_version=%s",
                        task.correlation_id,
                        len(image_paths),
                        parts_model_path,
                        damage_model_path,
                        parts_config_path,
                        damage_config_path,
                        model_id,
                        model_version,
                    )

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
                    log_analysis_result_summary(result)
                    payload = build_analysis_result_message(result).SerializeToString()
                    self._produce_with_retry(task.correlation_id, payload)
                    logger.info("Published completed analysis result for correlation_id=%s", task.correlation_id)
                    self._consumer.commit(message=msg, asynchronous=False)
                    logger.info(
                        "Committed Kafka request message correlation_id=%s topic=%s partition=%s offset=%s",
                        task.correlation_id,
                        msg.topic(),
                        msg.partition(),
                        msg.offset(),
                    )
                    self._health.mark_success()
                except Exception as exc:  # noqa: BLE001
                    correlation_id = ""
                    try:
                        correlation_id = task.correlation_id
                    except Exception:  # noqa: BLE001
                        pass
                    logger.exception("Failed to process message: %s", exc)
                    self._health.mark_error(str(exc))
                    failed = build_failed_result(correlation_id, str(exc))
                    payload = build_analysis_result_message(failed).SerializeToString()
                    self._produce_with_retry(correlation_id, payload)
                    logger.info("Published failed analysis result for correlation_id=%s", correlation_id)
                    self._consumer.commit(message=msg, asynchronous=False)
                    logger.info(
                        "Committed failed Kafka request message correlation_id=%s topic=%s partition=%s offset=%s",
                        correlation_id,
                        msg.topic(),
                        msg.partition(),
                        msg.offset(),
                    )
        finally:
            if self._health_server is not None:
                self._health_server.stop()
            self._producer.flush(5)
            self._consumer.close()


def main() -> None:
    worker = KafkaWorker()
    worker.run_forever()


def log_analysis_result_summary(result) -> None:
    image_count = len(result.results)
    damage_count = sum(len(image.damage_instances) for image in result.results)
    summary_count = sum(len(image.parts_summary) for image in result.results)

    logger.info(
        "Analysis result summary correlation_id=%s status=%s model_id=%s model_version=%s batch_id=%s images=%s damages=%s part_summaries=%s",
        result.correlation_id,
        result.status,
        result.model_id,
        result.model_version,
        result.batch_id,
        image_count,
        damage_count,
        summary_count,
    )

    for image in result.results:
        logger.info(
            "Image result image_id=%s image_uri=%s size=%sx%s damages=%s part_summaries=%s",
            image.image_id,
            image.image_uri,
            image.width,
            image.height,
            len(image.damage_instances),
            len(image.parts_summary),
        )
        for damage in image.damage_instances:
            logger.info(
                "Damage id=%s type=%s confidence=%.4f bbox=%s parts=%s",
                damage.id,
                damage.damage_type,
                damage.confidence,
                damage.bbox,
                [
                    {
                        "name": part.name,
                        "side": part.side,
                        "confidence": round(float(part.confidence), 4),
                    }
                    for part in damage.parts
                ],
            )
        for summary in image.parts_summary:
            logger.info(
                "Part summary name=%s side=%s damage_count=%s damage_types=%s",
                summary.name,
                summary.side,
                summary.damage_count,
                summary.damage_types,
            )


if __name__ == "__main__":
    main()
