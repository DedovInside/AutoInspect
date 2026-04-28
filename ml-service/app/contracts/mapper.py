from __future__ import annotations

from app.generated.analysis.v1 import request_pb2, result_pb2
from app.inference.models import (
    AnalysisResult,
    AnalysisTask,
    CarInfo,
    DamageInstance,
    ImageAnalysisResult,
    PartAssociation,
)


def parse_analysis_request(payload: bytes) -> AnalysisTask:
    msg = request_pb2.AnalysisRequest()
    msg.ParseFromString(payload)
    return request_to_analysis_task(msg)


def request_to_analysis_task(request: request_pb2.AnalysisRequest) -> AnalysisTask:
    if not request.correlation_id:
        raise ValueError("correlation_id is required")
    if not request.user_id:
        raise ValueError("user_id is required")
    if not request.HasField("car_info"):
        raise ValueError("car_info is required")
    if not request.parts_model_s3_key:
        raise ValueError("parts_model_s3_key is required")
    if not request.parts_config_s3_key:
        raise ValueError("parts_config_s3_key is required")

    car = request.car_info
    return AnalysisTask(
        correlation_id=request.correlation_id,
        user_id=request.user_id,
        car_info=CarInfo(
            make=car.make,
            model=car.model,
            generation=car.generation,
            year=int(car.year),
        ),
        image_s3_keys=list(request.image_s3_keys),
        parts_model_s3_key=request.parts_model_s3_key,
        parts_config_s3_key=request.parts_config_s3_key,
    )


def build_analysis_result_message(result: AnalysisResult) -> result_pb2.AnalysisResult:
    message = result_pb2.AnalysisResult(
        correlation_id=result.correlation_id,
        status=result.status,
        error_message=result.error_message,
        model_id=result.model_id,
        model_version=result.model_version,
        batch_id=result.batch_id,
    )

    for image_result in result.results:
        image_message = message.results.add(
            image_id=image_result.image_id,
            image_uri=image_result.image_uri,
        )
        image_message.image.width = int(image_result.width)
        image_message.image.height = int(image_result.height)

        for damage in image_result.damage_instances:
            _validate_bbox(damage.bbox)
            damage_message = image_message.damage_instances.add(
                id=damage.id,
                damage_type=damage.damage_type,
                confidence=float(damage.confidence),
            )
            damage_message.bbox.x_min = int(damage.bbox[0])
            damage_message.bbox.y_min = int(damage.bbox[1])
            damage_message.bbox.x_max = int(damage.bbox[2])
            damage_message.bbox.y_max = int(damage.bbox[3])

            for point in damage.polygon:
                _validate_point(point)
                damage_message.polygon.add(x=int(point[0]), y=int(point[1]))

            for part in damage.parts:
                part_message = damage_message.parts.add(
                    name=part.name,
                    confidence=float(part.confidence),
                )
                if part.side is not None:
                    part_message.side = part.side


    return message


def analysis_result_to_proto(result: AnalysisResult) -> result_pb2.AnalysisResult:
    return build_analysis_result_message(result)


def build_failed_result(correlation_id: str, error_message: str) -> AnalysisResult:
    return AnalysisResult(
        correlation_id=correlation_id,
        status="failed",
        error_message=error_message,
        model_id="",
        model_version="",
        batch_id=correlation_id or "",
        results=[],
    )


def _validate_bbox(bbox: list[int]) -> None:
    if len(bbox) != 4:
        raise ValueError("bbox must contain 4 integers [x_min, y_min, x_max, y_max]")


def _validate_point(point: list[int]) -> None:
    if len(point) != 2:
        raise ValueError("polygon point must contain 2 integers [x, y]")

