import csv
import os
from pathlib import Path
from typing import Dict, List, Optional, Tuple

import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from comet_ml import Optimizer
from PIL import Image, ImageOps, ImageStat
from sklearn.metrics import f1_score
from sklearn.model_selection import train_test_split
from torch.utils.data import DataLoader, Dataset
from torchvision import models, transforms
from tqdm import tqdm

CONFIG = {
    "project_name": "car-perspective",
    "workspace": "brshtsk",
    "hf_dataset_id": "mitbersh/car-view",
    "data_dir": "./car_view_dataset",
    "img_size": 224,
    "epochs": 3,
    "architecture": "resnet18",
    "seed": 42,
    "val_real_split": 0.2,
}

OPTIMIZER_CONFIG = {
    "algorithm": "bayes",
    "spec": {
        "maxCombo": 5,
        "objective": "maximize",
        "metric": "val_axes_macro_f1",
    },
    "parameters": {
        "learning_rate": {"type": "float", "scalingType": "loguniform", "min": 1e-5, "max": 1e-3},
        "batch_size": {"type": "discrete", "values": [16, 32, 64]},
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
            transforms.RandomRotation(degrees=7),
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
        return image, target_h, target_v


class TwoHeadPerspectiveModel(nn.Module):
    def __init__(self, architecture: str = "resnet18"):
        super().__init__()
        if architecture == "resnet34":
            backbone = models.resnet34(weights="DEFAULT")
        else:
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


def get_data_loaders(data_dir, batch_size, img_size, seed, val_real_split):
    meta_path = resolve_meta_csv(data_dir)
    dataset_root = str(Path(meta_path).parent)
    rows = read_meta(meta_path)
    train_rows, val_rows = split_rows(rows, seed=seed, val_real_split=val_real_split)

    train_transforms, val_transforms = build_transforms(img_size)
    train_dataset = PerspectiveMetaDataset(dataset_root, train_rows, transform=train_transforms)
    val_dataset = PerspectiveMetaDataset(dataset_root, val_rows, transform=val_transforms)

    print(f"Meta: {meta_path}")
    print(f"Train size: {len(train_dataset)} | Val size (real-only): {len(val_dataset)}")

    train_loader = DataLoader(
        train_dataset,
        batch_size=batch_size,
        shuffle=True,
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
        
        import shutil
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
    )

    model = TwoHeadPerspectiveModel(architecture=current_config.get("architecture", "resnet18"))
    model = model.to(device)

    criterion_h = nn.CrossEntropyLoss()
    criterion_v = nn.CrossEntropyLoss()
    optimizer = optim.Adam(model.parameters(), lr=current_config['learning_rate'])

    best_axes_score = -1.0

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

            for inputs, target_h, target_v in tqdm(train_loader, desc="Training"):
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

                train_loss += loss.item() * inputs.size(0)
                train_loss_h += loss_h.item() * inputs.size(0)
                train_loss_v += loss_v.item() * inputs.size(0)

                train_h_true.extend(target_h.cpu().tolist())
                train_h_pred.extend(pred_h.cpu().tolist())
                train_v_true.extend(target_v.cpu().tolist())
                train_v_pred.extend(pred_v.cpu().tolist())

            epoch_train_loss = train_loss / len(train_loader.dataset)
            epoch_train_loss_h = train_loss_h / len(train_loader.dataset)
            epoch_train_loss_v = train_loss_v / len(train_loader.dataset)
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

            with torch.no_grad():
                for inputs, target_h, target_v in val_loader:
                    inputs = inputs.to(device)
                    target_h = target_h.to(device)
                    target_v = target_v.to(device)

                    logits_h, logits_v = model(inputs)
                    loss_h = criterion_h(logits_h, target_h)
                    loss_v = criterion_v(logits_v, target_v)
                    loss = 0.5 * (loss_h + loss_v)

                    pred_h = torch.argmax(logits_h, dim=1)
                    pred_v = torch.argmax(logits_v, dim=1)

                    val_loss += loss.item() * inputs.size(0)
                    val_loss_h += loss_h.item() * inputs.size(0)
                    val_loss_v += loss_v.item() * inputs.size(0)

                    val_h_true.extend(target_h.cpu().tolist())
                    val_h_pred.extend(pred_h.cpu().tolist())
                    val_v_true.extend(target_v.cpu().tolist())
                    val_v_pred.extend(pred_v.cpu().tolist())

            epoch_val_loss = val_loss / len(val_loader.dataset)
            epoch_val_loss_h = val_loss_h / len(val_loader.dataset)
            epoch_val_loss_v = val_loss_v / len(val_loader.dataset)

            horizontal_acc = accuracy(val_h_true, val_h_pred)
            vertical_acc = accuracy(val_v_true, val_v_pred)
            horizontal_macro_f1 = safe_macro_f1(val_h_true, val_h_pred)
            vertical_macro_f1 = safe_macro_f1(val_v_true, val_v_pred)
            val_axes_score = 0.5 * (horizontal_macro_f1 + vertical_macro_f1)

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
                    "val_axes_macro_f1": float(val_axes_score),
                },
                step=epoch + 1,
            )

            experiment.log_confusion_matrix(
                y_true=val_h_true,
                y_predicted=val_h_pred,
                labels=H_LABELS,
                title="Horizontal confusion matrix",
                step=epoch + 1,
            )
            experiment.log_confusion_matrix(
                y_true=val_v_true,
                y_predicted=val_v_pred,
                labels=V_LABELS,
                title="Vertical confusion matrix",
                step=epoch + 1,
            )

            if val_axes_score > best_axes_score:
                best_axes_score = val_axes_score
                torch.save(model.state_dict(), "best_car_view_model.pth")
                print("Model saved locally! (Improved mean axis macro-F1)")

                experiment.log_model("best_car_view", "best_car_view_model.pth")

    finally:
        print(f"Обучение завершено. Лучший axis-score: {best_axes_score:.4f}")


if __name__ == '__main__':
    opt = Optimizer(OPTIMIZER_CONFIG)

    experiment_iterator_kwargs = {"project_name": CONFIG["project_name"]}
    if CONFIG.get("workspace"):
        experiment_iterator_kwargs["workspace"] = CONFIG["workspace"]
    
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
        current_config["learning_rate"] = experiment.get_parameter("learning_rate")
        current_config["batch_size"] = experiment.get_parameter("batch_size")

        lr_str = f"{float(current_config['learning_rate']):.2e}"
        run_name = f"{lr_str} | {current_config['batch_size']}"
        experiment.set_name(run_name)
        print("Run name:", run_name)

        train_model(experiment, current_config)
        experiment.end()
