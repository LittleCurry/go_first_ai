#!/usr/bin/env python3
"""
查看ChromaDB中已存储的知识库数据
使用方法: python scripts/view_knowledge.py
"""

import sys
try:
    import pysqlite3

    sys.modules["sqlite3"] = pysqlite3
    sys.modules["sqlite3.dbapi2"] = pysqlite3
except ImportError:
    # pysqlite3 未安装时退回系统自带的 sqlite3
    pass
from pathlib import Path
import os

os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

sys.path.insert(0, str(Path(__file__).parent.parent))

import chromadb
from chromadb.config import Settings
import json

PROJECT_ROOT = Path(__file__).parent.parent
CHROMA_PERSIST_DIR = PROJECT_ROOT / "data" / "chroma"
COLLECTION_NAME = "customer_service_knowledge"

def view_collection():
    """查看collection中的所有数据"""
    # 连接Chroma
    client = chromadb.PersistentClient(
        path=str(CHROMA_PERSIST_DIR),
        settings=Settings(anonymized_telemetry=False)
    )

    # 获取collection
    try:
        collection = client.get_collection(COLLECTION_NAME)
    except Exception as e:
        print(f"❌ Collection不存在: {e}")
        return

    # 获取所有数据（不限制数量）
    results = collection.get(
        include=["documents", "metadatas", "embeddings"]
    )

    print("=" * 60)
    print(f"📚 Collection: {COLLECTION_NAME}")
    print(f"📊 总条目数: {len(results['ids'])}")
    print("=" * 60)

    for i, (doc_id, doc, meta) in enumerate(zip(
            results['ids'],
            results['documents'],
            results['metadatas']
    )):
        print(f"\n📌 条目 #{i+1} (ID: {doc_id})")
        print(f"   分类: {meta.get('category', 'N/A')}")
        print(f"   问题: {meta.get('question', 'N/A')}")
        print(f"   关键词: {meta.get('keywords', 'N/A')}")
        print(f"   答案预览: {doc[:100]}..." if len(doc) > 100 else f"   答案: {doc}")
        print("-" * 40)

def search_knowledge(query):
    """搜索知识库"""
    from sentence_transformers import SentenceTransformer

    client = chromadb.PersistentClient(
        path=str(CHROMA_PERSIST_DIR),
        settings=Settings(anonymized_telemetry=False)
    )
    collection = client.get_collection(COLLECTION_NAME)

    # 加载模型
    model = SentenceTransformer('BAAI/bge-small-zh-v1.5')

    # 生成查询向量
    embedding = model.encode([query], normalize_embeddings=True)

    # 搜索
    results = collection.query(
        query_embeddings=embedding.tolist(),
        n_results=5,
        include=["documents", "metadatas", "distances"]
    )

    print("\n" + "=" * 60)
    print(f"🔍 搜索: '{query}'")
    print("=" * 60)

    for i, (doc, meta, dist) in enumerate(zip(
            results['documents'][0],
            results['metadatas'][0],
            results['distances'][0]
    )):
        similarity = 1 - dist
        print(f"\n结果 {i+1} (相似度: {similarity:.3f})")
        print(f"   问题: {meta.get('question', 'N/A')}")
        print(f"   分类: {meta.get('category', 'N/A')}")
        print(f"   答案: {doc[:150]}..." if len(doc) > 150 else f"   答案: {doc}")

if __name__ == "__main__":
    # 查看所有数据
    view_collection()

    # 测试搜索
    search_knowledge("快递什么时候到")
    search_knowledge("怎么退钱")