#!/usr/bin/env python3
"""
Embedding HTTP服务
为Go提供向量生成API
启动: python scripts/embedding_server.py
"""

from flask import Flask, request, jsonify
from sentence_transformers import SentenceTransformer
import numpy as np
import os

# 使用国内镜像下载模型
os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

app = Flask(__name__)

# 加载模型
print("正在加载Embedding模型...")
model = SentenceTransformer('BAAI/bge-small-zh-v1.5')
print("模型加载完成，服务已就绪")

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "ok"})

@app.route('/embed', methods=['POST'])
def embed():
    data = request.json
    texts = data.get('input', [])

    if not texts:
        return jsonify({"error": "input is required"}), 400

    # 生成向量
    embeddings = model.encode(texts, normalize_embeddings=True)

    # 转换为列表
    if isinstance(embeddings, np.ndarray):
        embeddings = embeddings.tolist()

    return jsonify({
        "data": [{"embedding": emb} for emb in embeddings]
    })

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8002, debug=False)