// Code generated by hertz generator.

package main

import (
	"context"
	"log"
	"time"

	"server/api_service/biz/router/api_service"
	"server/framework"
	"server/framework/logger"
	ragservice "server/service/rag_svr/kitex_gen/rag_svr/ragservice"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/endpoint"
	"github.com/cloudwego/kitex/pkg/loadbalance"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
)

func main() {
	// 创建etcd服务发现组件
	err, _, etcdResolver := framework.InitService()
	if err != nil {
		log.Fatalf("创建etcd解析器失败: %v", err)
	}

	h := server.Default(
		server.WithHostPorts("0.0.0.0:8081"),    // 允许所有网络接口访问
		server.WithReadTimeout(time.Second*10),  // 设置读取超时
		server.WithWriteTimeout(time.Second*10), // 设置写入超时
	)

	// 初始化Kitex客户端并集成etcd服务发现
	ragSvrClient, err := ragservice.NewClient(
		"rag_svr",
		client.WithResolver(etcdResolver),
		client.WithRPCTimeout(10*time.Second), // RPC超时时间
		client.WithLoadBalancer(loadbalance.NewWeightedBalancer()), // 负载均衡策略
		client.WithMiddleware(func(next endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, req, resp interface{}) (err error) {
				err = next(ctx, req, resp) // 先发起RPC
				rpcInfo := rpcinfo.GetRPCInfo(ctx)
				if rpcInfo == nil {
					logger.Infof("本次请求没有下游服务端地址")
					return
				}
				to := rpcInfo.To()
				if to == nil || to.Address() == nil {
					logger.Infof("本次请求没有下游服务端地址")
					return
				}
				// 获取接口名
				invocation := rpcInfo.Invocation()
				if invocation != nil {
					logger.Infof("本次请求接口: %s, 下游服务端地址: %s",
						invocation.MethodName(),
						to.Address().String())
				} else {
					logger.Infof("本次请求下游服务端地址: %s", to.Address().String())
				}
				return
			}
		}),
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ragSvrTestClient, err := ragservice.NewClient(
		"rag_svr",
		client.WithResolver(etcdResolver),
		client.WithRPCTimeout(3*time.Second), // RPC超时时间
		client.WithLoadBalancer(loadbalance.NewWeightedBalancer()), // 负载均衡策略

		client.WithMiddleware(func(next endpoint.Endpoint) endpoint.Endpoint {
			return func(ctx context.Context, req, resp interface{}) (err error) {
				err = next(ctx, req, resp) // 先发起RPC
				rpcInfo := rpcinfo.GetRPCInfo(ctx)
				if rpcInfo == nil {
					logger.Infof("本次请求没有下游服务端地址")
					return
				}
				to := rpcInfo.To()
				if to == nil || to.Address() == nil {
					logger.Infof("本次请求没有下游服务端地址")
					return
				}
				// 获取接口名
				invocation := rpcInfo.Invocation()
				if invocation != nil {
					logger.Infof("本次请求接口: %s, 下游服务端地址: %s",
						invocation.MethodName(),
						to.Address().String())
				} else {
					logger.Infof("本次请求下游服务端地址: %s", to.Address().String())
				}
				return
			}
		}),
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	ragSvrTest2Client, err := ragservice.NewClient(
		"rag_svr",
		client.WithResolver(etcdResolver),
		client.WithRPCTimeout(3*time.Second), // RPC超时时间
		client.WithLoadBalancer(loadbalance.NewWeightedRandomBalancer()), // 使用加权随机负载均衡器
	)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 将客户端保存到上下文中，供handler使用
	h.Use(func(ctx context.Context, c *app.RequestContext) {
		c.Set("rag_svr_client", ragSvrClient)
		c.Set("rag_svr_test_client", ragSvrTestClient)
		c.Set("rag_svr_test2_client", ragSvrTest2Client)
		c.Next(ctx)
	})

	// 注册路由（移到中间件之后）
	api_service.Register(h)

	logger.Infof("api_service start succ!")

	h.Spin()
}
