import argparse
import json
import os
from typing import List, Tuple

import torch
import torch.nn as nn
from PIL import Image, ImageOps, ImageStat
from torchvision import datasets, models, transforms


IMAGENET_MEAN = [0.485, 0.456, 0.406]
IMAGENET_STD = [0.229, 0.224, 0.225]


def get_device(device_arg: str) -> torch.device:
    if device_arg != "auto":
        return torch.device(device_arg)

    if torch.cuda.is_available():
        return torch.device("cuda")
    if torch.backends.mps.is_available():
        return torch.device("mps")
    return torch.device("cpu")


def load_class_names(data_dir: str = "", classes_file: str = "") -> List[str]:
    if classes_file:
        if not os.path.exists(classes_file):
            raise FileNotFoundError(f"Classes file not found: {classes_file}")

        if classes_file.lower().endswith(".json"):
            with open(classes_file, "r", encoding="utf-8") as f:
                names = json.load(f)
            if not isinstance(names, list) or not all(isinstance(x, str) for x in names):
                raise ValueError("JSON must contain a list of class names (strings)")
            return names

        with open(classes_file, "r", encoding="utf-8") as f:
            names = [line.strip() for line in f if line.strip()]
        if not names:
            raise ValueError("Classes text file is empty")
        return names

    if data_dir and os.path.isdir(data_dir):
        dataset = datasets.ImageFolder(data_dir)
        if not dataset.classes:
            raise ValueError(f"No classes found in data dir: {data_dir}")
        return dataset.classes

    raise ValueError("Provide --data-dir or --classes-file to resolve class names")


def pad_to_square_with_mean_color(image: Image.Image) -> Tuple[Image.Image, Tuple[int, int, int]]:
    image = image.convert("RGB")
    stat = ImageStat.Stat(image)
    mean_vals = stat.mean[:3]
    mean_color = (int(mean_vals[0]), int(mean_vals[1]), int(mean_vals[2]))

    w, h = image.size
    if w == h:
        return image, mean_color

    if w > h:
        total_pad = w - h
        top = total_pad // 2
        bottom = total_pad - top
        padded = ImageOps.expand(image, border=(0, top, 0, bottom), fill=mean_color)
    else:
        total_pad = h - w
        left = total_pad // 2
        right = total_pad - left
        padded = ImageOps.expand(image, border=(left, 0, right, 0), fill=mean_color)

    return padded, mean_color


def build_transform(img_size: int) -> transforms.Compose:
    return transforms.Compose(
        [
            transforms.Resize((img_size, img_size)),
            transforms.ToTensor(),
            transforms.Normalize(IMAGENET_MEAN, IMAGENET_STD),
        ]
    )


def load_model(model_path: str, num_classes: int, device: torch.device) -> nn.Module:
    if not os.path.exists(model_path):
        raise FileNotFoundError(f"Model file not found: {model_path}")

    model = models.resnet18(weights=None)
    model.fc = nn.Linear(model.fc.in_features, num_classes)

    state_dict = torch.load(model_path, map_location=device)
    model.load_state_dict(state_dict)

    model = model.to(device)
    model.eval()
    return model


def infer_single_image(
    image_path: str,
    model: nn.Module,
    class_names: List[str],
    img_size: int,
    device: torch.device,
    top_k: int,
) -> None:
    if not os.path.exists(image_path):
        raise FileNotFoundError(f"Image not found: {image_path}")

    image = Image.open(image_path).convert("RGB")
    padded_image, mean_color = pad_to_square_with_mean_color(image)

    transform = build_transform(img_size)
    input_tensor = transform(padded_image).unsqueeze(0).to(device)

    with torch.no_grad():
        logits = model(input_tensor)
        probs = torch.softmax(logits, dim=1)[0]

    top_k = max(1, min(top_k, len(class_names)))
    confs, indices = torch.topk(probs, k=top_k)

    best_idx = indices[0].item()
    best_prob = confs[0].item()

    print("Inference complete")
    print(f"Image: {image_path}")
    print(f"Original size: {image.size[0]}x{image.size[1]}")
    print(f"Padding color (mean RGB): {mean_color}")
    print(f"Model input size: {img_size}x{img_size}")
    print(f"Predicted class: {class_names[best_idx]}")
    print(f"Confidence: {best_prob:.2%}")

    if top_k > 1:
        print("Top-k predictions:")
        for rank, (conf, idx) in enumerate(zip(confs.tolist(), indices.tolist()), start=1):
            print(f"  {rank}. {class_names[idx]}: {conf:.2%}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run inference for a single car perspective image")
    parser.add_argument("--image", required=True, help="Path to input image")
    parser.add_argument("--model", default="best_car_view_model.pth", help="Path to .pth model file")
    parser.add_argument(
        "--data-dir",
        default="./car_position_dataset",
        help="Dataset root directory to auto-resolve class names",
    )
    parser.add_argument(
        "--classes-file",
        default="",
        help="Optional .json or .txt file with ordered class names; overrides --data-dir",
    )
    parser.add_argument("--img-size", type=int, default=224, help="Model input size")
    parser.add_argument("--top-k", type=int, default=3, help="How many top predictions to print")
    parser.add_argument(
        "--device",
        default="auto",
        choices=["auto", "cpu", "cuda", "mps"],
        help="Computation device",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    device = get_device(args.device)
    class_names = load_class_names(data_dir=args.data_dir, classes_file=args.classes_file)
    model = load_model(args.model, num_classes=len(class_names), device=device)

    infer_single_image(
        image_path=args.image,
        model=model,
        class_names=class_names,
        img_size=args.img_size,
        device=device,
        top_k=args.top_k,
    )


if __name__ == "__main__":
    main()

