import unittest

from app.contracts.mapper import (
    analysis_result_to_proto,
    build_analysis_result_message,
    parse_analysis_request,
    request_to_analysis_task,
)
from app.generated.analysis.v1 import request_pb2
from app.inference.models import (
    AnalysisResult,
    DamageInstance,
    ImageAnalysisResult,
    PartAssociation,
)


class MapperTests(unittest.TestCase):
    def test_request_to_analysis_task(self) -> None:
        request = request_pb2.AnalysisRequest(
            correlation_id="corr-1",
            user_id="user-1",
            image_s3_keys=["uploads/a.jpg", "uploads/b.jpg"],
            parts_model_s3_key="models/v1/parts_segmentation.pt",
            parts_config_s3_key="configs/parts_config.json",
        )
        request.car_info.make = "Toyota"
        request.car_info.model = "Camry"
        request.car_info.generation = "XV70"
        request.car_info.year = 2022

        task = request_to_analysis_task(request)

        self.assertEqual(task.correlation_id, "corr-1")
        self.assertEqual(task.user_id, "user-1")
        self.assertEqual(task.car_info.make, "Toyota")
        self.assertEqual(task.image_s3_keys, ["uploads/a.jpg", "uploads/b.jpg"])
        self.assertEqual(task.parts_model_s3_key, "models/v1/parts_segmentation.pt")

    def test_parse_analysis_request(self) -> None:
        request = request_pb2.AnalysisRequest(
            correlation_id="corr-2",
            user_id="user-2",
            image_s3_keys=["uploads/c.jpg"],
            parts_model_s3_key="models/v2/parts_segmentation.pt",
            parts_config_s3_key="configs/parts_config.json",
        )
        request.car_info.make = "BMW"
        request.car_info.model = "X5"
        request.car_info.generation = "G05"
        request.car_info.year = 2021

        task = parse_analysis_request(request.SerializeToString())

        self.assertEqual(task.correlation_id, "corr-2")
        self.assertEqual(task.car_info.model, "X5")

    def test_analysis_result_to_proto(self) -> None:
        result = AnalysisResult(
            correlation_id="corr-1",
            status="ok",
            error_message="",
            model_id="general",
            model_version="v1.0.0",
            batch_id="batch-1",
            results=[
                ImageAnalysisResult(
                    image_id="img-1",
                    image_uri="s3://bucket/uploads/a.jpg",
                    width=640,
                    height=480,
                    damage_instances=[
                        DamageInstance(
                            id="dmg-1",
                            damage_type="dent",
                            polygon=[[10, 10], [30, 10], [30, 20]],
                            bbox=[10, 10, 30, 20],
                            confidence=0.95,
                            parts=[
                                PartAssociation(name="hood", side=None, confidence=0.91),
                                PartAssociation(name="front-door", side="left", confidence=0.7),
                            ],
                        )
                    ],
                )
            ],
        )

        proto = build_analysis_result_message(result)

        self.assertEqual(proto.correlation_id, "corr-1")
        self.assertEqual(proto.status, "ok")
        self.assertEqual(proto.model_id, "general")
        self.assertEqual(len(proto.results), 1)
        self.assertEqual(proto.results[0].image.width, 640)
        self.assertEqual(proto.results[0].damage_instances[0].bbox.x_min, 10)
        self.assertEqual(proto.results[0].damage_instances[0].parts[1].side, "left")

    def test_invalid_bbox_raises(self) -> None:
        result = AnalysisResult(
            correlation_id="corr-1",
            status="ok",
            error_message="",
            model_id="general",
            model_version="v1.0.0",
            batch_id="batch-1",
            results=[
                ImageAnalysisResult(
                    image_id="img-1",
                    image_uri="s3://bucket/uploads/a.jpg",
                    width=640,
                    height=480,
                    damage_instances=[
                        DamageInstance(
                            id="dmg-1",
                            damage_type="dent",
                            polygon=[[10, 10], [30, 10]],
                            bbox=[10, 10, 30],
                            confidence=0.95,
                            parts=[],
                        )
                    ],
                )
            ],
        )

        with self.assertRaises(ValueError):
            analysis_result_to_proto(result)


if __name__ == "__main__":
    unittest.main()
