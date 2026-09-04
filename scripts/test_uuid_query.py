#!/usr/bin/env python3
"""使用UUID测试ChromaDB查询"""

import sys
try:
    import pysqlite3
    sys.modules["sqlite3"] = pysqlite3
    sys.modules["sqlite3.dbapi2"] = pysqlite3
except ImportError:
    pass

import requests
import json
from sentence_transformers import SentenceTransformer
import os

os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

# 加载模型
print("正在加载模型...")
model = SentenceTransformer('BAAI/bge-small-zh-v1.5')
print("模型加载完成")

# 生成真实向量（512维）
query = "怎么查快递"
embedding = model.encode([query], normalize_embeddings=True)
print(f"向量维度: {len(embedding[0])}")

# 使用UUID进行查询
url = "http://localhost:8001/api/v2/tenants/default_tenant/databases/default_database/collections/b2b3048b-4151-4b9b-9433-cf0319469d17/query"

payload = {
    "tenant": "default_tenant",
    "database": "default_database",
    "query_embeddings": embedding.tolist(),
    "n_results": 3,
    "include": ["documents", "metadatas", "distances"]
}

print(f"查询: {query}")
response = requests.post(url, json=payload)
print(f"状态码: {response.status_code}")

if response.status_code == 200:
    result = response.json()
    print("\n查询结果:")
    for i, (doc, meta, dist) in enumerate(zip(
            result['documents'][0],
            result['metadatas'][0],
            result['distances'][0]
    )):
        similarity = 1 - dist
        print(f"结果 {i+1}: {meta['question']} (相似度: {similarity:.3f})")
        print(f"  答案: {doc[:100]}...")
else:
    print(f"错误: {response.text}")