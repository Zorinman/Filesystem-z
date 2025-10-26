package meta

import (
	mydb "filestore-server/db"
)

// FileMeta:文件源信息结构
type FileMeta struct {
	FileSha1 string
	FileName string
	FileSize int64
	Location string
	UploadAt string
}

var fileMetas map[string]FileMeta //声明一个名为 fileMetas 的变量，类型为 map[string]FileMeta（零值状态，未分配内存）

func init() { //init() 是 Go 语言中的特殊函数，用于包的初始化逻辑，在 main() 函数执行前自动调用
	fileMetas = make(map[string]FileMeta) //使用 make 初始化 fileMetas，分配底层内存空间

}

// UpdateFileMeta：新增/更新文件元信息
func UpdateFileMeta(fmeta FileMeta) { //传入一个 FileMeta 类型的参数 fmeta
	fileMetas[fmeta.FileSha1] = fmeta
}

// UpdateFileMetaDB 新增/更新文件元信息到mysql
func UpdateFileMetaDB(fmeta FileMeta) bool {
	return mydb.OnFileUploadFinished(
		fmeta.FileSha1, fmeta.FileName, fmeta.FileSize, fmeta.Location)
}

// GetFileMeta:通过sha1值获取文件元信息对象
func GetFileMeta(fileSha1 string) FileMeta { //传入一个 sha1 值，返回对应的 FileMeta 对象
	return fileMetas[fileSha1]
}

// GetFileMetaDB:通过sha1值从数据库获取文件元信息对象
func GetFileMetaDB(fileSha1 string) (FileMeta, error) {
	tfile, err := mydb.GetFileMeta(fileSha1)
	if err != nil {
		return FileMeta{}, err
	}
	fmeta := FileMeta{
		FileSha1: tfile.FileHash,
		FileName: tfile.FileName.String,
		FileSize: tfile.FileSize.Int64,
		Location: tfile.Fileaddr.String,
	}
	return fmeta, nil
}

// RemoveFileMeta 删除元信息
func RemoveFileMeta(fileSha1 string) {
	delete(fileMetas, fileSha1)

}
