FROM python:3.10.12-slim AS builder

# 设置pip镜像源
RUN pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple/

# 复制requirements文件
COPY py_server/agent/requirements.txt ./

# 安装Python依赖到临时目录
RUN pip install --no-cache-dir --user -r requirements.txt

FROM python:3.10.12-slim

# 设置工作目录
WORKDIR /app

# 从builder阶段复制安装的包
COPY --from=builder /root/.local /root/.local

# 设置PATH
ENV PATH=/root/.local/bin:$PATH

# 复制应用代码
COPY py_server/agent/ .

# 创建日志目录
RUN mkdir -p /app/logs

# 暴露端口
EXPOSE 8085

# 启动命令
CMD ["python", "agent.py"] 