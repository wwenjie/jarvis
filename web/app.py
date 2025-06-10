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

# API 网关配置
API_GATEWAY_BASE_URL = "http://localhost:8888"  # 根据实际部署情况修改

# 创建应用启动上下文管理器
@asynccontextmanager
async def lifespan(app: FastAPI):
    # 启动前执行
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
uploaded_documents: Dict[str, Dict] = {}  # {id: {name, content, path}}
chat_sessions: Dict[str, Dict] = {}  # {id: {summary, updated_at, messages}}

# 创建 HTTP 客户端
async def get_http_client():
    async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
        yield client

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
            "/knowledge/add",
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
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.get(
                "/knowledge/list",
                params={
                    "user_id": 1  # 这里需要根据实际用户ID修改
                }
            )
            
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
                f"/knowledge/{doc_id}",
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
    # 如果没有提供session_id，创建一个新的
    if not session_id:
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

    # 检索相关文档
    async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
        response = await client.get(
            "/knowledge/search",
            params={
                "user_id": 1,  # 固定用户ID
                "query": query,
                "top_k": 3
            }
        )
        
        if response.status_code != 200:
            raise HTTPException(status_code=500, detail="搜索文档失败")
            
        result = response.json()
        if result["code"] != 0:
            raise HTTPException(status_code=500, detail=result["msg"])
            
        documents = result["documents"]
        scores = result["scores"]
    
    # 构建上下文
    context = "相关文档:\n"
    for doc, score in zip(documents, scores):
        context += f"文档: {doc['title']} (相关度: {score:.2f})\n"
        context += f"内容: {doc['content']}\n\n"
    
    # 创建stream响应    
    async def generate():
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            # 使用流式接口
            async with client.stream(
                "POST",
                "/chat/stream",
                json={
                    "session_id": int(session_id),
                    "user_id": 1,  # 固定用户ID
                    "message": query,
                    "message_type": "text",
                    "context": context  # 添加检索到的文档上下文
                }
            ) as response:
                if response.status_code != 200:
                    yield f"data: {json.dumps({'content': '服务器错误', 'session_id': session_id, 'done': True})}\n\n"
                    return
                
                async for line in response.aiter_lines():
                    if line.startswith("data: "):
                        try:
                            data = json.loads(line[6:])
                            yield f"data: {json.dumps(data)}\n\n"
                            if data.get("done"):
                                break
                        except json.JSONDecodeError:
                            continue
                
    return StreamingResponse(
        generate(),
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
        async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
            response = await client.get(
                "/session/list",
                params={
                    "user_id": 1,  # 这里需要根据实际用户ID修改
                    "page": 1,
                    "page_size": 100
                }
            )
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取会话列表失败")
                
            result = response.json()
            if result["code"] != 0:
                raise HTTPException(status_code=500, detail=result["msg"])
                
            sessions = []
            for session in result["session_list"]:
                sessions.append({
                    "id": str(session["session_id"]),
                    "summary": session["summary"],
                    "updated_at": session["update_time"]
                })
            
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
