package main

import (
	"filestore-server/store/ceph"
	"os"
)

// 测试从ceph的bucket下载下来的内容是否和之前上传的内容相符合
func main() {
	bucket := ceph.GetCephBucket("userfile")

	d, _ := bucket.Get("/ceph/2ca9c55107000a268fef1bfa4dd6f8a34eb1859f")
	tmpFile, _ := os.Create("E:\\filestore-server\\tmp\\test")
	tmpFile.Write(d)
	return

}
