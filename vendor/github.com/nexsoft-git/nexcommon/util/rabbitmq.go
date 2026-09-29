package util

import (
	"fmt"
	"time"

	"github.com/nexsoft-git/nexlogger/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type rabbitMQConfig struct {
	url      string
	password string
	user     string
}

func NewRabbitMQConfig(host string, port int) (
	r *rabbitMQConfig,
) {
	result := rabbitMQConfig{}
	result.url = host
	if port > 0 {
		result.url = fmt.Sprintf("%s:%d", host, port)
	}

	return &result
}

func (r *rabbitMQConfig) User(user string) *rabbitMQConfig {
	r.user = user
	return r
}

func (r *rabbitMQConfig) Password(pass string) *rabbitMQConfig {
	r.password = pass
	return r
}

func (r *rabbitMQConfig) toURI() string {
	return fmt.Sprintf("amqp://%s:%s@%s/", r.user, r.password, r.url)
}

func ConnectRabbitMQ(
	config *rabbitMQConfig,
) (
	*RabbitMQ,
	error,
) {
	uri := config.toURI()
	conn, err := amqp.Dial(uri)

	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	notifyClose := conn.NotifyClose(make(chan *amqp.Error))
	reconnectionSuccess := make(chan struct{})

	mq := &RabbitMQ{
		uri:                 uri,
		Conn:                conn,
		Channel:             channel,
		notifyClose:         notifyClose,
		reconnectionSuccess: reconnectionSuccess,
	}

	go mq.reconnection()

	return mq, nil

}

type RabbitMQ struct {
	uri                 string
	Conn                *amqp.Connection
	Channel             *amqp.Channel
	state               []chan struct{}
	notifyClose         chan *amqp.Error
	reconnectionSuccess chan struct{}
}

func (r *RabbitMQ) reconnection() {
	var err error
	for {
		<-r.notifyClose

		log.Error().
			Err(err).
			Msg("Client RabbitMQ disconnected (Reconnecting)")

		for {
			r.Conn, err = amqp.Dial(r.uri)
			if err != nil {
				time.Sleep(time.Duration(5 * time.Second))
			} else {

				r.Channel, err = r.Conn.Channel()

				log.Info().
					Err(err).
					Msg("Reconnection to RabbitMQ Succesfully")

				r.notifyClose = r.Conn.NotifyClose(make(chan *amqp.Error))
				r.reconnectionSuccess <- struct{}{}
				break
			}
		}
	}

}
func NewConsumeQueueConfig(
	queueName string,
) *consumeQueueConfig {
	return &consumeQueueConfig{
		queue:     queueName,
		consumer:  "",
		autoAck:   true,
		exclusive: false,
		noLocal:   false,
		noWait:    false,
		args:      nil,
	}
}

type consumeQueueConfig struct {
	queue     string
	consumer  string
	autoAck   bool
	exclusive bool
	noLocal   bool
	noWait    bool
	args      amqp.Table
}

func (cc *consumeQueueConfig) Consumer(
	param string,
) *consumeQueueConfig {
	cc.consumer = param
	return cc
}

func (cc *consumeQueueConfig) AutoACK() *consumeQueueConfig {
	cc.autoAck = true
	return cc
}

func (cc *consumeQueueConfig) Exclusive() *consumeQueueConfig {
	cc.exclusive = true
	return cc
}

func (cc *consumeQueueConfig) NoLocal() *consumeQueueConfig {
	cc.noLocal = true
	return cc
}

func (cc *consumeQueueConfig) NoWait() *consumeQueueConfig {
	cc.noWait = true
	return cc
}

func (cc *consumeQueueConfig) Args(
	param amqp.Table,
) *consumeQueueConfig {
	cc.args = param
	return cc
}

func (r *RabbitMQ) Close() {
	_ = r.Channel.Close()
	_ = r.Conn.Close()

	for i := 0; i < len(r.state); i++ {
		r.state[i] <- struct{}{}
	}
}

func (r *RabbitMQ) createConnectionToMQ(
	cc *consumeQueueConfig,
) (
	<-chan amqp.Delivery,
	error,
) {

	return r.Channel.Consume(
		cc.queue,
		cc.consumer,
		cc.autoAck,
		cc.exclusive,
		cc.noLocal,
		cc.noWait,
		cc.args,
	)

}

func (r *RabbitMQ) ListenRabbitMQQueue(
	cc *consumeQueueConfig,
	doFunc func(amqp.Delivery),
) error {

	state := make(chan struct{})
	channel, err := r.createConnectionToMQ(cc)

	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case incoming := <-channel:
				if len(incoming.Body) == 0 {
					continue
				}

				doFunc(incoming)
			case <-r.reconnectionSuccess:
				for {
					channel, err = r.createConnectionToMQ(cc)
					if err == nil {
						break
					}
				}
			case <-state:
				return
			}
		}
	}()

	r.state = append(r.state, state)

	return nil

}
