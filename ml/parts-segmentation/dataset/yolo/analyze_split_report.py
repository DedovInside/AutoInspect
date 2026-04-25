from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

import matplotlib.pyplot as plt
import pandas as pd

SPLITS = ("train", "val", "test")

COUNT_BLOCKS = {
    "views": "view_counts",
    "classes": "class_counts",
    "base_classes": "base_class_counts",
    "sides": "side_counts",
    "side_classes": "side_class_counts",
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Build split balance charts from split_report.json."
    )
    parser.add_argument(
        "--report",
        type=Path,
        required=True,
        help="Path to split_report.json.",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=None,
        help="Directory for charts and CSV files. Default: <report_dir>/split_analysis.",
    )
    parser.add_argument(
        "--min-total",
        type=int,
        default=1,
        help="Skip labels with total count below this value.",
    )
    parser.add_argument(
        "--dpi",
        type=int,
        default=160,
    )
    return parser.parse_args()


def load_report(path: Path) -> dict[str, Any]:
    with path.open("r", encoding="utf-8") as file:
        return json.load(file)


def collect_category_df(report: dict[str, Any], count_key: str) -> pd.DataFrame:
    rows = []

    labels = set()
    for split in SPLITS:
        labels.update(report["splits"][split].get(count_key, {}).keys())

    for label in sorted(labels):
        counts = {
            split: int(report["splits"][split].get(count_key, {}).get(label, 0))
            for split in SPLITS
        }
        total = sum(counts.values())
        if total == 0:
            continue

        row = {
            "label": label,
            "total": total,
            **{f"{split}_count": counts[split] for split in SPLITS},
            **{f"{split}_pct": counts[split] / total * 100.0 for split in SPLITS},
        }
        rows.append(row)

    return pd.DataFrame(rows)


def add_target_columns(df: pd.DataFrame, ratios: dict[str, float]) -> pd.DataFrame:
    result = df.copy()

    for split in SPLITS:
        target_pct = ratios[split] * 100.0
        result[f"{split}_target_pct"] = target_pct
        result[f"{split}_delta_pct"] = result[f"{split}_pct"] - target_pct

    result["max_abs_delta_pct"] = result[
        [f"{split}_delta_pct" for split in SPLITS]
    ].abs().max(axis=1)

    return result.sort_values("max_abs_delta_pct", ascending=False)


def plot_split_percentages(
    df: pd.DataFrame,
    title: str,
    output_path: Path,
    ratios: dict[str, float],
    dpi: int,
) -> None:
    if df.empty:
        return

    plot_df = df.sort_values("label").set_index("label")
    pct_cols = [f"{split}_pct" for split in SPLITS]

    height = max(5.0, min(28.0, 0.35 * len(plot_df) + 2.5))

    ax = plot_df[pct_cols].plot(kind="barh", figsize=(12, height))
    ax.set_title(title)
    ax.set_xlabel("% of this label assigned to split")
    ax.set_ylabel("")
    ax.legend(["train", "val", "test"])

    for split in SPLITS:
        ax.axvline(ratios[split] * 100.0, linestyle="--", linewidth=1)

    plt.tight_layout()
    output_path.parent.mkdir(parents=True, exist_ok=True)
    plt.savefig(output_path, dpi=dpi)
    plt.close()


def plot_deviation_from_target(
    df: pd.DataFrame,
    title: str,
    output_path: Path,
    dpi: int,
) -> None:
    if df.empty:
        return

    plot_df = df.sort_values("max_abs_delta_pct", ascending=True).set_index("label")
    delta_cols = [f"{split}_delta_pct" for split in SPLITS]

    height = max(5.0, min(28.0, 0.35 * len(plot_df) + 2.5))

    ax = plot_df[delta_cols].plot(kind="barh", figsize=(12, height))
    ax.set_title(title)
    ax.set_xlabel("percentage points vs target split ratio")
    ax.set_ylabel("")
    ax.axvline(0.0, linewidth=1)
    ax.legend(["train", "val", "test"])

    plt.tight_layout()
    output_path.parent.mkdir(parents=True, exist_ok=True)
    plt.savefig(output_path, dpi=dpi)
    plt.close()


def write_summary(report: dict[str, Any], output_dir: Path) -> None:
    rows = []

    for split in SPLITS:
        split_data = report["splits"][split]
        rows.append(
            {
                "split": split,
                "image_count": split_data["image_count"],
                "object_count": split_data["object_count"],
                "left_count": split_data["side_counts"].get("left", 0),
                "right_count": split_data["side_counts"].get("right", 0),
                "neutral_count": split_data["side_counts"].get("neutral", 0),
            }
        )

    pd.DataFrame(rows).to_csv(output_dir / "split_summary.csv", index=False)


def main() -> None:
    # Run: python analyze_split_report.py --report ../../images/out/split_report.json
    args = parse_args()

    report = load_report(args.report)
    output_dir = args.output_dir or args.report.parent / "split_analysis"
    output_dir.mkdir(parents=True, exist_ok=True)

    ratios = report["ratios"]

    write_summary(report, output_dir)

    for category_name, count_key in COUNT_BLOCKS.items():
        df = collect_category_df(report, count_key)

        if args.min_total > 1:
            df = df[df["total"] >= args.min_total].copy()

        df = add_target_columns(df, ratios)

        csv_path = output_dir / f"{category_name}.csv"
        df.to_csv(csv_path, index=False)

        plot_split_percentages(
            df=df,
            title=f"{category_name}: split percentage per label",
            output_path=output_dir / f"{category_name}_split_percentages.png",
            ratios=ratios,
            dpi=args.dpi,
        )

        plot_deviation_from_target(
            df=df,
            title=f"{category_name}: deviation from target ratio",
            output_path=output_dir / f"{category_name}_target_deviation.png",
            dpi=args.dpi,
        )

    print(f"Saved split analysis to: {output_dir}")


if __name__ == "__main__":
    main()