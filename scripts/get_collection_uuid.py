#!/usr/bin/env python3
"""获取collection的UUID"""

# 处理sqlite3版本问题
try:
    import pysqlite3
    import sys
    sys.modules["sqlite3"] = pysqlite3
    sys.modules["sqlite3.dbapi2"] = pysqlite3
except ImportError:
    pass

import chromadb
import json
import os
from pathlib import Path

os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

# 连接ChromaDB
client = chromadb.HttpClient(host="localhost", port=8001)

# 获取指定collection的详细信息
try:
    collection = client.get_collection("customer_service_knowledge")
    print(f"Collection名称: {collection.name}")
    print(f"Collection ID (UUID): {collection.id}")
    print(f"元数据: {collection.metadata}")
    print(f"文档数量: {collection.count()}")
except Exception as e:
    print(f"获取失败: {e}")
    # 尝试列出所有collections
    print("\n尝试列出所有collections:")
    collections = client.list_collections()
    for col in collections:
        print(f"  - 名称: {col.name}, ID: {col.id}")