from __future__ import annotations


from confluent_kafka import KafkaException

from app.contracts.mapper import build_analysis_result_message, build_failed_result, parse_analysis_request
from app.inference.mock_pipeline import MockPipeline
from app.kafka.consumer import create_consumer
from app.kafka.producer import create_producer
from app.settings import get_settings


class KafkaWorker:
    def __init__(self) -> None:
        self._settings = get_settings()
        self._consumer = create_consumer(self._settings)
        self._producer = create_producer(self._settings)
        self._pipeline = MockPipeline()

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
                    result = self._pipeline.analyze(task)
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

