import argparse
import os
import sys
from typing import Dict, List, Tuple

import torch
import torch.nn as nn
from PIL import Image, ImageOps, ImageStat
from torchvision import models, transforms


IMAGENET_MEAN = [0.485, 0.456, 0.406]
IMAGENET_STD = [0.229, 0.224, 0.225]

H_LABELS = ["left", "center", "right"]
V_LABELS = ["front", "center", "back"]

VIEW_TO_AXES: Dict[str, Tuple[str, str]] = {
    "front": ("center", "front"),
    "front-left": ("left", "front"),
    "left": ("left", "center"),
    "back-left": ("left", "back"),
    "back": ("center", "back"),
    "back-right": ("right", "back"),
    "right": ("right", "center"),
    "front-right": ("right", "front"),
}

PAIR_TO_VIEW = {value: key for key, value in VIEW_TO_AXES.items()}
H_TO_ID = {name: idx for idx, name in enumerate(H_LABELS)}
V_TO_ID = {name: idx for idx, name in enumerate(V_LABELS)}


def get_device(device_arg: str) -> torch.device:
    if device_arg != "auto":
        return torch.device(device_arg)

    if torch.cuda.is_available():
        return torch.device("cuda")
    if torch.backends.mps.is_available():
        return torch.device("mps")
    return torch.device("cpu")


def pad_to_square_with_mean_color(image: Image.Image) -> Image.Image:
    image = image.convert("RGB")
    stat = ImageStat.Stat(image)
    mean_vals = stat.mean[:3]
    mean_color = (int(mean_vals[0]), int(mean_vals[1]), int(mean_vals[2]))

    w, h = image.size
    if w == h:
        return image

    if w > h:
        total_pad = w - h
        top = total_pad // 2
        bottom = total_pad - top
        return ImageOps.expand(image, border=(0, top, 0, bottom), fill=mean_color)

    total_pad = h - w
    left = total_pad // 2
    right = total_pad - left
    return ImageOps.expand(image, border=(left, 0, right, 0), fill=mean_color)


def build_transform(img_size: int) -> transforms.Compose:
    return transforms.Compose(
        [
            transforms.Resize((img_size, img_size)),
            transforms.ToTensor(),
            transforms.Normalize(IMAGENET_MEAN, IMAGENET_STD),
        ]
    )


class TwoHeadPerspectiveModel(nn.Module):
    def __init__(self):
        super().__init__()
        backbone = models.resnet18(weights=None)

        feat_dim = backbone.fc.in_features
        backbone.fc = nn.Identity()

        self.backbone = backbone
        self.horizontal_head = nn.Linear(feat_dim, len(H_LABELS))
        self.vertical_head = nn.Linear(feat_dim, len(V_LABELS))

    def forward(self, x: torch.Tensor) -> Tuple[torch.Tensor, torch.Tensor]:
        features = self.backbone(x)
        logits_h = self.horizontal_head(features)
        logits_v = self.vertical_head(features)
        return logits_h, logits_v


def load_model(model_path: str, device: torch.device) -> nn.Module:
    if not os.path.exists(model_path):
        raise FileNotFoundError(f"Model file not found: {model_path}")

    model = TwoHeadPerspectiveModel()

    state_dict = torch.load(model_path, map_location=device)
    model.load_state_dict(state_dict)

    model = model.to(device)
    model.eval()
    return model


def build_joint_predictions(probs_h: torch.Tensor, probs_v: torch.Tensor) -> List[dict]:
    predictions = []
    for h_idx, h_name in enumerate(H_LABELS):
        for v_idx, v_name in enumerate(V_LABELS):
            confidence = float(probs_h[h_idx].item() * probs_v[v_idx].item())
            pair = (h_name, v_name)
            view_name = PAIR_TO_VIEW.get(pair)
            predictions.append(
                {
                    "class": view_name if view_name is not None else "center-center",
                    "confidence": confidence,
                    "horizontal": h_name,
                    "vertical": v_name,
                    "is_impossible": view_name is None,
                }
            )

    predictions.sort(key=lambda item: item["confidence"], reverse=True)
    return predictions


def predict_from_pil_image(
    image: Image.Image,
    model: nn.Module,
    img_size: int,
    device: torch.device,
    top_k: int,
) -> dict:
    padded_image = pad_to_square_with_mean_color(image)
    transform = build_transform(img_size)
    input_tensor = transform(padded_image).unsqueeze(0).to(device)

    with torch.no_grad():
        logits_h, logits_v = model(input_tensor)
        probs_h = torch.softmax(logits_h, dim=1)[0]
        probs_v = torch.softmax(logits_v, dim=1)[0]

    all_joint_predictions = build_joint_predictions(probs_h, probs_v)
    top_pred = all_joint_predictions[0]
    warning = None

    if top_pred["is_impossible"]:
        fallback_pred = next((p for p in all_joint_predictions if not p["is_impossible"]), None)
        if fallback_pred is None:
            raise RuntimeError("Could not find a valid fallback class for impossible center-center prediction.")

        warning = (
            "Top prediction is impossible class 'center-center' "
            f"({top_pred['confidence']:.2%}). Falling back to next best valid class "
            f"'{fallback_pred['class']}' ({fallback_pred['confidence']:.2%})."
        )
        selected_pred = fallback_pred
    else:
        selected_pred = top_pred

    valid_predictions = [p for p in all_joint_predictions if not p["is_impossible"]]
    top_k = max(1, min(top_k, len(valid_predictions)))
    predictions = [
        {"class": item["class"], "confidence": float(item["confidence"])}
        for item in valid_predictions[:top_k]
    ]

    return {
        "predicted_class": selected_pred["class"],
        "confidence": selected_pred["confidence"],
        "predicted_axes": {
            "horizontal": selected_pred["horizontal"],
            "vertical": selected_pred["vertical"],
        },
        "top_k": predictions,
        "warning": warning,
    }


def predict_from_image_path(
    image_path: str,
    model: nn.Module,
    img_size: int,
    device: torch.device,
    top_k: int,
) -> dict:
    if not os.path.exists(image_path):
        raise FileNotFoundError(f"Image not found: {image_path}")

    image = Image.open(image_path).convert("RGB")
    return predict_from_pil_image(
        image=image,
        model=model,
        img_size=img_size,
        device=device,
        top_k=top_k,
    )


def infer_single_image(
    image_path: str,
    model: nn.Module,
    img_size: int,
    device: torch.device,
    top_k: int,
) -> None:
    result = predict_from_image_path(
        image_path=image_path,
        model=model,
        img_size=img_size,
        device=device,
        top_k=top_k,
    )

    print("Inference completed")
    print(f"Image: {image_path}")
    print(f"Predicted class: {result['predicted_class']}")
    print(f"Predicted axes: H={result['predicted_axes']['horizontal']}, V={result['predicted_axes']['vertical']}")
    print(f"Confidence: {result['confidence']:.2%}")

    if result["warning"]:
        print(f"WARNING: {result['warning']}", file=sys.stderr)

    if top_k > 1:
        print("Top-k predictions:")
        for rank, item in enumerate(result["top_k"], start=1):
            print(f"  {rank}. {item['class']}: {item['confidence']:.2%}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run car perspective inference for a single image")
    parser.add_argument("--image", required=True, help="Path to input image")
    parser.add_argument("--model", default="best_car_view_model.pth", help="Path to .pth checkpoint")
    parser.add_argument("--img-size", type=int, default=224, help="Model input size")
    parser.add_argument("--top-k", type=int, default=3, help="Number of top predictions to print")
    parser.add_argument(
        "--device",
        default="auto",
        choices=["auto", "cpu", "cuda", "mps"],
        help="Compute device",
    )
    return parser.parse_args()


def main() -> None:
    '''Run: python infer_perspective.py --image "C:\path\to\car.jpg" --model "C:\path\to\best_car_view_model.pth" --top-k 3 --device auto'''
    args = parse_args()
    device = get_device(args.device)
    model = load_model(args.model, device=device)

    infer_single_image(
        image_path=args.image,
        model=model,
        img_size=args.img_size,
        device=device,
        top_k=args.top_k,
    )


if __name__ == "__main__":
    main()
