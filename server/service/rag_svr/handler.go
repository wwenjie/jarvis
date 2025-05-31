package main

import (
	"context"
	rag_svr "server/service/rag_svr/kitex_gen/rag_svr"

	"github.com/cloudwego/kitex/pkg/klog"
)

// RagServiceImpl implements the last service interface defined in the IDL.
type RagServiceImpl struct{}

// Test implements the RagServiceImpl interface.
func (s *RagServiceImpl) Test(ctx context.Context, req *rag_svr.TestReq) (resp *rag_svr.TestRsp, err error) {
	klog.Infof("收到测试请求: seq_id=%s, msg=%s", req.GetSeqId(), req.GetMsg())

	return &rag_svr.TestRsp{
		RetCode: 0,
		RetMsg:  "success",
	}, nil
}
