package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"server/framework"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr/ragservice"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/pkg/transmeta"
)

func main() {
	h := server.Default()

	// 创建etcd服务发现组件
	r, err := framework.NewEtcdResolver()
	if err != nil {
		log.Fatalf("创建etcd解析器失败: %v", err)
	}

	// 初始化Kitex客户端并集成etcd服务发现
	kitexClient, err := rag_svr.NewClient(
		"rag_svr",
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
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}

	// 定义HTTP路由
	h.GET("/api/test", func(c context.Context, ctx *app.RequestContext) {
		msg := ctx.Query("msg")
		if msg == "" {
			msg = "default message"
		}

		req := &rag_svr.TestReq{
			SeqId: fmt.Sprintf("%d", time.Now().UnixNano()),
			Msg:   msg,
		}

		// 调用Kitex服务（自动从etcd获取服务实例）
		resp, err := kitexClient.Test(c, req)
		if err != nil {
			ctx.JSON(consts.StatusInternalServerError, utils.H{
				"error": fmt.Sprintf("调用服务失败: %v", err),
			})
			return
		}

		ctx.JSON(consts.StatusOK, utils.H{
			"ret_code": resp.RetCode,
			"ret_msg":  resp.RetMsg,
		})
	})

	h.Spin()
}
