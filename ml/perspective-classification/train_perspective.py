# Для работы в Kaggle:
# !pip install comet_ml scikit-learn torch torchvision tqdm
# import os
# from kaggle_secrets import UserSecretsClient
# user_secrets = UserSecretsClient()
# os.environ["COMET_API_KEY"] = user_secrets.get_secret("COMET_API_KEY")
# os.environ["HF_TOKEN"] = user_secrets.get_secret("HF_TOKEN")

# import torch
#
# print("Torch:", torch.__version__)
# print("Torch CUDA runtime:", torch.version.cuda)
# print("CUDA available:", torch.cuda.is_available())
#
# if torch.cuda.is_available():
#     idx = torch.cuda.current_device()
#     name = torch.cuda.get_device_name(idx)
#     major, minor = torch.cuda.get_device_capability(idx)
#     sm = f"sm_{major}{minor}"
#
#     print(f"GPU[{idx}]: {name}")
#     print(f"Compute Capability: {major}.{minor} ({sm})")
#
#     arch_list = torch.cuda.get_arch_list() if hasattr(torch.cuda, "get_arch_list") else []
#     print("Torch supports:", ", ".join(arch_list) if arch_list else "unknown")
#
#     if arch_list and sm not in arch_list:
#         print(f"WARNING: {sm} не поддерживается текущей сборкой torch -> вероятен crash на CUDA kernels.")
# else:
#     print("GPU не доступен, будет CPU.")

from comet_ml import Experiment, Optimizer

import torch
import torch.nn as nn
import torch.optim as optim
from torchvision import datasets, models, transforms
from torch.utils.data import DataLoader, WeightedRandomSampler, Subset
from sklearn.model_selection import train_test_split
from sklearn.metrics import f1_score, precision_score, recall_score, confusion_matrix
import numpy as np
import os
from collections import Counter
from tqdm import tqdm

CONFIG = {
    "project_name": "car-perspective",
    "hf_dataset_id": "mitbersh/car-position",
    "data_dir": "./car_position_dataset", # Локальная папка для загрузки
    "img_size": 224,
    "epochs": 3,
    "architecture": "resnet18",
    "seed": 42
}

OPTIMIZER_CONFIG = {
    "algorithm": "bayes",
    "spec": {
        "maxCombo": 5, # Максимальное число экспериментов (подборов параметров)
        "objective": "maximize",
        "metric": "val_f1_macro"
    },
    "parameters": {
        "learning_rate": {"type": "float", "scalingType": "loguniform", "min": 1e-5, "max": 1e-3},
        "batch_size": {"type": "discrete", "values": [16, 32, 64]} # Перебор размеров батча
    }
}

def set_seed(seed):
    torch.manual_seed(seed)
    np.random.seed(seed)


def get_data_loaders(data_dir, batch_size, img_size):
    """
    Загружает данные, делает сплит на Train/Val и создает сбалансированный самплер.
    """
    # Аугментации
    train_transforms = transforms.Compose([
        transforms.Resize((img_size, img_size)),
        transforms.ColorJitter(brightness=0.2, contrast=0.2),
        transforms.ToTensor(),
        transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225])
    ])

    val_transforms = transforms.Compose([
        transforms.Resize((img_size, img_size)),
        transforms.ToTensor(),
        transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225])
    ])

    full_dataset_train = datasets.ImageFolder(data_dir, transform=train_transforms)
    full_dataset_val = datasets.ImageFolder(data_dir, transform=val_transforms)

    targets = full_dataset_train.targets
    train_idx, val_idx = train_test_split(
        np.arange(len(full_dataset_train)),
        test_size=0.2,
        shuffle=True,
        stratify=targets,
        random_state=CONFIG['seed']
    )
    # ToDo: оставить меньше фото класса other (?)

    train_dataset = Subset(full_dataset_train, train_idx)
    val_dataset = Subset(full_dataset_val, val_idx)

    print(f"Train size: {len(train_dataset)}, Val size: {len(val_dataset)}")

    train_targets = [targets[i] for i in train_idx]
    class_counts = Counter(train_targets)
    class_weights = {cls: 1.0 / count for cls, count in class_counts.items()}
    sample_weights = [class_weights[t] for t in train_targets]

    sampler = WeightedRandomSampler(
        weights=sample_weights,
        num_samples=len(sample_weights),
        replacement=True
    )

    train_loader = DataLoader(
        train_dataset,
        batch_size=batch_size,
        sampler=sampler,
        num_workers=0,
        drop_last=True
    )

    val_loader = DataLoader(
        val_dataset,
        batch_size=batch_size,
        shuffle=False,
        num_workers=0
    )

    return train_loader, val_loader, len(full_dataset_train.classes)


def train_model(experiment, current_config):
    set_seed(current_config['seed'])

    # Автоматическая загрузка датасета с Hugging Face, если его нет локально
    if not os.path.exists(current_config['data_dir']) or len(os.listdir(current_config['data_dir'])) == 0:
        print(f"Датасет не найден локально. Скачиваю {current_config['hf_dataset_id']} с Hugging Face...")
        
        # Используем git clone вместо snapshot_download, так как в Kaggle 
        # скачивание тысяч мелких файлов через API часто зависает из-за работы с кэшем и симлинками
        import subprocess
        token = os.environ.get("HF_TOKEN", "")
        if token:
            repo_url = f"https://oauth2:{token}@huggingface.co/datasets/{current_config['hf_dataset_id']}"
        else:
            repo_url = f"https://huggingface.co/datasets/{current_config['hf_dataset_id']}"
            
        subprocess.run(["git", "clone", repo_url, current_config['data_dir']], check=True)
        
        # Очищаем скрытые папки (вроде .git), из-за которых падает ImageFolder
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

    train_loader, val_loader, num_classes = get_data_loaders(
        current_config['data_dir'], current_config['batch_size'], current_config['img_size']
    )

    model = models.resnet18(weights='DEFAULT')
    num_ftrs = model.fc.in_features
    model.fc = nn.Linear(num_ftrs, num_classes)
    model = model.to(device)

    criterion = nn.CrossEntropyLoss()
    optimizer = optim.Adam(model.parameters(), lr=current_config['learning_rate'])

    best_f1 = 0.0

    # Train loop
    try:
        for epoch in range(current_config['epochs']):
            print(f"\nEpoch {epoch + 1}/{current_config['epochs']}")

            model.train()
            train_loss = 0.0
            train_corrects = 0

            for inputs, labels in tqdm(train_loader, desc="Training"):
                inputs = inputs.to(device)
                labels = labels.to(device)

                optimizer.zero_grad()

                outputs = model(inputs)
                _, preds = torch.max(outputs, 1)
                loss = criterion(outputs, labels)

                loss.backward()
                optimizer.step()

                train_loss += loss.item() * inputs.size(0)
                train_corrects += torch.sum(preds == labels)

            epoch_train_loss = train_loss / len(train_loader.dataset)
            epoch_train_acc = train_corrects.double() / len(train_loader.dataset)

            # Val
            model.eval()
            val_loss = 0.0
            val_corrects = 0

            all_preds = []
            all_labels = []

            with torch.no_grad():
                for inputs, labels in val_loader:
                    inputs = inputs.to(device)
                    labels = labels.to(device)

                    outputs = model(inputs)
                    _, preds = torch.max(outputs, 1)
                    loss = criterion(outputs, labels)

                    val_loss += loss.item() * inputs.size(0)
                    val_corrects += torch.sum(preds == labels)

                    all_preds.extend(preds.cpu().numpy())
                    all_labels.extend(labels.cpu().numpy())

            epoch_val_loss = val_loss / len(val_loader.dataset)
            epoch_val_acc = val_corrects.double() / len(val_loader.dataset)

            # Вычисление дополнительных метрик
            val_f1_macro = f1_score(all_labels, all_preds, average='macro')
            val_precision = precision_score(all_labels, all_preds, average='macro', zero_division=0)
            val_recall = recall_score(all_labels, all_preds, average='macro', zero_division=0)

            print(f"Train Loss: {epoch_train_loss:.4f} Acc: {epoch_train_acc:.4f}")
            print(f"Val Loss: {epoch_val_loss:.4f} Acc: {epoch_val_acc:.4f} F1: {val_f1_macro:.4f}")

            experiment.log_metrics({
                "train_loss": float(epoch_train_loss),
                "train_acc": float(epoch_train_acc),
                "val_loss": float(epoch_val_loss),
                "val_acc": float(epoch_val_acc),
                "val_f1_macro": float(val_f1_macro),
                "val_precision": float(val_precision),
                "val_recall": float(val_recall)
            }, step=epoch + 1)
            
            # Логирование матрицы ошибок
            experiment.log_confusion_matrix(y_true=all_labels, y_predicted=all_preds, step=epoch + 1)

            if val_f1_macro > best_f1:
                best_f1 = val_f1_macro
                torch.save(model.state_dict(), "best_car_view_model.pth")
                print("Model saved locally! (Improved F1)")

                experiment.log_model("best_car_view", "best_car_view_model.pth")

    finally:
        print(f"Обучение завершено. Лучший F1-Macro: {best_f1:.4f}")


if __name__ == '__main__':
    # Настраиваем Comet ML Optimizer
    opt = Optimizer(OPTIMIZER_CONFIG)
    
    # Optimizer автоматически подставит разные параметры для каждого эксперимента
    for experiment in opt.get_experiments(project_name=CONFIG["project_name"]):
        print(f"\nЗапуск нового эксперимента: {experiment.id}")
        
        # Обновляем конфиг текущими гиперпараметрами от оптимайзера
        current_config = CONFIG.copy()
        current_config["learning_rate"] = experiment.get_parameter("learning_rate")
        current_config["batch_size"] = experiment.get_parameter("batch_size")
        
        train_model(experiment, current_config)
        experiment.end()
