import csv
import json
import os
import random
from collections import Counter
from pathlib import Path
from typing import Dict, List, Optional, Tuple
import shutil

from comet_ml import Optimizer
import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from PIL import Image, ImageOps, ImageStat
from sklearn.metrics import f1_score
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, Dataset, WeightedRandomSampler
from torchvision import models, transforms
from tqdm import tqdm

CONFIG = {
    "project_name": "car-perspective",
    "workspace": "brshtsk",
    "hf_dataset_id": "mitbersh/car-view",
    "data_dir": "./car_view_dataset",
    "img_size": 224,
    "epochs": 5,
    "seed": 42,
    "val_real_split": 0.2,
    "real_source_weight": 2.0,
    "weight_decay": 1e-4,
}

OPTIMIZER_CONFIG = {
    "algorithm": "bayes",
    "spec": {
        "maxCombo": 7,
        "objective": "maximize",
        "metric": "val_axes_macro_f1",
    },
    "parameters": {
        "learning_rate": {"type": "float", "scalingType": "loguniform", "min": 1e-5, "max": 1e-3},
        "batch_size": {"type": "discrete", "values": [16, 32, 64]},
        "real_source_weight": {"type": "discrete", "values": [1.0, 1.5, 2.0, 3.0]},
        "weight_decay": {"type": "float", "scalingType": "loguniform", "min": 1e-6, "max": 1e-3},
    },
}

IMAGENET_MEAN = [0.485, 0.456, 0.406]
IMAGENET_STD = [0.229, 0.224, 0.225]

H_LABELS = ["left", "center", "right"]
V_LABELS = ["front", "center", "back"]

VIEW_TO_AXES = {
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
COMBINED_VIEW_NAMES = list(VIEW_TO_AXES.keys())
COMBINED_VIEW_TO_ID = {view: idx for idx, view in enumerate(COMBINED_VIEW_NAMES)}

H_TO_ID = {name: idx for idx, name in enumerate(H_LABELS)}
V_TO_ID = {name: idx for idx, name in enumerate(V_LABELS)}


def set_seed(seed: int) -> None:
    torch.manual_seed(seed)
    random.seed(seed)
    np.random.seed(seed)
    if torch.cuda.is_available():
        torch.cuda.manual_seed_all(seed)


class PadToSquareWithMeanColor:
    """Приводит картинку к 1:1 как в infer, дополняя средним цветом изображения."""

    def __call__(self, image: Image.Image) -> Image.Image:
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


def build_transforms(img_size: int) -> Tuple[transforms.Compose, transforms.Compose]:
    train_transforms = transforms.Compose(
        [
            PadToSquareWithMeanColor(),
            transforms.Resize((img_size, img_size)),
            transforms.RandomRotation(degrees=5),
            transforms.ColorJitter(brightness=0.2, contrast=0.2),
            transforms.ToTensor(),
            transforms.Normalize(IMAGENET_MEAN, IMAGENET_STD),
        ]
    )
    val_transforms = transforms.Compose(
        [
            PadToSquareWithMeanColor(),
            transforms.Resize((img_size, img_size)),
            transforms.ToTensor(),
            transforms.Normalize(IMAGENET_MEAN, IMAGENET_STD),
        ]
    )
    return train_transforms, val_transforms


def resolve_meta_csv(data_dir: str) -> str:
    direct = os.path.join(data_dir, "meta.csv")
    if os.path.exists(direct):
        return direct

    blended = os.path.join(data_dir, "blended", "meta.csv")
    if os.path.exists(blended):
        return blended

    for root, _, files in os.walk(data_dir):
        if "meta.csv" in files:
            return os.path.join(root, "meta.csv")
    raise FileNotFoundError(f"meta.csv не найден в: {data_dir}")


def read_meta(meta_path: str) -> List[Dict[str, str]]:
    with open(meta_path, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        rows = [row for row in reader if row.get("view") in VIEW_TO_AXES]

    if not rows:
        raise ValueError(f"В {meta_path} нет валидных строк с view из поддерживаемого набора.")
    return rows


def split_rows(rows: List[Dict[str, str]], seed: int, val_real_split: float) -> Tuple[List[Dict[str, str]], List[Dict[str, str]]]:
    synthetic_rows = [row for row in rows if row.get("source") == "synthetic"]
    real_rows = [row for row in rows if row.get("source") == "real"]

    if not real_rows:
        raise ValueError("В meta.csv отсутствуют записи source=real. Невозможно собрать val real-only.")

    real_views = [row["view"] for row in real_rows]
    real_train, real_val = train_test_split(
        real_rows,
        test_size=val_real_split,
        random_state=seed,
        stratify=real_views,
    )

    train_rows = synthetic_rows + real_train
    val_rows = real_val
    return train_rows, val_rows


class PerspectiveMetaDataset(Dataset):
    def __init__(self, root_dir: str, rows: List[Dict[str, str]], transform: Optional[transforms.Compose] = None):
        self.root_dir = root_dir
        self.rows = rows
        self.transform = transform

    def __len__(self) -> int:
        return len(self.rows)

    def __getitem__(self, idx: int):
        row = self.rows[idx]
        relative_path = row["relative_path"].replace("\\", os.sep).replace("/", os.sep)
        image_path = os.path.join(self.root_dir, relative_path)

        image = Image.open(image_path).convert("RGB")
        if self.transform is not None:
            image = self.transform(image)

        horizontal_name, vertical_name = VIEW_TO_AXES[row["view"]]
        target_h = H_TO_ID[horizontal_name]
        target_v = V_TO_ID[vertical_name]
        source = row.get("source", "unknown")
        return image, target_h, target_v, relative_path, source


class TwoHeadPerspectiveModel(nn.Module):
    def __init__(self):
        super().__init__()
        backbone = models.resnet18(weights="DEFAULT")

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


def axes_to_combined_ids(h_ids: List[int], v_ids: List[int]) -> List[int]:
    combined_ids = []
    for h_id, v_id in zip(h_ids, v_ids):
        h_name = H_LABELS[h_id]
        v_name = V_LABELS[v_id]
        view_name = PAIR_TO_VIEW.get((h_name, v_name))
        combined_ids.append(COMBINED_VIEW_TO_ID[view_name] if view_name is not None else -1)
    return combined_ids


def safe_macro_f1(y_true: List[int], y_pred: List[int]) -> float:
    if not y_true:
        return 0.0
    return float(f1_score(y_true, y_pred, average="macro", zero_division=0))


def accuracy(y_true: List[int], y_pred: List[int]) -> float:
    if not y_true:
        return 0.0
    correct = sum(int(t == p) for t, p in zip(y_true, y_pred))
    return float(correct / len(y_true))


def get_experiment_url(experiment, workspace, project_name):
    """Возвращает ссылку на эксперимент Comet для удобного перехода в UI."""
    url = getattr(experiment, "url", None)
    if isinstance(url, str) and url.strip():
        return url

    experiment_id = getattr(experiment, "id", None)
    if workspace and project_name and experiment_id:
        return f"https://www.comet.com/{workspace}/{project_name}/experiments/{experiment_id}"


def combined_view_name_from_axes(h_id: int, v_id: int) -> str:
    view_name = PAIR_TO_VIEW.get((H_LABELS[h_id], V_LABELS[v_id]))
    return view_name if view_name is not None else "unknown"


def tensor_to_uint8_image(image_tensor: torch.Tensor) -> np.ndarray:
    mean = torch.tensor(IMAGENET_MEAN, dtype=image_tensor.dtype).view(3, 1, 1)
    std = torch.tensor(IMAGENET_STD, dtype=image_tensor.dtype).view(3, 1, 1)
    image = image_tensor.detach().cpu() * std + mean
    image = image.clamp(0, 1)
    image = (image.permute(1, 2, 0).numpy() * 255.0).astype(np.uint8)
    return image


def log_best_val_samples(experiment, val_samples: List[Dict], step: int) -> None:
    error_samples = []
    for sample in val_samples:
        true_view = combined_view_name_from_axes(sample["true_h"], sample["true_v"])
        pred_view = combined_view_name_from_axes(sample["pred_h"], sample["pred_v"])
        if true_view != pred_view:
            error_samples.append(sample)

    print(f"Логирую {len(error_samples)} ошибочных val-картинок для best-модели в Comet...")
    for idx, sample in enumerate(error_samples):
        true_view = combined_view_name_from_axes(sample["true_h"], sample["true_v"])
        pred_view = combined_view_name_from_axes(sample["pred_h"], sample["pred_v"])
        is_correct = true_view == pred_view
        correctness_tag = "ok" if is_correct else "err"

        probs_h = {label: float(prob) for label, prob in zip(H_LABELS, sample["probs_h"])}
        probs_v = {label: float(prob) for label, prob in zip(V_LABELS, sample["probs_v"])}
        pred_h_conf = probs_h[H_LABELS[sample["pred_h"]]]
        pred_v_conf = probs_v[V_LABELS[sample["pred_v"]]]

        metadata = {
            "relative_path": sample["relative_path"],
            "source": sample["source"],
            "true_horizontal": H_LABELS[sample["true_h"]],
            "true_vertical": V_LABELS[sample["true_v"]],
            "pred_horizontal": H_LABELS[sample["pred_h"]],
            "pred_vertical": V_LABELS[sample["pred_v"]],
            "true_view": true_view,
            "pred_view": pred_view,
            "is_correct": bool(is_correct),
            "softmax_horizontal": probs_h,
            "softmax_vertical": probs_v,
            "pred_horizontal_confidence": pred_h_conf,
            "pred_vertical_confidence": pred_v_conf,
        }

        image_uint8 = tensor_to_uint8_image(sample["image_tensor"])
        base_name = Path(sample["relative_path"]).name
        comet_image_name = (
            f"best_val/{idx:04d}_{correctness_tag}_{true_view}_pred-{pred_view}_{base_name}"
        ).replace("\\", "/")
        comet_meta_name = f"best_val_meta/{idx:04d}_{Path(base_name).stem}.json"

        experiment.log_image(
            image_data=image_uint8,
            name=comet_image_name,
            step=step,
            metadata=metadata,
        )
        experiment.log_asset_data(
            json.dumps(metadata, ensure_ascii=False, indent=2),
            name=comet_meta_name,
            step=step,
        )


def get_data_loaders(data_dir, batch_size, img_size, seed, val_real_split, real_source_weight):
    meta_path = resolve_meta_csv(data_dir)
    dataset_root = str(Path(meta_path).parent)
    rows = read_meta(meta_path)
    train_rows, val_rows = split_rows(rows, seed=seed, val_real_split=val_real_split)

    train_transforms, val_transforms = build_transforms(img_size)
    train_dataset = PerspectiveMetaDataset(dataset_root, train_rows, transform=train_transforms)
    val_dataset = PerspectiveMetaDataset(dataset_root, val_rows, transform=val_transforms)

    print(f"Meta: {meta_path}")
    print(f"Train size: {len(train_dataset)} | Val size (real-only): {len(val_dataset)}")

    class_counts = Counter(row["view"] for row in train_rows)
    source_counts = Counter(row.get("source", "unknown") for row in train_rows)

    sampler_weights = []
    for row in train_rows:
        source_name = row.get("source", "unknown")
        class_weight = 1.0 / np.sqrt(class_counts[row["view"]])
        real_boost = float(real_source_weight) if source_name == "real" else 1.0
        sampler_weights.append(class_weight * real_boost)

    sampler = WeightedRandomSampler(
        weights=sampler_weights,
        num_samples=len(sampler_weights),
        replacement=True,
    )

    train_loader = DataLoader(
        train_dataset,
        batch_size=batch_size,
        sampler=sampler,
        num_workers=0,
        drop_last=True,
    )
    val_loader = DataLoader(
        val_dataset,
        batch_size=batch_size,
        shuffle=False,
        num_workers=0,
    )
    return train_loader, val_loader


def train_model(experiment, current_config):
    set_seed(current_config['seed'])

    if not os.path.exists(current_config['data_dir']) or len(os.listdir(current_config['data_dir'])) == 0:
        print(f"Датасет не найден локально. Скачиваю {current_config['hf_dataset_id']} с Hugging Face...")
        
        import subprocess
        token = os.environ.get("HF_TOKEN", "")
        if token:
            repo_url = f"https://oauth2:{token}@huggingface.co/datasets/{current_config['hf_dataset_id']}"
        else:
            repo_url = f"https://huggingface.co/datasets/{current_config['hf_dataset_id']}"
            
        subprocess.run(["git", "clone", repo_url, current_config['data_dir']], check=True)
        
        for item in os.listdir(current_config['data_dir']):
            if item.startswith('.'):
                item_path = os.path.join(current_config['data_dir'], item)
                if os.path.isdir(item_path):
                    shutil.rmtree(item_path, ignore_errors=True)
                elif os.path.isfile(item_path):
                    os.remove(item_path)
                    
        print("Датасет успешно загружен и очищен!")


    experiment.log_parameters(current_config)

    if torch.backends.mps.is_available():
        device = torch.device("mps")
        print("MPS")
    elif torch.cuda.is_available():
        device = torch.device("cuda")
        print("CUDA")
    else:
        device = torch.device("cpu")
        print("CPU")

    train_loader, val_loader = get_data_loaders(
        current_config['data_dir'],
        current_config['batch_size'],
        current_config['img_size'],
        current_config['seed'],
        current_config['val_real_split'],
        current_config['real_source_weight'],
    )

    model = TwoHeadPerspectiveModel()
    model = model.to(device)

    criterion_h = nn.CrossEntropyLoss()
    criterion_v = nn.CrossEntropyLoss()
    optimizer = optim.AdamW(
        model.parameters(),
        lr=current_config['learning_rate'],
        weight_decay=current_config['weight_decay'],
    )

    best_axes_score = -1.0
    best_epoch = -1
    best_model_path = f"best_car_view_model_{experiment.id}.pth"
    best_run_metrics: Dict[str, float] = {}
    best_val_payload: Optional[Dict[str, List]] = None

    # Train loop
    try:
        for epoch in range(current_config['epochs']):
            print(f"\nEpoch {epoch + 1}/{current_config['epochs']}")

            model.train()
            train_loss = 0.0
            train_loss_h = 0.0
            train_loss_v = 0.0
            train_h_true, train_h_pred = [], []
            train_v_true, train_v_pred = [], []
            seen_train_samples = 0

            for inputs, target_h, target_v, _, _ in tqdm(train_loader, desc="Training"):
                inputs = inputs.to(device)
                target_h = target_h.to(device)
                target_v = target_v.to(device)

                optimizer.zero_grad()

                logits_h, logits_v = model(inputs)
                loss_h = criterion_h(logits_h, target_h)
                loss_v = criterion_v(logits_v, target_v)
                loss = 0.5 * (loss_h + loss_v)

                pred_h = torch.argmax(logits_h, dim=1)
                pred_v = torch.argmax(logits_v, dim=1)

                loss.backward()
                optimizer.step()

                seen_train_samples += inputs.size(0)
                train_loss += loss.item() * inputs.size(0)
                train_loss_h += loss_h.item() * inputs.size(0)
                train_loss_v += loss_v.item() * inputs.size(0)

                train_h_true.extend(target_h.cpu().tolist())
                train_h_pred.extend(pred_h.cpu().tolist())
                train_v_true.extend(target_v.cpu().tolist())
                train_v_pred.extend(pred_v.cpu().tolist())

            train_denominator = max(1, seen_train_samples)
            epoch_train_loss = train_loss / train_denominator
            epoch_train_loss_h = train_loss_h / train_denominator
            epoch_train_loss_v = train_loss_v / train_denominator
            train_horizontal_acc = accuracy(train_h_true, train_h_pred)
            train_vertical_acc = accuracy(train_v_true, train_v_pred)
            train_horizontal_f1 = safe_macro_f1(train_h_true, train_h_pred)
            train_vertical_f1 = safe_macro_f1(train_v_true, train_v_pred)

            model.eval()
            val_loss = 0.0
            val_loss_h = 0.0
            val_loss_v = 0.0
            val_h_true, val_h_pred = [], []
            val_v_true, val_v_pred = [], []
            seen_val_samples = 0
            impossible_combined_pred_count = 0
            val_samples = []

            with torch.no_grad():
                for inputs, target_h, target_v, relative_paths, sources in val_loader:
                    raw_inputs = inputs.clone()
                    inputs = inputs.to(device)
                    target_h = target_h.to(device)
                    target_v = target_v.to(device)

                    logits_h, logits_v = model(inputs)
                    loss_h = criterion_h(logits_h, target_h)
                    loss_v = criterion_v(logits_v, target_v)
                    loss = 0.5 * (loss_h + loss_v)

                    pred_h = torch.argmax(logits_h, dim=1)
                    pred_v = torch.argmax(logits_v, dim=1)
                    probs_h = torch.softmax(logits_h, dim=1).cpu()
                    probs_v = torch.softmax(logits_v, dim=1).cpu()

                    impossible_mask = (pred_h == H_TO_ID["center"]) & (pred_v == V_TO_ID["center"])
                    impossible_combined_pred_count += int(impossible_mask.sum().item())

                    seen_val_samples += inputs.size(0)
                    val_loss += loss.item() * inputs.size(0)
                    val_loss_h += loss_h.item() * inputs.size(0)
                    val_loss_v += loss_v.item() * inputs.size(0)

                    val_h_true.extend(target_h.cpu().tolist())
                    val_h_pred.extend(pred_h.cpu().tolist())
                    val_v_true.extend(target_v.cpu().tolist())
                    val_v_pred.extend(pred_v.cpu().tolist())

                    target_h_cpu = target_h.cpu().tolist()
                    target_v_cpu = target_v.cpu().tolist()
                    pred_h_cpu = pred_h.cpu().tolist()
                    pred_v_cpu = pred_v.cpu().tolist()

                    for sample_idx in range(len(relative_paths)):
                        val_samples.append(
                            {
                                "image_tensor": raw_inputs[sample_idx],
                                "relative_path": relative_paths[sample_idx],
                                "source": sources[sample_idx],
                                "true_h": target_h_cpu[sample_idx],
                                "true_v": target_v_cpu[sample_idx],
                                "pred_h": pred_h_cpu[sample_idx],
                                "pred_v": pred_v_cpu[sample_idx],
                                "probs_h": probs_h[sample_idx].tolist(),
                                "probs_v": probs_v[sample_idx].tolist(),
                            }
                        )

            val_denominator = max(1, seen_val_samples)
            epoch_val_loss = val_loss / val_denominator
            epoch_val_loss_h = val_loss_h / val_denominator
            epoch_val_loss_v = val_loss_v / val_denominator

            horizontal_acc = accuracy(val_h_true, val_h_pred)
            vertical_acc = accuracy(val_v_true, val_v_pred)
            horizontal_macro_f1 = safe_macro_f1(val_h_true, val_h_pred)
            vertical_macro_f1 = safe_macro_f1(val_v_true, val_v_pred)
            val_axes_score = 0.5 * (horizontal_macro_f1 + vertical_macro_f1)
            impossible_combined_pred_ratio = impossible_combined_pred_count / max(1, seen_val_samples)

            val_combined_true = axes_to_combined_ids(val_h_true, val_v_true)
            val_combined_pred = axes_to_combined_ids(val_h_pred, val_v_pred)
            valid_combined = [
                (t, p)
                for t, p in zip(val_combined_true, val_combined_pred)
                if t >= 0
            ]
            combined_view_acc = (
                sum(int(t == p) for t, p in valid_combined) / len(valid_combined)
                if valid_combined
                else 0.0
            )

            print(
                f"Train Loss: {epoch_train_loss:.4f} "
                f"(H: {epoch_train_loss_h:.4f}, V: {epoch_train_loss_v:.4f})"
            )
            print(
                f"Train H-Acc/F1: {train_horizontal_acc:.4f}/{train_horizontal_f1:.4f} | "
                f"V-Acc/F1: {train_vertical_acc:.4f}/{train_vertical_f1:.4f}"
            )
            print(
                f"Val Loss: {epoch_val_loss:.4f} "
                f"(H: {epoch_val_loss_h:.4f}, V: {epoch_val_loss_v:.4f})"
            )
            print(
                f"Val H-Acc/F1: {horizontal_acc:.4f}/{horizontal_macro_f1:.4f} | "
                f"V-Acc/F1: {vertical_acc:.4f}/{vertical_macro_f1:.4f} | "
                f"Combined Acc: {combined_view_acc:.4f}"
            )
            print(
                f"Val impossible (center-center): {impossible_combined_pred_count} "
                f"({impossible_combined_pred_ratio:.4%})"
            )

            experiment.log_metrics(
                {
                    "train_loss": float(epoch_train_loss),
                    "train_loss_h": float(epoch_train_loss_h),
                    "train_loss_v": float(epoch_train_loss_v),
                    "train_horizontal_acc": float(train_horizontal_acc),
                    "train_vertical_acc": float(train_vertical_acc),
                    "train_horizontal_macro_f1": float(train_horizontal_f1),
                    "train_vertical_macro_f1": float(train_vertical_f1),
                    "val_loss": float(epoch_val_loss),
                    "val_loss_h": float(epoch_val_loss_h),
                    "val_loss_v": float(epoch_val_loss_v),
                    "horizontal_acc": float(horizontal_acc),
                    "vertical_acc": float(vertical_acc),
                    "horizontal_macro_f1": float(horizontal_macro_f1),
                    "vertical_macro_f1": float(vertical_macro_f1),
                    "combined_view_acc": float(combined_view_acc),
                    "impossible_combined_pred_count": float(impossible_combined_pred_count),
                    "impossible_combined_pred_ratio": float(impossible_combined_pred_ratio),
                    "val_axes_macro_f1": float(val_axes_score),
                },
                step=epoch + 1,
            )

            if val_axes_score > best_axes_score:
                best_axes_score = val_axes_score
                best_epoch = epoch + 1
                torch.save(model.state_dict(), best_model_path)
                print(f"Model snapshot updated: {best_model_path} (epoch={best_epoch})")

                best_run_metrics = {
                    "val_axes_macro_f1": float(val_axes_score),
                    "val_loss": float(epoch_val_loss),
                    "combined_view_acc": float(combined_view_acc),
                    "horizontal_macro_f1": float(horizontal_macro_f1),
                    "vertical_macro_f1": float(vertical_macro_f1),
                    "impossible_combined_pred_ratio": float(impossible_combined_pred_ratio),
                }
                best_val_payload = {
                    "val_h_true": val_h_true.copy(),
                    "val_h_pred": val_h_pred.copy(),
                    "val_v_true": val_v_true.copy(),
                    "val_v_pred": val_v_pred.copy(),
                    "val_samples": list(val_samples),
                }

    finally:
        print(f"Обучение завершено. Лучший axis-score: {best_axes_score:.4f}")

        if best_epoch > 0:
            # One-time run summary for quick hyperparameter comparison in Comet.
            experiment.log_metric("run_best_val_axes_macro_f1", float(best_axes_score))
            experiment.log_metrics(
                {
                    "run_best_val_loss": float(best_run_metrics.get("val_loss", 0.0)),
                    "run_best_combined_view_acc": float(best_run_metrics.get("combined_view_acc", 0.0)),
                    "run_best_horizontal_macro_f1": float(best_run_metrics.get("horizontal_macro_f1", 0.0)),
                    "run_best_vertical_macro_f1": float(best_run_metrics.get("vertical_macro_f1", 0.0)),
                    "run_best_impossible_combined_pred_ratio": float(
                        best_run_metrics.get("impossible_combined_pred_ratio", 0.0)
                    ),
                }
            )
            experiment.log_other("run_best_epoch", best_epoch)
            experiment.log_other("run_best_model_path", best_model_path)
            experiment.log_other("run_selection_metric", "val_axes_macro_f1")

            experiment.log_model(
                "best_car_view",
                best_model_path,
                metadata={
                    "best_epoch": int(best_epoch),
                    "best_val_axes_macro_f1": float(best_axes_score),
                    "selection_metric": "val_axes_macro_f1",
                    "best_val_loss": float(best_run_metrics.get("val_loss", 0.0)),
                    "best_combined_view_acc": float(best_run_metrics.get("combined_view_acc", 0.0)),
                },
            )

            if best_val_payload is not None:
                experiment.log_confusion_matrix(
                    y_true=best_val_payload["val_h_true"],
                    y_predicted=best_val_payload["val_h_pred"],
                    labels=H_LABELS,
                    title=f"Horizontal confusion matrix (best epoch {best_epoch})",
                    step=best_epoch,
                )
                experiment.log_confusion_matrix(
                    y_true=best_val_payload["val_v_true"],
                    y_predicted=best_val_payload["val_v_pred"],
                    labels=V_LABELS,
                    title=f"Vertical confusion matrix (best epoch {best_epoch})",
                    step=best_epoch,
                )
                log_best_val_samples(experiment, best_val_payload["val_samples"], step=best_epoch)

    return {
        "run_id": str(experiment.id),
        "best_axes_score": float(best_axes_score),
        "best_epoch": int(best_epoch),
        "best_model_path": best_model_path,
        "selection_metric": "val_axes_macro_f1",
        "best_run_metrics": best_run_metrics,
    }


if __name__ == '__main__':
    opt = Optimizer(OPTIMIZER_CONFIG)

    experiment_iterator_kwargs = {"project_name": CONFIG["project_name"]}
    if CONFIG.get("workspace"):
        experiment_iterator_kwargs["workspace"] = CONFIG["workspace"]
    
    overall_best_score = -1.0
    overall_best_info: Optional[Dict] = None

    for experiment in opt.get_experiments(**experiment_iterator_kwargs):
        print(f"\nЗапуск нового эксперимента: {experiment.id}")
        print(
            "Comet URL:",
            get_experiment_url(
                experiment,
                workspace=CONFIG.get("workspace", ""),
                project_name=CONFIG["project_name"]
            )
        )
        
        current_config = CONFIG.copy()
        current_config["learning_rate"] = float(experiment.get_parameter("learning_rate"))
        current_config["batch_size"] = int(experiment.get_parameter("batch_size"))
        current_config["real_source_weight"] = float(experiment.get_parameter("real_source_weight"))
        current_config["weight_decay"] = float(experiment.get_parameter("weight_decay"))

        lr_str = f"{float(current_config['learning_rate']):.2e}"
        wd_str = f"{float(current_config['weight_decay']):.1e}"
        run_name = (
            f"lr={lr_str} | bs={current_config['batch_size']} | "
            f"rsw={current_config['real_source_weight']:.2f} | wd={wd_str}"
        )
        experiment.set_name(run_name)
        print("Run name:", run_name)

        run_result = train_model(experiment, current_config)

        is_new_overall_best = run_result["best_axes_score"] > overall_best_score
        if is_new_overall_best:
            overall_best_score = run_result["best_axes_score"]
            overall_best_info = {
                "run_id": run_result["run_id"],
                "run_name": run_name,
                "selection_metric": run_result["selection_metric"],
                "best_axes_score": run_result["best_axes_score"],
                "best_epoch": run_result["best_epoch"],
                "best_model_path": run_result["best_model_path"],
                "config": {
                    "learning_rate": float(current_config["learning_rate"]),
                    "batch_size": int(current_config["batch_size"]),
                    "real_source_weight": float(current_config["real_source_weight"]),
                    "weight_decay": float(current_config["weight_decay"]),
                },
            }
            experiment.log_other("was_best_so_far", True)
            experiment.log_metric("overall_best_val_axes_macro_f1_so_far", float(overall_best_score))
            experiment.log_other("overall_best_model_path_so_far", run_result["best_model_path"])
            shutil.copyfile(run_result["best_model_path"], "best_overall_car_view_model.pth")
        else:
            experiment.log_other("was_best_so_far", False)

        experiment.end()

    if overall_best_info is not None:
        overall_summary_path = "best_overall_run_summary.json"
        with open(overall_summary_path, "w", encoding="utf-8") as f:
            json.dump(overall_best_info, f, ensure_ascii=False, indent=2)
        print("Overall best run:")
        print(json.dumps(overall_best_info, ensure_ascii=False, indent=2))
        print(f"Overall best summary saved to: {overall_summary_path}")

