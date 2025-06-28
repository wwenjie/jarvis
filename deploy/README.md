# Docker Compose 配置

本目录包含各种服务的Docker Compose配置文件。以下是每个文件的简要介绍和使用说明。

## 配置文件说明

### etcd-docker-compose.yml

etcd集群。

### service-docker-compose.yml

jarvis后台服务。包括api_gateway和rag_svr

### milvus-docker-compose.yml

Milvus向量数据库单例。

### redis-docker-compose.yml

Redis集群。支持宿主机内多redis节点和跨宿主机的集群。本容器配置是用的host网络模式，而redis-docker-compose.yml.bak是用的桥接模式。

### mongodb-docker-compose.yml

MongoDB数据库集群。1个configsvr，1个mongos，3个mongod分片

### mysql-docker-compose.yml

mysql单例。

## 环境变量配置

在启动服务之前，请按照以下步骤配置环境变量：

### 1. 创建 .env 文件

在 `deploy` 目录下将 `.env.example` 复制为 `.env` 文件：

```bash
cd deploy
cp .env.example .env
```

然后将 `.env` 文件中的占位符替换为你的实际 API Key。

## 完整启动流程

### 1. 启动基础服务
```bash
# 启动etcd集群
docker-compose -f etcd-docker-compose.yml up -d

# 启动MySQL数据库
docker-compose -f mysql-docker-compose.yml up -d

# 启动Redis集群
docker-compose -f redis-docker-compose.yml up -d

# 启动MongoDB集群
docker-compose -f mongodb-docker-compose.yml up -d

# 启动Milvus向量数据库
docker-compose -f milvus-docker-compose.yml up -d
```

#### 1.1 检查是否启动成功
```bash
# 查看容器日志
docker-compose -f xxx-docker-compose.yml logs

# 查看容器状态
docker-compose -f xxx-docker-compose.yml ps
```

#### 1.2 初始化mysql
```bash
# 进入mysql容器
docker exec -it mysql bash

# 执行MySQL建表语句
mysql -uroot -proot123 < mysql_ddl.sql
```

#### 1.3 初始化milvus
```bash
# 初始化Milvus集合
python3 milvus_init.py
```

### 2. 启动业务服务

配置完成后，在 `deploy` 目录下启动服务：

```bash
# 启动所有业务服务
docker-compose -f service-docker-compose.yml up -d

# 查看服务状态
docker-compose -f service-docker-compose.yml ps

# 查看服务日志
docker-compose -f service-docker-compose.yml logs -f
```

### 3. 访问系统
- **Web界面**：http://YOUR_IP:8080
- **健康检查**：http://YOUR_IP:8080/health

### 4. 业务服务日志
```bash
# 查看api_gateway日志
docker-compose -f service-docker-compose.yml logs api_gateway

# 查看api_service日志
docker-compose -f service-docker-compose.yml logs api_service

# 查看rag_svr日志
docker-compose -f service-docker-compose.yml logs rag_svr

# 日志路径在服务配置文件config.yaml配置，默认写入到/app/logs目录下
```

### 5. 停止服务

```bash
# 停止业务服务
docker-compose -f service-docker-compose.yml down

# 停止所有基础服务
docker-compose -f etcd-docker-compose.yml down
docker-compose -f mysql-docker-compose.yml down
docker-compose -f redis-docker-compose.yml down
docker-compose -f mongodb-docker-compose.yml down
docker-compose -f milvus-docker-compose.yml down
```