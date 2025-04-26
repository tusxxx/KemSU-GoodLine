import torch
from PIL import Image
from transformers import CLIPProcessor, CLIPModel
import numpy as np
from sklearn.metrics.pairwise import cosine_similarity
import argparse
import base64
from io import BytesIO

def calculate_similarity(image_base64_1, image_base64_2):
    # Загружаем модель CLIP
    model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32")
    processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

    # Декодируем изображения из формата Base64
    image1 = Image.open(BytesIO(base64.b64decode(image_base64_1)))
    image2 = Image.open(BytesIO(base64.b64decode(image_base64_2)))

    # Обрабатываем изображения
    inputs1 = processor(images=image1, return_tensors="pt")
    inputs2 = processor(images=image2, return_tensors="pt")

    # Получаем эмбеддинги для изображений
    with torch.no_grad():
        embedding1 = model.get_image_features(**inputs1)
        embedding2 = model.get_image_features(**inputs2)

    # Нормализуем эмбеддинги
    embedding1 = embedding1 / embedding1.norm(p=2, dim=-1, keepdim=True)
    embedding2 = embedding2 / embedding2.norm(p=2, dim=-1, keepdim=True)

    # Вычисляем косинусное сходство между эмбеддингами
    similarity = cosine_similarity(embedding1.cpu().numpy(), embedding2.cpu().numpy())

    return similarity[0][0]

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Calculate cosine similarity between two images in Base64 format using CLIP.")
    parser.add_argument("image_base64_1", type=str, help="Base64 encoded string of the first image.")
    parser.add_argument("image_base64_2", type=str, help="Base64 encoded string of the second image.")
    
    args = parser.parse_args()
    
    similarity = calculate_similarity(args.image_base64_1, args.image_base64_2)
    print(f"Cosine similarity between images: {similarity}")