package main

import (
	cfg "filestore-server/config"
	"filestore-server/route"
)

func main() {
	//// 静态文件处理
	//fs := http.FileServer(http.Dir("./static")) // 假设静态文件存放在当前目录的 static 文件夹中
	//http.Handle("/static/", http.StripPrefix("/static/", fs))
	//
	////文件增删改查接口
	//http.HandleFunc("/file/upload", handler.HTTPInterceptor(handler.UploadHandler))
	//http.HandleFunc("/file/upload/suc", handler.HTTPInterceptor(handler.UploadSucHandler))
	//http.HandleFunc("/file/meta", handler.HTTPInterceptor(handler.GetFileMetaHandler))
	//http.HandleFunc("/file/download", handler.HTTPInterceptor(handler.DownloadHandler))
	//http.HandleFunc("/file/update", handler.HTTPInterceptor(handler.FileMetaUpdateHandler))
	//http.HandleFunc("/file/delete", handler.HTTPInterceptor(handler.FileDeleteHandler))
	//http.HandleFunc("/file/query", handler.HTTPInterceptor(handler.FileQueryHandler))
	//
	//// 秒传接口
	//http.HandleFunc("/file/fastupload", handler.HTTPInterceptor(handler.TryFastUploadHandler))
	//http.HandleFunc("/file/downloadurl", handler.HTTPInterceptor(handler.DownloadURLHandler))
	////用户相关接口
	//http.HandleFunc("/user/signup", handler.HTTPInterceptor(handler.SignupHandler))
	//http.HandleFunc("/user/signin", handler.SignInHandler)
	//http.HandleFunc("/user/info", handler.HTTPInterceptor(handler.UserInfoHandler))
	//// 分块上传接口
	//http.HandleFunc("/file/mpupload/init",
	//	handler.HTTPInterceptor(handler.InitialMultipartUploadHandler))
	//http.HandleFunc("/file/mpupload/uppart",
	//	handler.HTTPInterceptor(handler.UploadPartHandler))
	//http.HandleFunc("/file/mpupload/complete",
	//	(handler.CompleteUploadHandler))

	//gun framework
	router := route.Router()
	err := router.Run(cfg.UploadServiceHost)
	if err != nil {
		return
	}

	//fmt.Printf("上传服务启动中，开始监听监听[%s]...\n", cfg.UploadServiceHost)
	//// 启动服务并监听端口
	//err := http.ListenAndServe(cfg.UploadServiceHost, nil)
	//if err != nil {
	//	fmt.Printf("Failed to start server, err:%s", err.Error())
	//}
}
