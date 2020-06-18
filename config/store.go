package config

import (
	cmn "filestore-server/common"
)

const (
	// TempLocalRootDir : 本地临时存储地址的路径
	TempLocalRootDir = "/data/fileserver_tmp/"
	// MergeLocalRootDir : 本地存储地址的路径(包含普通上传及分块上传)
	MergeLocalRootDir = "/data/fileserver_merge/"
	// ChunckLocalRootDir : 分块存储地址的路径
	ChunckLocalRootDir = "/data/fileserver_chunk/"
	// CurrentStoreType : 设置当前文件的存储类型
	CurrentStoreType = cmn.StoreOSS
)
