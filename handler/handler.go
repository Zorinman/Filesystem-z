package handler

import (
	"bytes"
	"encoding/json"
	cmn "filestore-server/common"
	cfg "filestore-server/config"
	dblayer "filestore-server/db"
	"filestore-server/meta"
	"filestore-server/mq"
	"filestore-server/store/ceph"
	"filestore-server/store/oss"
	"filestore-server/util"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/amz.v1/s3"
)

// UploadHandler : 响应上传页面
func UploadHandler(c *gin.Context) {
	data, err := os.ReadFile("static/view/index.html")
	if err != nil {
		c.String(404, `网页不存在`)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}

// DoUploadHandler ： 处理post文件上传
func DoUploadHandler(c *gin.Context) {
	errCode := 0
	defer func() {
		if errCode < 0 {
			c.JSON(http.StatusOK, gin.H{
				"code": errCode,
				"msg":  "Upload failed",
			})
		}
	}()

	// 1. 从form表单中获得文件内容句柄
	file, head, err := c.Request.FormFile("file")
	if err != nil {
		fmt.Printf("Failed to get form data, err:%s\n", err.Error())
		errCode = -1
		return
	}
	defer file.Close()

	// 2. 把文件内容转为[]byte
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		fmt.Printf("Failed to get file data, err:%s\n", err.Error())
		errCode = -2
		return
	}

	// 3. 构建文件元信息
	fileMeta := meta.FileMeta{
		FileName: head.Filename,
		FileSha1: util.Sha1(buf.Bytes()), //　计算文件sha1
		FileSize: int64(len(buf.Bytes())),
		UploadAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	// 4. 将文件写入临时存储位置
	fileMeta.Location = cfg.TempLocalRootDir + head.Filename // 临时存储地址
	newFile, err := os.Create(fileMeta.Location)
	if err != nil {
		fmt.Printf("Failed to create file, err:%s\n", err.Error())
		errCode = -3
		return
	}

	defer newFile.Close()

	nByte, err := newFile.Write(buf.Bytes())
	if int64(nByte) != fileMeta.FileSize || err != nil {
		fmt.Printf("Failed to save data into file, writtenSize:%d, err:%s\n", nByte, err.Error())
		errCode = -4
		return
	}
	newFile.Seek(0, 0) // 游标重新回到文件头部
	if cfg.CurrentStoreType == cmn.StoreCeph {
		// 文件写入Ceph存储
		data, _ := io.ReadAll(newFile)
		bucket := ceph.GetCephBucket("userfile")
		cephPath := "/ceph/" + fileMeta.FileSha1 //这里的路径指的是bucket中的路径
		_ = bucket.Put(cephPath, data, "octet-stream", s3.PublicRead)
		fileMeta.Location = cephPath
	} else if cfg.CurrentStoreType == cmn.StoreOSS {
		// 文件写入OSS存储
		ossPath := "oss/" + fileMeta.FileName
		// 判断写入OSS为同步还是异步
		if !cfg.AsyncTransferEnable {
			err = oss.Bucket().PutObject(ossPath, newFile)
			if err != nil {
				fmt.Println(err.Error())
				errCode = -5
				return
			}

			fileMeta.Location = ossPath
		} else {
			// 写入异步转移任务队列
			data := mq.TransferData{
				FileHash:      fileMeta.FileSha1,
				CurLocation:   fileMeta.Location,
				DestLocation:  ossPath,
				DestStoreType: cmn.StoreOSS,
			}
			pubData, _ := json.Marshal(data)
			pubSuc := mq.Publish(
				cfg.TransExchangeName,
				cfg.TransOSSRoutingKey,
				pubData,
			)
			if !pubSuc {
				// TODO: 当前发送转移信息失败，稍后重试
			}
		}
	}

	_ = meta.UpdateFileMetaDB(fileMeta) // 更新元信息到数据库

	username := c.Request.FormValue("username")

	suc := dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize) // 更新用户文件表记录
	if suc {
		c.Redirect(http.StatusFound, "/static/view/home.html")
	} else {
		errCode = -6
	}
}

//// 处理文件上传
//func UploadHandler(w http.ResponseWriter, r *http.Request) {
//	if r.Method == "GET" {
//		//返回上传html页面
//		data, err := os.ReadFile("static/view/index.html")
//		if err != nil {
//			io.WriteString(w, "internel server error")
//			return
//		}
//		io.WriteString(w, string(data))
//	} else if r.Method == "POST" {
//		//接受文件流及存储到临时存储目录
//		file, head, err := r.FormFile("file")
//		if err != nil {
//			fmt.Printf("Failed to get data,err:%s\n", err.Error())
//			return
//		}
//		defer file.Close()
//
//		fileMeta := meta.FileMeta{
//			FileName: head.Filename,
//			Location: cfg.TempLocalRootDir + head.Filename, // 临时存储地址
//			UploadAt: time.Now().Format("2006-01-02 15:04:05")}
//
//		newFile, err := os.Create(fileMeta.Location)
//		if err != nil {
//			fmt.Printf("Failed to creat file ,err%s\n", err.Error())
//			return
//		}
//		defer newFile.Close()
//		fileMeta.FileSize, err = io.Copy(newFile, file)
//		if err != nil {
//			fmt.Printf("Failed to save data into file,err:%s\n", err.Error())
//			return
//		}
//
//		newFile.Seek(0, 0)
//		fileMeta.FileSha1 = util.FileSha1(newFile)
//
//		newFile.Seek(0, 0) // 游标重新回到文件头部
//		if cfg.CurrentStoreType == cmn.StoreCeph {
//			// 文件写入Ceph存储
//			data, _ := io.ReadAll(newFile)
//			bucket := ceph.GetCephBucket("userfile")
//			cephPath := "/ceph/" + fileMeta.FileSha1 //这里的路径指的是bucket中的路径
//			_ = bucket.Put(cephPath, data, "octet-stream", s3.PublicRead)
//			fileMeta.Location = cephPath
//		} else if cfg.CurrentStoreType == cmn.StoreOSS {
//			// 文件写入OSS存储
//			ossPath := "oss/" + fileMeta.FileSha1
//			// 判断写入OSS为同步还是异步
//			if !cfg.AsyncTransferEnable {
//				err = oss.Bucket().PutObject(ossPath, newFile)
//				if err != nil {
//					fmt.Println(err.Error())
//					w.Write([]byte("Upload failed!"))
//					return
//				}
//
//				fileMeta.Location = ossPath
//			} else {
//				// 写入异步转移任务队列
//				data := mq.TransferData{
//					FileHash:      fileMeta.FileSha1,
//					CurLocation:   fileMeta.Location,
//					DestLocation:  ossPath,
//					DestStoreType: cmn.StoreOSS,
//				}
//				pubData, _ := json.Marshal(data)
//				pubSuc := mq.Publish(
//					cfg.TransExchangeName,
//					cfg.TransOSSRoutingKey,
//					pubData,
//				)
//				if !pubSuc {
//					// TODO: 当前发送转移信息失败，稍后重试
//				}
//			}
//		}
//		//meta.UpdateFileMeta(fileMeta)
//		_ = meta.UpdateFileMetaDB(fileMeta)
//
//		//更新用户文件表记录
//		err = r.ParseForm()
//		if err != nil {
//			return
//		}
//		username := r.Form.Get("username")
//		if username == "" {
//			w.WriteHeader(http.StatusBadRequest)
//			w.Write([]byte("Upload Failed: Invalid username."))
//			return
//		}
//		suc := dblayer.OnUserFileUploadFinished(username, fileMeta.FileSha1, fileMeta.FileName, fileMeta.FileSize)
//		if suc {
//			http.Redirect(w, r, "/static/view/home.html", http.StatusFound)
//		} else {
//			w.Write([]byte("Upload Failed."))
//		}
//	}
//}

//// UploadSucHandler:上传完成
//func UploadSucHandler(w http.ResponseWriter, r *http.Request) {
//	io.WriteString(w, "Upload finished!")
//}

// UploadSucHandler : 上传已完成
func UploadSucHandler(c *gin.Context) {
	c.JSON(http.StatusOK,
		gin.H{
			"code": 0,
			"msg":  "Upload Finish!",
		})
}

//// GetFileMetaHandler//获取文件元信息
//func GetFileMetaHandler(w http.ResponseWriter, r *http.Request) {
//	r.ParseForm()
//	filehash := r.Form["filehash"][0]
//	//fMeta := meta.GetFileMeta(filehash)
//	fMeta, err := meta.GetFileMetaDB(filehash)
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	data, err := json.Marshal(fMeta)
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	w.Write(data)
//
//}

// GetFileMetaHandler : 获取文件元信息
func GetFileMetaHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	fMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Upload failed!",
			})
		return
	}

	data, err := json.Marshal(fMeta)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -3,
				"msg":  "Upload failed!",
			})
		return
	}
	c.Data(http.StatusOK, "application/json", data) //
}

//// FileQueryHandler:查询批量的文件元信息
//func FileQueryHandler(w http.ResponseWriter, r *http.Request) {
//	r.ParseForm()
//	limitCnt, _ := strconv.Atoi(r.Form.Get("limit"))
//	username := r.Form.Get("username")
//	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	data, err := json.Marshal(userFiles)
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	w.Write(data)
//
//}

// FileQueryHandler : 查询批量的文件元信息
func FileQueryHandler(c *gin.Context) {
	limitCnt, _ := strconv.Atoi(c.Request.FormValue("limit"))
	username := c.Request.FormValue("username")
	userFiles, err := dblayer.QueryUserFileMetas(username, limitCnt)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -1,
				"msg":  "Query failed!",
			})
		return
	}

	data, err := json.Marshal(userFiles)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			gin.H{
				"code": -2,
				"msg":  "Query failed!",
			})
		return
	}
	c.Data(http.StatusOK, "application/json", data)
}

//// DownloadHandler//下载文件
//func DownloadHandler(w http.ResponseWriter, r *http.Request) {
//	r.ParseForm()
//	fsha1 := r.Form.Get("filehash")
//	// 获取文件元数据
//	fm, err := meta.GetFileMetaDB(fsha1)
//	if err != nil {
//		log.Printf("Error get file meta %s: %s", fsha1, err)
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//	//判断文件存储位置是否在ceph
//	if strings.HasPrefix(fm.Location, "/ceph") {
//		var d []byte
//		bucket := ceph.GetCephBucket("userfile")
//		d, err = bucket.Get(fm.Location)
//		if err != nil {
//			log.Printf("Error getting file from Ceph at %s: %s", fm.Location, err)
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		//在某些情况下，浏览器会根据文件扩展名自动推断文件类型，即使没有设置 Content-Type。因此，文件仍然能够正常下载，并可能正确识别文件类型，特别是对于常见文件格式
//		w.Header().Set("Content-Type", "application/octet-stream")
//		//Content-Disposition 头主要确保浏览器将文件作为附件下载，而不是直接在浏览器中显示
//		w.Header().Set("Content-Disposition", "attachment;filename=\""+fm.FileName+"\"")
//		_, err = w.Write(d)
//		if err != nil {
//			log.Printf("Error writing response: %s", err)
//		}
//		//判断文件存储位置是否在本地
//	} else if strings.HasPrefix(fm.Location, "E:\\filestore-server\\tmp\\") {
//		var f *os.File
//		var data []byte
//		f, err = os.Open(fm.Location)
//		if err != nil {
//			log.Printf("Error opening file at %s: %s", fm.Location, err)
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		defer f.Close()
//		data, err = io.ReadAll(f)
//		//在某些情况下，浏览器会根据文件扩展名自动推断文件类型，即使没有设置 Content-Type。因此，文件仍然能够正常下载，并可能正确识别文件类型，特别是对于常见文件格式
//		w.Header().Set("Content-Type", "application/octet-stream")
//		//Content-Disposition 头主要确保浏览器将文件作为附件下载，而不是直接在浏览器中显示
//		w.Header().Set("Content-Disposition", "attachment;filename=\""+fm.FileName+"\"")
//		_, err = w.Write(data)
//		if err != nil {
//			log.Printf("Error writing response: %s", err)
//		}
//	}
//}

// DownloadHandler : 文件下载接口
func DownloadHandler(c *gin.Context) {
	fsha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	if fsha1 == "" || username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数不能为空"})
		return
	}
	fm, _ := meta.GetFileMetaDB(fsha1) // 从数据库获取文件元信息
	userFile, _ := dblayer.QueryUserFileMeta(username, fsha1)

	if strings.HasPrefix(fm.Location, cfg.TempLocalRootDir) { //判断字符串 fm.Location 是否以 cfg.TempLocalRootDir 开头
		// 本地文件， 直接下载
		c.FileAttachment(fm.Location, userFile.FileName) // 使用 c.FileAttachment() 自动设置 Content-Disposition 头，触发浏览器下载
	} else if strings.HasPrefix(fm.Location, cfg.CephRootDir) {
		// ceph中的文件，通过ceph api先下载
		bucket := ceph.GetCephBucket("userfile")
		data, _ := bucket.Get(fm.Location)
		//	c.Header("content-type", "application/octect-stream")
		c.Header("content-disposition", "attachment; filename=\""+userFile.FileName+"\"") //强制浏览器将服务器返回的内容视为附件下载， filename为下载的默认文件名为
		c.Data(http.StatusOK, "application/octect-stream", data)
	}
}

// // FileMetaUpdateHandler文件元信息重命名更新
//
//	func FileMetaUpdateHandler(w http.ResponseWriter, r *http.Request) {
//		r.ParseForm()
//		opType := r.Form.Get("op")
//		fileSha1 := r.Form.Get("filehash")
//		newFileName := r.Form.Get("filename")
//		//op参数通常用来指示所要执行的操作类型,op 的值是 "0"，则表示允许更新文件的元信息。op 的值不是 "0"，则表示禁止此操作，返回 403 Forbidden 状态
//		if opType != "0" {
//			w.WriteHeader(http.StatusForbidden)
//			return
//		}
//		if r.Method != "POST" {
//			w.WriteHeader(http.StatusMethodNotAllowed)
//			return
//		}
//
//		curFileMeta := meta.GetFileMeta(fileSha1)
//		curFileMeta.FileName = newFileName
//		meta.UpdateFileMeta(curFileMeta)
//		data, err := json.Marshal(curFileMeta)
//		if err != nil {
//			w.WriteHeader(http.StatusInternalServerError)
//			return
//		}
//		w.WriteHeader(http.StatusOK)
//		w.Write(data)
//	}
//
// FileMetaUpdateHandler ： 更新元信息接口(重命名)
func FileMetaUpdateHandler(c *gin.Context) {
	opType := c.Request.FormValue("op")
	fileSha1 := c.Request.FormValue("filehash")
	username := c.Request.FormValue("username")
	newFileName := c.Request.FormValue("filename")

	if opType != "0" || len(newFileName) < 1 { //  opType != "0"表示禁止更新文件的元信息
		c.Status(http.StatusForbidden)
		return
	}

	// 更新用户文件表tbl_user_file中的文件名，tbl_file的文件名不用修改
	_ = dblayer.RenameFileName(username, fileSha1, newFileName)

	// 返回最新的文件信息
	userFile, err := dblayer.QueryUserFileMeta(username, fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	// data, err := json.Marshal(userFile)  //将 Go 数据结构（如结构体、map）转换为 JSON 格式的字节数组（[]byte）
	// if err != nil {
	// 	c.Status(http.StatusInternalServerError)
	// 	return
	// }
	c.JSON(http.StatusOK, userFile) //将Go数据结构序列化为JSON格式并设置响应头，发送错误时自动返回 500 错误
}

//// FileDeleteHandler:删除文件以及元信息
//func FileDeleteHandler(w http.ResponseWriter, r *http.Request) {
//	r.ParseForm()
//	fileSha1 := r.Form.Get("filehash")
//	fMeta := meta.GetFileMeta(fileSha1)
//	os.Remove(fMeta.Location)
//	meta.RemoveFileMeta(fileSha1)
//
//	w.WriteHeader(http.StatusOK)
//}

// FileDeleteHandler : 删除文件及元信息
func FileDeleteHandler(c *gin.Context) {
	username := c.Request.FormValue("username")
	fileSha1 := c.Request.FormValue("filehash")

	fm, err := meta.GetFileMetaDB(fileSha1)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 删除本地文件
	err = os.Remove(fm.Location)
	if err != nil {
		return
	}
	// TODO: 可考虑删除Ceph/OSS上的文件
	// 可以不立即删除，加个超时机制，
	// 比如该文件10天后也没有用户再次上传，那么就可以真正的删除了

	// 删除文件表中的一条记录
	suc := dblayer.DeleteUserFile(username, fileSha1)
	if !suc {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

//// TryFastUploadHandler：尝试妙传接口
//func TryFastUploadHandler(w http.ResponseWriter, r *http.Request) {
//	r.ParseForm()
//
//	// 1. 解析请求参数
//	username := r.Form.Get("username")
//	filehash := r.Form.Get("filehash")
//	filename := r.Form.Get("filename")
//	filesize, _ := strconv.Atoi(r.Form.Get("filesize"))
//
//	// 2. 从文件表中查询相同hash的文件记录
//	fileMeta, err := meta.GetFileMetaDB(filehash)
//	if err != nil {
//		fmt.Println(err.Error())
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//
//	// 3. 查不到记录则返回秒传失败
//	if fileMeta.FileSha1 == "" {
//		resp := util.RespMsg{
//			Code: -1,
//			Msg:  "秒传失败，请访问普通上传接口",
//		}
//		w.Write(resp.JSONBytes())
//		return
//	}
//
//	// 4. 上传过则将文件信息写入用户文件表， 返回成功
//	suc := dblayer.OnUserFileUploadFinished(
//		username, filehash, filename, int64(filesize))
//	if suc {
//		resp := util.RespMsg{
//			Code: 0,
//			Msg:  "秒传成功",
//		}
//		w.Write(resp.JSONBytes())
//		return
//	} else {
//		resp := util.RespMsg{
//			Code: -2,
//			Msg:  "秒传失败，请稍后重试",
//		}
//		w.Write(resp.JSONBytes())
//		return
//	}
//
//}

// TryFastUploadHandler : 尝试秒传接口
func TryFastUploadHandler(c *gin.Context) {
	// 1. 解析请求参数
	username := c.Request.FormValue("username")
	filehash := c.Request.FormValue("filehash")
	filename := c.Request.FormValue("filename")
	filesize, _ := strconv.Atoi(c.Request.FormValue("filesize"))

	// 2. 从文件表中查询相同hash的文件记录
	fileMeta, err := meta.GetFileMetaDB(filehash)
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		return
	}

	// 3. 查不到记录则返回秒传失败
	if fileMeta.FileSha1 == "" {
		resp := util.RespMsg{
			Code: -1,
			Msg:  "秒传失败，请访问普通上传接口",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}

	// 4. 上传过则将文件信息写入用户文件表， 返回成功
	suc := dblayer.OnUserFileUploadFinished(
		username, filehash, filename, int64(filesize))
	if suc {
		resp := util.RespMsg{
			Code: 0,
			Msg:  "秒传成功",
		}
		c.Data(http.StatusOK, "application/json", resp.JSONBytes())
		return
	}
	resp := util.RespMsg{
		Code: -2,
		Msg:  "秒传失败，请稍后重试",
	}
	c.Data(http.StatusOK, "application/json", resp.JSONBytes())
	return
}

//// DownloadURLHandler : 生成文件的下载地址
//func DownloadURLHandler(w http.ResponseWriter, r *http.Request) {
//	filehash := r.Form.Get("filehash")
//	// 从文件表查找记录
//	row, _ := dblayer.GetFileMeta(filehash)
//
//	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
//	if strings.HasPrefix(row.Fileaddr.String, "E:\\filestore-server\\tmp\\") ||
//		strings.HasPrefix(row.Fileaddr.String, "/ceph") {
//		username := r.Form.Get("username")
//		token := r.Form.Get("token")
//		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
//			r.Host, filehash, username, token)
//		w.Write([]byte(tmpURL))
//	} else if strings.HasPrefix(row.Fileaddr.String, "oss/") {
//		// oss下载url
//		signedURL := oss.DownloadURL(row.Fileaddr.String)
//		w.Write([]byte(signedURL))
//	}
//}
// DownloadURLHandler : 生成文件的下载地址

// DownloadURLHandler : 生成文件的下载地址
func DownloadURLHandler(c *gin.Context) {
	filehash := c.Request.FormValue("filehash")
	// 从文件表查找记录
	row, _ := dblayer.GetFileMeta(filehash)

	// TODO: 判断文件存在OSS，还是Ceph，还是在本地
	if strings.HasPrefix(row.Fileaddr.String, cfg.TempLocalRootDir) ||
		strings.HasPrefix(row.Fileaddr.String, cfg.CephRootDir) {
		username := c.Request.FormValue("username")
		token := c.Request.FormValue("token")
		tmpURL := fmt.Sprintf("http://%s/file/download?filehash=%s&username=%s&token=%s",
			c.Request.Host, filehash, username, token)
		fmt.Println(tmpURL)
		c.Data(http.StatusOK, "octet-stream", []byte(tmpURL))
	} else if strings.HasPrefix(row.Fileaddr.String, "oss/") {
		// oss下载url
		signedURL := oss.DownloadURL(row.Fileaddr.String)
		fmt.Println(signedURL)
		fmt.Println(row.Fileaddr.String)
		c.Data(http.StatusOK, "octet-stream", []byte(signedURL))
	}
}
