package meta

import (
	mydb "filestore-server/db"
	"sort"
)

// FileMeta: 文件元信息结构
type FileMeta struct {
	FileSha1 string
	FileName string
	FileSize int64
	Location string
	UploadAt string
}

var fileMetas map[string]FileMeta

func init() {
	fileMetas = make(map[string]FileMeta)
}

// UpdateFileMeta: 新增/更新文件元信息
func UpdateFileMeta(fMeta FileMeta) {
	fileMetas[fMeta.FileSha1] = fMeta
}

// UpdateFileMetaDB: 新增/更新文件元信息到MySQL中
func UpdateFileMetaDB(fMeta FileMeta) bool {
	 return mydb.OnFileUploadFinished(
	 	fMeta.FileSha1, fMeta.FileName, fMeta.FileSize, fMeta.Location)
}

// GetFileMeta: 通过sha1值获取文件的元信息对象
func GetFileMeta(fileSha1 string) FileMeta {
	return fileMetas[fileSha1]
}

// GetFileMetaDB: 从MySQL获取元信息
func GetFileMetaDB(fileSha1 string) (FileMeta, error) {
	tfile, err := mydb.GetFileMeta(fileSha1)
	if err != nil {
		return FileMeta{}, err
	}
	fMeta := FileMeta{
		FileSha1: tfile.FileHash,
		FileName: tfile.FileName.String,
		FileSize: tfile.FileSize.Int64,
		Location: tfile.FileAddr.String,
	}
	return fMeta, nil
}

// GetLastFileMetas：获取批量文件元信息列表
func GetLastFileMetas(count int) []FileMeta {
	var fileMetaArray []FileMeta
	for _, v := range fileMetas {
		fileMetaArray = append(fileMetaArray, v)
	}
	sort.Sort(ByUploadTime(fileMetaArray))
	if count > len(fileMetaArray) {
		return fileMetaArray
	}
	return fileMetaArray[0:count]
}

// RemoveFileMeta: 删除
func RemoveFileMeta(fileSha1 string) {
	delete(fileMetas, fileSha1)
}