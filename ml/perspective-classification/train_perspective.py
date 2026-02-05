import torch
import torch.nn as nn
import torch.optim as optim
from torchvision import datasets, models, transforms
from torch.utils.data import DataLoader, WeightedRandomSampler, Subset
from sklearn.model_selection import train_test_split
import numpy as np
import os
from collections import Counter
import wandb
from tqdm import tqdm
import getpass

CONFIG = {
    "project_name": "car-perspective",
    "data_dir": "/Users/brshtsk/Documents/hse/course-project/dataset-photo-position/blended",
    "img_size": 224,
    "batch_size": 32,
    "epochs": 15,
    "learning_rate": 1e-4,
    "architecture": "resnet18",
    "seed": 42
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
        transforms.ColorJitter(brightness=0.2, contrast=0.2),  # Устойчивость к освещению
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

    # Получаем индексы для стратифицированного разбиения (чтобы во всех выборках были все классы)
    targets = full_dataset_train.targets
    train_idx, val_idx = train_test_split(
        np.arange(len(full_dataset_train)),
        test_size=0.2,
        shuffle=True,
        stratify=targets,
        random_state=CONFIG['seed']
    )
    # ToDo: оставить меньше фото класса other (?)

    # Создаем подмножества
    train_dataset = Subset(full_dataset_train, train_idx)
    val_dataset = Subset(full_dataset_val, val_idx)

    print(f"Train size: {len(train_dataset)}, Val size: {len(val_dataset)}")
    print(f"Classes: {full_dataset_train.classes}")

    # Балансировка
    train_targets = [targets[i] for i in train_idx]
    class_counts = Counter(train_targets)

    # Вес класса
    class_weights = {cls: 1.0 / count for cls, count in class_counts.items()}

    # Присваиваем вес каждому сэмплу в train датасете
    sample_weights = [class_weights[t] for t in train_targets]

    # Создаем Sampler
    sampler = WeightedRandomSampler(
        weights=sample_weights,
        num_samples=len(sample_weights),
        replacement=True  # Позволяет брать одни и те же картинки редкого класса несколько раз за эпоху
    )

    # Dataloaders
    train_loader = DataLoader(train_dataset, batch_size=batch_size, sampler=sampler, num_workers=2)
    val_loader = DataLoader(val_dataset, batch_size=batch_size, shuffle=False, num_workers=2)

    return train_loader, val_loader, len(full_dataset_train.classes)


def train_model():
    set_seed(CONFIG['seed'])

    wandb.init(project=CONFIG["project_name"], config=CONFIG)

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
        CONFIG['data_dir'], CONFIG['batch_size'], CONFIG['img_size']
    )

    model = models.resnet18(weights='DEFAULT')

    # Заменяем последний слой
    num_ftrs = model.fc.in_features
    model.fc = nn.Linear(num_ftrs, num_classes)

    model = model.to(device)

    criterion = nn.CrossEntropyLoss()
    optimizer = optim.Adam(model.parameters(), lr=CONFIG['learning_rate'])

    wandb.watch(model, log="all", log_freq=10)

    best_acc = 0.0

    # Train loop
    for epoch in range(CONFIG['epochs']):
        print(f"\nExample {epoch + 1}/{CONFIG['epochs']}")

        model.train()
        train_loss = 0.0
        train_corrects = 0

        # Tqdm для красивой полоски прогресса
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
            train_corrects += torch.sum(preds == labels.data)

        epoch_train_loss = train_loss / len(train_loader.dataset)
        epoch_train_acc = train_corrects.double() / len(train_loader.dataset)

        # Val
        model.eval()
        val_loss = 0.0
        val_corrects = 0

        with torch.no_grad():
            for inputs, labels in val_loader:
                inputs = inputs.to(device)
                labels = labels.to(device)

                outputs = model(inputs)
                _, preds = torch.max(outputs, 1)
                loss = criterion(outputs, labels)

                val_loss += loss.item() * inputs.size(0)
                val_corrects += torch.sum(preds == labels.data)

        epoch_val_loss = val_loss / len(val_loader.dataset)
        epoch_val_acc = val_corrects.double() / len(val_loader.dataset)

        print(f"Train Loss: {epoch_train_loss:.4f} Acc: {epoch_train_acc:.4f}")
        print(f"Val Loss: {epoch_val_loss:.4f} Acc: {epoch_val_acc:.4f}")

        wandb.log({
            "epoch": epoch + 1,
            "train_loss": epoch_train_loss,
            "train_acc": epoch_train_acc,
            "val_loss": epoch_val_loss,
            "val_acc": epoch_val_acc
        })

        if epoch_val_acc > best_acc:
            best_acc = epoch_val_acc
            torch.save(model.state_dict(), "best_car_view_model.pth")
            print("Model saved!")

    print(f"Обучение завершено. Лучшая точность: {best_acc:.4f}")
    wandb.finish()


if __name__ == '__main__':
    train_model()
