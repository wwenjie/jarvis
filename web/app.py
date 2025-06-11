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
1. 始终以用户为中心，提供个性化服务
2. 保持对话的连贯性和上下文理解
3. 主动学习和适应用户的习惯
4. 保护用户隐私和数据安全
5. 提供准确、及时、有用的信息
6. 在合适的时候主动提供建议
7. 保持友好、专业、耐心的态度

你的记忆类型：
1. 事实性记忆：用户的基本信息、重要事实
2. 提醒类记忆：任务、日程、提醒事项
3. 用户偏好：使用习惯、交互偏好
4. 上下文记忆：当前对话的上下文信息

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
        if result["code"] != 0:
            raise HTTPException(status_code=500, detail=result["msg"])
            
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
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=result["msg"])
                
            return [{"id": str(doc["doc_id"]), "name": doc["title"]} for doc in result["documents"]]
            
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
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=result["msg"])
            
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

async def process_stream_request(query: str, session_id: str = None):
    try:
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
                if result["code"] != 0:
                    raise HTTPException(status_code=500, detail=result["msg"])
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
                if result["code"] != 0:
                    raise HTTPException(status_code=500, detail=f"搜索文档失败: {result['msg']}")
                    
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
    
        # 创建stream响应    
        async def generate():
            print("开始生成流式响应...")
            
            try:
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
                
                # 直接调用 qwen 模型
                stream = openai_client.chat.completions.create(
                    model="qwen-turbo",
                    messages=[
                        {"role": "system", "content": SYSTEM_PROMPT},
                        {"role": "user", "content": user_prompt}
                    ],
                    functions=FUNCTIONS,  # 添加函数定义
                    stream=True
                )
                
                full_response = ""
                function_calls = []
                for chunk in stream:
                    if chunk.choices[0].delta.content:
                        content = chunk.choices[0].delta.content
                        full_response += content
                        yield f"data: {json.dumps({'content': content, 'session_id': session_id})}\n\n"
                        await asyncio.sleep(0.01)  # 添加小延迟确保流式输出
                    
                    # 处理函数调用
                    if chunk.choices[0].delta.function_call:
                        function_call = chunk.choices[0].delta.function_call
                        if function_call.name:
                            print(f"函数调用: {function_call.name}")
                            function_calls.append({
                                "name": function_call.name,
                                "arguments": function_call.arguments
                            })
                            # 处理函数调用
                            if function_call.name == "add_memory":
                                args = json.loads(function_call.arguments)
                                async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                                    response = await client.post(
                                        "/memory/add",
                                        json=args
                                    )
                                    if response.status_code == 200:
                                        result = response.json()
                                        if result["code"] == 0:
                                            print(f"添加记忆成功: {result['data']['memory_id']}")
                                        else:
                                            print(f"添加记忆失败: {result['msg']}")
                                    else:
                                        print(f"添加记忆请求失败: {response.status_code}")
                            
                            elif function_call.name == "get_memory":
                                args = json.loads(function_call.arguments)
                                async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                                    response = await client.get(
                                        "/memory/get",
                                        params={"memory_id": args["memory_id"]}
                                    )
                                    if response.status_code == 200:
                                        result = response.json()
                                        if result["code"] == 0:
                                            print(f"获取记忆成功: {result['data']['memory']}")
                                        else:
                                            print(f"获取记忆失败: {result['msg']}")
                                    else:
                                        print(f"获取记忆请求失败: {response.status_code}")
                            
                            elif function_call.name == "search_memories":
                                args = json.loads(function_call.arguments)
                                async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                                    response = await client.get(
                                        "/memory/search",
                                        params={
                                            "query": args["query"],
                                            "limit": args.get("limit", 10)
                                        }
                                    )
                                    if response.status_code == 200:
                                        result = response.json()
                                        if result["code"] == 0:
                                            print(f"搜索记忆成功: {result['data']['memories']}")
                                        else:
                                            print(f"搜索记忆失败: {result['msg']}")
                                    else:
                                        print(f"搜索记忆请求失败: {response.status_code}")
                            
                            elif function_call.name == "delete_memory":
                                args = json.loads(function_call.arguments)
                                async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                                    response = await client.delete(
                                        f"/memory/{args['memory_id']}",
                                        json={"reason": args["reason"]}
                                    )
                                    if response.status_code == 200:
                                        result = response.json()
                                        if result["code"] == 0:
                                            print(f"删除记忆成功: {args['memory_id']}")
                                        else:
                                            print(f"删除记忆失败: {result['msg']}")
                                    else:
                                        print(f"删除记忆请求失败: {response.status_code}")
                        
                    if chunk.choices[0].finish_reason is not None:
                        yield f"data: {json.dumps({'content': '', 'session_id': session_id, 'done': True})}\n\n"
                        break
                        
                # 响应完成后，将完整会话保存到数据库
                if session_id:
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
                                if result["code"] != 0:
                                    print(f"保存聊天记录失败: {result['msg']}")
                                else:
                                    print(f"聊天记录保存成功: chat_id={result['chat_id']}")
                    except Exception as e:
                        print(f"保存聊天记录时发生错误: {str(e)}")
            except Exception as e:
                print(f"生成响应时发生错误: {str(e)}")
                yield f"data: {json.dumps({'content': f'生成响应时发生错误: {str(e)}', 'session_id': session_id, 'error': True})}\n\n"
                
        return StreamingResponse(
            generate(),
            media_type="text/event-stream",
            headers={
                "Cache-Control": "no-cache",
                "Connection": "keep-alive",
                "Transfer-Encoding": "chunked"
            }
        )
    except Exception as e:
        error_msg = f"处理流式请求失败: {str(e)}"
        print(error_msg)
        raise HTTPException(status_code=500, detail=error_msg)

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
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=f"获取会话列表失败: {result['msg']}")
                
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
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=result["msg"])
                
            messages = []
            for record in result["session_info"]["chat_records"]:
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
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=result["msg"])
            
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
