#!/usr/bin/env python3
# -*- coding: utf-8 -*-

from pymilvus import connections, FieldSchema, CollectionSchema, DataType, Collection, utility
import sys

# Milvus 配置
MILVUS_HOST = "localhost"
MILVUS_PORT = "19530"

# 集合配置
COLLECTIONS_CONFIG = {
    "chat_memory": {
        "dim": 1024,
        "description": "对话记忆向量集合",
        "shards_num": 1
    },
    "document": {
        "dim": 1024,
        "description": "文档向量集合",
        "shards_num": 1
    }
}

# 索引配置
INDEX_PARAMS = {
    "metric_type": "L2",
    "index_type": "HNSW",
    "params": {
        "M": 8,           # 图中每个节点的最大边数
        "efConstruction": 64  # 建图时的搜索宽度
    }
}

# 搜索配置
SEARCH_PARAMS = {
    "metric_type": "L2",
    "params": {"ef": 64}  # 搜索时的搜索宽度，越大结果越精确
}

def connect_milvus():
    """连接到 Milvus 服务"""
    try:
        connections.connect(
            alias="default",
            host=MILVUS_HOST,
            port=MILVUS_PORT
        )
        print(f"成功连接到 Milvus 服务 ({MILVUS_HOST}:{MILVUS_PORT})")
    except Exception as e:
        print(f"连接 Milvus 失败: {str(e)}")
        sys.exit(1)

def create_collection(collection_name):
    """创建集合并建立索引"""
    if collection_name not in COLLECTIONS_CONFIG:
        print(f"错误：未定义的集合 {collection_name}")
        sys.exit(1)

    config = COLLECTIONS_CONFIG[collection_name]
    
    # 检查集合是否已存在
    if utility.has_collection(collection_name):
        print(f"集合 {collection_name} 已存在，正在删除...")
        utility.drop_collection(collection_name)
    
    # 定义字段
    fields = [
        FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
        FieldSchema(name="vector", dtype=DataType.FLOAT_VECTOR, dim=config["dim"])
    ]

    # 创建集合模式
    schema = CollectionSchema(
        fields=fields,
        description=config["description"]
    )

    # 创建集合
    collection = Collection(
        name=collection_name,
        schema=schema,
        using="default",
        shards_num=config["shards_num"]
    )

    # 创建索引
    collection.create_index(
        field_name="vector",
        index_params=INDEX_PARAMS,
        index_name="hnsw_index"
    )

    # 加载集合到内存
    collection.load()
    print(f"成功创建集合 {collection_name} 并建立索引")

def main():
    """主函数"""
    # 连接 Milvus
    connect_milvus()
    
    # 创建所有集合
    for collection_name in COLLECTIONS_CONFIG:
        create_collection(collection_name)
        print("\n" + "="*50 + "\n")

    # 查询当前已有的全部数据
    # print("查询当前已有的全部数据")
    collection = Collection(name="chat_memory")
    results = collection.query(
        expr="id > 0",  # 查询所有数据n  
        output_fields=["id", "vector"]
    )
    print(f"总数据量: {len(results)}")
    for i, result in enumerate(results):
        print(f"\n记录 {i+1}:")
        print(f"ID: {result['id']}")
        print(f"向量维度: {len(result['vector'])}")
        # 删除
        collection.delete(expr=f"id == '{result['id']}'")
    collection.flush()

if __name__ == "__main__":
    main() 