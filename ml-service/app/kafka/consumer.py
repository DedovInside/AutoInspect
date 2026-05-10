from __future__ import annotations

from confluent_kafka import Consumer

from app.settings import Settings


def create_consumer(settings: Settings) -> Consumer:
    config = {
        "bootstrap.servers": settings.kafka_bootstrap_servers,
        "group.id": settings.kafka_consumer_group,
        "auto.offset.reset": "earliest",
        "enable.auto.commit": False,
    }
    return Consumer(config)

