# 构建阶段
FROM golang:1.23.5 AS builder

WORKDIR /app

# 复制 go.mod 和 go.sum
COPY go.mod ./
COPY go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN ./build.sh

# 运行阶段
FROM ubuntu:22.04

WORKDIR /app

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /app/output/bin/rag_svr /app/rag_svr
COPY --from=builder /app/output/script/bootstrap.sh /app/bootstrap.sh

# 设置执行权限
RUN chmod +x /app/bootstrap.sh

# 暴露端口
EXPOSE 8888

# 启动服务
CMD ["/app/bootstrap.sh"] 