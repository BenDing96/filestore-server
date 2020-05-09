package handler

import (
	"filestore-server/util"
	"fmt"
	"math"
	"net/http"
	"strconv"
	rPool "filestore-server/cache/redis"
	"time"
)


// MultipartUploadInfo: 初始化信息
type MultipartUploadInfo struct {
	FileHash string
	FileSize int
	UploadID string
	ChunkSize int
	ChunkCount int
}

// 初始化分块上传
func IntialMutltipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求
	r.ParseForm()
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.Write(util.NewRespMsg(-1,"params invalid", nil).JSONBytes())
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 生成分块上传的初始化信息
	upInfo := MultipartUploadInfo{
		FileHash:filehash,
		FileSize:filesize,
		UploadID:username+fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize: 5 * 1024 * 1024, // 5MB
		ChunkCount: int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
	}

	// 4. 将初始化信息写入到redis缓存
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "chunkcount", upInfo.ChunkCount)
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "filehash", upInfo.FileHash)
	rConn.Do("HSET", "MP_" + upInfo.UploadID, "filesize", upInfo.FileSize)

	// 5. 将响应初始化的数据返回到客户端
	w.Write(util.NewRespMsg(0,"OK", upInfo).JSONBytes())
}

func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	username := r.Form.Get("username")
	// 2. 或得redis连接池中的一个连接
	// 3. 获得文件句柄，用于存储分块内容
	// 4. 更新redis缓存状态
	// 5. 返回处理结果到客户端
}