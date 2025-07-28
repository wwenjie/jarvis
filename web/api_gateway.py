# api_gateway.py
import logging
import os
from fastapi import FastAPI, Request, HTTPException, UploadFile, File, Query
from fastapi.responses import StreamingResponse, RedirectResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
import httpx
from contextlib import asynccontextmanager

# 配置日志
def setup_logging():
    # 创建logs目录
    os.makedirs('/app/logs/api_gateway', exist_ok=True)
    
    # 配置日志格式
    log_format = '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    
    # 创建logger
    logger = logging.getLogger(__name__)
    logger.setLevel(logging.DEBUG)  # 设置logger级别为DEBUG，捕获所有级别
    
    # 清除已有的handlers（避免重复添加）
    logger.handlers.clear()
    
    # 创建不同级别的文件处理器
    debug_handler = logging.FileHandler('/app/logs/api_gateway/debug.log', encoding='utf-8')
    debug_handler.setLevel(logging.DEBUG)
    debug_handler.setFormatter(logging.Formatter(log_format))
    
    info_handler = logging.FileHandler('/app/logs/api_gateway/info.log', encoding='utf-8')
    info_handler.setLevel(logging.INFO)
    info_handler.setFormatter(logging.Formatter(log_format))
    
    warning_handler = logging.FileHandler('/app/logs/api_gateway/warning.log', encoding='utf-8')
    warning_handler.setLevel(logging.WARNING)
    warning_handler.setFormatter(logging.Formatter(log_format))
    
    error_handler = logging.FileHandler('/app/logs/api_gateway/error.log', encoding='utf-8')
    error_handler.setLevel(logging.ERROR)
    error_handler.setFormatter(logging.Formatter(log_format))
    
    # 控制台处理器
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    console_handler.setFormatter(logging.Formatter(log_format))
    
    # 添加所有处理器到logger
    logger.addHandler(debug_handler)
    logger.addHandler(info_handler)
    logger.addHandler(warning_handler)
    logger.addHandler(error_handler)
    logger.addHandler(console_handler)
    
    return logger

# 初始化日志
logger = setup_logging()

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

# 初始化函数
def init():    
    logger.info("API网关初始化")

@app.get("/")
async def redirect_to_document():
    return RedirectResponse(url="/static/documents.html")

# 文档管理 API
@app.post("/api/upload_doc")
async def upload_document(file: UploadFile = File(...)):
    try:
        # 读取文件内容
        content = await file.read()
        
        # 转发到 agent.py 服务
        async with httpx.AsyncClient(timeout=60.0) as client:
            files = {'file': (file.filename, content, file.content_type)}
            response = await client.post("http://10.1.12.17:8085/api/upload_doc", files=files)
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"Agent服务异常: {response.text}")
            
            return response.json()
    except Exception as e:
        logger.error(f"文档上传接口异常: {str(e)}")
        raise HTTPException(status_code=500, detail=f"文档上传接口异常: {str(e)}")

@app.get("/api/list_docs")
async def list_documents():
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get("http://10.1.12.17:8085/api/list_docs")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取文档列表失败")
                
            return response.json()
            
    except Exception as e:
        logger.error(f"获取文档列表失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取文档列表失败: {str(e)}")

@app.delete("/api/documents/{doc_id}")
async def delete_document(doc_id: str):
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.delete(f"http://10.1.12.17:8085/api/documents/{doc_id}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="删除文档失败")
                
            return response.json()
            
    except Exception as e:
        logger.error(f"删除文档失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"删除文档失败: {str(e)}")

@app.post("/api/stream")
async def stream_post(request: Request):
    try:
        # 解析请求体中的 JSON 数据
        req_data = await request.json()
        
        # 转发到 agent.py 服务
        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.post("http://10.1.12.17:8085/api/stream", json=req_data)
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"Agent服务异常: {response.text}")
            
            # 返回流式响应
            return StreamingResponse(
                response.aiter_bytes(),
                media_type="text/event-stream",
                headers={
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                    "Transfer-Encoding": "chunked"
                }
            )
    except Exception as e:
        error_msg = str(e)
        logger.error(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

@app.get("/api/stream")
async def stream_get(query: str = Query(None), session_id: str = Query(None), web_search: bool = Query(False)):
    try:
        if not query:
            raise HTTPException(status_code=400, detail="Missing query parameter")
        
        # 构建请求参数
        params = {"query": query}
        if session_id:
            params["session_id"] = session_id
        if web_search:
            params["web_search"] = web_search
        
        # 转发到 agent.py 服务
        async with httpx.AsyncClient(timeout=60.0) as client:
            response = await client.get("http://10.1.12.17:8085/api/stream", params=params)
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"Agent服务异常: {response.text}")
            
            # 返回流式响应
            return StreamingResponse(
                response.aiter_bytes(),
                media_type="text/event-stream",
                headers={
                    "Cache-Control": "no-cache",
                    "Connection": "keep-alive",
                    "Transfer-Encoding": "chunked"
                }
            )
    except Exception as e:
        error_msg = str(e)
        logger.error(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

# 会话历史记录 API
@app.get("/api/chat/history")
async def get_chat_history():
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get("http://10.1.12.17:8085/api/chat/history")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取聊天历史失败")
                
            return response.json()
    except Exception as e:
        logger.error(f"获取聊天历史失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取聊天历史失败: {str(e)}")

@app.get("/api/chat/session/{session_id}")
async def get_session(session_id: str):
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.get(f"http://10.1.12.17:8085/api/chat/session/{session_id}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取会话详情失败")
                
            return response.json()
    except Exception as e:
        logger.error(f"获取会话详情失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"获取会话详情失败: {str(e)}")

# 删除会话
@app.delete("/api/chat/session/{session_id}")
async def delete_session(session_id: str):
    try:
        async with httpx.AsyncClient(timeout=5.0) as client:
            response = await client.delete(f"http://10.1.12.17:8085/api/chat/session/{session_id}")
            
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="删除会话失败")
                
            return response.json()
            
    except Exception as e:
        logger.error(f"删除会话失败: {str(e)}")
        raise HTTPException(status_code=500, detail=f"删除会话失败: {str(e)}")

# 花朵识别API
@app.post("/api/flower_infer")
async def flower_infer(file: UploadFile = File(...)):
    try:
        # 读取图片内容
        image_bytes = await file.read()
        # 转发到 flower_infer.py 的 /infer 接口
        async with httpx.AsyncClient(timeout=10.0) as client:
            files = {'file': (file.filename, image_bytes, file.content_type)}
            response = await client.post("http://10.1.20.9:8082/infer", files=files)
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"花朵识别服务异常: {response.text}")
            return response.json()
    except Exception as e:
        logger.error(f"花朵识别接口异常: {str(e)}")
        raise HTTPException(status_code=500, detail=f"花朵识别接口异常: {str(e)}")

# OCR文字识别API
@app.post("/api/ocr")
async def ocr_recognize(file: UploadFile = File(...)):
    try:
        logger.info(f"开始OCR文字识别处理: {file.filename}")
        
        # 检查文件类型
        if not file.content_type or not file.content_type.startswith('image/'):
            raise HTTPException(status_code=400, detail="只支持图片文件")
        
        # 读取图片内容
        image_bytes = await file.read()
        logger.info(f"图片读取完成，大小: {len(image_bytes)} bytes")
        
        # 转发到 agent.py 服务
        async with httpx.AsyncClient(timeout=30.0) as client:
            files = {'file': (file.filename, image_bytes, file.content_type)}
            response = await client.post("http://10.1.12.17:8085/api/ocr", files=files)
            
            if response.status_code != 200:
                logger.error(f"Agent服务响应异常: {response.status_code} - {response.text}")
                raise HTTPException(status_code=500, detail=f"Agent服务异常: {response.text}")
            
            result = response.json()
            logger.info(f"OCR识别完成，识别到 {result.get('total_texts', 0)} 个文本区域")
            
            return result
            
    except HTTPException:
        # 重新抛出HTTP异常
        raise
    except Exception as e:
        logger.error(f"OCR文字识别接口异常: {str(e)}")
        raise HTTPException(status_code=500, detail=f"OCR文字识别接口异常: {str(e)}")

# 健康检查接口
@app.get("/health")
def health_check():
    logger.info("health check")
    return {"status": "healthy"}

# 运行服务
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "api_gateway:app",
        host="0.0.0.0",
        port=8080,
        reload=False,  # 关闭热重载
        workers=1,     # 使用单进程
        limit_concurrency=50,  # 降低并发限制
        timeout_keep_alive=30,
    )
