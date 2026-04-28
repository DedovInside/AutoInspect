from __future__ import annotations

from app.inference.models import (
    AnalysisResult,
    AnalysisTask,
    DamageInstance,
    ImageAnalysisResult,
    PartAssociation,
    PartSummary,
)


class MockPipeline:
    def analyze(self, task: AnalysisTask) -> AnalysisResult:
        image_key = task.image_s3_keys[0] if task.image_s3_keys else ""
        image_id = "image-1"

        damage = DamageInstance(
            id=f"{image_id}_damage_1",
            damage_type="scratch",
            polygon=[[10, 10], [40, 10], [40, 30], [10, 30]],
            bbox=[10, 10, 40, 30],
            confidence=0.9,
            parts=[PartAssociation(name="hood", side=None, confidence=0.85)],
        )

        summary = PartSummary(
            name="hood",
            side=None,
            damage_count=1,
            damage_types={"scratch": 1},
        )

        image_result = ImageAnalysisResult(
            image_id=image_id,
            image_uri=image_key,
            width=640,
            height=480,
            damage_instances=[damage],
            parts_summary=[summary],
        )

        return AnalysisResult(
            correlation_id=task.correlation_id,
            status="ok",
            error_message="",
            model_id="mock",
            model_version="v0",
            batch_id=task.correlation_id or "batch-1",
            results=[image_result],
        )

