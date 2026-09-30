package util

import (
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nexsoft-git/nexlogger/log"
)

var NatsTimeoutDuration = 5 * time.Second

func ConnectNatsJs(
	host string,
) (
	*nats.Conn,
	nats.JetStreamContext,
	error,
) {

	address := strings.Split(host, ":")
	if len(address) == 2 {
		port, _ := strconv.Atoi(address[1])
		if port == 0 {
			host = address[0]
		}
	}

	nc, err := ConnectNats(
		host,
	)

	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream()

	if err != nil {
		return nil, nil, err
	}

	return nc, js, nil
}

func ConnectNats(
	host string,
) (
	*nats.Conn,
	error,
) {

	nc, err := nats.Connect(
		host,
		nats.Timeout(NatsTimeoutDuration),
	)

	if err != nil {
		return nil, err
	}

	return nc, nil
}

func CreateStream(
	js nats.JetStreamContext,
	streamName string,
	subject []string,
) error {

	stream, err := js.StreamInfo(streamName)
	if err != nil {
		log.Printf("creating new stream : %s", streamName)
	}

	if stream == nil {
		stream, err = js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Retention: nats.WorkQueuePolicy,
			Subjects:  subject,
		})

		if err != nil {
			return err
		}
	}

	if stream.Config.Retention != nats.WorkQueuePolicy {

		err = js.DeleteStream(streamName)

		if err != nil {
			return err
		}

		_, err = js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Retention: nats.WorkQueuePolicy,
			Subjects:  subject,
		})

		if err != nil {
			return err
		}
	}

	return err
}
