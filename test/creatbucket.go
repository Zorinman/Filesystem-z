package main

import (
	"filestore-server/store/ceph"
	"fmt"

	"gopkg.in/amz.v1/s3"
)

// 创建一个bucket
func main() {
	bucketName := "userfile" // 指定要创建的 bucket 名称

	// 获取 Ceph bucket 对象
	bucket := ceph.GetCephBucket(bucketName)

	// 创建一个新的 bucket
	err := bucket.PutBucket(s3.PublicRead) // 使用 s3.PublicRead 设置权限
	if err != nil {
		fmt.Printf("create bucket err: %v\n", err)
	} else {
		fmt.Printf("Bucket '%s' created successfully\n", bucketName)
	}
}
