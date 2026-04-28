from __future__ import annotations

import argparse
from pathlib import Path
import sys

ROOT_DIR = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT_DIR))

from google.protobuf.json_format import MessageToJson

from app.contracts.mapper import build_analysis_result_message, parse_analysis_request
from app.inference.mock_pipeline import MockPipeline


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run mock pipeline for a protobuf request")
    parser.add_argument("--request", required=True, help="Path to request .bin file")
    parser.add_argument("--out", help="Optional output .bin file for result payload")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    payload = Path(args.request).read_bytes()
    task = parse_analysis_request(payload)
    result = MockPipeline().analyze(task)
    message = build_analysis_result_message(result)

    if args.out:
        out_path = Path(args.out)
        out_path.write_bytes(message.SerializeToString())
        print(f"Wrote result payload to {out_path}")

    print(MessageToJson(message, indent=2))


if __name__ == "__main__":
    main()
