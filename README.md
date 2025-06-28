# Jarvis - 智能AI助手系统

Jarvis是一个基于微服务架构的智能AI助手系统，具备会话管理、简单的长期记忆和知识库管理功能。

## 🚀 核心特性

### 1. 长期记忆能力
- **事实性记忆**：记住用户的基本信息和重要事实
- **上下文记忆**：保持对话的连贯性和上下文理解

### 2. 知识库管理
- **文档上传**：支持TXT、PDF、DOCX格式文档
- **智能检索**：基于向量相似度的文档搜索
- **知识关联**：建立文档间的关联关系

### 3. 多模态功能
- **天气查询**：实时天气、24小时预报、15天预报
- **网络搜索**：基于博查API的联网搜索能力
- **会话管理**：多会话支持，历史记录保存

## 🏗️ 系统架构

### 调用链路
```
网页端 → api_gateway.py → api_service(Hertz服务) → rag_svr(Kitex服务)
```

### 技术栈
- **前端**：HTML/CSS/JavaScript
- **API网关**：FastAPI (Python)
- **API服务**：Hertz (Go)
- **RAG服务**：Kitex (Go)
- **数据库**：MySQL、MongoDB、Redis
- **向量数据库**：Milvus
- **服务发现**：etcd
- **AI模型**：阿里云通义千问

## 📁 项目结构

```
jarvis/
├── web/                   # 前端和API网关
│   ├── api_gateway.py     # FastAPI网关服务
│   └── static/            # 静态文件
├── server/                # 后端服务
│   ├── api_service/       # Hertz API服务
│   ├── service/           # Kitex 微服务，包括rag服务
│   ├── framework/         # 框架组件
│   └── idl/               # 协议定义文件
├── deploy/                # 部署配置
│   ├── docker-compose/    # Docker编排文件
│   └── README.md          # 部署说明
└── README.md              # 项目说明
```

## 🔧 核心组件

### 1. API网关 (api_gateway.py)
- **功能**：前端请求入口，处理流式对话
- **技术**：FastAPI + OpenAI兼容接口
- **特性**：
  - AI对话生成
  - 流式响应支持
  - 大模型function calling处理
  - 联网搜索
  - 天气查询

### 2. API服务 (api_service)
- **功能**：业务逻辑处理，HTTP API提供
- **技术**：Hertz框架 + etcd服务发现
- **接口**：
  - 会话管理 (/session/*)
  - 文档管理 (/document/*)
  - 记忆管理 (/memory/*)

### 3. RAG服务 (rag_svr)
- **功能**：核心AI服务，向量检索和记忆管理
- **技术**：Kitex框架 + 多数据库支持
- **能力**：
  - 会话管理
  - 文档向量化存储
  - 语义搜索
  - 记忆存储检索

## 🗄️ 数据存储

### MySQL
- 用户信息管理
- 会话记录存储
- 聊天历史保存
- 文档存储
- 记忆存储
- id生成管理

### MongoDB
- 非结构化数据
- 暂未用到

### Redis
- 缓存加速
- 会话状态
- 临时数据

### Milvus
- 文档向量存储和检索
- 记忆向量存储和检索

## 🚀 快速开始

### 1. 环境准备
```bash
# 配置环境变量
cd deploy
cp .env.example .env
# 编辑.env文件，填入你的 API Key
```

### 2. 启动基础服务
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

#### 2.1 检查是否启动成功
```bash
# 查看容器日志
docker-compose -f xxx-docker-compose.yml logs -f

# 查看容器状态
docker-compose -f xxx-docker-compose.yml ps
```

#### 2.2 初始化mysql
```bash
# 进入mysql容器
docker exec -it mysql bash

# 执行MySQL建表语句
mysql -uroot -proot123 < mysql_ddl.sql
```

#### 2.3 初始化milvus
```bash
# 初始化Milvus集合
python3 milvus_init.py
```

### 3. 启动业务服务
```bash
# 启动所有业务服务
docker-compose -f service-docker-compose.yml up -d

# 查看服务状态
docker-compose -f service-docker-compose.yml ps

# 查看容器日志
docker-compose -f service-docker-compose.yml logs -f
```

### 4. 访问系统
- **Web界面**：http://YOUR_IP:8080
- **健康检查**：http://YOUR_IP:8080/health

### 5. 业务服务日志
```bash
# 查看api_gateway日志
docker-compose -f service-docker-compose.yml logs api_gateway

# 查看api_service日志
docker-compose -f service-docker-compose.yml logs api_service

# 查看rag_svr日志
docker-compose -f service-docker-compose.yml logs rag_svr

# 日志路径在服务配置文件config.yaml配置，默认写入到/app/logs目录下
```

### 6. 停止服务
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