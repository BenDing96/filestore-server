package db

import (
	"database/sql"
	mydb "filestore-server/db/mysql"
	"fmt"
)

// OnFileUploadFinished: 文件上传完成
func OnFileUploadFinished(
	filehash, filename string,
	filesize int64,
	fileaddr string) bool {
		statement, err := mydb.DBConn().Prepare(
			"insert ignore into tbl_file (`file_sha1`,`file_name`,`file_size`," +
				"`file_addr`,`status`) values (?,?,?,?,1)")
		if err != nil {
			fmt.Println("Failed to prepare statement, err: " + err.Error())
			return false
		}
		defer statement.Close()

		ret, err := statement.Exec(filehash, filename, filesize, fileaddr)

		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		if rf, err := ret.RowsAffected(); nil == err {
			if rf <= 0 {
				fmt.Printf("File with hash:%s has been uploaded before", filehash)
			}
			return true
		}
		return false
}

type TableFile struct {
	FileHash string
	FileName sql.NullString
	FileSize sql.NullInt64
	FileAddr sql.NullString

}

// GetFileMeta: 从MySQL获取元信息
func GetFileMeta(filehash string) (*TableFile, error) {
	statement, err := mydb.DBConn().Prepare(
		"select file_sha1,file_addr,file_name,file_size from tbl_file " +
			"where file_sha1=? and status=1 limit 1")
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	defer statement.Close()

	tfile := TableFile{}
	statement.QueryRow(filehash).Scan(
		&tfile.FileHash,&tfile.FileAddr,&tfile.FileName,&tfile.FileName)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return &tfile, nil
}