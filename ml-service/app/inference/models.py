from __future__ import annotations

from dataclasses import dataclass, field


@dataclass(frozen=True)
class CarInfo:
    make: str
    model: str
    generation: str
    year: int


@dataclass(frozen=True)
class AnalysisTask:
    correlation_id: str
    user_id: str
    car_info: CarInfo
    image_s3_keys: list[str]
    parts_model_s3_key: str
    parts_config_s3_key: str


@dataclass(frozen=True)
class PartAssociation:
    name: str
    side: str | None
    confidence: float


@dataclass(frozen=True)
class DamageInstance:
    id: str
    damage_type: str
    polygon: list[list[int]]
    bbox: list[int]
    confidence: float
    parts: list[PartAssociation] = field(default_factory=list)


@dataclass(frozen=True)
class PartSummary:
    name: str
    side: str | None
    damage_count: int
    damage_types: dict[str, int] = field(default_factory=dict)


@dataclass(frozen=True)
class ImageAnalysisResult:
    image_id: str
    image_uri: str
    width: int
    height: int
    damage_instances: list[DamageInstance] = field(default_factory=list)
    parts_summary: list[PartSummary] = field(default_factory=list)


@dataclass(frozen=True)
class AnalysisResult:
    correlation_id: str
    status: str
    error_message: str
    model_id: str
    model_version: str
    batch_id: str
    results: list[ImageAnalysisResult] = field(default_factory=list)
