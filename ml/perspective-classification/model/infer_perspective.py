import argparse
import os

import torch
import torch.nn as nn
from PIL import Image, ImageOps, ImageStat
from torchvision import models, transforms


IMAGENET_MEAN = [0.485, 0.456, 0.406]
IMAGENET_STD = [0.229, 0.224, 0.225]

CLASS_NAMES = [
    "back",
    "back-left",
    "back-right",
    "front",
    "front-left",
    "front-right",
    "left",
    "other",
    "right",
]


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


def load_model(model_path: str, device: torch.device) -> nn.Module:
    if not os.path.exists(model_path):
        raise FileNotFoundError(f"Файл модели не найден: {model_path}")

    model = models.resnet18(weights=None)
    model.fc = nn.Linear(model.fc.in_features, len(CLASS_NAMES))

    state_dict = torch.load(model_path, map_location=device)
    model.load_state_dict(state_dict)

    model = model.to(device)
    model.eval()
    return model


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
        logits = model(input_tensor)
        probs = torch.softmax(logits, dim=1)[0]

    top_k = max(1, min(top_k, len(CLASS_NAMES)))
    confs, indices = torch.topk(probs, k=top_k)

    predictions = []
    for conf, idx in zip(confs.tolist(), indices.tolist()):
        predictions.append({"class": CLASS_NAMES[idx], "confidence": float(conf)})

    return {
        "predicted_class": predictions[0]["class"],
        "confidence": predictions[0]["confidence"],
        "top_k": predictions,
    }


def predict_from_image_path(
    image_path: str,
    model: nn.Module,
    img_size: int,
    device: torch.device,
    top_k: int,
) -> dict:
    if not os.path.exists(image_path):
        raise FileNotFoundError(f"Изображение не найдено: {image_path}")

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

    print("Инференс завершен")
    print(f"Изображение: {image_path}")
    print(f"Предсказанный класс: {result['predicted_class']}")
    print(f"Уверенность: {result['confidence']:.2%}")

    if top_k > 1:
        print("Топ-k предсказаний:")
        for rank, item in enumerate(result["top_k"], start=1):
            print(f"  {rank}. {item['class']}: {item['confidence']:.2%}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Инференс ракурса автомобиля для одного изображения")
    parser.add_argument("--image", required=True, help="Путь к картинке")
    parser.add_argument("--model", default="best_car_view_model.pth", help="Путь к модели pth")
    parser.add_argument("--img-size", type=int, default=224, help="Размер входа модели")
    parser.add_argument("--top-k", type=int, default=3, help="Сколько лучших предсказаний вывести")
    parser.add_argument(
        "--device",
        default="auto",
        choices=["auto", "cpu", "cuda", "mps"],
        help="Устройство вычислений",
    )
    return parser.parse_args()


def main() -> None:
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
