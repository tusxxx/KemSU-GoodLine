import grpc
from concurrent import futures
import torch
from PIL import Image
from transformers import CLIPProcessor, CLIPModel
import numpy as np
from sklearn.metrics.pairwise import cosine_similarity
import base64
from io import BytesIO
import similarity_pb2
import similarity_pb2_grpc

class SimilarityService(similarity_pb2_grpc.SimilarityServiceServicer):
    def __init__(self):
        self.model = CLIPModel.from_pretrained("openai/clip-vit-base-patch32")
        self.processor = CLIPProcessor.from_pretrained("openai/clip-vit-base-patch32")

    def CalculateSimilarity(self, request, context):
        image_base64_1 = request.image_base64_1
        image_base64_2 = request.image_base64_2
        
        # Decode images from Base64
        image1 = Image.open(BytesIO(base64.b64decode(image_base64_1)))
        image2 = Image.open(BytesIO(base64.b64decode(image_base64_2)))

        # Process images
        inputs1 = self.processor(images=image1, return_tensors="pt")
        inputs2 = self.processor(images=image2, return_tensors="pt")

        # Get embeddings for images
        with torch.no_grad():
            embedding1 = self.model.get_image_features(**inputs1)
            embedding2 = self.model.get_image_features(**inputs2)

        # Normalize embeddings
        embedding1 = embedding1 / embedding1.norm(p=2, dim=-1, keepdim=True)
        embedding2 = embedding2 / embedding2.norm(p=2, dim=-1, keepdim=True)

        # Calculate cosine similarity between embeddings
        similarity = cosine_similarity(embedding1.cpu().numpy(), embedding2.cpu().numpy())

        return similarity_pb2.SimilarityResponse(similarity=similarity[0][0])

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    similarity_pb2_grpc.add_SimilarityServiceServicer_to_server(SimilarityService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("Server is running on port 50051...")
    server.wait_for_termination()

if __name__ == "__main__":
    serve()