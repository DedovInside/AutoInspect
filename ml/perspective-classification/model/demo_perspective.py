import torch
import torch.nn as nn
from torchvision import datasets, models, transforms
import random
import os

def demo_model(data_dir="./car_position_dataset", model_path="best_car_view_model.pth", n=5):
    # Настройки
    img_size = 224
    device = torch.device("cuda" if torch.cuda.is_available() else ("mps" if torch.backends.mps.is_available() else "cpu"))

    # Трансформации (такие же, как при валидации во время обучения)
    val_transforms = transforms.Compose([
        transforms.Resize((img_size, img_size)),
        transforms.ToTensor(),
        transforms.Normalize([0.485, 0.456, 0.406], [0.229, 0.224, 0.225])
    ])

    # Загружаем датасет, чтобы получить доступ к фото и списку классов
    if not os.path.exists(data_dir):
        print(f"Ошибка: Папка с датасетом '{data_dir}' не найдена.")
        return

    dataset = datasets.ImageFolder(data_dir, transform=val_transforms)
    class_names = dataset.classes

    # Инициализация модели
    print(f"Загрузка модели из {model_path} на {device}...")
    model = models.resnet18(weights=None)
    num_ftrs = model.fc.in_features
    model.fc = nn.Linear(num_ftrs, len(class_names))
    
    if os.path.exists(model_path):
        model.load_state_dict(torch.load(model_path, map_location=device))
    else:
        print(f"Ошибка: Файл модели '{model_path}' не найден. Сначала обучите модель.")
        return

    model = model.to(device)
    model.eval()

    # Выбираем n случайных индексов
    n = min(n, len(dataset))
    indices = random.sample(range(len(dataset)), n)

    print(f"\nДелаем предсказания для {n} случайных изображений...\n")
    print("-" * 50)
    
    with torch.no_grad():
        for idx in indices:
            image, label = dataset[idx]
            image_batch = image.unsqueeze(0).to(device) # Добавляем размерность батча
            
            # Предсказание
            outputs = model(image_batch)
            probabilities = torch.nn.functional.softmax(outputs, dim=1)[0]
            confidence, predicted_idx = torch.max(probabilities, 0)
            
            true_class = class_names[label]
            predicted_class = class_names[predicted_idx.item()]
            
            # Результаты
            print(f"Истинный класс: {true_class:<15} | Предсказанный класс: {predicted_class:<15} | Уверенность: {confidence.item():.2%}")
            
    print("-" * 50)

if __name__ == "__main__":
    demo_model()

