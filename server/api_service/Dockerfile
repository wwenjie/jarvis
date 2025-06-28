# 第一阶段：构建 Go 应用
FROM golang:1.23.5 AS builder

WORKDIR /app

# 使用国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制 go.mod 和 go.sum
COPY server/api_service/go.mod server/api_service/go.sum ./

# 复制本地依赖模块（注意路径要和 go.mod 里的 replace 保持一致）
COPY server/framework /framework
COPY server/service/rag_svr /service/rag_svr

# 下载依赖
RUN go mod download

# 复制 api_service 源码
COPY server/api_service .

# 编译（假设你有 build.sh 脚本）
RUN ./build.sh

# 第二阶段：精简运行环境
FROM ubuntu:22.04

WORKDIR /app

# 只复制编译好的产物，不带源码
COPY --from=builder /app/output /app
COPY --from=builder /app/config/config.yaml /app/config/config.yaml

EXPOSE 8081

CMD ["./bootstrap.sh"] 