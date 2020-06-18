package mq

import (
	"filestore-server/config"
	"github.com/streadway/amqp"
	"log"
)
var conn *amqp.Connection
var channel *amqp.Channel

func initChannel() bool {
	// 1. 判断channel是否已经创建
	if channel != nil {
		return true
	}

	// 2. 获得一个rabbitmq的一个连接
	conn, err := amqp.Dial(config.RabbitURL)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	//
}