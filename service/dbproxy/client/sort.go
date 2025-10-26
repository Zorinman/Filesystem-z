package client

//该文件实现了一个自定义排序器 ByUploadTime
// 用于对 FileMeta 切片按上传时间降序排序。
// 通过实现 sort.Interface 接口，可以直接使用 Go 的 sort.Sort 函数对数据进行排序。
import "time"

const baseFormat = "2006-01-02 15:04:05"

type ByUploadTime []FileMeta

func (a ByUploadTime) Len() int {
	return len(a)
}

func (a ByUploadTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByUploadTime) Less(i, j int) bool {
	iTime, _ := time.Parse(baseFormat, a[i].UploadAt)
	jTime, _ := time.Parse(baseFormat, a[j].UploadAt)
	return iTime.UnixNano() > jTime.UnixNano()
}
