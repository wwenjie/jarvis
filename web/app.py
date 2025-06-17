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

from fastmcp import Client as mcp_client

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
                    "description": "记忆内容，如果涉及日期，请使用实际日期，不要用相对日期"
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
        },
        "returns": {
            "type": "object",
            "properties": {
                "memory_id": {
                    "type": "integer",
                    "description": "新创建的记忆ID"
                }
            }
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
        },
        "returns": {
            "type": "object",
            "properties": {
                "memory": {
                    "type": "object",
                    "description": "记忆信息，包含memory_id、content、memory_type等字段"
                }
            }
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
        },
        "returns": {
            "type": "object",
            "properties": {
                "memories": {
                    "type": "array",
                    "description": "记忆列表，每条记忆包含memory_id、content、memory_type等字段",
                    "items": {
                        "type": "object",
                        "properties": {
                            "memory_id": {
                                "type": "integer",
                                "description": "记忆ID，用于后续删除或更新操作"
                            },
                            "content": {
                                "type": "string",
                                "description": "记忆内容"
                            },
                            "memory_type": {
                                "type": "string",
                                "description": "记忆类型"
                            },
                            "importance": {
                                "type": "number",
                                "description": "重要性"
                            },
                            "create_time": {
                                "type": "string",
                                "description": "创建时间"
                            },
                            "expire_time": {
                                "type": "string",
                                "description": "过期时间"
                            }
                        }
                    }
                }
            }
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
        },
        "returns": {
            "type": "object",
            "properties": {
                "memory_id": {
                    "type": "integer",
                    "description": "被删除的记忆ID"
                }
            }
        }
    },
    {
        "name": "search_document",
        "description": "搜索知识库中的文档",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {
                    "type": "string",
                    "description": "搜索查询"
                },
                "top_k": {
                    "type": "integer",
                    "description": "返回结果数量"
                }
            },
            "required": ["query"]
        },
        "returns": {
            "type": "object",
            "properties": {
                "documents": {
                    "type": "array",
                    "description": "文档列表，每个文档包含title、content等字段",
                    "items": {
                        "type": "object",
                        "properties": {
                            "title": {
                                "type": "string",
                                "description": "文档标题"
                            },
                            "content": {
                                "type": "string",
                                "description": "文档中相关的内容"
                            },
                            "score": {
                                "type": "number",
                                "description": "相关度分数"
                            }
                        }
                    }
                }
            }
        }
    },
    {
        "name": "get_weather",
        "description": "获取指定位置的实时天气信息",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "位置名称，如：北京、上海、广州等"
                }
            },
            "required": ["location"]
        },
        "returns": {
            "type": "object",
            "properties": {
                "weather": {
                    "type": "object",
                    "description": "天气信息，包含温度、天气状况等",
                    "properties": {
                        "temperature": {
                            "type": "number",
                            "description": "温度（摄氏度）"
                        },
                        "condition": {
                            "type": "string",
                            "description": "天气状况"
                        },
                        "humidity": {
                            "type": "number",
                            "description": "湿度（百分比）"
                        }
                    }
                }
            }
        }
    },
    {
        "name": "get_hourly_weather",
        "description": "获取指定位置的24小时天气预报",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "位置名称，如：北京、上海、广州等"
                }
            },
            "required": ["location"]
        },
        "returns": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "位置名称"
                },
                "hourly": {
                    "type": "array",
                    "description": "24小时天气预报",
                    "items": {
                        "type": "object",
                        "properties": {
                            "time": {
                                "type": "string",
                                "description": "时间"
                            },
                            "weather": {
                                "type": "string",
                                "description": "天气状况"
                            },
                            "temperature": {
                                "type": "number",
                                "description": "温度（摄氏度）"
                            },
                            "humidity": {
                                "type": "number",
                                "description": "湿度（百分比）"
                            },
                            "wind_speed": {
                                "type": "number",
                                "description": "风速（米/秒）"
                            },
                            "wind_dir": {
                                "type": "string",
                                "description": "风向"
                            }
                        }
                    }
                }
            }
        }
    },
    {
        "name": "get_daily_weather",
        "description": "获取指定位置的15天天气预报",
        "parameters": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "位置名称，如：北京、上海、广州等"
                }
            },
            "required": ["location"]
        },
        "returns": {
            "type": "object",
            "properties": {
                "location": {
                    "type": "string",
                    "description": "位置名称"
                },
                "daily": {
                    "type": "array",
                    "description": "15天天气预报",
                    "items": {
                        "type": "object",
                        "properties": {
                            "date": {
                                "type": "string",
                                "description": "日期"
                            },
                            "text_day": {
                                "type": "string",
                                "description": "白天天气"
                            },
                            "text_night": {
                                "type": "string",
                                "description": "夜间天气"
                            },
                            "high_temp": {
                                "type": "number",
                                "description": "最高温度（摄氏度）"
                            },
                            "low_temp": {
                                "type": "number",
                                "description": "最低温度（摄氏度）"
                            },
                            "rainfall": {
                                "type": "number",
                                "description": "降雨量（毫米）"
                            },
                            "precip": {
                                "type": "number",
                                "description": "降水概率（百分比）"
                            },
                            "wind_dir": {
                                "type": "string",
                                "description": "风向"
                            },
                            "wind_speed": {
                                "type": "number",
                                "description": "风速（米/秒）"
                            },
                            "wind_scale": {
                                "type": "string",
                                "description": "风力等级"
                            },
                            "humidity": {
                                "type": "number",
                                "description": "湿度（百分比）"
                            }
                        }
                    }
                }
            }
        }
    },
    {
        "name": "web_search",
        "description": "使用博查API进行联网网页搜索，返回摘要和结果",
        "parameters": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "需要搜索的关键词"},
                "count": {"type": "integer", "description": "返回结果数量，最大50，默认10"},
                "freshness": {"type": "string", "description": "时间范围，可选值：noLimit, oneDay, oneWeek, oneMonth, oneYear"}
            },
            "required": ["query"]
        },
        "returns": {
            "type": "object",
            "properties": {
                "results": {
                    "type": "array",
                    "description": "搜索结果列表",
                    "items": {
                        "type": "object",
                        "properties": {
                            "title": {"type": "string", "description": "网页标题"},
                            "url": {"type": "string", "description": "网页链接"},
                            "snippet": {"type": "string", "description": "简短摘要"},
                            "siteName": {"type": "string", "description": "网站名"},
                            "datePublished": {"type": "string", "description": "发布时间"}
                        }
                    }
                }
            }
        }
    }
]

# 系统提示词
SYSTEM_PROMPT = """你是一个带长远记忆的 AI 助手，名字叫 Jarvis。

你的主要特点：
1. 长期记忆能力
   - 能记住用户的重要信息
   - 能记住重要的对话历史
   - 能记住用户的任务和提醒，并及时清理过期的提醒

2. 个性化交互
   - 根据用户习惯调整回复风格
   - 记住用户的常用表达方式
   - 保持对话的连贯性

3. 智能学习
   - 从对话中学习用户习惯
   - 记住有效的解决方案

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

BOCHA_API_KEY = os.getenv("BOCHA_API_KEY")

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
                "/document/delete",
                params={
                    "doc_id": doc_id,
                    "user_id": 1
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
        web_search = req_data.get("web_search")
        return await process_stream_request(query, session_id, web_search)
    except Exception as e:
        error_msg = str(e)
        print(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

@app.get("/api/stream")
async def stream_get(query: str = Query(None), session_id: str = Query(None), web_search: bool = Query(False)):
    try:
        if not query:
            raise HTTPException(status_code=400, detail="Missing query parameter")
        return await process_stream_request(query, session_id, web_search)
    except Exception as e:
        error_msg = str(e)
        print(f"聊天接口错误: {error_msg}")
        raise HTTPException(status_code=500, detail=error_msg)

# async def process_web_search_by_mcp(arguments: dict):
#     """
#     通过 mcp_client 访问博查 MCP Server 实现联网搜索
#     """
#     try:
#         # 1. 初始化 mcp_client（建议全局只初始化一次，这里仅为演示）
#         # endpoint、api_key 建议放到配置文件或环境变量
#         endpoint = os.getenv("BOCHA_MCP_ENDPOINT", "https://api.bochaai.com/v1/")
#         api_key = os.getenv("BOCHA_API_KEY")
#         client = mcp_client(endpoint=endpoint, api_key=api_key)

#         tools = client.list_tools()
#         print("可用工具：", tools)

#         # 2. 构造请求参数
#         payload = {
#             "query": arguments["query"],
#             "summary": True,
#             "count": arguments.get("count", 10),
#             "freshness": arguments.get("freshness", "noLimit")
#         }

#         # 3. 调用 mcp_client 的方法（假设为 call，实际方法名请查阅 fastmcp 文档）
#         # tool_name 可能是 "web_search" 或 "bocha_web_search"，具体看博查 MCP Server 的 tool 列表
#         tool_name = "web_search"
#         # 有的 mcp_client 是同步的，如果是同步的用 await asyncio.to_thread(...)
#         result = await client.call(tool_name, **payload)

#         # 4. 解析结果，适配 function call 返回结构
#         results = []
#         for item in result.get("results", []):
#             results.append({
#                 "title": item.get("title", ""),
#                 "url": item.get("url", ""),
#                 "snippet": item.get("snippet", ""),
#                 "siteName": item.get("siteName", ""),
#                 "datePublished": item.get("datePublished", "")
#             })
#         return {
#             "name": "web_search",
#             "result": "success",
#             "results": results
#         }
#     except Exception as e:
#         return {
#             "name": "web_search",
#             "result": "error",
#             "error": f"通过MCP访问博查时出错: {str(e)}"
#         }

async def process_web_search(arguments: dict):
    print(f"开始处理网页搜索: {arguments}")
    try:
        import requests

        headers = {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {BOCHA_API_KEY}'
        }

        payload = {
            "query": arguments["query"],
            "summary": True,
            "count": arguments.get("count", 10),
            "freshness": arguments.get("freshness", "noLimit")
        }

        response = requests.post("https://api.bochaai.com/v1/web-search", headers=headers, json=payload)

        if response.status_code != 200:
            return {
                "name": "web_search",
                "result": "error",
                "error": f"搜索失败，状态码: {response.status_code}, 详情: {response.text}"
            }

        json_data = response.json()
        results = []
        for item in json_data.get("data", {}).get("webPages", {}).get("value", []):
            results.append({
                "title": item.get("name", ""),
                "url": item.get("url", ""),
                "snippet": item.get("snippet", ""),
                "siteName": item.get("siteName", ""),
                "datePublished": item.get("dateLastCrawled", "")
            })
        return {
            "name": "web_search",
            "result": "success",
            "results": results
        }
    except Exception as e:
        return {
            "name": "web_search",
            "result": "error",
            "error": f"执行网络搜索时出错: {str(e)}"
        }

async def process_function_call(func_call: dict, api_gateway_base_url: str) -> dict:
    """处理单个函数调用
    
    Args:
        func_call: 函数调用信息，包含name和arguments
        api_gateway_base_url: API网关基础URL
        
    Returns:
        dict: 函数调用结果
    """
    try:
        if func_call["name"] == "web_search":
            # return await process_web_search_by_mcp(func_call["arguments"])
            return await process_web_search(func_call["arguments"])
        elif func_call["name"] == "get_weather":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/weather/get",
                    params={
                        "location": func_call["arguments"]["location"]
                    }
                )
                if response.status_code == 200:
                    result = response.json()
                    if result.get("code", 0) == 0:
                        weather = result.get("weather", {})
                        return {
                            "name": "get_weather",
                            "result": "success",
                            "weather": weather
                        }
                    else:
                        return {
                            "name": "get_weather",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "get_weather",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "search_document":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/document/search",
                    params={
                        "user_id": 1,  # 固定用户ID
                        "query": func_call["arguments"]["query"],
                        "top_k": func_call["arguments"].get("top_k", 5)
                    }
                )
                if response.status_code == 200:
                    result = response.json()
                    if result.get("code", 0) == 0:
                        documents = result.get("results", [])
                        return {
                            "name": "search_document",
                            "result": "success",
                            "documents": documents,
                            "is_empty": len(documents) == 0
                        }
                    else:
                        return {
                            "name": "search_document",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "search_document",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "search_memories":
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
                    "/memory/delete",
                    params={
                        "memory_id": func_call["arguments"]["memory_id"],
                        "user_id": 1,
                        "reason": func_call["arguments"]["reason"]
                    }
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
        
        elif func_call["name"] == "get_hourly_weather":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/weather/hourly",
                    params={
                        "location": func_call["arguments"]["location"]
                    }
                )
                if response.status_code == 200:
                    result = response.json()
                    if result.get("code", 0) == 0:
                        return {
                            "name": "get_hourly_weather",
                            "result": "success",
                            "location": result.get("location"),
                            "hourly": result.get("hourly", [])
                        }
                    else:
                        return {
                            "name": "get_hourly_weather",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "get_hourly_weather",
                        "result": "error",
                        "error": f"请求失败: {response.status_code}"
                    }
        
        elif func_call["name"] == "get_daily_weather":
            async with httpx.AsyncClient(base_url=api_gateway_base_url) as client:
                response = await client.get(
                    "/weather/daily",
                    params={
                        "location": func_call["arguments"]["location"]
                    }
                )
                if response.status_code == 200:
                    result = response.json()
                    if result.get("code", 0) == 0:
                        return {
                            "name": "get_daily_weather",
                            "result": "success",
                            "location": result.get("location"),
                            "daily": result.get("daily", [])
                        }
                    else:
                        return {
                            "name": "get_daily_weather",
                            "result": "error",
                            "error": result.get("msg", "未知错误")
                        }
                else:
                    return {
                        "name": "get_daily_weather",
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

async def process_stream_request(query: str, session_id: str = None, web_search: bool = False):
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

    # # 检索相关文档
    # print("开始检索相关文档...")
    # try:
    #     async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL, timeout=30.0) as client:
    #         print(f"请求 API 网关: {API_GATEWAY_BASE_URL}/document/search")
    #         print(f"请求参数: user_id=1, query={query}, top_k=3")
            
    #         response = await client.get(
    #             "/document/search",
    #             params={
    #                 "user_id": 1,  # 固定用户ID
    #                 "query": query,
    #                 "top_k": 3
    #             }
    #         )
            
    #         print(f"API 网关响应状态码: {response.status_code}")
    #         print(f"API 网关响应内容: {response.text}")
            
    #         if response.status_code != 200:
    #             raise HTTPException(status_code=500, detail=f"搜索文档失败: HTTP {response.status_code}")
                
    #         result = response.json()
    #         if result.get("code", 0) != 0:
    #             raise HTTPException(status_code=500, detail=f"搜索文档失败: {result.get('msg', '未知错误')}")
                
    #         documents = result.get("results", [])
    #         print(f"检索到 {len(documents)} 个相关文档")
    # except httpx.RequestError as e:
    #     print(f"请求 API 网关失败: {str(e)}")
    #     raise HTTPException(status_code=500, detail=f"请求 API 网关失败: {str(e)}")
    # except Exception as e:
    #     print(f"搜索文档时发生错误: {str(e)}")
    #     raise HTTPException(status_code=500, detail=f"搜索文档时发生错误: {str(e)}")

    # 构建上下文
    context = {
        "documents": []
    }
    # for doc in documents:
    #     context["documents"].append({
    #         "title": doc['title'],
    #         "content": doc['content'],
    #         "score": doc.get('score', 0)
    #     })

    # 获取当前时间
    current_time = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    
    # 获取历史消息
    history_messages = []
    if session_id:
        try:
            async with httpx.AsyncClient(base_url=API_GATEWAY_BASE_URL) as client:
                resp = await client.get(
                    "/chat/records/get",
                    params={
                        "session_id": session_id,
                        "page": 1,
                        "page_size": 20
                    }
                )
                if resp.status_code == 200:
                    result = resp.json()
                    for record in result.get("chat_records", []):
                        if record["message_type"] == "text":
                            history_messages.append({"role": "user", "content": record["message"]})
                        else:
                            history_messages.append({"role": "assistant", "content": record["response"]})
                else:
                    print(f"获取历史消息失败: {resp.status_code}")
        except Exception as e:
            print(f"获取历史消息失败: {str(e)}")
    
    # user_prompt 只用本轮 query
    user_prompt = query
    
    # 拼接 messages：system + 历史消息 + 当前 user
    system_prompt = f"""{SYSTEM_PROMPT}
当前时间：{current_time}
"""
    messages = [
        {"role": "system", "content": system_prompt}
    ]
    messages.extend(history_messages)
    messages.append({"role": "user", "content": user_prompt})
    
    # 外层循环：处理整个问答过程
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
                        "context": json.dumps(history_messages, ensure_ascii=False),
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

def format_timestamp(ts):
    if not ts:
        return ""
    try:
        return datetime.fromtimestamp(int(ts)).strftime("%Y/%m/%d %H:%M:%S")
    except Exception:
        return str(ts)

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
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail=f"获取会话列表失败: HTTP {response.status_code}")
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=f"获取会话列表失败: {result.get('msg', '未知错误')}")
            sessions = []
            for session in result.get("sessions", []):
                sessions.append({
                    "id": str(session["session_id"]),
                    "summary": session.get("summary", ""),
                    "updated_at": format_timestamp(session.get("update_time"))
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
                "/session/get",
                params={
                    "user_id": 1,
                    "session_id": session_id
                }
            )
            if response.status_code != 200:
                raise HTTPException(status_code=500, detail="获取会话详情失败")
            result = response.json()
            if result.get("code", 0) != 0:
                raise HTTPException(status_code=500, detail=result.get("msg", "未知错误"))
            messages = []
            for record in result.get("chat_records", []):
                messages.append({
                    "role": "user" if record["message_type"] == "text" else "bot",
                    "content": record["message"] if record["message_type"] == "text" else record["response"],
                    "created_at": format_timestamp(record.get("create_time"))
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
