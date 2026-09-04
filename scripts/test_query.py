#!/usr/bin/env python3
"""测试ChromaDB查询"""
import chromadb
from sentence_transformers import SentenceTransformer
import os

os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

client = chromadb.HttpClient(host="localhost", port=8001)
collection = client.get_collection("customer_service_knowledge")

model = SentenceTransformer('BAAI/bge-small-zh-v1.5')

query = "怎么查快递"
embedding = model.encode([query], normalize_embeddings=True)

results = collection.query(
    query_embeddings=embedding.tolist(),
    n_results=3,
    include=["documents", "metadatas", "distances"]
)

print(f"查询: {query}")
for i, (doc, meta, dist) in enumerate(zip(
        results['documents'][0],
        results['metadatas'][0],
        results['distances'][0]
)):
    similarity = 1 - dist
    print(f"结果 {i+1}: {meta['question']} (相似度: {similarity:.3f})")
    print(f"  答案: {doc[:100]}...")