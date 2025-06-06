from pymilvus import connections, FieldSchema, CollectionSchema, DataType, Collection, utility
import random

# 连接到 Milvus 服务
connections.connect(
    alias="default",
    host="localhost",
    port="19530"
)

# 定义集合结构（类似 MySQL 的 CREATE TABLE）
def create_collection_test1():
    collection_name = "test1"
    
    # 检查集合是否已存在
    if utility.has_collection(collection_name):
        print(f"集合 {collection_name} 已存在！")
        return Collection(collection_name)
        
    # 定义字段
    fields = [
        FieldSchema(name="id", dtype=DataType.INT64, is_primary=True, auto_id=False),
        FieldSchema(name="text", dtype=DataType.VARCHAR, max_length=65535),
        FieldSchema(name="embedding", dtype=DataType.FLOAT_VECTOR, dim=128)
    ]

    # 创建集合模式
    schema = CollectionSchema(
        fields=fields,
        description="just test"
    )

    # 创建集合
    collection = Collection(
        name=collection_name,
        schema=schema,
        using="default",
        shards_num=1
    )

    # 创建索引（可选但推荐）
    index_params = {
        "metric_type": "L2",
        "index_type": "HNSW",
        "params": {"M": 8, "efConstruction": 64}
    }
    collection.create_index(
        field_name="embedding",
        index_params=index_params,
        index_name="hnsw_index"
    )

    print("集合创建成功！")
    return collection

# 随机插入num条数据
def insert_data(collection, num=100):
    # 准备数据
    ids = [i for i in range(num)]  # 主键id
    texts = [f"这是第{i}条测试文本" for i in range(num)]  # 文本内容
    embeddings = [[random.random() for _ in range(128)] for _ in range(num)]  # 128维向量
    
    # 插入数据
    data = [
        ids,        # id字段
        texts,      # text字段
        embeddings  # embedding字段
    ]
    
    collection.insert(data)
    collection.flush()  # 强制刷新内存，确保数据可见
    print(f"成功插入 {len(ids)} 条数据！")

# 根据向量相似度搜索
def search_data(collection, vector, topk=5):
    search_params = {
        "metric_type": "L2",
        "params": {"nprobe": 10}
    }
    results = collection.search(
        data=vector,
        anns_field="embedding",  # 向量字段名
        param=search_params,  # 搜索参数
        limit=topk,  # 返回前5个最相似的结果
        output_fields=["id", "text"]  # 要返回的字段
    )
    return results

if __name__ == "__main__":
    # 创建集合
    collection = create_collection_test1()

    # 插入随机数据
    # insert_data(collection, 100)

    # 把数据加载到milvus内存中，不做这一步，搜索会报错
    # collection.load()

    # 随机搜索向量
    # vector = [[random.random() for _ in range(128)]]

    # 搜索
    # results = search_data(collection, vector, 5)
    # print(results)