package util

import (
	"time"

	"github.com/IBM/sarama"
	"github.com/nexsoft-git/nexlogger/log"
)

var maxRetryCount int = 3
var errCounter int = 0
var retryTime time.Duration = 1 * time.Second

func SetKafkaMaxRetryCount(x int) {
	maxRetryCount = x
}

func SetKafkaRetryDuration(x time.Duration) {
	retryTime = x
}

type Kafka struct {
	ConsumerGroup sarama.ConsumerGroup
}

var kafkaConnectListener chan (error)

func (k *Kafka) Ping() string {
	if k.ConsumerGroup == nil {
		return "DOWN"
	}

	errors := k.ConsumerGroup.Errors()
	if len(errors) > 0 {
		return "DOWN"
	}

	return "UP"
}

func DoNothingWhenKafkaError() {
	go func() {
		err := <-kafkaConnectListener
		close(kafkaConnectListener)
		if err != nil {
			log.Error().
				Err(err).
				Caller().
				Msg("Error Found when connect to Kafka")
		}
	}()
}

func BreakApplicationWhenKafkaError() {
	go func() {
		err := <-kafkaConnectListener
		close(kafkaConnectListener)
		if err != nil {
			log.Fatal().
				Err(err).
				Caller().
				Msg("Error Found when connect to Kafka")
		}
	}()
}

func NewKafkaConnection(
	host []string,
	group string,
	errorHandler func(),
) (
	kafkaConn *Kafka,
) {
	configKafka := sarama.NewConfig()
	configKafka.Consumer.Return.Errors = true
	kafkaConn = &Kafka{}

	go func() {
		for i := 0; i < maxRetryCount; i++ {
			kafkaConnectListener = make(chan error)
			errorHandler()

			conn, err := sarama.NewConsumerGroup(host, group, configKafka)
			kafkaConn.ConsumerGroup = conn
			kafkaConnectListener <- err

			if err != nil {
				time.Sleep(retryTime)
			} else {
				close(kafkaConnectListener)
				return
			}
		}

	}()

	return
}
