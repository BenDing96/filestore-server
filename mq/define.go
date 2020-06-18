package mq

import "filestore-server/common"


// TransferData: 转移队列中消息载体的结构格式
type TransferData struct {
	FileHash string
	CurLocation string
	DestLocation string
	DestStoreType common.StoreType
}