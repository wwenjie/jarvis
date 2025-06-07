# Docker Compose 配置

本目录包含各种服务的Docker Compose配置文件。以下是每个文件的简要介绍和使用说明。

## etcd-docker-compose.yml

该文件使用Docker Compose配置etcd集群。Etcd是一个分布式键值存储，提供了一种可靠的方式来在机器集群中存储数据。

### 使用方法

启动etcd集群，运行：
```bash
docker-compose -f etcd-docker-compose.yml up -d
```

停止集群，运行：
```bash
docker-compose -f etcd-docker-compose.yml down
```

## service-docker-compose.yml

该文件用于配置和运行特定服务。请将"service"替换为实际的服务名称或描述。

### 使用方法

启动服务，运行：
```bash
docker-compose -f service-docker-compose.yml up -d
```

停止服务，运行：
```bash
docker-compose -f service-docker-compose.yml down
```

## milvus-docker-compose.yml

该文件使用Docker Compose设置Milvus实例。Milvus是一个开源向量数据库，用于嵌入相似性搜索和AI应用。

### 使用方法

启动Milvus，运行：
```bash
docker-compose -f milvus-docker-compose.yml up -d
```

停止Milvus，运行：
```bash
docker-compose -f milvus-docker-compose.yml down
```

## redis-docker-compose.yml

该文件使用Docker Compose配置Redis集群。Redis是一个内存数据结构存储，用作缓存。
本配置的网络模式是的host模式，而redis-docker-compose.yml.bak是用的桥接模式。

### 使用方法

启动Redis集群，运行：
```bash
docker-compose -f redis-docker-compose.yml up -d
```

停止集群，运行：
```bash
docker-compose -f redis-docker-compose.yml down
```

## mongo-docker-compose.yml

该文件使用Docker Compose设置MongoDB实例。MongoDB是一个NoSQL数据库，以灵活的、类似JSON的文档存储数据。

### 使用方法

启动MongoDB，运行：
```bash
docker-compose -f mongo-docker-compose.yml up -d
```

停止MongoDB，运行：
```bash
docker-compose -f mongo-docker-compose.yml down
```

## mysql-docker-compose.yml

该文件使用Docker Compose配置MySQL实例。MySQL是一个流行的开源关系数据库管理系统。

### 使用方法

启动MySQL，运行：
```bash
docker-compose -f mysql-docker-compose.yml up -d
```

停止MySQL，运行：
```bash
docker-compose -f mysql-docker-compose.yml down
``` 