# Jarvis - 智能AI助手系统

Jarvis是一个智能AI助手系统，具备会话管理、简单的长期记忆和知识库管理功能。

## 🚀 核心特性

### 1. 记忆能力
- **上下文记忆**：保持对话的连贯性和上下文理解
- **长期记忆**：记住与用户对话中的重要信息并持久化存储

### 2. 知识库管理
- **文档上传**：目前只支持TXT格式文档
- **智能检索**：基于向量相似度的文档搜索

### 3. 扩展功能
- **天气查询**：实时天气、24小时预报、15天预报
- **网络搜索**：基于博查API的联网搜索能力
- **会话管理**：多会话支持，历史聊天记录保存

### 4. 图片花朵识别
- **花朵识别**：支持识别图片花朵类别，所用模型基于EfficientNet-B1并在Flower-102数据集上训练，支持识别102种花朵的模型

## 🏗️ 系统架构

### 调用链路
```
网页端 → api_gateway.py → api_service(Hertz服务) → rag_svr(Kitex服务)
           ↓
        flower_infer.py（花朵识别服务）
```

### 技术栈
- **前端**：Vue.js
- **API网关**：FastAPI (Python)
- **API服务**：Hertz (Go)
- **RAG微服务**：Kitex (Go)
- **数据库**：MySQL、MongoDB
- **向量数据库**：Milvus
- **缓存**：Redis
- **服务发现**：Etcd 
- **AI语言模型**：Qwen-turbo
- **花朵识别模型**：EfficientNet-B1（基于Flower-102数据集训练，本地ONNX部署）

## 🔧 核心组件

### 1. API网关 (api_gateway.py)
- **功能**：前端请求入口，处理流式对话、花朵图片识别
- **技术**：FastAPI + OpenAI兼容接口
- **特性**：
  - AI对话生成
  - 流式响应支持
  - 大模型function calling处理
  - 联网搜索
  - 天气查询
  - 花朵图片识别（转发图片到flower_infer服务，返回识别结果）

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

### 4. 花朵识别服务 (flower_infer.py)
- **功能**：图片花朵识别，支持Flower-102数据集的102种花卉分类
- **技术**：FastAPI + ONNXRuntime + EfficientNet-B1
- **模型说明**：本服务使用EfficientNet-B1作为基础模型，结合Flower-102数据集进行迁移学习训练，最终导出ONNX模型用于高效推理。
- **接口**：
  - /infer：上传图片，返回花朵类别及ID

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
- 记忆向量存储和检索
- 文档语句向量存储和检索


## 🚀 服务部署

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

# 查看flower_infer日志
docker-compose -f service-docker-compose.yml logs flower_infer

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