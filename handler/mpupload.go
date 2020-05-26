package handler

import (
	rPool "filestore-server/cache/redis"
	dblayer "filestore-server/db"
	"filestore-server/util"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	// ChunkDir: 上传的分块所在目录
	ChunkDir = "/tmp/chunks/"
	// MergeDir: 合并后的文件所在目录
	MergeDir = "/tmp/merge/"
	// ChunkKeyPrefix: 分块信息对应的redis键前缀
	ChunkKeyPrefix    = "chkidx_"
	HashUpIDKeyPrefix = "HASH_UPID_"
)

func init() {
	if err := os.MkdirAll(ChunkDir, 0744); err != nil {
		fmt.Println("无法制定目录用于存储分块文件： " + ChunkDir)
		os.Exit(1)
	}

	if err := os.MkdirAll(MergeDir, 0744); err != nil {
		fmt.Println("无法指定目录用于存储合并后文件： " + MergeDir)
		os.Exit(1)
	}
}

// MultipartUploadInfo: 初始化信息
type MultipartUploadInfo struct {
	FileHash   string
	FileSize   int
	UploadID   string
	ChunkSize  int
	ChunkCount int
	// 已经上传完成的分块索引列表
	ChunkExists []int
}

// IntialMutltipartUploadHandler：初始化分块上传
func IntialMutltipartUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求
	r.ParseForm()
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize, err := strconv.Atoi(r.Form.Get("filesize"))
	if err != nil {
		w.Write(util.NewRespMsg(-1, "params invalid", nil).JSONBytes())
		return
	}

	// 2. 获得redis的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过文件hash判断是否断点续传，并获取uploadID
	uploadID := ""
	keyExists, _ := redis.Bool(rConn.Do("EXISTS", HashUpIDKeyPrefix+filehash))
	if keyExists {
		uploadID, err = redis.String(rConn.Do("GET", HashUpIDKeyPrefix+filehash))
		if err != nil {
			w.Write(util.NewRespMsg(-1, "Upload part failed", err.Error()).JSONBytes())
			return
		}
	}

	// 4.1 首次上传新建uploadID
	// 4.2 断点续传则根据uploadID获取已上传的文件分块列表
	chunksExists := []int{}
	if uploadID == "" {
		uploadID = username + fmt.Sprintf("%x", time.Now().UnixNano())
	} else {
		chunks, err := redis.Values(rConn.Do("HGETALL", ChunkKeyPrefix+uploadID))
		if err != nil {
			w.Write(util.NewRespMsg(-1, "Upload part failed", err.Error()).JSONBytes())
			return
		}
		for i := 0; i < len(chunks); i += 2 {
			k := string(chunks[i].([]byte))
			v := string(chunks[i+1].([]byte))
			if strings.HasPrefix(k, ChunkKeyPrefix) && v == "1" {
				// chkidx_6 -> 6
				chunkIdx, _ := strconv.Atoi(k[7:])
				chunksExists = append(chunksExists, chunkIdx)
			}
		}
	}
	// 5. 生成分块上传的初始化信息
	upInfo := MultipartUploadInfo{
		FileHash:    filehash,
		FileSize:    filesize,
		UploadID:    username + fmt.Sprintf("%x", time.Now().UnixNano()),
		ChunkSize:   5 * 1024 * 1024, // 5MB
		ChunkCount:  int(math.Ceil(float64(filesize) / (5 * 1024 * 1024))),
		ChunkExists: chunksExists,
	}

	// 6. 将初始化信息写入到redis缓存
	if len(upInfo.ChunkExists) <= 0 {
		hKey := "MP_" + upInfo.UploadID
		rConn.Do("HSET", hKey, "chunkcount", upInfo.ChunkCount)
		rConn.Do("HSET", hKey, "filehash", upInfo.FileHash)
		rConn.Do("HSET", hKey, "filesize", upInfo.FileSize)
		rConn.Do("EXPIRE", hKey, 43200)
		rConn.Do("SET", HashUpIDKeyPrefix+filehash, upInfo.UploadID, "EX", 43200)
	}

	// 7. 将响应初始化的数据返回到客户端
	w.Write(util.NewRespMsg(0, "OK", upInfo).JSONBytes())

}

// UploadPartHandler: 上传分块接口
func UploadPartHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	uploadID := r.Form.Get("uploadid")
	chunkIndex := r.Form.Get("index")

	// 2. 或得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 获得文件句柄，用于存储分块内容
	fpath := "/data/" + uploadID + "/" + chunkIndex
	os.MkdirAll(path.Dir(fpath), 0744)
	fd, err := os.Create(fpath)
	if err != nil {
		w.Write(util.NewRespMsg(-1, "Upload part failed", nil).JSONBytes())
	}
	defer fd.Close()

	buf := make([]byte, 1024*1024)
	for {
		n, err := r.Body.Read(buf)
		fd.Write(buf[:n])
		if err != nil {
			break
		}
	}

	// 4. 更新redis缓存状态
	rConn.Do("HSET", "MP_"+uploadID, "chkidx_"+chunkIndex, 1)

	// 5. 返回处理结果到客户端
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}

// 通知上传合并
func CompleteUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析请求参数
	r.ParseForm()
	upid := r.Form.Get("uploadid")
	username := r.Form.Get("username")
	filehash := r.Form.Get("filehash")
	filesize := r.Form.Get("filesize")
	filename := r.Form.Get("filename")

	// 2. 获得redis连接池中的一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 通过uploadid查询redis并判断是否所有的分块上传完成
	data, err := redis.Values(rConn.Do("HGETALL", "MP_"+upid))
	if err != nil {
		w.Write(util.NewRespMsg(-1, "complete upload failed", nil).JSONBytes())
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
		w.Write(util.NewRespMsg(-2, "invalid request", nil).JSONBytes())
		return
	}

	// 4. TODO: 合并分块

	// 5. 更新唯一文件表及用户表
	fsize, _ := strconv.Atoi(filesize)
	dblayer.OnFileUploadFinished(filehash, filename, int64(fsize), "")
	dblayer.OnUserFileUploadFinished(username, filehash, filename, int64(fsize))

	// 6. 响应处理结果
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}

// CancelUploadHandler: 文件取消上传接口
func CancelUploadHandler(w http.ResponseWriter, r *http.Request) {
	// 1. 解析用户请求参数
	r.ParseForm()
	filehash := r.Form.Get("filehash")

	// 2. 获得redis一个连接
	rConn := rPool.RedisPool().Get()
	defer rConn.Close()

	// 3. 检查uploadID是否存在，如果存在则删除
	uploadID, err := redis.String(rConn.Do("GET", HashUpIDKeyPrefix+filehash))
	if err != nil || uploadID == "" {
		w.Write(util.NewRespMsg(-2, "Cancel upload part failed", nil).JSONBytes())
		return
	}

	_, delHashErr := rConn.Do("DEL", HashUpIDKeyPrefix+filehash)
	_, delUploadInfoErr := rConn.Do("DEL", ChunkKeyPrefix+uploadID)
	if delHashErr != nil || delUploadInfoErr != nil {
		w.Write(util.NewRespMsg(-2, "Cancel upload part failed", nil).JSONBytes())
		return
	}

	// 4. 删除已已上传的分块文件
	delChkRes := util.RemovePathByShell(ChunkDir + uploadID)
	if !delChkRes {
		fmt.Printf("Failed to delete chunks as upload canceled, uplloadID:%s\n", uploadID)
	}

	// 5. 响应客户端
	w.Write(util.NewRespMsg(0, "OK", nil).JSONBytes())
}
