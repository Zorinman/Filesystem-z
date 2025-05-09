package config

const (
	// MySQLSource : 要连接的数据库源；
	// 其中root:123456 是用户名密码；
	// 192.168.0.105:13306 是ip及端口；
	// fileserver 是数据库名;
	// charset=utf8 指定了数据以utf8字符编码进行传输
	MySQLSource = "root:123456@tcp(192.168.0.105:13306)/fileserver?charset=utf8"
)
