from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Any


@dataclass(frozen=True)
class InstancePrediction:
    """Prediction for a single object from Ultralytics Results."""

    class_id: int
    class_name: str
    confidence: float
    bbox_xyxy: list[int]
    polygon_xy: list[list[int]]
    mask: Any


@dataclass(frozen=True)
class ImagePredictions:
    """All predictions for a single image."""

    image_path: Path | None
    width: int
    height: int
    instances: list[InstancePrediction]
