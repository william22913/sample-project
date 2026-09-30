package health

import (
	"fmt"

	"github.com/nexsoft-git/nexcommon/util"
)

type rabbitMQ struct {
	rabbitMQ *util.RabbitMQ
}

func (d *rabbitMQ) Ping() string {
	return fmt.Sprintf("%v", !d.rabbitMQ.Conn.IsClosed())
}

func NewRabbitMQConnChecker(
	conn *util.RabbitMQ,
) *rabbitMQ {
	return &rabbitMQ{
		rabbitMQ: conn,
	}
}
