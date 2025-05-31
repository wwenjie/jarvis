module server/api_gateway

go 1.23.5

require (
	server v0.0.0
	server/framework v0.0.0
	server/service/rag_svr v0.0.0
	github.com/cloudwego/hertz v0.10.0
	github.com/cloudwego/kitex v0.13.1
	github.com/kitex-contrib/registry-etcd v0.2.6
)

replace (
	server => ../
	server/framework => ../framework
	server/service/rag_svr => ../service/rag_svr
) 