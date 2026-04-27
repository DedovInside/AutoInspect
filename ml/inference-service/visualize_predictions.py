"""Визуализация backend JSON."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any
from urllib.parse import unquote, urlparse

import cv2
import numpy as np

IMAGE_EXTENSIONS = {".jpg", ".jpeg", ".png", ".bmp", ".webp", ".tif", ".tiff"}
REMOTE_SCHEMES = {"s3", "gs", "http", "https", "azure", "az", "ftp"}

# BGR colors for OpenCV. Reused cyclically for damage instances.
PALETTE: list[tuple[int, int, int]] = [
    (64, 160, 255),
    (80, 220, 120),
    (255, 120, 80),
    (220, 120, 255),
    (255, 210, 80),
    (80, 220, 220),
    (180, 180, 255),
    (160, 255, 160),
]


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Draw AutoInspect damage predictions and matched parts on source images."
    )
    parser.add_argument(
        "--predictions",
        required=True,
        help="Path to AutoInspect JSON produced by infer.py.",
    )
    parser.add_argument(
        "--images-root",
        default=None,
        help=(
            "Local folder with source images. Used when image_uri in JSON is not a local path "
            "or points to remote storage like s3://..."
        ),
    )
    parser.add_argument(
        "--output-dir",
        default="outputs/visualized",
        help="Directory for annotated images.",
    )
    parser.add_argument(
        "--alpha",
        type=float,
        default=0.35,
        help="Transparency of damage polygon fill. Use 0 to disable fill.",
    )
    parser.add_argument(
        "--no-fill",
        action="store_true",
        help="Do not fill damage polygons, draw only outlines and labels.",
    )
    parser.add_argument(
        "--no-bbox",
        action="store_true",
        help="Do not draw damage bounding boxes.",
    )
    parser.add_argument(
        "--label-max-parts",
        type=int,
        default=3,
        help="Maximum number of matched parts to show in the label for each damage.",
    )
    parser.add_argument(
        "--thickness",
        type=int,
        default=2,
        help="Line thickness for polygons and bounding boxes.",
    )
    parser.add_argument(
        "--recursive-search",
        action="store_true",
        help="Search source images recursively under --images-root when direct lookup fails.",
    )
    parser.add_argument(
        "--copy-empty",
        action="store_true",
        help="Also save images with no damage instances, with a small 'no damages' label.",
    )
    return parser.parse_args()


def uri_basename(uri: str) -> str:
    """Return filename from local paths, URLs, or s3://bucket/key style URIs."""
    parsed = urlparse(uri)
    if parsed.scheme:
        return Path(unquote(parsed.path)).name
    return Path(uri).name


def is_probably_remote(uri: str) -> bool:
    parsed = urlparse(uri)
    return parsed.scheme.lower() in REMOTE_SCHEMES


def resolve_image_path(image_result: dict[str, Any], images_root: Path | None, recursive_search: bool) -> Path | None:
    """Resolve the original image path for one JSON image result.

    Resolution order:
    1. Treat image_uri as a local path if it exists.
    2. Try --images-root / basename(image_uri).
    3. Try --images-root / image_id with common image extensions.
    4. Optional recursive search by basename and image_id.
    """
    image_uri = str(image_result.get("image_uri") or "")
    image_id = str(image_result.get("image_id") or "")
    basename = uri_basename(image_uri) if image_uri else ""

    if image_uri and not is_probably_remote(image_uri):
        local_candidate = Path(image_uri)
        if local_candidate.exists() and local_candidate.is_file():
            return local_candidate

    if images_root is None:
        return None

    candidates: list[Path] = []
    if basename:
        candidates.append(images_root / basename)
    if image_id:
        candidates.extend(images_root / f"{image_id}{ext}" for ext in sorted(IMAGE_EXTENSIONS))

    for candidate in candidates:
        if candidate.exists() and candidate.is_file():
            return candidate

    if recursive_search:
        if basename:
            matches = [p for p in images_root.rglob(basename) if p.is_file()]
            if matches:
                return matches[0]
        if image_id:
            matches = [p for p in images_root.rglob(f"{image_id}.*") if p.suffix.lower() in IMAGE_EXTENSIONS]
            if matches:
                return matches[0]

    return None


def clamp(value: int, low: int, high: int) -> int:
    return max(low, min(high, value))


def safe_bbox(bbox: list[Any] | None, polygon: list[list[Any]] | None, width: int, height: int) -> list[int]:
    if bbox and len(bbox) == 4:
        x1, y1, x2, y2 = [int(round(float(v))) for v in bbox]
    elif polygon:
        xs = [int(round(float(p[0]))) for p in polygon]
        ys = [int(round(float(p[1]))) for p in polygon]
        x1, y1, x2, y2 = min(xs), min(ys), max(xs), max(ys)
    else:
        x1, y1, x2, y2 = 0, 0, 0, 0

    return [
        clamp(x1, 0, max(0, width - 1)),
        clamp(y1, 0, max(0, height - 1)),
        clamp(x2, 0, max(0, width - 1)),
        clamp(y2, 0, max(0, height - 1)),
    ]


def polygon_points(polygon: list[list[Any]] | None, width: int, height: int) -> np.ndarray | None:
    if not polygon or len(polygon) < 3:
        return None

    points: list[list[int]] = []
    for point in polygon:
        if len(point) < 2:
            continue
        x = clamp(int(round(float(point[0]))), 0, max(0, width - 1))
        y = clamp(int(round(float(point[1]))), 0, max(0, height - 1))
        points.append([x, y])

    if len(points) < 3:
        return None
    return np.asarray(points, dtype=np.int32).reshape((-1, 1, 2))


def fmt_conf(value: Any) -> str:
    try:
        return f"{float(value):.2f}"
    except (TypeError, ValueError):
        return "n/a"


def format_part(part: dict[str, Any]) -> str:
    name = str(part.get("name", "unknown"))
    side = part.get("side")
    title = f"{side} {name}" if side else name
    return f"{title} {fmt_conf(part.get('confidence'))}"


def make_label_lines(damage: dict[str, Any], max_parts: int) -> list[str]:
    damage_id = str(damage.get("id") or "damage")
    damage_type = str(damage.get("damage_type") or "unknown")
    damage_conf = fmt_conf(damage.get("confidence"))

    parts = damage.get("parts") or []
    parts_text = ", ".join(format_part(part) for part in parts[:max_parts])
    if len(parts) > max_parts:
        parts_text += f", +{len(parts) - max_parts} more"
    if not parts_text:
        parts_text = "unknown"

    # Two compact lines are usually readable even on mobile-sized images.
    return [
        f"{damage_id} | {damage_type} | conf {damage_conf}",
        f"parts: {parts_text}",
    ]


def draw_label(
    image: np.ndarray,
    anchor_bbox: list[int],
    lines: list[str],
    color: tuple[int, int, int],
    font_scale: float,
    thickness: int,
) -> None:
    font = cv2.FONT_HERSHEY_SIMPLEX
    height, width = image.shape[:2]
    padding = max(4, int(round(6 * font_scale)))
    line_gap = max(3, int(round(5 * font_scale)))

    text_sizes = [cv2.getTextSize(line, font, font_scale, thickness)[0] for line in lines]
    label_width = max(size[0] for size in text_sizes) + padding * 2
    label_height = sum(size[1] for size in text_sizes) + line_gap * (len(lines) - 1) + padding * 2

    x1, y1, x2, y2 = anchor_bbox
    label_x = clamp(x1, 0, max(0, width - label_width - 1))
    preferred_y = y1 - label_height - 4
    label_y = preferred_y if preferred_y >= 0 else min(height - label_height - 1, y2 + 4)
    label_y = clamp(label_y, 0, max(0, height - label_height - 1))

    bg_color = tuple(max(0, int(channel * 0.28)) for channel in color)
    cv2.rectangle(
        image,
        (label_x, label_y),
        (label_x + label_width, label_y + label_height),
        bg_color,
        thickness=-1,
    )
    cv2.rectangle(
        image,
        (label_x, label_y),
        (label_x + label_width, label_y + label_height),
        color,
        thickness=max(1, thickness - 1),
    )

    cursor_y = label_y + padding
    for line, size in zip(lines, text_sizes):
        cursor_y += size[1]
        cv2.putText(
            image,
            line,
            (label_x + padding, cursor_y),
            font,
            font_scale,
            (255, 255, 255),
            thickness,
            lineType=cv2.LINE_AA,
        )
        cursor_y += line_gap


def draw_no_damages_label(image: np.ndarray) -> None:
    font = cv2.FONT_HERSHEY_SIMPLEX
    height, width = image.shape[:2]
    font_scale = max(0.45, min(0.8, width / 1200))
    thickness = 2
    text = "no damages"
    text_size = cv2.getTextSize(text, font, font_scale, thickness)[0]
    padding = 8
    cv2.rectangle(image, (10, 10), (10 + text_size[0] + padding * 2, 10 + text_size[1] + padding * 2), (40, 40, 40), -1)
    cv2.putText(image, text, (10 + padding, 10 + padding + text_size[1]), font, font_scale, (255, 255, 255), thickness, cv2.LINE_AA)


def visualize_one_image(
    image_path: Path,
    image_result: dict[str, Any],
    output_dir: Path,
    alpha: float,
    fill_polygons: bool,
    draw_bbox: bool,
    label_max_parts: int,
    thickness: int,
) -> Path:
    image = cv2.imread(str(image_path))
    if image is None:
        raise ValueError(f"Could not read image: {image_path}")

    height, width = image.shape[:2]
    damages = image_result.get("damage_instances") or []

    font_scale = max(0.42, min(0.72, width / 1350))
    line_thickness = max(1, thickness)

    # First pass: draw only transparent polygon fills.
    # Second pass: draw outlines, bboxes and labels on top, so text stays sharp.
    canvas = image.copy()
    if damages and fill_polygons and alpha > 0:
        overlay = image.copy()
        for idx, damage in enumerate(damages, start=1):
            color = PALETTE[(idx - 1) % len(PALETTE)]
            points = polygon_points(damage.get("polygon") or [], width, height)
            if points is not None:
                cv2.fillPoly(overlay, [points], color)
        canvas = cv2.addWeighted(overlay, float(alpha), image, 1.0 - float(alpha), 0)

    for idx, damage in enumerate(damages, start=1):
        color = PALETTE[(idx - 1) % len(PALETTE)]
        polygon = damage.get("polygon") or []
        bbox = safe_bbox(damage.get("bbox"), polygon, width, height)
        points = polygon_points(polygon, width, height)

        if points is not None:
            cv2.polylines(canvas, [points], isClosed=True, color=color, thickness=line_thickness, lineType=cv2.LINE_AA)

        if draw_bbox:
            cv2.rectangle(canvas, (bbox[0], bbox[1]), (bbox[2], bbox[3]), color, line_thickness)

        draw_label(
            image=canvas,
            anchor_bbox=bbox,
            lines=make_label_lines(damage, max_parts=label_max_parts),
            color=color,
            font_scale=font_scale,
            thickness=1,
        )

    if not damages:
        draw_no_damages_label(canvas)

    image_id = str(image_result.get("image_id") or image_path.stem)
    output_name = f"{image_id}_{image_path.stem}_annotated{image_path.suffix.lower() or '.jpg'}"
    output_path = output_dir / output_name
    output_path.parent.mkdir(parents=True, exist_ok=True)

    ok = cv2.imwrite(str(output_path), canvas)
    if not ok:
        raise ValueError(f"Could not write output image: {output_path}")
    return output_path

def main() -> None:
    args = parse_args()
    predictions_path = Path(args.predictions)
    images_root = Path(args.images_root) if args.images_root else None
    output_dir = Path(args.output_dir)

    with predictions_path.open("r", encoding="utf-8") as f:
        payload = json.load(f)

    image_results = payload.get("results") or []
    if not image_results:
        raise ValueError(f"No results[] found in predictions JSON: {predictions_path}")

    saved_paths: list[Path] = []
    missing_images: list[str] = []
    skipped_empty = 0

    for image_result in image_results:
        damages = image_result.get("damage_instances") or []
        if not damages and not args.copy_empty:
            skipped_empty += 1
            continue

        image_path = resolve_image_path(
            image_result=image_result,
            images_root=images_root,
            recursive_search=args.recursive_search,
        )
        if image_path is None:
            missing_images.append(str(image_result.get("image_uri") or image_result.get("image_id") or "unknown"))
            continue

        saved_paths.append(
            visualize_one_image(
                image_path=image_path,
                image_result=image_result,
                output_dir=output_dir,
                alpha=args.alpha,
                fill_polygons=not args.no_fill,
                draw_bbox=not args.no_bbox,
                label_max_parts=args.label_max_parts,
                thickness=args.thickness,
            )
        )

    for path in saved_paths:
        print(f"Saved visualization: {path}")

    if skipped_empty:
        print(f"Skipped images without damages: {skipped_empty}. Use --copy-empty to save them too.")

    if missing_images:
        print("Could not resolve source images for:")
        for item in missing_images:
            print(f"  - {item}")
        raise SystemExit(1)

    print(f"Done. Saved {len(saved_paths)} annotated image(s) to: {output_dir}")


if __name__ == "__main__":
    main()
