from __future__ import annotations

from confluent_kafka import Producer

from app.settings import Settings


def create_producer(settings: Settings) -> Producer:
    config = {
        "bootstrap.servers": settings.kafka_bootstrap_servers,
        "message.send.max.retries": settings.kafka_produce_retries,
        "retry.backoff.ms": settings.kafka_retry_backoff_ms,
    }
    return Producer(config)
