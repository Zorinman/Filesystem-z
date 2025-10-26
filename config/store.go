package config

import (
	cmn "filestore-server/common"
)

const (
	// TempLocalRootDir : 本地临时存储地址的路径
	// TempLocalRootDir = "/data/fileserver/"
	TempLocalRootDir = "E:\\filestore-server\\tmp\\"
	//TempPartRootDir : 分块文件在本地临时存储地址的路径
	TempPartRootDir = "\"E:\\\\filestore-server\\\\tmp-part\\\\"
	// 设置当前文件的存储类型

	CurrentStoreType = cmn.StoreOSS //设置当前文件存储类型是本地，Ceph还是OSS
)
