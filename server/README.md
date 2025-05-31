# Jarvis Server

## 服务启动步骤

### 1. 启动 etcd 服务
```bash
cd deploy
docker-compose -f etcd-docker-compose.yml up -d
```

#### 验证 etcd 启动情况
```bash
# 设置 etcd 客户端 API 版本
export ETCDCTL_API=3

# 设置 etcd 集群端点
export ETCDCTL_ENDPOINTS="http://localhost:2379,http://localhost:2381,http://localhost:2383"

# 测试读写
etcdctl put test_key "test_value"
etcdctl get test_key
```

### 2. 启动 Kitex 服务
```bash
go run kitex_service/main.go
```

### 3. 启动 Hertz API 网关
```bash
go run api_gateway/main.go
```

### 4. 测试服务
```bash
curl http://localhost:8080/api/users/123
```

## 高级配置说明

### 自定义权重与元数据
在服务注册时能够添加额外标签，像版本、区域、实例规格等信息。

### 连接池配置
```go
client.WithConnectionPool(
    pool.NewFixedSizePool(10, 30*time.Second),
)
```

### 重试策略
```go
client.WithFailureRetry(retry.NewFailurePolicy(
    retry.WithMaxRetryTimes(3),
    retry.WithPerRetryTimeout(2*time.Second),
))
```

### 熔断器
```go
client.WithCircuitBreaker(
    breaker.NewSentinelBreaker(),
)
```