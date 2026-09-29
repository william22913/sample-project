package health

import (
	"github.com/nats-io/nats.go"
)

type natsConn struct {
	nats *nats.Conn
}

func (d *natsConn) Ping() string {
	return d.nats.Status().String()
}

func NewNatsConnChecker(
	nats *nats.Conn,
) *natsConn {
	return &natsConn{
		nats: nats,
	}
}
