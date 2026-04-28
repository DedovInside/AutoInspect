from __future__ import annotations

import argparse
import sys
from pathlib import Path

ROOT_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT_DIR))

from google.protobuf.json_format import MessageToJson

from app.contracts.mapper import build_analysis_result_message
from app.inference.adapter import AutoInspectPipeline
from app.inference.core.yolo_utils import collect_images


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run real inference adapter on local files")
    parser.add_argument("--source", required=True, help="Image file or directory")
    parser.add_argument("--parts-model", required=True, help="Path to parts model .pt")
    parser.add_argument("--damage-model", required=True, help="Path to damage model .pt")
    parser.add_argument("--parts-config", required=True, help="Path to parts_config.json")
    parser.add_argument("--damage-config", required=True, help="Path to damage_config.json")
    parser.add_argument("--matching-config", required=True, help="Path to matching_config.json")
    parser.add_argument("--model-id", default="general")
    parser.add_argument("--model-version", default="v1")
    parser.add_argument("--batch-id", default="batch-local")
    parser.add_argument("--correlation-id", default="corr-local")
    parser.add_argument("--out", help="Optional output .bin file for result protobuf")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    image_paths = collect_images(args.source)
    image_uris = [str(path) for path in image_paths]

    pipeline = AutoInspectPipeline()
    result = pipeline.analyze_batch(
        image_paths=image_paths,
        image_uris=image_uris,
        parts_model_path=Path(args.parts_model),
        damage_model_path=Path(args.damage_model),
        parts_inference_config_path=Path(args.parts_config),
        damage_inference_config_path=Path(args.damage_config),
        matching_config_path=Path(args.matching_config),
        model_id=args.model_id,
        model_version=args.model_version,
        batch_id=args.batch_id,
        correlation_id=args.correlation_id,
    )

    message = build_analysis_result_message(result)
    if args.out:
        out_path = Path(args.out)
        out_path.write_bytes(message.SerializeToString())
        print(f"Wrote result payload to {out_path}")

    print(MessageToJson(message, indent=2))


if __name__ == "__main__":
    main()
