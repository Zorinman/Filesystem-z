package handler

import (
	rPool "filestore-server/cache/redis"
	dblayer "filestore-server/db"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gomodule/redigo/redis"
)

// MultipartUploadInfo : 初始化信息
type MultipartUploadInfo struct {
	FileHash   string
	FileSize   int
	UploadID   string
	ChunkSize  int
	ChunkCount int
}

//// InitialMultipartUploadHandler : 初始化分块上传
//func InitialMultipartUploadHandler(w http.ResponseWriter, r *http.Request) {
//	// 1. 解析用户请求参数
//	r.ParseForm()
//	username := r.Form.Get("username")
//	filehash := r.Form.Get("filehash")
//	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
//	if err != nil {
//		w.Write(util.NewRespMsg(-1, "params invalid", nil).JSONBytes())
//		return
//	}
//
//	// 2. 获得redis的一个连接
//	rConn := rPool.RedisPool().Get()
//	defer rConn.Close()
//
//	// 3. 生成分块上传的初始化信息
//	upInfo := MultipartUploadInfo{
//		FileHash:   filehash,
//		FileSize:   filesize,
//		UploadID:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
//		ChunkSize:  5 * 1024 * 1024, // 5MB
//		ChunkCount: int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
//	}
//
//	// 4. 将初始化信息写入到redis缓存
//	rConn.Do("HSET", "MP_"+upInfo.UploadID, "chunkcount", upInfo.ChunkCount)
//	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filehash", upInfo.FileHash)
//	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filesize", upInfo.FileSize)
//
//	// 5. 将响应初始化数据返回到客户端
//	w.Write(util.NewRespMsg(0, "OK", upInfo).JSONBytes())
//}

func InitialMultipartUploadHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize, err := strconv.Atoi(c.Request.FormValue("filesize"))
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": -1,
				"msg":  "params invalid",
			})
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 生成分块上传的初始化信息
	upInfo := MultipartUploadInfo{
		FileHash:   filehash,
		FileSize:   filesize,
		UploadID:   username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:  5 * 1024 * 1024, // 5MB
		ChunkCount: int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
	}

	// 4. 将初始化信息写入到redis缓存
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "chunkcount", upInfo.ChunkCount)
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filehash", upInfo.FileHash)
	rConn.Do("HSET", "MP_"+upInfo.UploadID, "filesize", upInfo.FileSize)

	// 5. 将响应初始化数据返回到客户端
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": upInfo,
		})
}

//// UploadPartHandler : 上传文件分块
//func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
//	// 1. 解析用户请求参数
//	r.ParseForm()
//	//	username := r.Form.Get("username")
//	uploadID := r.Form.Get("uploadid")
//	chunkIndex := r.Form.Get("index")
//
//	// 2. 获得redis连接池中的一个连接
//	rConn := rPool.RedisPool().Get()
//	defer rConn.Close()
//
//	// 3. 获得文件句柄，用于存储分块内容
//	fpath := "E:\\filestore-server\\data\\" + uploadID + "\\" + chunkIndex
//	os.MkdirAll(path.Dir(fpath), 0744)
//	fd, err := os.Create(fpath)
//	if err != nil {
//		w.Write(util.NewRespMsg(-1, "Upload part failed", nil).JSONBytes())
//		return
//	}
//	defer fd.Close()
//
//	buf := make([]byte, 1024*1024)
//	for {
//		n, err := r.Body.Read(buf)
//		fd.Write(buf[:n])
//		if err != nil {
//			break
//		}
//	}
//
//	// 4. 更新redis缓存状态
//	rConn.Do("HSET", "MP_"+uploadID, "chkidx_"+chunkIndex, 1)
//
//	// 5. 返回处理结果到客户端
//	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
//}

// UploadPartHandler : 上传文件分块
func UploadPartHandler(c *gin.Context) {
	// 1. 解析用户请求参数
	//	username := c.Request.FormValue("username")
	uploadID := c.Request.FormValue("uploadid")
	chunkIndex := c.Request.FormValue("index")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 获得文件句柄，用于存储分块内容
	fpath := "E:\\filestore-server\\data\\" + uploadID + "/" + chunkIndex // 分块文件存储路径赋值给fpath
	os.MkdirAll(path.Dir(fpath), 0744)                                    //递归创建目录
	fd, err := os.Create(fpath)                                           //在目录下创建文件
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": 0,
				"msg":  "Upload part failed",
				"data": nil,
			})
		return
	}
	defer fd.Close()

	buf := make([]byte, 1024*1024) // 创建1MB缓冲区
	for {
		n, err := c.Request.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			break
		}
	}

	// 4. 更新redis缓存状态
	rConn.Do("HSET", "MP_"+uploadID, "chkidx_"+chunkIndex, 1) // 更新分块上传状态，以 uploadID 为哈希表的键名，chkidx_<chunkIndex> 为字段名，如果字段不存在，则创建该字段；如果字段已存在，则更新其值

	// 5. 返回处理结果到客户端
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": nil,
		})
}

//// CompleteUploadHandler : 通知上传合并
//func CompleteUploadHandler(w http.ResponseWriter, r *http.Request) {
//	// 1. 解析请求参数
//	r.ParseForm()
//	upid := r.Form.Get("uploadid")
//	username := r.Form.Get("username")
//	filehash := r.Form.Get("filehash")
//	filesize := r.Form.Get("filesize")
//	filename := r.Form.Get("filename")
//
//	// 2. 获得redis连接池中的一个连接
//	rConn := rPool.RedisPool().Get()
//	defer rConn.Close()
//
//	// 3. 通过uploadid查询redis并判断是否所有分块上传完成
//	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+upid))
//	if err != nil {
//		w.Write(util.NewRespMsg(-1, "complete upload failed", nil).JSONBytes())
//		return
//	}
//	totalCount := 0
//	chunkCount := 0
//	for i := 0; i < len(data); i += 2 {
//		k := string(data[i].([]byte))
//		v := string(data[i+1].([]byte))
//		if k == "chunkcount" {
//			totalCount, _ = strconv.Atoi(v)
//		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
//			chunkCount++
//		}
//	}
//	if totalCount != chunkCount {
//		w.Write(util.NewRespMsg(-2, "invalid request", nil).JSONBytes())
//		return
//	}
//
//	// 4. TODO：合并分块
//
//	// 5. 更新唯一文件表及用户文件表
//	fsize, _ := strconv.Atoi(filesize)
//	dblayer.OnFileUploadFinished(filehash, filename, int64(fsize), "")
//	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))
//
//	// 6. 响应处理结果
//	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
//}

// CompleteUploadHandler : 通知上传合并
func CompleteUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	upid := c.Request.FormValue("uploadid")
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filesize := c.Request.FormValue("filesize")
	filename := c.Request.FormValue("filename")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过uploadid查询redis并判断是否所有分块上传完成
	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+upid))
	if err != nil {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": -1,
				"msg":  "OK",
				"data": nil,
			})
		return
	}
	totalCount := 0
	chunkCount := 0
	for i := 0; i < len(data); i += 2 {
		k := string(data[i].([]byte))
		v := string(data[i+1].([]byte))
		if k == "chunkcount" {
			totalCount, _ = strconv.Atoi(v)
		} else if strings.HasPrefix(k, "chkidx_") && v == "1" {
			chunkCount++
		}
	}
	if totalCount != chunkCount {
		c.JSON(
			http.StatusOK,
			gin.H{
				"code": -2,
				"msg":  "OK",
				"data": nil,
			})
		return
	}

	// 4. TODO：合并分块
	// 合并分块文件
	targetFilePath := path.Join("E:\\filestore-server\\data", filename)
	if err := mergeChunks(upid, targetFilePath, totalCount); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			gin.H{
				"code": -3,
				"msg":  "Failed to merge chunks",
				"data": nil,
			})
		return
	}
	// 5. 更新唯一文件表及用户文件表
	fsize, _ := strconv.Atoi(filesize)
	dblayer.OnFileUploadFinished(filehash, filename, int64(fsize), "")
	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))

	// 6. 响应处理结果
	c.JSON(
		http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "OK",
			"data": nil,
		})
}

// mergeChunks : 合并分块文件
func mergeChunks(uploadID, targetFilePath string, chunkCount int) error {
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		return err
	}
	defer targetFile.Close()

	for i := 0; i < chunkCount; i++ {
		chunkPath := path.Join("E:\\filestore-server\\data", uploadID, strconv.Itoa(i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		io.Copy(targetFile, chunkFile)
		chunkFile.Close()
		os.Remove(chunkPath) // 删除分块文件
	}
	return nil
}
