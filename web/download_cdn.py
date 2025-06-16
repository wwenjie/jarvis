import os
import requests
from urllib.parse import urlparse
import shutil

# CDN 资源列表
CDN_RESOURCES = [
    # chat.html用
    # Bootstrap
    "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css",
    "https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js",
    
    # Vue.js
    "https://cdn.jsdelivr.net/npm/vue@3.2.47/dist/vue.global.prod.js",
    
    # Axios
    "https://cdn.jsdelivr.net/npm/axios@1.4.0/dist/axios.min.js",
    
    # Marked
    "https://cdn.jsdelivr.net/npm/marked@5.0.2/marked.min.js",
    
    # Highlight.js
    "https://cdn.jsdelivr.net/npm/highlightjs@9.16.2/highlight.pack.min.js",
    "https://cdn.jsdelivr.net/npm/highlightjs@9.16.2/styles/monokai-sublime.min.css",
    
    # Bootstrap Icons
    "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.5/font/bootstrap-icons.css",
    "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.5/font/fonts/bootstrap-icons.woff",
    "https://cdn.jsdelivr.net/npm/bootstrap-icons@1.10.5/font/fonts/bootstrap-icons.woff2",
    
    
    # documents.html用
    # 新增：Vue 开发版本（用于调试）
    "https://cdn.jsdelivr.net/npm/vue@3.2.47/dist/vue.global.js",
    
    # 新增：Vue 生产版本（已存在，不需要重复添加）
    # "https://cdn.jsdelivr.net/npm/vue@3.2.47/dist/vue.global.prod.js",
    
    # 新增：Axios 最新版本
    "https://cdn.jsdelivr.net/npm/axios@1.6.2/dist/axios.min.js",
    
    # 新增：Bootstrap 最新版本
    "https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css",
    "https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"
]

def download_file(url, local_dir):
    """下载文件到指定目录"""
    try:
        # 解析 URL 获取文件名
        parsed_url = urlparse(url)
        filename = os.path.basename(parsed_url.path)
        
        # 如果是字体文件，需要创建 fonts 目录
        if 'fonts' in url:
            local_dir = os.path.join(local_dir, 'fonts')
            os.makedirs(local_dir, exist_ok=True)
        
        # 构建本地文件路径
        local_path = os.path.join(local_dir, filename)
        
        # 检查文件是否已存在
        if os.path.exists(local_path):
            print(f"文件已存在，跳过下载: {local_path}")
            return filename
        
        # 下载文件
        print(f"正在下载: {url}")
        response = requests.get(url, stream=True)
        response.raise_for_status()
        
        # 保存文件
        with open(local_path, 'wb') as f:
            response.raw.decode_content = True
            shutil.copyfileobj(response.raw, f)
        
        print(f"下载完成: {local_path}")
        return filename
        
    except Exception as e:
        print(f"下载失败 {url}: {str(e)}")
        return None

def main():
    # 创建 static 目录
    static_dir = os.path.join(os.path.dirname(__file__), 'static')
    os.makedirs(static_dir, exist_ok=True)
    
    # 下载所有资源
    downloaded_files = {}
    for url in CDN_RESOURCES:
        filename = download_file(url, static_dir)
        if filename:
            downloaded_files[url] = filename
    
    # 生成 HTML 引用代码
    print("\n本地资源引用代码:")
    print("<!-- Bootstrap CSS -->")
    print(f'<link href="/static/{downloaded_files[CDN_RESOURCES[0]]}" rel="stylesheet">')
    print("\n<!-- Vue.js -->")
    print(f'<script src="/static/{downloaded_files[CDN_RESOURCES[2]]}"></script>')
    print("\n<!-- Axios -->")
    print(f'<script src="/static/{downloaded_files[CDN_RESOURCES[3]]}"></script>')
    print("\n<!-- Marked -->")
    print(f'<script src="/static/{downloaded_files[CDN_RESOURCES[4]]}"></script>')
    print("\n<!-- Highlight.js -->")
    print(f'<script src="/static/{downloaded_files[CDN_RESOURCES[5]]}"></script>')
    print(f'<link href="/static/{downloaded_files[CDN_RESOURCES[6]]}" rel="stylesheet">')
    print("\n<!-- Bootstrap Icons -->")
    print(f'<link href="/static/{downloaded_files[CDN_RESOURCES[7]]}" rel="stylesheet">')
    print("\n<!-- Bootstrap JS -->")
    print(f'<script src="/static/{downloaded_files[CDN_RESOURCES[1]]}"></script>')

if __name__ == "__main__":
    main() 