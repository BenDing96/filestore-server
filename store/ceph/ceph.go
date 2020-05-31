package ceph

import (
	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"
)

var cephConn *s3.S3

// GetCephPermission: 获取Ceph连接
func GetCephPermission() *s3.S3 {
	if cephConn != nil {
		return cephConn
	}
	// 1. 初始化ceph的信息
	auth := aws.Auth{
		AccessKey: "",
		SecretKey: "",
	}

	curRegion := aws.Region{
		Name:                 "default",
		EC2Endpoint:          "http://127.0.0.1:9080",
		S3Endpoint:           "http://127.0.0.1:9080",
		S3BucketEndpoint:     "",
		S3LocationConstraint: false,
		S3LowercaseBucket:    false,
		Sign:                 aws.SignV2,
	}

	// 2. 创建S3类型的连接
	return s3.New(auth, curRegion)
}

// GetCephConnection: 获取Ceph连接
func GetCephConnection(bucket string) *s3.Bucket {
	conn := GetCephPermission()
	return conn.Bucket(bucket)
}
