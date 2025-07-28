# 多阶段构建 Dockerfile for EasyOCR 非量化模型

# 第一阶段：构建环境
FROM python:3.10-slim AS builder

# 安装系统依赖（使用腾讯云镜像源）
RUN rm -f /etc/apt/sources.list.d/* && \
    echo "deb http://mirrors.cloud.tencent.com/debian/ bookworm main" > /etc/apt/sources.list && \
    echo "deb http://mirrors.cloud.tencent.com/debian-security/ bookworm-security main" >> /etc/apt/sources.list && \
    echo "deb http://mirrors.cloud.tencent.com/debian/ bookworm-updates main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y \
    gcc \
    g++ \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 复制依赖文件
COPY py_server/ocr_svr/requirements.txt .

# 先安装setuptools和wheel
RUN pip install -i https://mirrors.cloud.tencent.com/pypi/simple --no-cache-dir setuptools wheel

# 先安装numpy 1.24.3版本（与Python 3.10兼容）
RUN pip install -i https://mirrors.cloud.tencent.com/pypi/simple --no-cache-dir "numpy==1.24.3"

# 先安装PyTorch CPU版本（使用稳定的版本）
RUN pip install torch==2.1.0+cpu torchvision==0.16.0+cpu --index-url https://download.pytorch.org/whl/cpu

# 使用腾讯云pip镜像源安装其他 Python 依赖
RUN pip install -i https://mirrors.cloud.tencent.com/pypi/simple --no-cache-dir fastapi==0.100.0 uvicorn==0.23.2 easyocr==1.7.0 pillow==9.5.0 python-multipart==0.0.6 opencv-python==4.8.0.76

# 强制重新安装numpy 1.24.3版本以确保兼容性
RUN pip install -i https://mirrors.cloud.tencent.com/pypi/simple --no-cache-dir --force-reinstall "numpy==1.24.3"

# 第二阶段：运行环境
FROM python:3.10-slim

# 安装运行时依赖（OpenCV和PyTorch需要的图形库）
RUN rm -f /etc/apt/sources.list.d/* && \
    echo "deb http://mirrors.cloud.tencent.com/debian/ bookworm main" > /etc/apt/sources.list && \
    echo "deb http://mirrors.cloud.tencent.com/debian-security/ bookworm-security main" >> /etc/apt/sources.list && \
    echo "deb http://mirrors.cloud.tencent.com/debian/ bookworm-updates main" >> /etc/apt/sources.list && \
    apt-get update && \
    apt-get install -y \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 设置工作目录
WORKDIR /app

# 从构建阶段复制 Python 包
COPY --from=builder /usr/local/lib/python3.10/site-packages /usr/local/lib/python3.10/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# 复制模型文件
COPY models/.EasyOCR /app/.EasyOCR

# 复制应用代码
COPY py_server/ocr_svr/ocr_svr.py .

# 创建日志目录
RUN mkdir -p /app/logs

# 设置环境变量
ENV PYTHONPATH=/app
ENV EASYOCR_MODULE_PATH=/app/.EasyOCR
ENV OMP_NUM_THREADS=1
ENV MKL_NUM_THREADS=1

# 暴露端口
EXPOSE 8083

# 健康检查
HEALTHCHECK --interval=30s --timeout=30s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8083/ || exit 1

# 启动命令
CMD ["python", "-m", "uvicorn", "ocr_svr:app", "--host", "0.0.0.0", "--port", "8083", "--workers", "1"]

# 云服务器内存不够，环境变量OMP_NUM_THREADS、MKL_NUM_THREADS设为1，以及--workers设为1，大概减少几百MB内存占用