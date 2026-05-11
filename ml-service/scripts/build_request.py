from __future__ import annotations

import argparse
from pathlib import Path
import sys

ROOT_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT_DIR))

from app.generated.analysis.v1 import request_pb2


def build_request(args: argparse.Namespace) -> request_pb2.AnalysisRequest:
    request = request_pb2.AnalysisRequest(
        correlation_id=args.correlation_id,
        user_id=args.user_id,
        image_s3_keys=args.image_s3_keys,
        parts_model_s3_key=args.parts_model_s3_key,
        parts_config_s3_key=args.parts_config_s3_key,
    )
    request.car_info.make = args.car_make
    request.car_info.model = args.car_model
    request.car_info.generation = args.car_generation
    request.car_info.year = args.car_year
    return request


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Build AnalysisRequest protobuf payload")
    parser.add_argument("--out", required=True, help="Output .bin file for request payload")
    parser.add_argument("--correlation-id", required=True, help="Example: corr-1")
    parser.add_argument("--user-id", required=True, help="Example: user-1")
    parser.add_argument("--image-s3-keys", nargs="+", required=True, help="Example: uploads/a.jpg uploads/b.jpg")
    parser.add_argument("--parts-model-s3-key", required=True, help="Example: models/v1/parts_segmentation.pt")
    parser.add_argument("--parts-config-s3-key", required=True, help="Example: configs/parts_config.json")
    parser.add_argument("--car-make", required=True, help="Example: Toyota")
    parser.add_argument("--car-model", required=True, help="Example: Camry")
    parser.add_argument("--car-generation", required=True, help="Example: XV70")
    parser.add_argument("--car-year", type=int, required=True, help="Example: 2022")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    request = build_request(args)
    out_path = Path(args.out)
    out_path.write_bytes(request.SerializeToString())
    print(f"Wrote request payload to {out_path}")


if __name__ == "__main__":
    main()
