from __future__ import annotations

from collections import Counter, defaultdict
from typing import Any

import numpy as np

from .schema import ImagePredictions, InstancePrediction
from .yolo_utils import split_part_name


def polygon_bbox(polygon: list[list[int]]) -> list[int]:
    if not polygon:
        return [0, 0, 0, 0]
    xs = [p[0] for p in polygon]
    ys = [p[1] for p in polygon]
    return [min(xs), min(ys), max(xs), max(ys)]


def bbox_area(bbox: list[int]) -> int:
    x1, y1, x2, y2 = bbox
    return max(0, x2 - x1) * max(0, y2 - y1)


def bbox_intersection_area(a: list[int], b: list[int]) -> int:
    ax1, ay1, ax2, ay2 = a
    bx1, by1, bx2, by2 = b
    x1 = max(ax1, bx1)
    y1 = max(ay1, by1)
    x2 = min(ax2, bx2)
    y2 = min(ay2, by2)
    return max(0, x2 - x1) * max(0, y2 - y1)


def overlap_by_mask(damage: InstancePrediction, part: InstancePrediction) -> float:
    """Доля повреждения на детали авто"""
    if damage.mask is not None and part.mask is not None:
        damage_mask = damage.mask.astype(bool)
        part_mask = part.mask.astype(bool)
        damage_area = int(damage_mask.sum())
        if damage_area <= 0:
            return 0.0
        intersection = int(np.logical_and(damage_mask, part_mask).sum())
        return intersection / damage_area

    # Fallback
    damage_area = bbox_area(damage.bbox_xyxy)
    if damage_area <= 0:
        return 0.0
    return bbox_intersection_area(damage.bbox_xyxy, part.bbox_xyxy) / damage_area


def assignment_score(overlap_ratio: float, part_confidence: float) -> float:
    """Оценка для API: насколько уверенно это повреждение относится к этой детали.

    Умножение намеренно консервативное: малое пересечение или низкая уверенность по детали
    снижают итоговую уверенность привязки"""
    return round(float(overlap_ratio) * float(part_confidence), 6)


def build_damage_record(
    image_id: str,
    damage_idx: int,
    damage: InstancePrediction,
    parts: list[InstancePrediction],
    min_overlap: float,
    min_assignment_score: float,
    max_parts_per_damage: int,
) -> dict[str, Any]:
    matched_parts: list[dict[str, Any]] = []

    for part in parts:
        overlap = overlap_by_mask(damage, part)
        if overlap < min_overlap:
            continue

        score = assignment_score(overlap, part.confidence)
        if score < min_assignment_score:
            continue

        part_name, side = split_part_name(part.class_name)
        part_record: dict[str, Any] = {
            "name": part_name,
            "confidence": score,
        }
        if side:
            part_record["side"] = side

        part_record["overlap_ratio"] = round(float(overlap), 6)
        matched_parts.append(part_record)

    matched_parts.sort(key=lambda item: item["confidence"], reverse=True)
    matched_parts = matched_parts[:max_parts_per_damage]

    return {
        "id": f"{image_id}_damage_{damage_idx}",
        "damage_type": damage.class_name,
        "polygon": damage.polygon_xy,
        "bbox": damage.bbox_xyxy,
        "confidence": damage.confidence,
        "parts": matched_parts,
    }


def build_parts_summary(damage_records: list[dict[str, Any]]) -> list[dict[str, Any]]:
    summary: dict[tuple[str, str | None], Counter[str]] = defaultdict(Counter)

    for damage in damage_records:
        damage_type = damage["damage_type"]
        for part in damage.get("parts", []):
            key = (part["name"], part.get("side"))
            summary[key][damage_type] += 1

    records: list[dict[str, Any]] = []
    for (name, side), counter in sorted(summary.items(), key=lambda item: (item[0][0], item[0][1] or "")):
        item: dict[str, Any] = {
            "name": name,
            "damage_count": int(sum(counter.values())),
            "damage_types": dict(counter),
        }
        if side:
            item["side"] = side
        records.append(item)

    return records


def build_image_result(
    image_id: str,
    image_uri: str,
    parts_predictions: ImagePredictions,
    damage_predictions: ImagePredictions,
    min_overlap: float,
    min_assignment_score: float,
    max_parts_per_damage: int,
) -> dict[str, Any]:
    damage_records = [
        build_damage_record(
            image_id=image_id,
            damage_idx=idx,
            damage=damage,
            parts=parts_predictions.instances,
            min_overlap=min_overlap,
            min_assignment_score=min_assignment_score,
            max_parts_per_damage=max_parts_per_damage,
        )
        for idx, damage in enumerate(damage_predictions.instances, start=1)
    ]

    return {
        "image_id": image_id,
        "image_uri": image_uri,
        "image": {
            "width": int(damage_predictions.width or parts_predictions.width),
            "height": int(damage_predictions.height or parts_predictions.height),
        },
        "damage_instances": damage_records,
        "parts_summary": build_parts_summary(damage_records),
    }


def build_batch_summary(image_results: list[dict[str, Any]]) -> dict[str, Any]:
    damage_type_counter: Counter[str] = Counter()
    part_keys: set[tuple[str, str | None]] = set()
    damage_count = 0

    for image_result in image_results:
        for damage in image_result.get("damage_instances", []):
            damage_count += 1
            damage_type_counter[damage["damage_type"]] += 1
            for part in damage.get("parts", []):
                part_keys.add((part["name"], part.get("side")))

    parts: list[dict[str, Any]] = []
    for name, side in sorted(part_keys, key=lambda item: (item[0], item[1] or "")):
        item: dict[str, Any] = {"name": name}
        if side:
            item["side"] = side
        parts.append(item)

    return {
        "image_count": len(image_results),
        "damage_count": damage_count,
        "damage_types": dict(damage_type_counter),
        "parts": parts,
    }
