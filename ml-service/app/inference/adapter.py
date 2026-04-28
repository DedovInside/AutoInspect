from __future__ import annotations

from dataclasses import dataclass
import json
from pathlib import Path
from typing import List, Optional

from app.inference.core import matcher, yolo_utils
from app.inference.core.yolo_utils import load_yolo, predict_one_image

from app.inference.models import (
    AnalysisResult,
    CarInfo,
    DamageInstance,
    ImageAnalysisResult,
    PartAssociation,
    PartSummary,
)


@dataclass
class PipelineSettings:
    parts_imgsz: int = 768
    damage_imgsz: int = 896
    parts_conf: float = 0.25
    damage_conf: float = 0.25
    parts_iou: float = 0.70
    damage_iou: float = 0.70
    parts_device: Optional[str] = None
    damage_device: Optional[str] = None
    parts_retina_masks: bool = True
    damage_retina_masks: bool = True
    min_overlap: float = 0.05
    min_assignment_score: float = 0.05
    max_parts_per_damage: int = 3
    parts_max_det: int = 300
    damage_max_det: int = 300


class AutoInspectPipeline:
    def __init__(self, settings: PipelineSettings | None = None):
        self.settings = settings or PipelineSettings()
        self._parts_model_cache: dict[str, object] = {}
        self._damage_model = None

    def _get_parts_model(self, model_path: Path):
        key = str(model_path)
        if key in self._parts_model_cache:
            return self._parts_model_cache[key]
        model = load_yolo(str(model_path), yolo_utils.PARTS_REPO_ID, yolo_utils.PARTS_FILENAME)
        self._parts_model_cache[key] = model
        return model

    def _get_damage_model(self, model_path: Path):
        key = str(model_path)
        if getattr(self, "_damage_model_key", None) == key and self._damage_model is not None:
            return self._damage_model
        model = load_yolo(str(model_path), yolo_utils.DAMAGE_REPO_ID, yolo_utils.DAMAGE_FILENAME)
        self._damage_model = model
        self._damage_model_key = key
        return model

    def _apply_inference_config(self, config_path: Path, target: str) -> None:
        with config_path.open("r", encoding="utf-8") as handle:
            payload = json.load(handle)

        inference = payload.get("inference", {})
        if not isinstance(inference, dict):
            raise ValueError("parts_config.inference must be an object")

        if target == "parts":
            self.settings.parts_imgsz = int(inference.get("imgsz", self.settings.parts_imgsz))
            self.settings.parts_conf = float(inference.get("conf", self.settings.parts_conf))
            self.settings.parts_iou = float(inference.get("iou", self.settings.parts_iou))
            self.settings.parts_max_det = int(inference.get("max_det", self.settings.parts_max_det))
            self.settings.parts_retina_masks = bool(
                inference.get("retina_masks", self.settings.parts_retina_masks)
            )

            device = inference.get("device", self.settings.parts_device)
            if device in (None, "", "auto"):
                self.settings.parts_device = None
            else:
                self.settings.parts_device = str(device)
            return

        if target == "damage":
            self.settings.damage_imgsz = int(inference.get("imgsz", self.settings.damage_imgsz))
            self.settings.damage_conf = float(inference.get("conf", self.settings.damage_conf))
            self.settings.damage_iou = float(inference.get("iou", self.settings.damage_iou))
            self.settings.damage_max_det = int(inference.get("max_det", self.settings.damage_max_det))
            self.settings.damage_retina_masks = bool(
                inference.get("retina_masks", self.settings.damage_retina_masks)
            )

            device = inference.get("device", self.settings.damage_device)
            if device in (None, "", "auto"):
                self.settings.damage_device = None
            else:
                self.settings.damage_device = str(device)
            return

        raise ValueError("target must be 'parts' or 'damage'")

    def _apply_matching_config(self, payload: dict) -> None:
        if "min_overlap" in payload:
            self.settings.min_overlap = float(payload["min_overlap"])
        if "min_assignment_score" in payload:
            self.settings.min_assignment_score = float(payload["min_assignment_score"])
        if "max_parts_per_damage" in payload:
            self.settings.max_parts_per_damage = int(payload["max_parts_per_damage"])

    def analyze_batch(
        self,
        image_paths: List[Path],
        image_uris: List[str],
        parts_model_path: Path,
        damage_model_path: Path,
        parts_inference_config_path: Path,
        damage_inference_config_path: Path,
        model_id: str,
        model_version: str,
        batch_id: str,
        correlation_id: str,
    ) -> AnalysisResult:
        # validate inputs
        if not parts_inference_config_path.exists():
            raise FileNotFoundError(f"parts_config not found: {parts_inference_config_path}")
        self._apply_inference_config(parts_inference_config_path, target="parts")

        if not damage_inference_config_path.exists():
            raise FileNotFoundError(f"damage_config not found: {damage_inference_config_path}")
        with damage_inference_config_path.open("r", encoding="utf-8") as handle:
            damage_payload = json.load(handle)
        if not isinstance(damage_payload, dict):
            raise ValueError("damage_config must be an object")
        self._apply_inference_config(damage_inference_config_path, target="damage")
        matching = damage_payload.get("matching")
        if matching is not None:
            if not isinstance(matching, dict):
                raise ValueError("damage_config.matching must be an object")
            self._apply_matching_config(matching)

        parts_model = self._get_parts_model(parts_model_path)
        damage_model = self._get_damage_model(damage_model_path)

        image_results: list[ImageAnalysisResult] = []

        for idx, image_path in enumerate(image_paths):
            parts_predictions = predict_one_image(
                parts_model,
                image_path,
                imgsz=self.settings.parts_imgsz,
                conf=self.settings.parts_conf,
                iou=self.settings.parts_iou,
                device=self.settings.parts_device,
                retina_masks=self.settings.parts_retina_masks,
                max_det=self.settings.parts_max_det,
            )

            damage_predictions = predict_one_image(
                damage_model,
                image_path,
                imgsz=self.settings.damage_imgsz,
                conf=self.settings.damage_conf,
                iou=self.settings.damage_iou,
                device=self.settings.damage_device,
                retina_masks=self.settings.damage_retina_masks,
                max_det=self.settings.damage_max_det,
            )

            image_uri = image_uris[idx] if idx < len(image_uris) else str(image_path)

            image_dict = matcher.build_image_result(
                image_id=str(idx + 1),
                image_uri=image_uri,
                parts_predictions=parts_predictions,
                damage_predictions=damage_predictions,
                min_overlap=self.settings.min_overlap,
                min_assignment_score=self.settings.min_assignment_score,
                max_parts_per_damage=self.settings.max_parts_per_damage,
            )

            # convert dicts to dataclasses
            damage_instances: list[DamageInstance] = []
            for dmg in image_dict.get("damage_instances", []):
                parts: list[PartAssociation] = []
                for p in dmg.get("parts", []):
                    parts.append(
                        PartAssociation(name=p["name"], side=p.get("side"), confidence=float(p["confidence"]))
                    )
                damage_instances.append(
                    DamageInstance(
                        id=dmg["id"],
                        damage_type=dmg["damage_type"],
                        polygon=dmg.get("polygon", []),
                        bbox=dmg.get("bbox", [0, 0, 0, 0]),
                        confidence=float(dmg.get("confidence", 0.0)),
                        parts=parts,
                    )
                )

            parts_summary: list[PartSummary] = []
            for ps in image_dict.get("parts_summary", []):
                parts_summary.append(
                    PartSummary(
                        name=ps["name"],
                        side=ps.get("side"),
                        damage_count=int(ps.get("damage_count", 0)),
                        damage_types=dict(ps.get("damage_types", {})),
                    )
                )

            image_result = ImageAnalysisResult(
                image_id=image_dict.get("image_id"),
                image_uri=image_dict.get("image_uri"),
                width=int(image_dict.get("image", {}).get("width", 0)),
                height=int(image_dict.get("image", {}).get("height", 0)),
                damage_instances=damage_instances,
                parts_summary=parts_summary,
            )

            image_results.append(image_result)

        result = AnalysisResult(
            correlation_id=correlation_id,
            status="ok",
            error_message="",
            model_id=model_id,
            model_version=model_version,
            batch_id=batch_id,
            results=image_results,
        )

        return result

