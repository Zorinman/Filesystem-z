package config

const (
	// CephAccessKey : 访问Key ，ceph创建rgw用户时得到的信息
	CephAccessKey = ""
	// CephSecretKey : 访问密钥 ，ceph创建rgw用户时得到的信息
	CephSecretKey = ""
	// CephGWEndpoint : gateway地址  后面7480是容器rgw的端口
	CephGWEndpoint = "http://192.168.0.105:7480"
	//CephRootDir:存储的前缀地址
	CephRootDir = "/ceph"
)
