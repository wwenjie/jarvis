# 第一阶段：构建 Go 应用
FROM golang:1.23.5 AS builder

WORKDIR /app

# 使用国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum
COPY server/service/rag_svr/go.mod ./
COPY server/service/rag_svr/go.sum ./

# 复制本地依赖模块
COPY server/framework /framework

# 下载依赖
RUN go mod download

# 复制全部源码（包含 build.sh、script、config、kitex_gen 等）
COPY server/service/rag_svr/ ./

# 编译（假设你有 build.sh 脚本）
RUN ./build.sh

# 第二阶段：精简运行环境
FROM ubuntu:22.04

WORKDIR /app

# 安装CA证书库和时区设置
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 只复制编译好的二进制和启动脚本，不带源码
COPY --from=builder /app/output/bin/rag_svr /app/bin/rag_svr
COPY --from=builder /app/output/bootstrap.sh /app/bootstrap.sh
COPY --from=builder /app/config/config.yaml /app/config/config.yaml

RUN chmod +x /app/bootstrap.sh

EXPOSE 8888

CMD ["/app/bootstrap.sh"] 