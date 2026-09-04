#!/usr/bin/env python3
"""
知识库向量化脚本 - 使用ChromaDB SDK（自动适配v2 API）
启动Chroma服务后运行: python scripts/init_knowledge.py
"""

import os
import json
import sys
# ChromaDB 要求 sqlite3 >= 3.35.0。
# macOS python.org 版 Python 的 sqlite3 是"静态链接"进 _sqlite3 扩展的
# （ctypes 预加载新版 libsqlite3.dylib 对它无效），本机 Python 3.9.0 内置 3.32.3。
# 因此用 pysqlite3（捆绑新版 SQLite）在导入 chromadb 之前替换标准库 sqlite3。
try:
    import pysqlite3

    sys.modules["sqlite3"] = pysqlite3
    sys.modules["sqlite3.dbapi2"] = pysqlite3
except ImportError:
    # pysqlite3 未安装时退回系统自带的 sqlite3
    pass
from pathlib import Path
# huggingface.co 在本机网络不可达，默认走国内镜像下载 Embedding 模型；
# 若需官方源，可先 export HF_ENDPOINT=https://huggingface.co 覆盖。
os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")

sys.path.insert(0, str(Path(__file__).parent.parent))

import chromadb
from chromadb.config import Settings
from sentence_transformers import SentenceTransformer

PROJECT_ROOT = Path(__file__).parent.parent
KNOWLEDGE_FILE = PROJECT_ROOT / "data" / "knowledge" / "faq.json"
CHROMA_HOST = "localhost"
CHROMA_PORT = 8001
COLLECTION_NAME = "customer_service_knowledge"

def load_knowledge():
    with open(KNOWLEDGE_FILE, 'r', encoding='utf-8') as f:
        return json.load(f)

def init_chroma():
    """通过HTTP连接ChromaDB"""
    client = chromadb.HttpClient(
        host=CHROMA_HOST,
        port=CHROMA_PORT,
        # settings=Settings(anonymized_telemetry=False)
    )
    return client

def vectorize_knowledge():
    print("=" * 50)
    print("开始初始化知识库向量库 (v2 API)")
    print("=" * 50)

    # 1. 加载知识
    knowledge = load_knowledge()
    print(f"📚 加载了 {len(knowledge)} 条知识条目")

    # 2. 连接Chroma
    client = init_chroma()

    # 查看已有collections
    collections = client.list_collections()
    print(f"📋 已有collections: {[c.name for c in collections]}")

    # 删除已存在的collection
    try:
        client.delete_collection(COLLECTION_NAME)
        print(f"🗑️  删除已存在的collection: {COLLECTION_NAME}")
    except:
        pass

    # 创建新collection
    collection = client.create_collection(
        name=COLLECTION_NAME,
        metadata={"hnsw:space": "cosine"}
    )
    print(f"✅ 创建collection: {COLLECTION_NAME}")

    # 3. 加载Embedding模型
    print("正在加载Embedding模型...")
    model = SentenceTransformer('BAAI/bge-small-zh-v1.5')

    # 4. 准备数据
    ids = []
    documents = []
    metadatas = []
    texts_to_embed = []

    for item in knowledge:
        search_text = item['question']
        if item.get('similar_questions'):
            search_text += " " + " ".join(item['similar_questions'])
        if item.get('keywords'):
            search_text += " " + " ".join(item['keywords'])

        ids.append(item['id'])
        documents.append(item['answer'])
        metadatas.append({
            "id": item['id'],
            "category": item['category'],
            "question": item['question'],
            "keywords": ",".join(item.get('keywords', []))
        })
        texts_to_embed.append(search_text)

    # 5. 批量生成向量
    print(f"🔄 正在生成 {len(texts_to_embed)} 条向量的Embedding...")
    embeddings = model.encode(texts_to_embed, normalize_embeddings=True)

    # 6. 存入Chroma
    print("💾 正在存入向量数据库...")
    collection.add(
        ids=ids,
        documents=documents,
        metadatas=metadatas,
        embeddings=embeddings.tolist()
    )

    print(f"✅ 成功存入 {len(knowledge)} 条知识到向量库")
    print("=" * 50)
    print("🎉 知识库初始化完成！")

    # 测试查询
    print("\n🔍 验证存入的数据...")
    count = collection.count()
    print(f"📊 collection中条数: {count}")

    # 随机取一条查看
    sample = collection.get(limit=1)
    if sample['ids']:
        print(f"📌 示例数据:")
        print(f"   ID: {sample['ids'][0]}")
        if sample['metadatas']:
            print(f"   元数据: {sample['metadatas'][0]}")

if __name__ == "__main__":
    vectorize_knowledge()




# #!/usr/bin/env python3
# """
# 知识库向量化脚本
# 将FAQ知识库转换为向量并存入ChromaDB
# 使用方法: python scripts/init_knowledge.py
# """
#
# import sys
#
# # ChromaDB 要求 sqlite3 >= 3.35.0。
# # macOS python.org 版 Python 的 sqlite3 是"静态链接"进 _sqlite3 扩展的
# # （ctypes 预加载新版 libsqlite3.dylib 对它无效），本机 Python 3.9.0 内置 3.32.3。
# # 因此用 pysqlite3（捆绑新版 SQLite）在导入 chromadb 之前替换标准库 sqlite3。
# try:
#     import pysqlite3
#
#     sys.modules["sqlite3"] = pysqlite3
#     sys.modules["sqlite3.dbapi2"] = pysqlite3
# except ImportError:
#     # pysqlite3 未安装时退回系统自带的 sqlite3
#     pass
#
# import json
# import os
# from pathlib import Path
#
# # huggingface.co 在本机网络不可达，默认走国内镜像下载 Embedding 模型；
# # 若需官方源，可先 export HF_ENDPOINT=https://huggingface.co 覆盖。
# os.environ.setdefault("HF_ENDPOINT", "https://hf-mirror.com")
#
# # 添加项目根目录到路径
# sys.path.insert(0, str(Path(__file__).parent.parent))
#
# import chromadb
# from chromadb.config import Settings
# from sentence_transformers import SentenceTransformer
# import uuid
#
# # 配置
# PROJECT_ROOT = Path(__file__).parent.parent
# KNOWLEDGE_FILE = PROJECT_ROOT / "data" / "knowledge" / "faq.json"
# CHROMA_PERSIST_DIR = PROJECT_ROOT / "data" / "chroma"
# COLLECTION_NAME = "customer_service_knowledge"
#
# def load_knowledge():
#     """加载知识库JSON"""
#     with open(KNOWLEDGE_FILE, 'r', encoding='utf-8') as f:
#         return json.load(f)
#
# def init_chroma():
#     """初始化ChromaDB客户端"""
#     # 确保目录存在
#     CHROMA_PERSIST_DIR.mkdir(parents=True, exist_ok=True)
#
#     client = chromadb.PersistentClient(
#         path=str(CHROMA_PERSIST_DIR),
#         settings=Settings(anonymized_telemetry=False)
#     )
#     return client
#
# def create_embeddings():
#     """创建Embedding模型（使用本地模型，免费无需API）"""
#     # 使用BGE-small，轻量级中文模型，Mac上运行流畅
#     print("正在加载Embedding模型（首次加载需要下载，约400MB）...")
#     model = SentenceTransformer('BAAI/bge-small-zh-v1.5')
#     return model
#
# def vectorize_knowledge():
#     """主函数：向量化知识库"""
#     print("=" * 50)
#     print("开始初始化知识库向量库")
#     print("=" * 50)
#
#     # 1. 加载知识
#     knowledge = load_knowledge()
#     print(f"📚 加载了 {len(knowledge)} 条知识条目")
#
#     # 2. 初始化Chroma
#     client = init_chroma()
#
#     # 删除已存在的collection（如果需要重新初始化）
#     try:
#         client.delete_collection(COLLECTION_NAME)
#         print(f"🗑️  删除已存在的collection: {COLLECTION_NAME}")
#     except:
#         pass
#
#     # 创建新collection
#     collection = client.create_collection(
#         name=COLLECTION_NAME,
#         metadata={"hnsw:space": "cosine"}  # 使用余弦相似度
#     )
#     print(f"✅ 创建collection: {COLLECTION_NAME}")
#
#     # 3. 加载Embedding模型
#     model = create_embeddings()
#
#     # 4. 准备数据
#     ids = []
#     documents = []
#     metadatas = []
#     texts_to_embed = []
#
#     for item in knowledge:
#         # 构建用于检索的文本：问题 + 相似问法 + 关键词
#         search_text = item['question']
#         if item.get('similar_questions'):
#             search_text += " " + " ".join(item['similar_questions'])
#         if item.get('keywords'):
#             search_text += " " + " ".join(item['keywords'])
#
#         ids.append(item['id'])
#         documents.append(item['answer'])  # 返回的答案
#         metadatas.append({
#             "id": item['id'],
#             "category": item['category'],
#             "question": item['question'],
#             "keywords": ",".join(item.get('keywords', []))
#         })
#         texts_to_embed.append(search_text)
#
#     # 5. 批量生成向量
#     print(f"🔄 正在生成 {len(texts_to_embed)} 条向量的Embedding...")
#     embeddings = model.encode(texts_to_embed, normalize_embeddings=True)
#
#     # 6. 存入Chroma
#     print("💾 正在存入向量数据库...")
#     collection.add(
#         ids=ids,
#         documents=documents,
#         metadatas=metadatas,
#         embeddings=embeddings.tolist()
#     )
#
#     print(f"✅ 成功存入 {len(knowledge)} 条知识到向量库")
#     print(f"📁 数据存储位置: {CHROMA_PERSIST_DIR}")
#     print("=" * 50)
#     print("🎉 知识库初始化完成！")
#
#     # 测试查询
#     test_query(collection, model)
#
# def test_query(collection, model):
#     """测试查询功能"""
#     print("\n🔍 测试查询...")
#     test_questions = [
#         "怎么查我的快递",
#         "我想退货",
#         "密码忘了"
#     ]
#
#     for q in test_questions:
#         embedding = model.encode([q], normalize_embeddings=True)
#         results = collection.query(
#             query_embeddings=embedding.tolist(),
#             n_results=2
#         )
#         print(f"\n问题: {q}")
#         for i, (doc, meta, dist) in enumerate(zip(
#             results['documents'][0],
#             results['metadatas'][0],
#             results['distances'][0]
#         )):
#             # 距离越小越相似
#             similarity = 1 - dist  # 转换为相似度
#             print(f"  结果 {i+1}: {meta['question']} (相似度: {similarity:.3f})")
#
# if __name__ == "__main__":
#     vectorize_knowledge()