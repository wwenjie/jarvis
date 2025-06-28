# Docker Compose 配置

本目录包含各种服务的Docker Compose配置文件。以下是每个文件的简要介绍和使用说明。

## etcd-docker-compose.yml

etcd集群。

## service-docker-compose.yml

jarvis后台服务。包括api_gateway和rag_svr

## milvus-docker-compose.yml

Milvus向量数据库单例。

## redis-docker-compose.yml

Redis集群。支持宿主机内多redis节点和跨宿主机的集群。本容器配置是用的host网络模式，而redis-docker-compose.yml.bak是用的桥接模式。

## mongodb-docker-compose.yml

MongoDB数据库集群。1个configsvr，1个mongos，3个mongod分片

## mysql-docker-compose.yml

mysql单例。

## 环境变量配置

在启动服务之前，请按照以下步骤配置环境变量：

### 1. 创建 .env 文件

在 `deploy` 目录下将 `.env.example` 复制为 `.env` 文件：

```bash
cd deploy
cp .env.example .env
```

然后将 `.env` 文件中的占位符替换为你的实际 API Key：

### 2. 启动服务

配置完成后，在 `deploy` 目录下启动服务：

```bash
# 启动所有服务
docker-compose -f service-docker-compose.yml up -d

# 查看服务状态
docker-compose -f service-docker-compose.yml ps

# 查看服务日志
docker-compose -f service-docker-compose.yml logs -f
```

### 3. 停止服务

```bash
# 停止所有服务
docker-compose -f service-docker-compose.yml down
```