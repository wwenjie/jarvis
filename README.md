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

### 5. 文字识别与翻译
- **文字识别**：基于本地部署的EasyOCR模型，支持简体中文和英文的文字识别
- **自动翻译**：识别到的英文文字自动翻译为中文
- **识别信息**：提供识别置信度和文字位置信息

## 🏗️ 系统架构

### 整体架构图
```
┌─────────────────┐    HTTP/WebSocket    ┌─────────────────┐
│   前端界面       │ ───────────────────→ │   API网关       │
│   (Vue.js)      │                      │ (api_gateway.py)│
└─────────────────┘                      └─────────────────┘
                                                │
                                                ▼
                    ┌─────────────────────────────────────────────────┐
                    │              智能对话服务                        │
                    │            (agent.py)                           │
                    │  • AI对话生成 • 流式响应 • Function Calling      │
                    │  • 联网搜索 • 天气查询 • 记忆管理                 │
                    └─────────────────────────────────────────────────┘
                                │
                    ┌───────────┼───────────┐
                    ▼           ▼           ▼
        ┌─────────────────┐ ┌─────────────┐ ┌───────────────────┐
        │  文字识别        │ │  花朵识别    │ │   API服务         │
        │  (ocr_svr)      │ │(flower_infer)│ │ (api_service)    │
        │  • EasyOCR      │ │• EfficientNet│ │ • Hertz框架       │
        │  • 中英文识别    │ │• Flower-102  │ │ • golang微服务入口│
        │  • 自动翻译      │ │• 102种花卉   │ │ • 请求转发        │
        └─────────────────┘ └─────────────┘ └───────────────────┘
                                                    │
                                                    ▼
                                        ┌─────────────────┐
                                        │   RAG微服务      │
                                        │ (rag_svr)       │
                                        │ • Kitex框架     │
                                        │ • 记忆管理      │
                                        │ • 文档管理      │
                                        │ • 向量检索      │
                                        └─────────────────┘
                                                    │
                    ┌────────────────────────────────────────────────────────┐
                    ▼                 ▼                 ▼                    ▼
        ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
        │     MySQL       │ │     Milvus      │ │     Redis       │ │     Etcd        │
        │  • 会话记录      │ │  • 向量存储      │ │  • 缓存加速      │ │ • 服务注册和发现 │
        │  • 聊天历史      │ │  • 相似度检索    │ │  • 会话状态      │ │                 │
        │  • 记忆存储      │ │  • 记忆向量      │ │  • 临时数据      │ │                 │
        │  • 文档存储      │ │  • 文档向量      │ │                 │ │                 │
        └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘
```

### 技术栈
- **前端**：Vue.js
- **API网关**：FastAPI (Python)
- **API服务**：Hertz (Go)
- **RAG微服务**：Kitex (Go)
- **常规数据库**：MySQL、MongoDB
- **向量数据库**：Milvus
- **缓存**：Redis
- **服务发现**：Etcd 
- **AI语言模型**：Qwen-turbo
- **图像花朵识别模型**：EfficientNet-B1（基于Flower-102数据集训练）
- **图像文字识别模型**：EasyOCR

## 🔧 核心组件

### 1. API网关 (api_gateway.py)
- **功能**：前端请求入口，路由分发到各个后端服务
- **技术**：FastAPI + 静态文件服务
- **特性**：
  - 静态文件服务（HTML、CSS、JS）
  - 请求路由转发

### 2. Agent服务 (agent.py)
- **功能**：智能对话处理，AI交互核心服务
- **技术**：FastAPI + OpenAI兼容接口
- **特性**：
  - AI对话生成
  - 流式响应支持
  - 大模型function calling处理
  - 联网搜索
  - 天气查询
  - 记忆管理
  - 文档搜索
  - 图像花朵识别
  - 图像文字识别和翻译

### 3. API服务 (api_service)
- **功能**：golang微服务的api入口
- **技术**：Hertz框架 + etcd服务发现

### 4. RAG服务 (rag_svr)
- **功能**：核心AI服务，向量检索和记忆管理
- **技术**：Kitex框架 + 多数据库支持
- **能力**：
  - 会话管理
  - 文档存储和检索
  - 记忆存储和检索
  - 向量存储和检索

### 5. 花朵识别服务 (flower_infer.py)
- **功能**：图片花朵识别，支持Flower-102数据集的102种花卉分类
- **技术**：FastAPI + ONNXRuntime + EfficientNet-B1
- **模型说明**：本服务使用EfficientNet-B1作为基础模型，结合Flower-102数据集进行迁移学习训练，最终导出ONNX模型用于高效推理。
- **接口**：
  - /infer：上传图片，返回花朵类别及ID

### 6. 文字识别服务 (ocr_svr.py)
- **功能**：图片文字识别，支持英文和简体中文
- **技术**：FastAPI + EasyOCR + OpenAI翻译
- **模型说明**：使用EasyOCR模型进行文字识别。
- **接口**：
  - /ocr：上传图片，返回识别结果（包含文本、置信度、坐标）

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

# 查看ocr_svr日志
docker-compose -f service-docker-compose.yml logs ocr_svr

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