package main

import (
	"filestore-server/util" // 确保路径正确
	"fmt"
	"log"
	"os"
)

func main() {
	// 指定需要计算哈希值的文件路径
	filePath := "E:\\filestore-server\\tmp\\周林犷2112333047.docx" // 替换为你的文件路径
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	// 计算文件的 SHA1 哈希值
	fileSha1 := util.FileSha1(file)
	fmt.Printf("SHA1: %s\n", fileSha1)
}
