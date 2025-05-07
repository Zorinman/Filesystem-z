package main

//该文件的主要功能是启动一个基于 go-micro 的 RPC 服务，服务名称为 "go.micro.service.dbproxy"，并注册了一个服务处理器。
import (
	"log"
	"time"

	"github.com/micro/go-micro"

	dbProxy "filestore-server/service/dbproxy/proto" // rpc服务的协议定义
	dbRpc "filestore-server/service/dbproxy/rpc"     //  rpc服务具体逻辑的实现
)

func startRpcService() {
	service := micro.NewService( // 创建一个rpc微服务
		micro.Name("go.micro.service.dbproxy"), // 在注册中心中的服务名称
		micro.RegisterTTL(time.Second*10),      // 声明超时时间, 避免consul不主动删掉已失去心跳的服务节点
		micro.RegisterInterval(time.Second*5),
	)
	service.Init() // 初始化服务，解析命令行参数等

	dbProxy.RegisterDBProxyServiceHandler(service.Server(), new(dbRpc.DBProxy)) // 将服务处理器 new(dbRpc.DBProxy) 注册到微服务服务器 service.Server() 中，使其能够处理 RPC 请求，处理器负责处理业务逻辑
	if err := service.Run(); err != nil {
		log.Println(err)
	}
}

func main() {
	startRpcService() // 启动rpc服务
	// res, err := mapper.FuncCall("/user/UserExist", []interface{}{"haha"}...)
	// log.Printf("error: %+v\n", err)
	// log.Printf("result: %+v\n", res[0].Interface())

	// res, err = mapper.FuncCall("/user/UserExist", []interface{}{"admin"}...)
	// log.Printf("error: %+v\n", err)
	// log.Printf("result: %+v\n", res[0].Interface())
}
