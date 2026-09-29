package model

import (
	"database/sql"
	"time"
)

type PushDataWithDelayedProcessingTime struct {
	Data            interface{}
	ProcessingAfter time.Time
}

func NewQueueProcessingError(isBlocker bool, msg string) QueueProcessingError {
	return QueueProcessingError{
		isBlockerError: isBlocker,
		message:        msg,
	}
}

func NewDoneProcessingWithMessage(msg string) QueueProcessingError {
	return QueueProcessingError{
		isDone:  true,
		message: msg,
	}
}

type QueueProcessingError struct {
	isBlockerError bool
	isDone         bool
	message        string
}

func (q QueueProcessingError) Error() string {
	return q.message
}

func (q QueueProcessingError) IsBlockerError() bool {
	return q.isBlockerError
}

func (q QueueProcessingError) IsDone() bool {
	return q.isDone
}

type DBQueueModel struct {
	ID                sql.NullInt64
	MessageID         sql.NullString
	IncomingMessageID sql.NullString
	UniqueIdentifier  sql.NullString
	AckTopic          sql.NullString
	QueueType         sql.NullString
	Payload           sql.NullString
	RetryCounter      sql.NullInt16
	MaxRetry          sql.NullInt16
	ProcessMessage    sql.NullString
	CreatedAt         sql.NullTime
	ProcessedAfter    sql.NullTime
	UpdatedAt         sql.NullTime
}

type NatsQueueMessage struct {
	MessageID string      `json:"message_id"`
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	AckTopic  string      `json:"ack_topics"`
}

type IncomingNatsMessageFromQueuePublisher struct {
	MessageID string      `json:"message_id"`
	Payload   interface{} `json:"payload"`
	AckTopic  string      `json:"ack_topic"`
}
