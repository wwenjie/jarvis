package main

import (
	"context"
	"fmt"
	"log"
	"time"

	client "your_project/kitex_gen/your_service/client"
	"your_project/kitex_gen/your_service/proto"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/transmeta"
	etcd "github.com/kitex-contrib/registry-etcd"
)

func main() {
	h := server.Default()

	// 创建etcd服务发现组件
	r, err := etcd.NewEtcdResolver([]string{"localhost:2379"})
	if err != nil {
		log.Fatalf("创建etcd解析器失败: %v", err)
	}

	// 初始化Kitex客户端并集成etcd服务发现
	kitexClient := client.MustNewClient(
		"your-service-name",
		client.WithResolver(r),
		client.WithRPCTimeout(3*time.Second),
		client.WithMiddleware(func(next client.Invoker) client.Invoker {
			return func(ctx context.Context, method string, req, resp interface{}, opts ...rpcinfo.Option) error {
				// 示例：添加请求头信息
				ctx = transmeta.WithClientTransportHeader(ctx, "x-request-id", "123456")
				return next(ctx, method, req, resp, opts...)
			}
		}),
	)

	// 定义HTTP路由
	h.GET("/api/users/:id", func(c context.Context, ctx *app.RequestContext) {
		id := ctx.Param("id")

		req := &proto.GetUserRequest{
			Id: id,
		}

		// 调用Kitex服务（自动从etcd获取服务实例）
		resp, err := kitexClient.GetUser(c, req)
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{
				"error": fmt.Sprintf("调用用户服务失败: %v", err),
			})
			return
		}

		ctx.JSON(consts.StatusOK, utils.H{
			"user": resp.GetUser(),
		})
	})

	h.Spin()
}
