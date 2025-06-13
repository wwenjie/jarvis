# app.py
from fastapi import FastAPI, Request, HTTPException, UploadFile, File, Query
from fastapi.responses import StreamingResponse, RedirectResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
import httpx
import json
import uuid
from datetime import datetime
import asyncio
import os
from typing import List, Dict
from contextlib import asynccontextmanager
from openai import OpenAI

# API 网关配置
API_GATEWAY_BASE_URL = "http://localhost:8081"  # 根据实际部署情况修改

# 创建应用启动上下文管理器
@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动前执行
    init()
    yield
    # 关闭时执行

# 创建FastAPI应用
app = FastAPI(lifespan=lifespan)

# 添加静态文件服务
app.mount("/static", StaticFiles(directory="static"), name="static")

# 添加CORS中间件允许跨域请求
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # 在生产环境中应该设置为特定域名
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 全局变量
openai_client = None  # 重命名为 openai_client 以避免混淆

# 函数定义
FUNCTIONS = [
    {
        "name": "add_memory",
        "description": "添加新的记忆",
        "parameters": {
            "type": "object",
            "properties": {
                "user_id": {
                    "type": "integer",
                    "description": "用户ID"
                },
                "content": {
                    "type": "string",
                    "description": "记忆内容"
                },
                "memory_type": {
                    "type": "string",
                    "description": "记忆类型",
                    "enum": ["fact", "reminder", "preference", "context"]
                },
                "importance": {
                    "type": "number",
                    "description": "重要性(0-1)"
                },
                "metadata": {
                    "type": "object",
                    "description": "元数据"
                }
            },
            "required": ["user_id", "content", "memory_type", "importance"]
        }
    },
    {
        "name": "get_memory",
        "description": "获取指定ID的记忆信息",
        "parameters": {
            "type": "object",
            "properties": {
                "memory_id": {
                    "type": "integer",
                    "description": "记忆ID"
                }
            },
            "required": ["memory_id"]
        }
    },
    {
        "name": "search_memories",
        "description": "搜索相关记忆",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "搜索查询"
                },
                "limit": {
                    "type": "integer",
                    "description": "返回结果数量"
                }
            },
            "required": ["query"]
        }
    },
    {
        "name": "delete_memory",
        "description": "删除指定ID的记忆",
        "parameters": {
            "type": "object",
            "properties": {
                "memory_id": {
                    "type": "integer",
                    "description": "记忆ID"
                },
                "reason": {
                    "type": "string",
                    "description": "删除原因"
                }
            },
            "required": ["memory_id", "reason"]
        }
    }
]

# 系统提示词
SYSTEM_PROMPT = """你是一个带长远记忆的 AI 助手，名字叫 Jarvis。

你的主要特点：
1. 长期记忆能力
   - 能记住用户的重要信息
   - 能记住用户的偏好设置
   - 能记住重要的对话历史
   - 能记住用户的任务和提醒

2. 个性化交互
   - 根据用户习惯调整回复风格
   - 记住用户的常用表达方式
   - 适应用户的交流节奏
   - 保持对话的连贯性

3. 智能学习
   - 从对话中学习用户习惯
   - 优化回复策略
   - 记住有效的解决方案
   - 避免重复的错误

4. 任务管理
   - 记住并提醒用户的任务
   - 跟踪任务的完成状态
   - 提供任务相关的建议
   - 帮助用户规划时间

5. 情感理解
   - 理解用户的情感状态
   - 提供适当的情感支持
   - 记住用户的情感偏好
   - 调整回复的语气和风格

6. 知识积累
   - 记住用户分享的知识
   - 积累问题解决方案
   - 建立知识关联网络
   - 持续优化知识体系

7. 隐私保护
   - 保护用户的隐私信息
   - 遵守数据安全规范
   - 谨慎处理敏感内容
   - 及时清理过期信息

你的行为准则：
1. 保持对话的连贯性和上下文理解
2. 主动学习和适应用户的习惯
3. 提供准确、及时、有用的信息
4. 在合适的时候主动提供建议

你的记忆类型：
1. 事实性记忆：用户的基本信息、重要事实
2. 提醒类记忆：任务、日程、提醒事项
3. 用户偏好：使用习惯、交互偏好
4. 上下文记忆：当前对话的上下文信息

函数调用格式要求：
1. 调用函数时，必须生成完整的 JSON 格式参数
2. 每个函数调用的参数必须在一行内完成，不要分多次生成
3. JSON 格式必须完整，包含开始和结束的大括号
4. 所有必需的参数都必须提供
5. 参数值必须符合指定的类型要求

搜索结果处理：
1. 当搜索结果为空时，即memories为[]，直接告诉用户没有找到相关信息
2. 不要重复搜索相同或类似的内容

例如，搜索记忆的函数调用应该是：
{"query": "搜索关键词","limit": 5}

请记住：你的目标是成为用户的得力助手，通过长期记忆和个性化服务，提供更好的用户体验。"""

# 用户提示词模板
USER_PROMPT_TEMPLATE = """用户信息：
- 用户ID：{user_id}
- 会话ID：{session_id}
- 当前时间：{current_time}

历史记忆：
{memory}

当前对话：
{query}

请根据以上信息，结合你的长期记忆，为用户提供最合适的回复。"""

# 初始化函数
def init():
    global openai_client
    
    # 初始化OpenAI客户端
    openai_client = OpenAI(
        api_key=os.getenv("DASHSCOPE_API_KEY"),
        base_url="https://dashscope.aliyuncs.com/compatible-mode/v1",
        default_headers={
            "X-DashScope-SSE": "enable"
        }
    )

@app.get("/")
async def redirect_to_document():
    return RedirectResponse(url="/static/documents.html")

# 文档管理 API
@app.post("/api/upload_doc")
async def upload_document(file: UploadFile = File(...)):
    if not file.filename.endswith((".txt", ".pdf", ".docx")):
        raise HTTPException(status_code=400, detail="仅支持.txt或.pdf或.docx文件")
    
    # 确保docs目录存在
    os.makedirs("docs", exist_ok=True)
    
    # 保存文件到docs目录
    file_path = os.path.join("docs", file.filename)
    
    # 检查文件名是否重复，如果重复则添加时间戳
    if os.path.exists(file_path):
        filename, extension = os.path.splitext(file.filename)
        timestamp = datetime.now().strftime("%Y%m%d%H%M%S")
        file_path = os.path.join("docs", f"{filename}_{timestamp}{extension}")
        file_name = f"{filename}_{timestamp}{extension}"
    else:
        file_name = file.filename
        
    # 读取文件内容
    content = await file.read()
    
    # 保存文件到磁盘
    with open(file_path, "wb") as f:
        f.write(content)
    
    # 读取文件内容（用于存储在内存中）
    if file.filename.endswith(".txt"):
        try:
            content_text = content.decode("utf-8")
        except UnicodeDecodeError:
            content_text = content.decode("gbk", errors="ignore")
    else:
        content_text = f"文件内容已保存到 {file_path}"  # 实际需用pdf/docx解析库
    
    # 调用 API 网关添加文档
    async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
        response = await client.post(
            "/document/add",
            json={
                "user_id": 1,  # 这里需要根据实际用户ID修改
                "title": file_name,
                "content": content_text,
                "metadata": json.dumps({"path": file_path})
            }
        )
        
        if response.status_code != 200:
            raise HTTPException(status_code=500, detail="添加文档到知识库失败")
            
        result = response.json()
        if result.get("code", 0) != 0:
            raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
            
        doc_id = result["doc_id"]
    
    return {"id": doc_id, "name": file_name}

@app.get("/api/list_docs")
async def list_documents():
    try:
        print(f"正在请求 API 网关: {API_GATEWAY_BASE_URL}/document/list")
        print(f"请求参数: user_id=1, page=1, page_size=100")
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.get(
                "/document/list",
                params={
                    "user_id": 1,  # 这里需要根据实际用户ID修改
                    "page": 1,     # 添加页码参数
                    "page_size": 100  # 添加每页大小参数
                }
            )
            
            print(f"API 网关响应状态码: {response.status_code}")
            print(f"API 网关响应内容: {response.text}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取文档列表失败")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
                
            return [{"id": str(doc["doc_id"]), "name": doc["title"]} for doc in result.get("documents", [])]
            
    except Exception as e:
        print(f"获取文档列表失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取文档列表失败: {str(e)}")

@app.delete("/api/documents/{doc_id}")
async def delete_document(doc_id: str):
    try:
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.delete(
                f"/document/{doc_id}",
                params={
                    "user_id": 1  # 这里需要根据实际用户ID修改
                }
            )
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="删除文档失败")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
            
            return {"message": "删除成功"}
            
    except Exception as e:
        print(f"删除文档失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"删除文档失败: {str(e)}")

@app.post("/api/stream")
async def stream_post(request: Request):
    try:
        # 解析请求体中的 JSON 数据
        req_data = await request.json()
        query = req_data.get("query")
        session_id = req_data.get("session_id")  # 获取会话ID
        return await process_stream_request(query, session_id)
    except Exception as e:
        error_msg = str(e)
        print(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

@app.get("/api/stream")
async def stream_get(query: str = Query(None), session_id: str = Query(None)):
    try:
        if not query:
            raise HTTPException(status_code=400, detail="Missing query parameter")
        return await process_stream_request(query, session_id)
    except Exception as e:
        error_msg = str(e)
        print(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

async def process_function_call(func_call: dict, api_gateway_base_url: str) -> dict:
    """处理单个函数调用
    
    Args:
        func_call: 函数调用信息，包含name和arguments
        api_gateway_base_url: API网关基础URL
        
    Returns:
        dict: 函数调用结果
    """
    try:
        if func_call["name"] == "search_memories":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/memory/search",
                    params={
                        "query": func_call["arguments"]["query"],
                        "limit": func_call["arguments"].get("limit", 10)
                    }
                )
                if response.status_code == 200:
                    result = response.json()
                    if result.get("code", 0) == 0:
                        memories = result.get("memories", [])
                        return {
                            "name": "search_memories",
                            "result": "success",
                            "memories": memories,
                            "is_empty": len(memories) == 0
                        }
                    else:
                        return {
                            "name": "search_memories",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "search_memories",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "add_memory":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                # 确保参数类型正确
                args = func_call["arguments"]
                request_data = {
                    "user_id": 1,  # 使用固定的用户ID
                    "content": str(args["content"]),
                    "memory_type": str(args["memory_type"]),
                    "importance": float(args["importance"]),
                }
                # 如果有 metadata，转换为 JSON 字符串
                if "metadata" in args:
                    request_data["metadata"] = json.dumps(args["metadata"])
                
                print(f"发送添加记忆请求: {request_data}")  # 添加日志
                response = await client.post(
                    "/memory/add",
                    json=request_data
                )
                print(f"添加记忆响应: {response.status_code} - {response.text}")  # 添加日志
                if response.status_code == 200:
                    result = response.json()
                    # 兼容不同的响应格式
                    if result.get("code", 0) == 0 or result.get("msg") == "success":
                        return {
                            "name": "add_memory",
                            "result": "success",
                            "memory_id": result.get("memory_id")
                        }
                    else:
                        return {
                            "name": "add_memory",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "add_memory",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "get_memory":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/memory/get",
                    params={"memory_id": func_call["arguments"]["memory_id"]}
                )
                if response.status_code == 200:
                    result = response.json()
                    if result["code"] == 0:
                        return {
                            "name": "get_memory",
                            "result": "success",
                            "memory": result.get("memory")
                        }
                    else:
                        return {
                            "name": "get_memory",
                            "result": "error",
                            "error": result.get("msg")
                        }
                else:
                    return {
                        "name": "get_memory",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "delete_memory":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.delete(
                    f"/memory/{func_call['arguments']['memory_id']}",
                    json={"reason": func_call["arguments"]["reason"]}
                )
                if response.status_code == 200:
                    result = response.json()
                    if result["code"] == 0:
                        return {
                            "name": "delete_memory",
                            "result": "success",
                            "memory_id": func_call["arguments"]["memory_id"]
                        }
                    else:
                        return {
                            "name": "delete_memory",
                            "result": "error",
                            "error": result.get("msg")
                        }
                else:
                    return {
                        "name": "delete_memory",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        return {
            "name": func_call["name"],
            "result": "error",
            "error": f"未知的函数调用: {func_call['name']}"
        }
        
    except Exception as e:
        return {
            "name": func_call["name"],
            "result": "error",
            "error": str(e)
        }

async def process_stream_request(query: str, session_id: str = None):
    print(f"开始处理流式请求: query={query}, session_id={session_id}")
    
    # 如果没有提供session_id，创建一个新的
    if not session_id:
        print("创建新会话...")
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.post(
                "/session/create",
                json={
                    "user_id": 1,  # 固定用户ID
                }
            )
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="创建会话失败")
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
            session_id = str(result["session_id"])
            print(f"新会话创建成功: session_id={session_id}")

    # 检索相关文档
    print("开始检索相关文档...")
    try:
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL, timeout=30.0) as client:
            print(f"请求 API 网关: {API_GATEWAY_BASE_URL}/document/search")
            print(f"请求参数: user_id=1, query={query}, top_k=3")
            
            response = await client.get(
                "/document/search",
                params={
                    "user_id": 1,  # 固定用户ID
                    "query": query,
                    "top_k": 3
                }
            )
            
            print(f"API 网关响应状态码: {response.status_code}")
            print(f"API 网关响应内容: {response.text}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"搜索文档失败: HTTP {response.status_code}")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=f"搜索文档失败: {result.get('msg', '未知错误')}")
                
            documents = result.get("results", [])
            print(f"检索到 {len(documents)} 个相关文档")
    except httpx.RequestError as e:
        print(f"请求 API 网关失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"请求 API 网关失败: {str(e)}")
    except Exception as e:
        print(f"搜索文档时发生错误: {str(e)}")
        raise HTTPException(status_code=500, detail=f"搜索文档时发生错误: {str(e)}")

    # 构建上下文
    context = {
        "documents": []
    }
    for doc in documents:
        context["documents"].append({
            "title": doc['title'],
            "content": doc['content'],
            "score": doc.get('score', 0)
        })

    # 获取当前时间
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    # 构建用户提示词
    user_prompt = USER_PROMPT_TEMPLATE.format(
        user_id=1,
        session_id=session_id,
        current_time=current_time,
        memory=json.dumps(context, ensure_ascii=False),
        query=query
    )
    
    # 外层循环：处理整个问答过程
    messages = [
        {"role": "system", "content": SYSTEM_PROMPT},
        {"role": "user", "content": user_prompt}
    ]
    
    while True:
        # 调用 qwen 模型（非流式模式）
        response = openai_client.chat.completions.create(
            model="qwen-turbo",
            messages=messages,
            functions=FUNCTIONS,  # 添加函数定义
            stream=False  # 不使用流式模式
        )
        
        # 打印完整的AI响应
        print("\n=== AI 完整响应 ===")
        print(f"Content: {response.choices[0].message.content}")
        if response.choices[0].message.function_call:
            print(f"Function Call: {response.choices[0].message.function_call.name}")
            print(f"Arguments: {response.choices[0].message.function_call.arguments}")
        print("==================\n")
        
        # 获取完整响应
        full_response = response.choices[0].message.content or ""
        function_calls = []
        
        # 处理函数调用
        if response.choices[0].message.function_call:
            function_call = response.choices[0].message.function_call
            try:
                args = json.loads(function_call.arguments)
                function_calls.append({
                    "name": function_call.name,
                    "arguments": args
                })
            except json.JSONDecodeError as e:
                print(f"解析函数调用参数失败: {str(e)}")
                continue
        
        # 处理函数调用结果
        if function_calls:
            print(f"处理函数调用结果: {function_calls}")
            function_results = []
            
            # 处理每个函数调用
            for func_call in function_calls:
                result = await process_function_call(func_call, API_GATEWAY_BASE_URL)
                function_results.append(result)
            
            # 将函数调用结果添加到对话历史中
            if function_results:
                print(f"函数调用结果: {function_results}")
                # 添加AI的函数调用请求
                messages.append({
                    "role": "assistant",
                    "content": None,
                    "function_call": {
                        "name": function_call.name,
                        "arguments": function_call.arguments
                    }
                })
                # 添加函数调用结果
                messages.append({
                    "role": "function",
                    "name": function_call.name,
                    "content": json.dumps(function_results, ensure_ascii=False)
                })
                # 继续外层循环，重新调用模型
                continue
        
        # 如果没有函数调用，或者函数调用已经处理完成，跳出循环
        break
    
    # 保存聊天记录
    if session_id and full_response:
        try:
            async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                response = await client.post(
                    "/chat/record",
                    json={
                        "session_id": int(session_id),
                        "user_id": 1,
                        "message": query,
                        "response": full_response,
                        "context": json.dumps(context, ensure_ascii=False),
                        "function_calls": json.dumps(function_calls, ensure_ascii=False),
                        "metadata": json.dumps({
                            "model": "qwen-turbo",
                            "timestamp": current_time
                        }, ensure_ascii=False)
                    }
                )
                if response.status_code != 200:
                    print(f"保存聊天记录失败: HTTP {response.status_code}")
                else:
                    result = response.json()
                    if result.get("code", 0) != 0:
                        print(f"保存聊天记录失败: {result.get('msg', '未知错误')}")
                    else:
                        print(f"聊天记录保存成功: chat_id={result.get('chat_id')}")
        except Exception as e:
            print(f"保存聊天记录时发生错误: {str(e)}")
    
    # 流式返回最终回答
    async def generate_final_response():
        # 模拟流式输出（逐字或按块）
        chunk_size = 1  # 每次返回的字符数
        for i in range(0, len(full_response), chunk_size):
            chunk = full_response[i:i+chunk_size]
            yield f"data: {json.dumps({'content': chunk, 'session_id': session_id})}\n\n"
            await asyncio.sleep(0.01)  # 控制输出速度
        
        # 发送结束标记
        yield f"data: {json.dumps({'content': '', 'session_id': session_id, 'done': True})}\n\n"
    
    # 返回流式响应
    return StreamingResponse(
        generate_final_response(),
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "Connection": "keep-alive",
            "Transfer-Encoding": "chunked"
        }
    )

# 会话历史记录 API
@app.get("/api/chat/history")
async def get_chat_history():
    try:
        print("开始获取聊天历史...")
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            print(f"请求 API 网关: {API_GATEWAY_BASE_URL}/session/list")
            response = await client.get(
                "/session/list",
                params={
                    "user_id": 1,  # 这里需要根据实际用户ID修改
                    "page": 1,
                    "page_size": 100
                }
            )
            
            print(f"API 网关响应状态码: {response.status_code}")
            print(f"API 网关响应内容: {response.text}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"获取会话列表失败: HTTP {response.status_code}")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=f"获取会话列表失败: {result.get('msg', '未知错误')}")
                
            sessions = []
            for session in result.get("sessions", []):  # 修改为 sessions
                sessions.append({
                    "id": str(session["session_id"]),
                    "summary": session.get("summary", ""),
                    "updated_at": session.get("update_time", session.get("last_active_time", ""))
                })
            
            print(f"成功获取到 {len(sessions)} 个会话")
            return sessions
            
    except Exception as e:
        print(f"获取聊天历史失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取聊天历史失败: {str(e)}")

@app.get("/api/chat/session/{session_id}")
async def get_session(session_id: str):
    try:
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.get(
                f"/session/{session_id}",
                params={
                    "user_id": 1  # 这里需要根据实际用户ID修改
                }
            )
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取会话详情失败")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
                
            messages = []
            for record in result.get("session_info", {}).get("chat_records", []):
                messages.append({
                    "role": "user" if record["message_type"] == "text" else "bot",
                    "content": record["message"] if record["message_type"] == "text" else record["response"]
                })
            
            return {"messages": messages}
            
    except Exception as e:
        print(f"获取会话详情失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取会话详情失败: {str(e)}")

# 删除会话
@app.delete("/api/chat/session/{session_id}")
async def delete_session(session_id: str):
    try:
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.post(
                "/session/end",
                json={
                    "session_id": int(session_id)
                }
            )
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="删除会话失败")
                
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
            
            return {"message": "会话已删除"}
            
    except Exception as e:
        print(f"删除会话失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"删除会话失败: {str(e)}")

# 健康检查接口
@app.get("/health")
def health_check():
    print("health check")
    return {"status": "healthy"}

# 运行服务
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "app:app",
        host="0.0.0.0",
        port=8080,
        reload=False,  # 关闭热重载
        workers=1,     # 使用单进程
        limit_concurrency=50,  # 降低并发限制
        timeout_keep_alive=30,
    )
